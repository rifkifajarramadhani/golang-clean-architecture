package identity

import (
	"context"
	"errors"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// RegisterUser runs dispatch inside the transaction so a user row never exists
// without its verification having been queued: a dispatch failure rolls the
// registration back. The cost is that row locks are held for the duration of
// the dispatch call, so dispatchers must stay fast and honor ctx deadlines.
func (r *Repository) RegisterUser(
	ctx context.Context,
	account *user.User,
	token *auth.EmailVerificationToken,
	dispatch func() error,
) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var pending int64
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Model(&userModel{}).
			Where("pending_email = ?", account.Email).Count(&pending).Error; err != nil {
			return err
		}
		if pending > 0 {
			return user.ErrDuplicateEmail
		}
		model := fromUser(account)
		if err := tx.Create(&model).Error; err != nil {
			return mapWriteError(err)
		}
		account.ID = model.ID
		token.UserID = model.ID
		if err := tx.Create(&emailVerificationTokenModel{
			UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
		}).Error; err != nil {
			return err
		}
		if dispatch != nil {
			if err := dispatch(); err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	record := refreshTokenModel{UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	token.ID = record.ID
	return nil
}

func (r *Repository) GetActiveRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
	var record refreshTokenModel
	err := r.db.WithContext(ctx).Where(
		"token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now(),
	).First(&record).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &auth.RefreshToken{
		ID: record.ID, UserID: record.UserID, TokenHash: record.TokenHash, ExpiresAt: record.ExpiresAt,
		RevokedAt: record.RevokedAt, CreatedAt: record.CreatedAt, UpdatedAt: record.UpdatedAt,
	}, nil
}

func (r *Repository) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return requireAffected(r.db.WithContext(ctx).Model(&refreshTokenModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", time.Now()))
}

func (r *Repository) RotateRefreshToken(ctx context.Context, currentHash string, replacement *auth.RefreshToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&refreshTokenModel{}).
			Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", currentHash, now).
			Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return auth.ErrUnauthorized
		}
		record := refreshTokenModel{
			UserID: replacement.UserID, TokenHash: replacement.TokenHash, ExpiresAt: replacement.ExpiresAt,
		}
		if err := tx.Create(&record).Error; err != nil {
			return err
		}
		replacement.ID = record.ID
		return nil
	})
}

func (r *Repository) ReplaceEmailVerificationToken(ctx context.Context, token *auth.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", token.UserID).Delete(&emailVerificationTokenModel{}).Error; err != nil {
			return err
		}
		return tx.Create(&emailVerificationTokenModel{
			UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
		}).Error
	})
}

func (r *Repository) CanIssueEmailVerification(ctx context.Context, userID int, issuedAfter time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&emailVerificationTokenModel{}).
		Where("user_id = ? AND created_at > ?", userID, issuedAfter).Count(&count).Error
	return count == 0, err
}

func (r *Repository) VerifyEmail(
	ctx context.Context,
	tokenHash string,
	now time.Time,
	rolePolicy auth.VerificationRolePolicy,
) (*auth.EmailVerificationResult, error) {
	var result *auth.EmailVerificationResult
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var token emailVerificationTokenModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("token_hash = ? AND expires_at > ?", tokenHash, now).First(&token).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return auth.ErrInvalidToken
			}
			return err
		}
		var record userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, token.UserID).Error; err != nil {
			return mapUserNotFound(err)
		}
		account := toUser(record)
		firstVerification := account.VerifyEmail(now)
		var admins int64
		if err := tx.Model(&userModel{}).Where("role = ?", user.RoleAdmin).Count(&admins).Error; err != nil {
			return err
		}
		if rolePolicy != nil {
			role, err := rolePolicy(*account, admins)
			if err != nil {
				return err
			}
			if err := account.AssignRole(role); err != nil {
				return err
			}
		}
		updates := map[string]any{
			"email": account.Email, "pending_email": account.PendingEmail, "email_verified_at": account.EmailVerifiedAt,
			"role": account.Role, "token_version": account.TokenVersion,
		}
		if err := mapWriteError(tx.Model(&record).Updates(updates).Error); err != nil {
			return err
		}
		if err := tx.Where("user_id = ?", record.ID).Delete(&emailVerificationTokenModel{}).Error; err != nil {
			return err
		}
		if err := tx.Model(&refreshTokenModel{}).Where("user_id = ? AND revoked_at IS NULL", record.ID).
			Update("revoked_at", now).Error; err != nil {
			return err
		}
		if err := tx.First(&record, record.ID).Error; err != nil {
			return err
		}
		result = &auth.EmailVerificationResult{User: toUser(record), FirstVerification: firstVerification}
		return nil
	})
	return result, err
}

func (r *Repository) DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before).
		Delete(&refreshTokenModel{})
	return result.RowsAffected, result.Error
}
