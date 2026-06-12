package identity

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type repositoryFake struct {
	user      *User
	active    map[string]RefreshToken
	createErr error
}

func (r *repositoryFake) CreateUser(_ context.Context, user *User) error {
	if r.createErr != nil {
		return r.createErr
	}
	user.ID = 7
	r.user = user
	return nil
}
func (r *repositoryFake) GetUserByEmail(context.Context, string) (*User, error) {
	if r.user == nil {
		return nil, ErrNotFound
	}
	return r.user, nil
}
func (r *repositoryFake) GetUserByID(context.Context, int) (*User, error) {
	if r.user == nil {
		return nil, ErrNotFound
	}
	return r.user, nil
}
func (r *repositoryFake) CreateRefreshToken(_ context.Context, token RefreshToken) error {
	if r.active == nil {
		r.active = make(map[string]RefreshToken)
	}
	r.active[token.TokenHash] = token
	return nil
}
func (r *repositoryFake) RotateRefreshToken(_ context.Context, old string, replacement RefreshToken) error {
	if _, ok := r.active[old]; !ok {
		return ErrNotFound
	}
	delete(r.active, old)
	r.active[replacement.TokenHash] = replacement
	return nil
}
func (r *repositoryFake) RevokeRefreshToken(_ context.Context, hash string) error {
	if _, ok := r.active[hash]; !ok {
		return ErrNotFound
	}
	delete(r.active, hash)
	return nil
}
func (*repositoryFake) DeleteExpiredOrRevokedRefreshTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}

type tokenServiceFake struct{ next int }

func (t *tokenServiceFake) GenerateAccessToken(int) (string, time.Time, error) {
	t.next++
	return "access", time.Now().Add(time.Minute), nil
}
func (t *tokenServiceFake) GenerateRefreshToken(int) (string, time.Time, error) {
	t.next++
	return "refresh-" + string(rune('0'+t.next)), time.Now().Add(time.Hour), nil
}
func (*tokenServiceFake) ValidateToken(raw, expected string) (*TokenClaims, error) {
	if raw == "" || expected != TokenTypeRefresh {
		return nil, ErrInvalidToken
	}
	return &TokenClaims{UserID: 7, TokenType: expected}, nil
}

func TestRefreshIsSingleUseAndLogoutRevokes(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct-password"), bcrypt.MinCost)
	repo := &repositoryFake{user: &User{ID: 7, Email: "user@example.com", PasswordHash: string(hash)}}
	service := NewService(repo, &tokenServiceFake{}, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	login, err := service.Login(context.Background(), "user@example.com", "correct-password")
	if err != nil {
		t.Fatal(err)
	}
	refreshed, err := service.Refresh(context.Background(), login.RefreshToken)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Refresh(context.Background(), login.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("replay error = %v, want invalid token", err)
	}
	if err := service.Logout(context.Background(), refreshed.RefreshToken); err != nil {
		t.Fatal(err)
	}
	if err := service.Logout(context.Background(), refreshed.RefreshToken); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("second logout error = %v, want invalid token", err)
	}
}

func TestRegisterReturnsDuplicateError(t *testing.T) {
	repo := &repositoryFake{createErr: ErrDuplicateEmail}
	service := NewService(repo, &tokenServiceFake{}, nil, slog.Default())
	if _, err := service.Register(context.Background(), "user", "user@example.com", "long-enough-password"); !errors.Is(err, ErrDuplicateEmail) {
		t.Fatalf("error = %v", err)
	}
}
