package usecase

import (
	"context"
	"time"
)

type MaintenanceRepository interface {
	DeleteExpiredOrRevokedRefreshTokens(ctx context.Context, before time.Time) (int64, error)
}

type MaintenanceUsecase struct {
	repo MaintenanceRepository
}

func NewMaintenanceUsecase(repo MaintenanceRepository) *MaintenanceUsecase {
	return &MaintenanceUsecase{repo: repo}
}

func (u *MaintenanceUsecase) CleanupRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	return u.repo.DeleteExpiredOrRevokedRefreshTokens(ctx, before)
}
