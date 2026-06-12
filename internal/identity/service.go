package identity

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	repo   Repository
	tokens TokenService
	mail   WelcomeDispatcher
	logger *slog.Logger
}

func NewService(repo Repository, tokens TokenService, mail WelcomeDispatcher, logger *slog.Logger) *Service {
	return &Service{repo: repo, tokens: tokens, mail: mail, logger: logger}
}

func (s *Service) Register(ctx context.Context, username, email, password string) (*User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user := &User{
		Username:     strings.TrimSpace(username),
		Email:        strings.ToLower(strings.TrimSpace(email)),
		PasswordHash: string(hash),
	}
	if err := s.repo.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	if s.mail != nil {
		mailCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
		defer cancel()
		if err := s.mail.DispatchWelcome(mailCtx, user.Username, user.Email); err != nil {
			s.logger.Warn("welcome email dispatch failed", "user_id", user.ID, "error", err)
		}
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, email, password string) (*Tokens, error) {
	user, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)) != nil {
		return nil, ErrInvalidCredentials
	}
	return s.issueTokens(ctx, user.ID, "")
}

func (s *Service) Refresh(ctx context.Context, refreshToken string) (*Tokens, error) {
	claims, err := s.tokens.ValidateToken(refreshToken, TokenTypeRefresh)
	if err != nil {
		return nil, ErrInvalidToken
	}
	return s.issueTokens(ctx, claims.UserID, hashToken(refreshToken))
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if _, err := s.tokens.ValidateToken(refreshToken, TokenTypeRefresh); err != nil {
		return ErrInvalidToken
	}
	if err := s.repo.RevokeRefreshToken(ctx, hashToken(refreshToken)); err != nil {
		if errors.Is(err, ErrNotFound) {
			return ErrInvalidToken
		}
		return err
	}
	return nil
}

func (s *Service) Me(ctx context.Context, userID int) (*User, error) {
	user, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return nil, ErrUnauthorized
	}
	return user, nil
}

func (s *Service) CleanupRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	return s.repo.DeleteExpiredOrRevokedRefreshTokens(ctx, before)
}

func (s *Service) issueTokens(ctx context.Context, userID int, previousHash string) (*Tokens, error) {
	accessToken, accessExp, err := s.tokens.GenerateAccessToken(userID)
	if err != nil {
		return nil, err
	}
	refreshToken, refreshExp, err := s.tokens.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}
	replacement := RefreshToken{UserID: userID, TokenHash: hashToken(refreshToken), ExpiresAt: refreshExp}
	if previousHash == "" {
		err = s.repo.CreateRefreshToken(ctx, replacement)
	} else {
		err = s.repo.RotateRefreshToken(ctx, previousHash, replacement)
	}
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrInvalidToken
		}
		return nil, err
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
