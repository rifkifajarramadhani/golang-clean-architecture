package user

import (
	"context"
	"testing"
)

type repositoryFake struct {
	created *User
	updated *User
}

func (r *repositoryFake) Create(_ context.Context, account *User) error {
	r.created = account
	return nil
}
func (*repositoryFake) GetAll(context.Context) ([]*User, error)     { return nil, nil }
func (*repositoryFake) GetByID(context.Context, int) (*User, error) { return nil, nil }
func (r *repositoryFake) Update(_ context.Context, account *User) error {
	r.updated = account
	return nil
}
func (*repositoryFake) Delete(context.Context, int) error { return nil }

type passwordFake struct{}

func (passwordFake) Hash(string) (string, error) { return "hashed", nil }

func TestServiceHashesPasswords(t *testing.T) {
	repo := &repositoryFake{}
	service := NewService(repo, passwordFake{})
	account := &User{Password: "plain"}
	if err := service.Create(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if repo.created.Password != "hashed" {
		t.Fatalf("password = %q", repo.created.Password)
	}
	account.Password = ""
	if err := service.Update(context.Background(), account); err != nil {
		t.Fatal(err)
	}
	if repo.updated.Password != "" {
		t.Fatalf("empty update password changed to %q", repo.updated.Password)
	}
}
