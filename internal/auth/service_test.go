package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
)

type userStoreFake struct {
	created *user.User
	account *user.User
	token   *EmailVerificationToken
}

func (r *userStoreFake) RegisterUser(_ context.Context, account *user.User, token *EmailVerificationToken, dispatch func() error) error {
	account.ID = 42
	token.UserID = account.ID
	r.token = token
	copy := *account
	r.created = &copy
	return dispatch()
}
func (r *userStoreFake) GetUserByID(context.Context, int) (*user.User, error) {
	if r.account == nil {
		return nil, user.ErrNotFound
	}
	copy := *r.account
	return &copy, nil
}
func (r *userStoreFake) GetUserByEmail(context.Context, string) (*user.User, error) {
	return r.GetUserByID(context.Background(), 1)
}
func (r *userStoreFake) GetUserByEmailOrPending(context.Context, string) (*user.User, error) {
	return r.GetUserByID(context.Background(), 1)
}

type refreshTokenRepoFake struct {
	stored    *RefreshToken
	revoked   bool
	revokeErr error
}

func (r *refreshTokenRepoFake) CreateRefreshToken(_ context.Context, token *RefreshToken) error {
	r.stored = token
	return nil
}
func (r *refreshTokenRepoFake) GetActiveRefreshTokenByHash(context.Context, string) (*RefreshToken, error) {
	if r.stored == nil {
		return nil, errors.New("not found")
	}
	return r.stored, nil
}
func (r *refreshTokenRepoFake) RevokeRefreshTokenByHash(context.Context, string) error {
	if r.revokeErr != nil {
		return r.revokeErr
	}
	r.revoked = true
	return nil
}
func (r *refreshTokenRepoFake) RotateRefreshToken(_ context.Context, _ string, token *RefreshToken) error {
	r.revoked = true
	r.stored = token
	return nil
}

type verificationRepoFake struct {
	token  *EmailVerificationToken
	result *EmailVerificationResult
	err    error
}

func (r *verificationRepoFake) ReplaceEmailVerificationToken(_ context.Context, token *EmailVerificationToken) error {
	r.token = token
	return nil
}
func (r *verificationRepoFake) VerifyEmail(context.Context, string, time.Time, VerificationRolePolicy) (*EmailVerificationResult, error) {
	return r.result, r.err
}
func (*verificationRepoFake) CanIssueEmailVerification(context.Context, int, time.Time) (bool, error) {
	return true, nil
}

type passwordFake struct{}

func (passwordFake) Hash(string) (string, error)  { return "hashed", nil }
func (passwordFake) Compare(string, string) error { return nil }

type tokenServiceFake struct {
	claims Claims
}

func (tokenServiceFake) GenerateAccessToken(int, int) (string, time.Time, error) {
	return "access", time.Now().Add(time.Minute), nil
}
func (tokenServiceFake) GenerateRefreshToken(int, int) (string, time.Time, error) {
	return "refresh", time.Now().Add(time.Hour), nil
}
func (t tokenServiceFake) ValidateRefreshToken(string) (Claims, error) { return t.claims, nil }

type notifierFake struct {
	called bool
	token  string
	email  string
}

type welcomeNotifierFake struct {
	called  int
	account user.User
}

func (n *welcomeNotifierFake) NotifyWelcome(_ context.Context, account user.User) error {
	n.called++
	n.account = account
	return nil
}

func (n *notifierFake) NotifyVerification(_ context.Context, account user.User, token string) error {
	n.called, n.token, n.email = true, token, account.Email
	return nil
}

func TestResendVerificationTargetsPendingEmail(t *testing.T) {
	verified := time.Now()
	users := &userStoreFake{account: &user.User{
		ID: 42, Email: "old@example.com", PendingEmail: "new@example.com", EmailVerifiedAt: &verified,
	}}
	notifier := &notifierFake{}
	service := newTestService(users, &refreshTokenRepoFake{}, &verificationRepoFake{}, tokenServiceFake{}, notifier)
	if err := service.ResendVerification(context.Background(), "new@example.com"); err != nil {
		t.Fatal(err)
	}
	if notifier.email != "new@example.com" {
		t.Fatalf("verification sent to %q", notifier.email)
	}
}

func newTestService(users *userStoreFake, refresh *refreshTokenRepoFake, verification *verificationRepoFake, tokens tokenServiceFake, notifier VerificationNotifier) *Service {
	return NewService(users, refresh, verification, tokens, passwordFake{}, notifier, nil, time.Hour, "")
}

