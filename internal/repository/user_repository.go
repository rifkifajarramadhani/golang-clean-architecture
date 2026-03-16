package repository

import (
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/models"
	"gorm.io/gorm"
)

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *userRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(user *domain.User) error {
	model := models.User{
		Username: user.Username,
		Email:    user.Email,
		Password: user.Password,
	}

	if err := r.db.Create(&model).Error; err != nil {
		return err
	}

	user.ID = model.ID

	return nil
}

func (r *userRepository) GetAll() ([]*domain.User, error) {
	var modelsUsers []models.User
	if err := r.db.Find(&modelsUsers).Error; err != nil {
		return nil, err
	}

	users := make([]*domain.User, 0, len(modelsUsers))
	for _, userModel := range modelsUsers {
		users = append(users, &domain.User{
			ID:       userModel.ID,
			Username: userModel.Username,
			Email:    userModel.Email,
			Password: userModel.Password,
		})
	}

	return users, nil
}

func (r *userRepository) GetByID(id int) (*domain.User, error) {
	var model models.User
	if err := r.db.First(&model, id).Error; err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       model.ID,
		Username: model.Username,
		Email:    model.Email,
		Password: model.Password,
	}, nil
}

func (r *userRepository) GetByEmail(email string) (*domain.User, error) {
	var model models.User
	if err := r.db.Where("email = ?", email).First(&model).Error; err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       model.ID,
		Username: model.Username,
		Email:    model.Email,
		Password: model.Password,
	}, nil
}

func (r *userRepository) GetByUsername(username string) (*domain.User, error) {
	var model models.User
	if err := r.db.Where("username = ?", username).First(&model).Error; err != nil {
		return nil, err
	}

	return &domain.User{
		ID:       model.ID,
		Username: model.Username,
		Email:    model.Email,
		Password: model.Password,
	}, nil
}

func (r *userRepository) EmailExists(email string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Where("email = ?", email).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *userRepository) UsernameExists(username string) (bool, error) {
	var count int64
	if err := r.db.Model(&models.User{}).Where("username = ?", username).Count(&count).Error; err != nil {
		return false, err
	}

	return count > 0, nil
}

func (r *userRepository) Update(user *domain.User) error {
	updates := map[string]interface{}{
		"username": user.Username,
		"email":    user.Email,
	}
	if user.Password != "" {
		updates["password"] = user.Password
	}

	return r.db.Model(&models.User{}).Where("id = ?", user.ID).Updates(updates).Error
}

func (r *userRepository) Delete(id int) error {
	return r.db.Delete(&models.User{}, id).Error
}

func (r *userRepository) CreateRefreshToken(token *domain.RefreshToken) error {
	model := models.RefreshToken{
		UserID:    token.UserID,
		TokenHash: token.TokenHash,
		ExpiresAt: token.ExpiresAt,
	}
	if err := r.db.Create(&model).Error; err != nil {
		return err
	}
	token.ID = model.ID
	return nil
}

func (r *userRepository) GetActiveRefreshTokenByHash(tokenHash string) (*domain.RefreshToken, error) {
	var model models.RefreshToken
	if err := r.db.Where("token_hash = ? AND revoked_at IS NULL AND expires_at > ?", tokenHash, time.Now()).First(&model).Error; err != nil {
		return nil, err
	}

	return &domain.RefreshToken{
		ID:        model.ID,
		UserID:    model.UserID,
		TokenHash: model.TokenHash,
		ExpiresAt: model.ExpiresAt,
		RevokedAt: model.RevokedAt,
		CreatedAt: model.CreatedAt,
		UpdatedAt: model.UpdatedAt,
	}, nil
}

func (r *userRepository) RevokeRefreshTokenByHash(tokenHash string) error {
	now := time.Now()
	result := r.db.Model(&models.RefreshToken{}).Where("token_hash = ? AND revoked_at IS NULL", tokenHash).Update("revoked_at", now)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
