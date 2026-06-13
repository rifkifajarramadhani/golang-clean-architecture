package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type userStoreFake struct {
	created        *user.User
	emailExists    bool
	usernameExists bool
}

func (r *userStoreFake) CreateUser(_ context.Context, account *user.User) error {
	account.ID = 42
	r.created = account
	return nil
}
func (*userStoreFake) GetUserByEmail(context.Context, string) (*user.User, error) {
	return nil, errors.New("not implemented")
}
func (*userStoreFake) GetUserByUsername(context.Context, string) (*user.User, error) {
	return nil, errors.New("not implemented")
}
func (r *userStoreFake) EmailExists(context.Context, string) (bool, error) {
	return r.emailExists, nil
}
func (r *userStoreFake) UsernameExists(context.Context, string) (bool, error) {
	return r.usernameExists, nil
}

type refreshTokenRepoFake struct{}

func (*refreshTokenRepoFake) CreateRefreshToken(context.Context, *RefreshToken) error { return nil }
func (*refreshTokenRepoFake) GetActiveRefreshTokenByHash(context.Context, string) (*RefreshToken, error) {
	return nil, errors.New("not implemented")
}
func (*refreshTokenRepoFake) RevokeRefreshTokenByHash(context.Context, string) error { return nil }

type passwordFake struct{}

func (passwordFake) Hash(string) (string, error)  { return "hashed", nil }
func (passwordFake) Compare(string, string) error { return nil }

type tokenServiceFake struct{}

func (tokenServiceFake) GenerateAccessToken(string, int) (string, time.Time, error) {
	return "access", time.Now().Add(time.Minute), nil
}
func (tokenServiceFake) GenerateRefreshToken(string, int) (string, time.Time, error) {
	return "refresh", time.Now().Add(time.Hour), nil
}
func (tokenServiceFake) ValidateRefreshToken(string) (Claims, error) { return Claims{}, nil }

type notifierFake struct{ called bool }

func (n *notifierFake) NotifyWelcome(context.Context, user.User) { n.called = true }

func TestRegister(t *testing.T) {
	repo := &userStoreFake{}
	notifier := &notifierFake{}
	service := NewService(repo, &refreshTokenRepoFake{}, tokenServiceFake{}, passwordFake{}, notifier)
	account := &user.User{Username: "rifki", Email: "rifki@example.com", Password: "secret"}
	if err := service.Register(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if repo.created == nil || account.Password != "hashed" || !notifier.called {
		t.Fatalf("unexpected registration: account=%+v notified=%v", account, notifier.called)
	}
}

func TestRegisterRejectsDuplicates(t *testing.T) {
	tests := []struct {
		name string
		repo *userStoreFake
		want error
	}{
		{"email", &userStoreFake{emailExists: true}, ErrDuplicateEmail},
		{"username", &userStoreFake{usernameExists: true}, ErrDuplicateUsername},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewService(test.repo, &refreshTokenRepoFake{}, tokenServiceFake{}, passwordFake{}, nil).Register(
				context.Background(), &user.User{Email: "x", Username: "x", Password: "x"},
			)
			if !errors.Is(err, test.want) {
				t.Fatalf("got %v want %v", err, test.want)
			}
		})
	}
}
