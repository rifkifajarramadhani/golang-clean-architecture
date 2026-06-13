package bootstrap

import (
	"log/slog"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jobs"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/jwt"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/password"
	queueadapter "github.com/rifkifajarramadhani/golang-clean-architecture/internal/adapter/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/auth"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
	appmail "github.com/rifkifajarramadhani/golang-clean-architecture/internal/mail"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/queue"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/user"
	"gorm.io/gorm"
)

// HTTPServices holds application services wired for the HTTP server.
type HTTPServices struct {
	Users  *user.Service
	Auth   *auth.Service
	Tokens *jwt.Service
}

// WireHTTPServices wires HTTP-facing application services.
func WireHTTPServices(cfg *config.Config, db *gorm.DB, logger *slog.Logger, dispatcher queue.Dispatcher) HTTPServices {
	repository := mysqlRepository(db)
	hasher := password.Bcrypt{}
	users := user.NewService(repository, hasher)
	tokens := jwt.NewService(
		cfg.Auth.JWTAccessSecret,
		cfg.Auth.JWTRefreshSecret,
		cfg.Auth.AccessTTLMinutes,
		cfg.Auth.RefreshTTLHours,
	)
	mailer := appmail.NewMailer(
		appmail.Address{Name: cfg.Mail.FromName, Address: cfg.Mail.FromAddress},
		nil,
		queueadapter.NewMailDispatcher(dispatcher),
	)
	authService := auth.NewService(
		repository,
		repository,
		tokens,
		hasher,
		jobs.NewWelcomeNotifier(mailer, logger),
	)
	return HTTPServices{Users: users, Auth: authService, Tokens: tokens}
}
