package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/domain"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
)

type authRepositoryFake struct {
	created *domain.User
}

func (r *authRepositoryFake) Create(user *domain.User) error {
	user.ID = 42
	r.created = user
	return nil
}
func (*authRepositoryFake) GetByEmail(string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (*authRepositoryFake) GetByUsername(string) (*domain.User, error) {
	return nil, errors.New("not implemented")
}
func (*authRepositoryFake) EmailExists(string) (bool, error)              { return false, nil }
func (*authRepositoryFake) UsernameExists(string) (bool, error)           { return false, nil }
func (*authRepositoryFake) CreateRefreshToken(*domain.RefreshToken) error { return nil }
func (*authRepositoryFake) GetActiveRefreshTokenByHash(string) (*domain.RefreshToken, error) {
	return nil, errors.New("not implemented")
}
func (*authRepositoryFake) RevokeRefreshTokenByHash(string) error { return nil }

type authDispatcherFake struct {
	job queue.Job
	err error
}

func (d *authDispatcherFake) Dispatch(_ context.Context, job queue.Job, _ queue.DispatchOptions) (*queue.JobInfo, error) {
	d.job = job
	return &queue.JobInfo{ID: "mail-1", Queue: "mail"}, d.err
}

func TestRegisterQueuesWelcomeEmail(t *testing.T) {
	repo := &authRepositoryFake{}
	dispatcher := &authDispatcherFake{}
	mailer := appmail.NewMailer(appmail.Address{Address: "hello@example.com"}, nil, dispatcher)
	auth := NewAuthUsecase(repo, nil, mailer)
	user := &domain.User{Username: "rifki", Email: "rifki@example.com", Password: "secret123"}

	if err := auth.Register(user); err != nil {
		t.Fatal(err)
	}
	if dispatcher.job == nil || dispatcher.job.Type() != appmail.TypeSend {
		t.Fatalf("welcome email was not queued: %+v", dispatcher.job)
	}
}

func TestRegisterSucceedsWhenWelcomeDispatchFails(t *testing.T) {
	repo := &authRepositoryFake{}
	dispatcher := &authDispatcherFake{err: errors.New("queue unavailable")}
	mailer := appmail.NewMailer(appmail.Address{Address: "hello@example.com"}, nil, dispatcher)
	auth := NewAuthUsecase(repo, nil, mailer)

	if err := auth.Register(&domain.User{Username: "rifki", Email: "rifki@example.com", Password: "secret123"}); err != nil {
		t.Fatalf("registration failed because mail dispatch failed: %v", err)
	}
	if repo.created == nil {
		t.Fatal("user was not created")
	}
}
