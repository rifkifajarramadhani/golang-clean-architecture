package bootstrap

import (
	"log/slog"
	"time"

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

type HTTPServices struct {
	Users  *user.Service
	Auth   *auth.Service
	Tokens *jwt.Service
}

func WireHTTPServices(cfg *config.Config, db *gorm.DB, logger *slog.Logger, dispatcher queue.Dispatcher) HTTPServices {
	repository := mysqlRepository(db)
	hasher := password.Bcrypt{}
	users := user.NewService(repository, hasher)
	tokens := jwt.NewService(
		cfg.Auth.JWTAccessSecret,
		cfg.Auth.JWTRefreshSecret,
		cfg.Auth.AccessTTLMinutes,
		cfg.Auth.RefreshTTLHours,
		cfg.Auth.Issuer,
		cfg.Auth.Audience,
	)
	mailer := appmail.NewMailer(
		appmail.Address{Name: cfg.Mail.FromName, Address: cfg.Mail.FromAddress},
		nil,
		queueadapter.NewMailDispatcher(dispatcher),
	)
	authService := auth.NewService(
		repository,
		repository,
		repository,
		tokens,
		hasher,
		jobs.NewVerificationNotifier(mailer, logger),
		jobs.NewWelcomeNotifier(mailer, logger),
		time.Duration(cfg.Auth.VerificationTTLHours)*time.Hour,
		cfg.Auth.BootstrapAdminEmail,
	)
	return HTTPServices{Users: users, Auth: authService, Tokens: tokens}
}
