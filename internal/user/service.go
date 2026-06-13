package user

import (
	"context"
	"fmt"
)

type Repository interface {
	Create(context.Context, *User) error
	GetAll(context.Context) ([]*User, error)
	GetByID(context.Context, int) (*User, error)
	Update(context.Context, *User) error
	Delete(context.Context, int) error
}

type PasswordHasher interface {
	Hash(string) (string, error)
}

type Service struct {
	repo     Repository
	password PasswordHasher
}

func NewService(repo Repository, password PasswordHasher) *Service {
	return &Service{repo: repo, password: password}
}

func (s *Service) Create(ctx context.Context, account *User) error {
	hashedPassword, err := s.password.Hash(account.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	account.Password = hashedPassword
	return s.repo.Create(ctx, account)
}

func (s *Service) GetAll(ctx context.Context) ([]*User, error) {
	return s.repo.GetAll(ctx)
}

func (s *Service) GetByID(ctx context.Context, id int) (*User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, account *User) error {
	if account.Password != "" {
		hashedPassword, err := s.password.Hash(account.Password)
		if err != nil {
			return fmt.Errorf("hash password: %w", err)
		}
		account.Password = hashedPassword
	}
	return s.repo.Update(ctx, account)
}

func (s *Service) Delete(ctx context.Context, id int) error {
	return s.repo.Delete(ctx, id)
}
