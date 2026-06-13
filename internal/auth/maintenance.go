package auth

import (
	"context"
	"time"
)

type MaintenanceRepository interface {
	DeleteExpiredOrRevokedRefreshTokens(context.Context, time.Time) (int64, error)
}

type MaintenanceService struct {
	repo MaintenanceRepository
}

func NewMaintenanceService(repo MaintenanceRepository) *MaintenanceService {
	return &MaintenanceService{repo: repo}
}

func (s *MaintenanceService) CleanupRefreshTokens(ctx context.Context, before time.Time) (int64, error) {
	return s.repo.DeleteExpiredOrRevokedRefreshTokens(ctx, before)
}
