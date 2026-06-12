package httpserver

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity"
	identityhttp "github.com/rifkifajarramadhani/golang-clean-architecture/internal/identity/http"
	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/platform/config"
	"gorm.io/gorm"
)

type Server struct {
	App      *fiber.App
	requests atomic.Uint64
}

func New(cfg *config.Config, logger *slog.Logger, db *gorm.DB, redisClient *redis.Client, service *identity.Service, tokens identity.TokenService) *Server {
	server := &Server{}
	app := fiber.New(fiber.Config{
		BodyLimit:    cfg.HTTP.BodyLimitBytes,
		ErrorHandler: errorHandler(logger),
	})
	server.App = app
	app.Use(recoverPanics(logger))
	app.Use(cors(cfg.HTTP.CORSAllowedOrigins))
	app.Use(server.requestMiddleware(logger, cfg.HTTP.RequestTimeoutSeconds))
	app.Use(securityHeaders)
	app.Get("/health/live", func(c fiber.Ctx) error { return c.JSON(fiber.Map{"status": "ok"}) })
	app.Get("/health/ready", readiness(db, redisClient))
	app.Get("/metrics", server.metrics)

	handler := identityhttp.NewHandler(service)
	api := app.Group("/api/v1")
	auth := api.Group("/auth", newRateLimiter(cfg.HTTP.AuthRateLimit))
	auth.Post("/register", handler.Register)
	auth.Post("/login", handler.Login)
	auth.Post("/refresh", handler.Refresh)
	auth.Post("/logout", handler.Logout)
	api.Get("/me", identityhttp.Authenticate(tokens), handler.Me)
	return server
}

func (s *Server) requestMiddleware(logger *slog.Logger, timeoutSeconds int) fiber.Handler {
	return func(c fiber.Ctx) error {
		start := time.Now()
		ctx, cancel := context.WithTimeout(c.Context(), time.Duration(timeoutSeconds)*time.Second)
		defer cancel()
		c.SetContext(ctx)
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("X-Request-ID", requestID)
		s.requests.Add(1)
		err := c.Next()
		logger.Info("http request", "request_id", requestID, "method", c.Method(), "path", c.Path(),
			"status", c.Response().StatusCode(), "duration_ms", time.Since(start).Milliseconds())
		return err
	}
}

func recoverPanics(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) (err error) {
		defer func() {
			if recovered := recover(); recovered != nil {
				logger.Error("request panic", "recovered", recovered, "path", c.Path())
				err = fiber.ErrInternalServerError
			}
		}()
		return c.Next()
	}
}

func cors(allowed []string) fiber.Handler {
	origins := make(map[string]struct{}, len(allowed))
	for _, origin := range allowed {
		origins[origin] = struct{}{}
	}
	return func(c fiber.Ctx) error {
		origin := c.Get("Origin")
		if _, ok := origins[origin]; ok {
			c.Set("Access-Control-Allow-Origin", origin)
			c.Set("Vary", "Origin")
			c.Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Request-ID")
			c.Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		}
		if c.Method() == fiber.MethodOptions {
			return c.SendStatus(fiber.StatusNoContent)
		}
		return c.Next()
	}
}

func securityHeaders(c fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Referrer-Policy", "no-referrer")
	c.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
	return c.Next()
}

func errorHandler(logger *slog.Logger) fiber.ErrorHandler {
	return func(c fiber.Ctx, err error) error {
		status, message := fiber.StatusInternalServerError, "internal server error"
		switch {
		case errors.Is(err, identity.ErrValidation):
			status, message = fiber.StatusBadRequest, err.Error()
		case errors.Is(err, identity.ErrDuplicateEmail), errors.Is(err, identity.ErrDuplicateUsername):
			status, message = fiber.StatusConflict, err.Error()
		case errors.Is(err, identity.ErrInvalidCredentials), errors.Is(err, identity.ErrInvalidToken), errors.Is(err, identity.ErrUnauthorized):
			status, message = fiber.StatusUnauthorized, "unauthorized"
		default:
			var fiberErr *fiber.Error
			if errors.As(err, &fiberErr) {
				status, message = fiberErr.Code, fiberErr.Message
			} else {
				logger.Error("request failed", "error", err, "method", c.Method(), "path", c.Path())
			}
		}
		return c.Status(status).JSON(fiber.Map{"error": message})
	}
}

func readiness(db *gorm.DB, redisClient *redis.Client) fiber.Handler {
	return func(c fiber.Ctx) error {
		ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
		defer cancel()
		sqlDB, err := db.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil || redisClient.Ping(ctx).Err() != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "not_ready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	}
}

func (s *Server) metrics(c fiber.Ctx) error {
	c.Type("text/plain", "utf-8")
	return c.SendString(fmt.Sprintf("# TYPE http_requests_total counter\nhttp_requests_total %d\n", s.requests.Load()))
}

type rateLimiter struct {
	mu      sync.Mutex
	entries map[string]*rateEntry
	limit   int
}

type rateEntry struct {
	window time.Time
	count  int
}

func newRateLimiter(limit int) fiber.Handler {
	limiter := &rateLimiter{entries: make(map[string]*rateEntry), limit: limit}
	return func(c fiber.Ctx) error {
		now := time.Now()
		key := c.IP()
		limiter.mu.Lock()
		entry := limiter.entries[key]
		if entry == nil || now.Sub(entry.window) >= time.Minute {
			entry = &rateEntry{window: now}
			limiter.entries[key] = entry
		}
		entry.count++
		allowed := entry.count <= limiter.limit
		if len(limiter.entries) > 10000 {
			for candidate, candidateEntry := range limiter.entries {
				if now.Sub(candidateEntry.window) >= time.Minute {
					delete(limiter.entries, candidate)
				}
			}
		}
		limiter.mu.Unlock()
		if !allowed {
			return fiber.NewError(fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return c.Next()
	}
}
