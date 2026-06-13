package auth

import (
	"context"
	"testing"
	"time"
)

type maintenanceRepositoryFake struct {
	before time.Time
	count  int64
}

func (r *maintenanceRepositoryFake) DeleteExpiredOrRevokedRefreshTokens(_ context.Context, before time.Time) (int64, error) {
	r.before = before
	return r.count, nil
}

func TestCleanupRefreshTokens(t *testing.T) {
	repo := &maintenanceRepositoryFake{count: 4}
	service := NewMaintenanceService(repo)
	before := time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	count, err := service.CleanupRefreshTokens(context.Background(), before)
	if err != nil {
		t.Fatal(err)
	}
	if count != 4 || !repo.before.Equal(before) {
		t.Fatalf("got count=%d before=%s", count, repo.before)
	}
}
