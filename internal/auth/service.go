package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type UserStore interface {
	CreateUser(context.Context, *user.User) error
	GetUserByID(context.Context, int) (*user.User, error)
	GetUserByEmail(context.Context, string) (*user.User, error)
	GetUserByEmailOrPending(context.Context, string) (*user.User, error)
}

type RefreshTokenRepository interface {
	CreateRefreshToken(context.Context, *RefreshToken) error
	GetActiveRefreshTokenByHash(context.Context, string) (*RefreshToken, error)
	RevokeRefreshTokenByHash(context.Context, string) error
}

type VerificationRepository interface {
	ReplaceEmailVerificationToken(context.Context, *EmailVerificationToken) error
	VerifyEmail(context.Context, string, string, time.Time) (*EmailVerificationResult, error)
}

type TokenService interface {
	GenerateAccessToken(int, int) (string, time.Time, error)
	GenerateRefreshToken(int, int) (string, time.Time, error)
	ValidateRefreshToken(string) (Claims, error)
}

type PasswordHasher interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type VerificationNotifier interface {
	NotifyVerification(context.Context, user.User, string)
}

type WelcomeNotifier interface {
	NotifyWelcome(context.Context, user.User)
}

type Claims struct {
	UserID       int
	TokenVersion int
}

type Tokens struct {
	AccessToken      string
	AccessExpiresAt  time.Time
	RefreshToken     string
	RefreshExpiresAt time.Time
}

type Service struct {
	users           UserStore
	refresh         RefreshTokenRepository
	verification    VerificationRepository
	tokens          TokenService
	password        PasswordHasher
	notifier        VerificationNotifier
	welcome         WelcomeNotifier
	verificationTTL time.Duration
	bootstrapEmail  string
}

func NewService(
	users UserStore,
	refresh RefreshTokenRepository,
	verification VerificationRepository,
	tokens TokenService,
	password PasswordHasher,
	notifier VerificationNotifier,
	welcome WelcomeNotifier,
	verificationTTL time.Duration,
	bootstrapEmail string,
) *Service {
	return &Service{
		users: users, refresh: refresh, verification: verification, tokens: tokens, password: password,
		notifier: notifier, welcome: welcome, verificationTTL: verificationTTL, bootstrapEmail: user.NormalizeEmail(bootstrapEmail),
	}
}

func (s *Service) Register(ctx context.Context, account *user.User) error {
	if err := user.NormalizeAndValidate(account, true); err != nil {
		return err
	}
	hashedPassword, err := s.password.Hash(account.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	account.Password = hashedPassword
	account.Role = user.RoleUser
	account.TokenVersion = 1
	if err := s.users.CreateUser(ctx, account); err != nil {
		return fmt.Errorf("create user: %w", err)
	}
	return s.sendVerification(ctx, account)
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	account, err := s.users.GetUserByEmailOrPending(ctx, user.NormalizeEmail(email))
	if errors.Is(err, user.ErrNotFound) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("find user for verification: %w", err)
	}
	if account == nil {
		return nil
	}
	if account.EmailVerified() && account.PendingEmail == "" {
		return nil
	}
	return s.sendVerification(ctx, account)
}

func (s *Service) SendVerificationForUser(ctx context.Context, userID int) error {
	account, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if account.EmailVerified() && account.PendingEmail == "" {
		return nil
	}
	return s.sendVerification(ctx, account)
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	if token == "" {
		return ErrInvalidToken
	}
	result, err := s.verification.VerifyEmail(ctx, hashToken(token), s.bootstrapEmail, time.Now())
	if err != nil {
		if errors.Is(err, ErrInvalidToken) {
			return ErrInvalidToken
		}
		return fmt.Errorf("verify email: %w", err)
	}
	if result != nil && result.FirstVerification && result.User != nil && s.welcome != nil {
		s.welcome.NotifyWelcome(ctx, *result.User)
	}
	return nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*Tokens, error) {
	account, err := s.users.GetUserByEmail(ctx, user.NormalizeEmail(email))
	if errors.Is(err, user.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user for login: %w", err)
	}
	if account == nil {
		return nil, ErrInvalidCredentials
	}
	if s.password.Compare(account.Password, password) != nil {
		return nil, ErrInvalidCredentials
	}
	if !account.EmailVerified() {
		return nil, ErrEmailUnverified
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
	if err != nil || storedToken == nil || storedToken.UserID != claims.UserID {
		return nil, ErrUnauthorized
	}
	account, err := s.users.GetUserByID(ctx, claims.UserID)
	if err != nil || account == nil || account.TokenVersion != claims.TokenVersion || !account.EmailVerified() {
		return nil, ErrUnauthorized
	}
	if err := s.refresh.RevokeRefreshTokenByHash(ctx, tokenHash); err != nil {
		return nil, fmt.Errorf("revoke refresh token: %w", err)
	}
	return s.issueTokens(ctx, account)
}

func (s *Service) Me(ctx context.Context, userID int) (*user.User, error) {
	account, err := s.users.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	if account == nil {
		return nil, ErrUnauthorized
	}
	return account, nil
}

func (s *Service) sendVerification(ctx context.Context, account *user.User) error {
	token, err := randomToken()
	if err != nil {
		return fmt.Errorf("generate verification token: %w", err)
	}
	if err := s.verification.ReplaceEmailVerificationToken(ctx, &EmailVerificationToken{
		UserID: account.ID, TokenHash: hashToken(token), ExpiresAt: time.Now().Add(s.verificationTTL),
	}); err != nil {
		return fmt.Errorf("store verification token: %w", err)
	}
	if s.notifier != nil {
		recipient := *account
		if recipient.PendingEmail != "" {
			recipient.Email = recipient.PendingEmail
		}
		s.notifier.NotifyVerification(ctx, recipient, token)
	}
	return nil
}

func (s *Service) issueTokens(ctx context.Context, account *user.User) (*Tokens, error) {
	accessToken, accessExp, err := s.tokens.GenerateAccessToken(account.ID, account.TokenVersion)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	refreshToken, refreshExp, err := s.tokens.GenerateRefreshToken(account.ID, account.TokenVersion)
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

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func hashToken(token string) string {
	hashed := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hashed[:])
}
