package mysqladapter

import (
	"context"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
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

func (r *UserRepository) CreateUser(ctx context.Context, account *user.User) error {
	return r.createUser(ctx, account)
}

func (r *UserRepository) createUser(ctx context.Context, account *user.User) error {
	model := userModel{Username: account.Username, Email: account.Email, Password: account.Password}
	if err := r.db.WithContext(ctx).Create(&model).Error; err != nil {
		return err
	}
	account.ID = model.ID
	return nil
}

func (r *UserRepository) GetAll(ctx context.Context) ([]*user.User, error) {
	var records []userModel
	if err := r.db.WithContext(ctx).Find(&records).Error; err != nil {
		return nil, err
	}
	accounts := make([]*user.User, 0, len(records))
	for _, record := range records {
		accounts = append(accounts, toUser(record))
	}
	return accounts, nil
}

func (r *UserRepository) GetByID(ctx context.Context, id int) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).First(&record, id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&record).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*user.User, error) {
	var record userModel
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&record).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return toUser(record), nil
}

func (r *UserRepository) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&userModel{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&userModel{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *UserRepository) Update(ctx context.Context, account *user.User) error {
	updates := map[string]any{"username": account.Username, "email": account.Email}
	if account.Password != "" {
		updates["password"] = account.Password
	}
	return r.db.WithContext(ctx).Model(&userModel{}).Where("id = ?", account.ID).Updates(updates).Error
}

func (r *UserRepository) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&userModel{}, id).Error
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
	result := r.db.WithContext(ctx).Model(&refreshTokenModel{}).
		Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", time.Now())
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepository) DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("expires_at < ? OR (revoked_at IS NOT NULL AND revoked_at < ?)", before, before).
		Delete(&refreshTokenModel{})
	return result.RowsAffected, result.Error
}

func toUser(record userModel) *user.User {
	return &user.User{ID: record.ID, Username: record.Username, Email: record.Email, Password: record.Password}
}

var (
	_ auth.Repository            = (*UserRepository)(nil)
	_ auth.MaintenanceRepository = (*UserRepository)(nil)
	_ user.Repository            = (*UserRepository)(nil)
)
