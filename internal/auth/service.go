package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type UserStore interface {
	CreateUser(context.Context, *user.User) error
	GetUserByEmail(context.Context, string) (*user.User, error)
	GetUserByUsername(context.Context, string) (*user.User, error)
	EmailExists(context.Context, string) (bool, error)
	UsernameExists(context.Context, string) (bool, error)
}

type RefreshTokenRepository interface {
	CreateRefreshToken(context.Context, *RefreshToken) error
	GetActiveRefreshTokenByHash(context.Context, string) (*RefreshToken, error)
	RevokeRefreshTokenByHash(context.Context, string) error
}

type TokenService interface {
	GenerateAccessToken(string, int) (string, time.Time, error)
	GenerateRefreshToken(string, int) (string, time.Time, error)
	ValidateRefreshToken(string) (Claims, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type WelcomeNotifier interface {
	NotifyWelcome(context.Context, user.User)
}

type Claims struct {
	Subject string
	UserID  int
}

type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type Service struct {
	users    UserStore
	refresh  RefreshTokenRepository
	tokens   TokenService
	password PasswordHasher
	welcome  WelcomeNotifier
}

func NewService(
	users UserStore,
	refresh RefreshTokenRepository,
	tokens TokenService,
	password PasswordHasher,
	welcome WelcomeNotifier,
) *Service {
	return &Service{users: users, refresh: refresh, tokens: tokens, password: password, welcome: welcome}
}

func (s *Service) Register(ctx context.Context, account *user.User) error {
	emailExists, err := s.users.EmailExists(ctx, account.Email)
	if err != nil {
		return fmt.Errorf("check email: %w", err)
	}
	if emailExists {
		return ErrDuplicateEmail
	}

	usernameExists, err := s.users.UsernameExists(ctx, account.Username)
	if err != nil {
		return fmt.Errorf("check username: %w", err)
	}
	if usernameExists {
		return ErrDuplicateUsername
	}

	hashedPassword, err := s.password.Hash(account.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	account.Password = hashedPassword

	if err := s.users.CreateUser(ctx, account); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	if s.welcome != nil {
		s.welcome.NotifyWelcome(ctx, *account)
	}
	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*Tokens, error) {
	account, err := s.users.GetUserByEmail(ctx, email)
	if err != nil || account == nil {
		return nil, ErrInvalidCredentials
	}
	if s.password.Compare(account.Password, password) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, account)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	claims, err := s.tokens.ValidateRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, ErrExpiredToken) {
			return nil, ErrUnauthorized
		}
		return nil, ErrInvalidToken
	}

	tokenHash := hashToken(refreshToken)
	storedToken, err := s.refresh.GetActiveRefreshTokenByHash(ctx, tokenHash)
	if err != nil || storedToken.UserID != claims.UserID {
		return nil, ErrUnauthorized
	}
	if err := s.refresh.RevokeRefreshTokenByHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}

	account, err := s.users.GetUserByUsername(ctx, claims.Subject)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return s.issueTokens(ctx, account)
}

func (s *Service) Me(ctx context.Context, username string) (*user.User, error) {
	account, err := s.users.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return account, nil
}

func (s *Service) issueTokens(ctx context.Context, account *user.User) (*Tokens, error) {
	accessToken, accessExp, err := s.tokens.GenerateAccessToken(account.Username, account.ID)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, refreshExp, err := s.tokens.GenerateRefreshToken(account.Username, account.ID)
	if err != nil {
		return nil, fmt.Errorf("generate refresh token: %w", err)
	}
	if err := s.refresh.CreateRefreshToken(ctx, &RefreshToken{
		UserID: account.ID, TokenHash: hashToken(refreshToken), ExpiresAt: refreshExp,
	}); err != nil {
		return nil, fmt.Errorf("store refresh token: %w", err)
	}
	return &Tokens{
		AccessToken: accessToken, AccessExpiresAt: accessExp,
		RefreshToken: refreshToken, RefreshExpiresAt: refreshExp,
	}, nil
}

func hashToken(token string) string {
	hashed := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashed[:])
}
