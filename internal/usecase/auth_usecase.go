package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
)

type AuthRepository interface {
	Create(user *domain.User) error
	GetByEmail(email string) (*domain.User, error)
	GetByUsername(username string) (*domain.User, error)
	EmailExists(email string) (bool, error)
	UsernameExists(username string) (bool, error)
	CreateRefreshToken(token *domain.RefreshToken) error
	GetActiveRefreshTokenByHash(tokenHash string) (*domain.RefreshToken, error)
	RevokeRefreshTokenByHash(tokenHash string) error
}

type AuthUsecase struct {
	repo         AuthRepository
	tokenService *security.JWTService
}

type AuthTokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

func NewAuthUsecase(repo AuthRepository, tokenService *security.JWTService) *AuthUsecase {
	return &AuthUsecase{repo: repo, tokenService: tokenService}
}

func (u *AuthUsecase) Register(user *domain.User) error {
	emailExists, err := u.repo.EmailExists(user.Email)
	if err != nil {
		return err
	}
	if emailExists {
		return ErrDuplicateEmail
	}

	usernameExists, err := u.repo.UsernameExists(user.Username)
	if err != nil {
		return err
	}
	if usernameExists {
		return ErrDuplicateUsername
	}

	hashedPassword, err := hashPassword(user.Password)
	if err != nil {
		return err
	}
	user.Password = hashedPassword

	return u.repo.Create(user)
}

func (u *AuthUsecase) Login(email, password string) (*AuthTokens, error) {
	user, err := u.repo.GetByEmail(email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if err := comparePassword(user.Password, password); err != nil {
		return nil, ErrInvalidCredentials
	}

	return u.issueTokens(user)
}

func (u *AuthUsecase) Refresh(refreshToken string) (*AuthTokens, error) {
	claims, err := u.tokenService.ValidateToken(refreshToken, security.TokenTypeRefresh)
	if err != nil {
		if errors.Is(err, security.ErrExpiredToken) {
			return nil, ErrUnauthorized
		}
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(refreshToken)
	storedToken, err := u.repo.GetActiveRefreshTokenByHash(tokenHash)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if storedToken.UserID != claims.UserID {
		return nil, ErrUnauthorized
	}

	if err := u.repo.RevokeRefreshTokenByHash(tokenHash); err != nil {
		return nil, err
	}

	user, err := u.repo.GetByUsername(claims.Subject)
	if err != nil {
		return nil, ErrUnauthorized
	}

	return u.issueTokens(user)
}

func (u *AuthUsecase) Me(username string) (*domain.User, error) {
	user, err := u.repo.GetByUsername(username)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (u *AuthUsecase) issueTokens(user *domain.User) (*AuthTokens, error) {
	accessToken, accessExp, err := u.tokenService.GenerateAccessToken(user.Username, user.ID)
	if err != nil {
		return nil, err
	}

	refreshToken, refreshExp, err := u.tokenService.GenerateRefreshToken(user.Username, user.ID)
	if err != nil {
		return nil, err
	}

	if err := u.repo.CreateRefreshToken(&domain.RefreshToken{
		UserID:    user.ID,
		TokenHash: hashToken(refreshToken),
		ExpiresAt: refreshExp,
	}); err != nil {
		return nil, err
	}

	return &AuthTokens{
		AccessToken:      accessToken,
		AccessExpiresAt:  accessExp,
		RefreshToken:     refreshToken,
		RefreshExpiresAt: refreshExp,
	}, nil
}

func hashToken(token string) string {
	hashed := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashed[:])
}
