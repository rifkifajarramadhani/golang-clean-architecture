package mysql

import (
	"context"
	"errors"
	"time"

	mysqlDriver "github.com/go-sql-driver/mysql"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
	"gorm.io/gorm"
)

type userModel struct {
	ID       int `gorm:"primaryKey"`
	Username string
	Email    string
	Password string
}

func (userModel) TableName() string { return "users" }

type refreshTokenModel struct {
	ID        int `gorm:"primaryKey"`
	UserID    int
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (refreshTokenModel) TableName() string { return "refresh_tokens" }

type Repository struct{ db *gorm.DB }

func NewRepository(db *gorm.DB) *Repository { return &Repository{db: db} }

func (r *Repository) CreateUser(ctx context.Context, user *identity.User) error {
	model := userModel{Username: user.Username, Email: user.Email, Password: user.PasswordHash}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		var mysqlErr *mysqlDriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			if mysqlErr.Message != "" && contains(mysqlErr.Message, "email") {
				return identity.ErrDuplicateEmail
			}
			return identity.ErrDuplicateUsername
		}
		return err
	}
	user.ID = model.ID
	return nil
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (*identity.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&model).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return toUser(model), nil
}

func (r *Repository) GetUserByID(ctx context.Context, id int) (*identity.User, error) {
	var model userModel
	if err := r.db.WithContext(ctx).First(&model, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return toUser(model), nil
}

func (r *Repository) CreateRefreshToken(ctx context.Context, token identity.RefreshToken) error {
	return r.db.WithContext(ctx).Create(&refreshTokenModel{
		UserID: token.UserID, TokenHash: token.TokenHash, ExpiresAt: token.ExpiresAt,
	}).Error
}

func (r *Repository) RotateRefreshToken(ctx context.Context, oldHash string, replacement identity.RefreshToken) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		now := time.Now()
		result := tx.Model(&refreshTokenModel{}).
			Where("token_hash = ? AND user_id = ? AND revoked_at IS NULL AND expires_at > ?", oldHash, replacement.UserID, now).
			Update("revoked_at", now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected != 1 {
			return identity.ErrNotFound
		}
		return tx.Create(&refreshTokenModel{
			UserID: replacement.UserID, TokenHash: replacement.TokenHash, ExpiresAt: replacement.ExpiresAt,
		}).Error
	})
}

func (r *Repository) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	result := r.db.WithContext(ctx).Model(&refreshTokenModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return identity.ErrNotFound
	}
	return nil
}

func (r *Repository) DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before).
		Delete(&refreshTokenModel{})
	return result.RowsAffected, result.Error
}

func toUser(model userModel) *identity.User {
	return &identity.User{ID: model.ID, Username: model.Username, Email: model.Email, PasswordHash: model.Password}
}

func mapNotFound(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return identity.ErrNotFound
	}
	return err
}

func contains(value, part string) bool {
	for i := 0; i+len(part) <= len(value); i++ {
		if value[i:i+len(part)] == part {
			return true
		}
	}
	return false
}
