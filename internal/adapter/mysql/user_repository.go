package mysqladapter

import (
	"context"
	"errors"
	"strings"
	"time"

	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, account *user.User) error {
	return r.createUser(ctx, account)
}

// RegisterUser runs dispatch inside the transaction so a user row never exists
// without its verification having been queued: a dispatch failure rolls the
// registration back. The cost is that row locks are held for the duration of
// the dispatch call, so dispatchers must stay fast and honor ctx deadlines.
func (r *UserRepository) RegisterUser(
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

func (r *UserRepository) createUser(ctx context.Context, account *user.User) error {
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
		return nil
	})
}

func (r *UserRepository) List(ctx context.Context, page, limit int) ([]*user.User, int64, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&userModel{}).Count(&count).Error; err != nil {
		return nil, 0, err
	}
	var records []userModel
	if err := r.db.WithContext(ctx).Order("id ASC").Offset((page - 1) * limit).Limit(limit).Find(&records).Error; err != nil {
		return nil, 0, err
	}
	accounts := make([]*user.User, 0, len(records))
	for _, record := range records {
		accounts = append(accounts, toUser(record))
	}
	return accounts, count, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*user.User, error) {
	return r.GetUserByID(ctx, id)
}

func (r *UserRepository) GetUserByID(ctx context.Context, id int) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&record).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) GetUserByEmailOrPending(ctx context.Context, email string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("email = ? OR pending_email = ?", email, email).First(&record).Error; err != nil {
		return nil, mapUserNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) CountAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&userModel{}).Where("role = ?", user.RoleAdmin).Count(&count).Error
	return count, err
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id int, username, email string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var record userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&record, id).Error; err != nil {
			return mapUserNotFound(err)
		}
		updates := map[string]any{"username": username}
		if email == record.Email {
			updates["pending_email"] = ""
		} else {
			var count int64
			if err := tx.Model(&userModel{}).Where("(email = ? OR pending_email = ?) AND id <> ?", email, email, id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return user.ErrDuplicateEmail
			}
			updates["pending_email"] = email
		}
		return mapWriteError(tx.Model(&record).Updates(updates).Error)
	})
}

func (r *UserRepository) ChangePassword(ctx context.Context, id int, hashedPassword string) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		result := tx.Model(&userModel{}).Where("id = ?", id).Updates(map[string]any{
			"password": hashedPassword, "token_version": gorm.Expr("token_version + 1"),
		})
		if err := requireAffected(result); err != nil {
			return err
		}
		return tx.Model(&refreshTokenModel{}).Where("user_id = ? AND revoked_at IS NULL", id).Update("revoked_at", time.Now()).Error
	})
}

// ChangeRole re-checks the admin-actor, promote-requires-verified, and
// last-admin invariants that user.Service already validated: the service
// checks are advisory (fast failure, business rules in core), while these run
// under FOR UPDATE locks and are authoritative against concurrent changes.
func (r *UserRepository) ChangeRole(ctx context.Context, actorID, targetID int, role user.Role) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var actor, target userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&actor, actorID).Error; err != nil {
			return mapUserNotFound(err)
		}
		if actor.Role != user.RoleAdmin {
			return user.ErrForbidden
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&target, targetID).Error; err != nil {
			return mapUserNotFound(err)
		}
		if role == user.RoleAdmin && target.EmailVerifiedAt == nil {
			return user.ErrForbidden
		}
		if target.Role == user.RoleAdmin && role != user.RoleAdmin {
			var admins []userModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role = ?", user.RoleAdmin).Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return user.ErrLastAdmin
			}
		}
		if err := tx.Model(&target).Updates(map[string]any{
			"role": role, "token_version": gorm.Expr("token_version + 1"),
		}).Error; err != nil {
			return err
		}
		return tx.Model(&refreshTokenModel{}).Where("user_id = ? AND revoked_at IS NULL", targetID).
			Update("revoked_at", time.Now()).Error
	})
}

// Delete re-checks the last-admin invariant under FOR UPDATE locks; see
// ChangeRole for why the user.Service pre-check is repeated here.
func (r *UserRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var target userModel
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&target, id).Error; err != nil {
			return mapUserNotFound(err)
		}
		if target.Role == user.RoleAdmin {
			var admins []userModel
			if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("role = ?", user.RoleAdmin).Find(&admins).Error; err != nil {
				return err
			}
			if len(admins) <= 1 {
				return user.ErrLastAdmin
			}
		}
		return requireAffected(tx.Delete(&target))
	})
}

func (r *UserRepository) CreateRefreshToken(ctx context.Context, token *auth.RefreshToken) error {
	record := refreshTokenModel{UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt}
	if err := r.db.WithContext(ctx).Create(&record).Error; err != nil {
		return err
	}
	token.ID = record.ID
	return nil
}

func (r *UserRepository) GetActiveRefreshTokenByHash(ctx context.Context, tokenHash string) (*auth.RefreshToken, error) {
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

func (r *UserRepository) RevokeRefreshTokenByHash(ctx context.Context, tokenHash string) error {
	return requireAffected(r.db.WithContext(ctx).Model(&refreshTokenModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", time.Now()))
}

func (r *UserRepository) RotateRefreshToken(ctx context.Context, currentHash string, replacement *auth.RefreshToken) error {
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

func (r *UserRepository) ReplaceEmailVerificationToken(ctx context.Context, token *auth.EmailVerificationToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", token.UserID).Delete(&emailVerificationTokenModel{}).Error; err != nil {
			return err
		}
		return tx.Create(&emailVerificationTokenModel{
			UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
		}).Error
	})
}

func (r *UserRepository) CanIssueEmailVerification(ctx context.Context, userID int, issuedAfter time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&emailVerificationTokenModel{}).
		Where("user_id = ? AND created_at > ?", userID, issuedAfter).Count(&count).Error
	return count == 0, err
}

func (r *UserRepository) VerifyEmail(
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

func (r *UserRepository) DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before).
		Delete(&refreshTokenModel{})
	return result.RowsAffected, result.Error
}

func fromUser(account *user.User) userModel {
	return userModel{
		ID: account.ID, Username: account.Username, Email: account.Email, Password: account.Password, Role: account.Role,
		EmailVerifiedAt: account.EmailVerifiedAt, PendingEmail: account.PendingEmail, TokenVersion: account.TokenVersion,
	}
}

func toUser(record userModel) *user.User {
	return &user.User{
		ID: record.ID, Username: record.Username, Email: record.Email, Password: record.Password, Role: record.Role,
		EmailVerifiedAt: record.EmailVerifiedAt, PendingEmail: record.PendingEmail, TokenVersion: record.TokenVersion,
	}
}

func mapUserNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return user.ErrNotFound
	}
	return err
}

func mapWriteError(err error) error {
	var mysqlErr *mysqldriver.MySQLError
	if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
		if strings.Contains(mysqlErr.Message, "username") {
			return user.ErrDuplicateUsername
		}
		return user.ErrDuplicateEmail
	}
	return err
}

func requireAffected(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return user.ErrNotFound
	}
	return nil
}

var (
	_ auth.UserStore              = (*UserRepository)(nil)
	_ auth.RefreshTokenRepository = (*UserRepository)(nil)
	_ auth.VerificationRepository = (*UserRepository)(nil)
	_ auth.MaintenanceRepository  = (*UserRepository)(nil)
	_ user.Repository             = (*UserRepository)(nil)
)
