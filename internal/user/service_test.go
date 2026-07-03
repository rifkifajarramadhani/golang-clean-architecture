package user

import (
	"context"
	"errors"
	"testing"
)

type repositoryFake struct {
	created     *User
	account     *User
	changedHash string
	changedRole Role
	deletedID   int
}

func (r *repositoryFake) Create(_ context.Context, account *User) error {
	copy := *account
	r.created = &copy
	return nil
}
func (*repositoryFake) List(context.Context, int, int) ([]*User, int64, error) { return nil, 0, nil }
func (r *repositoryFake) GetByID(context.Context, int) (*User, error) {
	if r.account == nil {
		return nil, ErrNotFound
	}
	copy := *r.account
	return &copy, nil
}
func (r *repositoryFake) CountAdmins(context.Context) (int64, error)             { return 1, nil }
func (*repositoryFake) UpdateProfile(context.Context, int, string, string) error { return nil }
func (r *repositoryFake) ChangePassword(_ context.Context, _ int, hash string) error {
	r.changedHash = hash
	return nil
}
func (r *repositoryFake) ChangeRole(_ context.Context, _, _ int, role Role) error {
	r.changedRole = role
	return nil
}
func (r *repositoryFake) Delete(_ context.Context, id int) error {
	r.deletedID = id
	return nil
}

type passwordFake struct{}

func (passwordFake) Hash(string) (string, error) { return "hashed", nil }
func (passwordFake) Compare(hashed, plain string) error {
	if hashed == "stored" && plain == "current-password" {
		return nil
	}
	return errors.New("mismatch")
}

func TestServiceCreatesNormalizedUserWithHashedPassword(t *testing.T) {
	repo := &repositoryFake{}
	service := NewService(repo, passwordFake{})
	account := &User{Username: " rifki_1 ", Email: " USER@Example.COM ", Password: "long-password"}
	if err := service.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if repo.created.Password != "hashed" || repo.created.Email != "user@example.com" ||
		repo.created.Role != RoleUser || repo.created.TokenVersion != 1 {
		t.Fatalf("created = %+v", repo.created)
	}
}

func TestServiceRejectsInvalidInput(t *testing.T) {
	service := NewService(&repositoryFake{}, passwordFake{})
	for _, account := range []*User{
		{Username: "x", Email: "user@example.com", Password: "long-password"},
		{Username: "valid_name", Email: "bad", Password: "long-password"},
		{Username: "valid_name", Email: "User <user@example.com>", Password: "long-password"},
		{Username: "valid_name", Email: "user@example.com", Password: "short"},
	} {
		if err := service.Create(context.Background(), account); !errors.Is(err, ErrInvalidInput) {
			t.Fatalf("error = %v", err)
		}
	}
}

func TestChangePasswordRequiresCurrentPassword(t *testing.T) {
	repo := &repositoryFake{account: &User{Password: "stored"}}
	service := NewService(repo, passwordFake{})
	if err := service.ChangePassword(context.Background(), 1, "wrong", "new-password-1"); !errors.Is(err, ErrInvalidPassword) {
		t.Fatalf("error = %v", err)
	}
	if err := service.ChangePassword(context.Background(), 1, "current-password", "new-password-1"); err != nil {
		t.Fatal(err)
	}
	if repo.changedHash != "hashed" {
		t.Fatalf("hash = %q", repo.changedHash)
	}
}

func TestChangeRoleRejectsSelfAndUnknownRoles(t *testing.T) {
	service := NewService(&repositoryFake{}, passwordFake{})
	if err := service.ChangeRole(context.Background(), 1, 1, RoleUser); !errors.Is(err, ErrForbidden) {
		t.Fatalf("self role error = %v", err)
	}
	if err := service.ChangeRole(context.Background(), 1, 2, "owner"); !errors.Is(err, ErrInvalidRole) {
		t.Fatalf("role error = %v", err)
	}
}
