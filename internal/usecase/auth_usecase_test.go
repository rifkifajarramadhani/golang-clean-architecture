package usecase

import (
	"errors"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/infrastructure/security"
)

type authRepoMock struct {
	usersByEmail    map[string]*domain.User
	usersByUsername map[string]*domain.User
	refreshTokens   map[string]*domain.RefreshToken
	nextUserID      int
}

func newAuthRepoMock() *authRepoMock {
	return &authRepoMock{
		usersByEmail:    map[string]*domain.User{},
		usersByUsername: map[string]*domain.User{},
		refreshTokens:   map[string]*domain.RefreshToken{},
		nextUserID:      1,
	}
}

func (m *authRepoMock) Create(user *domain.User) error {
	user.ID = m.nextUserID
	m.nextUserID++
	cloned := *user
	m.usersByEmail[user.Email] = &cloned
	m.usersByUsername[user.Username] = &cloned
	return nil
}

func (m *authRepoMock) GetByEmail(email string) (*domain.User, error) {
	user, ok := m.usersByEmail[email]
	if !ok {
		return nil, errors.New("not found")
	}
	cloned := *user
	return &cloned, nil
}

func (m *authRepoMock) GetByUsername(username string) (*domain.User, error) {
	user, ok := m.usersByUsername[username]
	if !ok {
		return nil, errors.New("not found")
	}
	cloned := *user
	return &cloned, nil
}

func (m *authRepoMock) EmailExists(email string) (bool, error) {
	_, ok := m.usersByEmail[email]
	return ok, nil
}

func (m *authRepoMock) UsernameExists(username string) (bool, error) {
	_, ok := m.usersByUsername[username]
	return ok, nil
}

func (m *authRepoMock) CreateRefreshToken(token *domain.RefreshToken) error {
	cloned := *token
	m.refreshTokens[token.TokenHash] = &cloned
	return nil
}

func (m *authRepoMock) GetActiveRefreshTokenByHash(tokenHash string) (*domain.RefreshToken, error) {
	token, ok := m.refreshTokens[tokenHash]
	if !ok {
		return nil, errors.New("not found")
	}
	if token.RevokedAt != nil || time.Now().After(token.ExpiresAt) {
		return nil, errors.New("not active")
	}
	cloned := *token
	return &cloned, nil
}

func (m *authRepoMock) RevokeRefreshTokenByHash(tokenHash string) error {
	token, ok := m.refreshTokens[tokenHash]
	if !ok {
		return errors.New("not found")
	}
	now := time.Now()
	token.RevokedAt = &now
	return nil
}

func TestAuthUsecaseRegisterRejectsDuplicateEmail(t *testing.T) {
	repo := newAuthRepoMock()
	tokens := security.NewJWTService("a", "b", 15, 24)
	uc := NewAuthUsecase(repo, tokens)

	seed := domain.User{Username: "john", Email: "john@example.com", Password: "secret"}
	if err := uc.Register(&seed); err != nil {
		t.Fatalf("seed register failed: %v", err)
	}

	dupe := domain.User{Username: "john2", Email: "john@example.com", Password: "secret"}
	err := uc.Register(&dupe)
	if !errors.Is(err, ErrDuplicateEmail) {
		t.Fatalf("expected ErrDuplicateEmail, got %v", err)
	}
}

func TestAuthUsecaseLoginAndRefresh(t *testing.T) {
	repo := newAuthRepoMock()
	tokens := security.NewJWTService("access-secret", "refresh-secret", 15, 24)
	uc := NewAuthUsecase(repo, tokens)

	user := domain.User{Username: "john", Email: "john@example.com", Password: "secret123"}
	if err := uc.Register(&user); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	result, err := uc.Login("john@example.com", "secret123")
	if err != nil {
		t.Fatalf("login failed: %v", err)
	}
	if result.AccessToken == "" || result.RefreshToken == "" {
		t.Fatal("expected access and refresh tokens")
	}

	refreshed, err := uc.Refresh(result.RefreshToken)
	if err != nil {
		t.Fatalf("refresh failed: %v", err)
	}
	if refreshed.AccessToken == "" || refreshed.RefreshToken == "" {
		t.Fatal("expected refreshed access and refresh tokens")
	}

	if _, err := uc.Refresh(result.RefreshToken); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected revoked refresh token to fail with ErrUnauthorized, got %v", err)
	}
}

func TestAuthUsecaseLoginInvalidPassword(t *testing.T) {
	repo := newAuthRepoMock()
	tokens := security.NewJWTService("a", "b", 15, 24)
	uc := NewAuthUsecase(repo, tokens)

	user := domain.User{Username: "john", Email: "john@example.com", Password: "secret123"}
	if err := uc.Register(&user); err != nil {
		t.Fatalf("register failed: %v", err)
	}

	_, err := uc.Login("john@example.com", "wrong")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("expected ErrInvalidCredentials, got %v", err)
	}
}
