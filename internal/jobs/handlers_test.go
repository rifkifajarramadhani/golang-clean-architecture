package jobs

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/usecase"
)

type maintenanceRepositoryFake struct{}

func (maintenanceRepositoryFake) DeleteExpiredOrRevokedRefreshTokens(context.Context, time.Time) (int64, error) {
	return 0, nil
}

func TestDemoHandlerRejectsMalformedPayload(t *testing.T) {
	registry := queue.NewHandlerRegistry()
	if err := RegisterHandlers(registry, usecase.NewMaintenanceUsecase(maintenanceRepositoryFake{})); err != nil {
		t.Fatal(err)
	}
	handler := registry.Handlers()[TypeDemoLog]
	if err := handler(context.Background(), json.RawMessage("{")); err == nil {
		t.Fatal("expected malformed payload error")
	}
}
