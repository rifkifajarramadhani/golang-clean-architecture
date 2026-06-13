package user

import (
	"context"
	"fmt"
	"net/mail"
	"regexp"
	"strings"
)

const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

var usernamePattern = regexp.MustCompile(`^[A-Za-z0-9_-]{3,50}$`)

type Repository interface {
	Create(context.Context, *User) error
	List(context.Context, int, int) ([]*User, int64, error)
	GetByID(context.Context, int) (*User, error)
	UpdateProfile(context.Context, int, string, string) error
	ChangePassword(context.Context, int, string) error
	ChangeRole(context.Context, int, int, string) error
	Delete(context.Context, int) error
}

type PasswordService interface {
	Hash(string) (string, error)
	Compare(string, string) error
}

type Service struct {
	repo     Repository
	password PasswordService
}

func NewService(repo Repository, password PasswordService) *Service {
	return &Service{repo: repo, password: password}
}

func (s *Service) Create(ctx context.Context, account *User) error {
	if err := NormalizeAndValidate(account, true); err != nil {
		return err
	}
	hashedPassword, err := s.password.Hash(account.Password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	account.Password = hashedPassword
	account.Role = RoleUser
	account.TokenVersion = 1
	return s.repo.Create(ctx, account)
}

func (s *Service) List(ctx context.Context, page, limit int) ([]*User, int64, error) {
	page, limit = NormalizePagination(page, limit)
	return s.repo.List(ctx, page, limit)
}

func (s *Service) GetByID(ctx context.Context, id int) (*User, error) {
	if id <= 0 {
		return nil, ErrNotFound
	}
	return s.repo.GetByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id int, username, email string) error {
	account := &User{Username: username, Email: email}
	if err := NormalizeAndValidate(account, false); err != nil {
		return err
	}
	return s.repo.UpdateProfile(ctx, id, account.Username, account.Email)
}

func (s *Service) ChangePassword(ctx context.Context, id int, currentPassword, newPassword string) error {
	if err := ValidatePassword(newPassword); err != nil {
		return err
	}
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if s.password.Compare(account.Password, currentPassword) != nil {
		return ErrInvalidPassword
	}
	hashed, err := s.password.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.ChangePassword(ctx, id, hashed)
}

func (s *Service) ChangeRole(ctx context.Context, actorID, targetID int, role string) error {
	role = strings.ToLower(strings.TrimSpace(role))
	if role != RoleUser && role != RoleAdmin {
		return ErrInvalidRole
	}
	if actorID == targetID {
		return ErrForbidden
	}
	return s.repo.ChangeRole(ctx, actorID, targetID, role)
}

func (s *Service) Delete(ctx context.Context, actorID, targetID int) error {
	if actorID == targetID {
		return ErrForbidden
	}
	return s.repo.Delete(ctx, targetID)
}

func (s *Service) DeleteSelf(ctx context.Context, id int, currentPassword string) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if s.password.Compare(account.Password, currentPassword) != nil {
		return ErrInvalidPassword
	}
	return s.repo.Delete(ctx, id)
}

func NormalizeAndValidate(account *User, requirePassword bool) error {
	account.Username = strings.TrimSpace(account.Username)
	account.Email = NormalizeEmail(account.Email)
	if !usernamePattern.MatchString(account.Username) {
		return fmt.Errorf("%w: username must be 3-50 letters, digits, underscores, or hyphens", ErrInvalidInput)
	}
	address, err := mail.ParseAddress(account.Email)
	if err != nil || address.Address != account.Email || !strings.Contains(account.Email, "@") {
		return fmt.Errorf("%w: invalid email", ErrInvalidInput)
	}
	if requirePassword {
		return ValidatePassword(account.Password)
	}
	return nil
}

func ValidatePassword(password string) error {
	if len(password) < 12 || len(password) > 72 {
		return fmt.Errorf("%w: password must be 12-72 bytes", ErrInvalidInput)
	}
	return nil
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func NormalizePagination(page, limit int) (int, int) {
	if page < 1 {
		page = DefaultPage
	}
	if limit < 1 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}
	return page, limit
}