func TestRegisterHashesPasswordAndSendsSingleUseVerification(t *testing.T) {
	users := &userStoreFake{}
	verification := &verificationRepoFake{}
	notifier := &notifierFake{}
	service := newTestService(users, &refreshTokenRepoFake{}, verification, tokenServiceFake{}, notifier)
	account := &user.User{Username: "rifki", Email: "RIFKI@example.com", Password: "long-password"}
	if err := service.Register(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if users.created == nil || account.Password != "hashed" || account.Role != user.RoleUser || !notifier.called {
		t.Fatalf("unexpected registration: account=%+v notified=%v", account, notifier.called)
	}
	if users.token == nil || users.token.TokenHash == notifier.token {
		t.Fatal("verification token was not stored as a hash")
	}
}

func TestVerifyEmailSendsWelcomeOnlyForFirstVerification(t *testing.T) {
	account := &user.User{ID: 42, Username: "rifki", Email: "rifki@example.com"}
	tests := []struct {
		name              string
		firstVerification bool
		wantWelcome       int
	}{
		{"first verification", true, 1},
		{"pending email verification", false, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			verification := &verificationRepoFake{result: &EmailVerificationResult{
				User: account, FirstVerification: test.firstVerification,
			}}
			welcome := &welcomeNotifierFake{}
			service := NewService(
				&userStoreFake{}, &refreshTokenRepoFake{}, verification, tokenServiceFake{}, passwordFake{},
				nil, welcome, time.Hour, "",
			)
			if err := service.VerifyEmail(context.Background(), "token"); err != nil {
				t.Fatal(err)
			}
			if welcome.called != test.wantWelcome {
				t.Fatalf("welcome calls = %d, want %d", welcome.called, test.wantWelcome)
			}
		})
	}
}

func TestVerifyEmailSucceedsWithBestEffortWelcomeNotifier(t *testing.T) {
	verification := &verificationRepoFake{result: &EmailVerificationResult{
		User: &user.User{ID: 42}, FirstVerification: true,
	}}
	service := NewService(
		&userStoreFake{}, &refreshTokenRepoFake{}, verification, tokenServiceFake{}, passwordFake{},
		nil, &welcomeNotifierFake{}, time.Hour, "",
	)
	if err := service.VerifyEmail(context.Background(), "token"); err != nil {
		t.Fatal(err)
	}
}

func TestLoginRequiresVerifiedEmail(t *testing.T) {
	users := &userStoreFake{account: &user.User{ID: 42, Password: "hashed", TokenVersion: 1}}
	service := newTestService(users, &refreshTokenRepoFake{}, &verificationRepoFake{}, tokenServiceFake{}, nil)
	if _, err := service.Login(context.Background(), "user@example.com", "password"); !errors.Is(err, ErrEmailUnverified) {
		t.Fatalf("error = %v", err)
	}
}

func TestLoginComparesPasswordForUnknownUser(t *testing.T) {
	spy := &passwordSpy{}
	service := NewService(
		&userStoreFake{}, &refreshTokenRepoFake{}, &verificationRepoFake{}, tokenServiceFake{}, spy,
		nil, nil, time.Hour, "",
	)
	if _, err := service.Login(context.Background(), "unknown@example.com", "password"); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("error = %v", err)
	}
	if spy.compares != 1 {
		t.Fatalf("password compares = %d, want 1 (timing-equalizing dummy compare)", spy.compares)
	}
}

type passwordSpy struct {
	compares int
}

func (*passwordSpy) Hash(string) (string, error) { return "hashed", nil }
func (s *passwordSpy) Compare(string, string) error {
	s.compares++
	return errors.New("mismatch")
}

func TestLogoutRevokesTokenAndIsIdempotent(t *testing.T) {
	tests := []struct {
		name        string
		token       string
		revokeErr   error
		wantErr     bool
		wantRevoked bool
	}{
		{"revokes active token", "refresh-token", nil, false, true},
		{"unknown token is success", "refresh-token", user.ErrNotFound, false, false},
		{"empty token is success", "", nil, false, false},
		{"storage failure surfaces", "refresh-token", errors.New("db down"), true, false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			refresh := &refreshTokenRepoFake{revokeErr: test.revokeErr}
			service := newTestService(&userStoreFake{}, refresh, &verificationRepoFake{}, tokenServiceFake{}, nil)
			err := service.Logout(context.Background(), test.token)
			if (err != nil) != test.wantErr {
				t.Fatalf("error = %v, wantErr %v", err, test.wantErr)
			}
			if refresh.revoked != test.wantRevoked {
				t.Fatalf("revoked = %v, want %v", refresh.revoked, test.wantRevoked)
			}
		})
	}
}

func TestRefreshUsesStableUserIDAndRejectsStaleTokenVersion(t *testing.T) {
	verified := time.Now()
	users := &userStoreFake{account: &user.User{
		ID: 42, Username: "new-name", EmailVerifiedAt: &verified, TokenVersion: 2,
	}}
	refresh := &refreshTokenRepoFake{stored: &RefreshToken{UserID: 42}}
	service := newTestService(users, refresh, &verificationRepoFake{}, tokenServiceFake{
		claims: Claims{UserID: 42, TokenVersion: 1},
	}, nil)
	if _, err := service.Refresh(context.Background(), "old-refresh"); !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("stale token error = %v", err)
	}
	if refresh.revoked {
		t.Fatal("stale token should not be rotated")
	}

	service.tokens = tokenServiceFake{claims: Claims{UserID: 42, TokenVersion: 2}}
	if _, err := service.Refresh(context.Background(), "valid-refresh"); err != nil {
		t.Fatal(err)
	}
	if !refresh.revoked {
		t.Fatal("refresh token was not rotated")
	}
}
