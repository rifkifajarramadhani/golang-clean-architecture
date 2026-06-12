package config

import "testing"

func TestValidateRejectsWeakOrMatchingJWTSecrets(t *testing.T) {
	cfg := validConfig()
	cfg.Auth.JWTAccessSecret = "short"
	if err := cfg.validate(); err == nil {
		t.Fatal("expected weak secret validation error")
	}
	cfg = validConfig()
	cfg.Auth.JWTRefreshSecret = cfg.Auth.JWTAccessSecret
	if err := cfg.validate(); err == nil {
		t.Fatal("expected matching secret validation error")
	}
}

func TestValidateRejectsProductionWithoutDatabasePassword(t *testing.T) {
	cfg := validConfig()
	cfg.App.Environment = EnvironmentProduction
	cfg.Database.Password = ""
	if err := cfg.validate(); err == nil {
		t.Fatal("expected production password validation error")
	}
}

func validConfig() Config {
	return Config{
		App:      AppConfig{Name: "test", Environment: EnvironmentTest, Port: "8080"},
		HTTP:     HTTPConfig{BodyLimitBytes: 1024, RequestTimeoutSeconds: 1, AuthRateLimit: 2},
		Database: DatabaseConfig{Host: "db", Port: 3306, User: "app", Name: "app"},
		Auth: AuthConfig{
			JWTAccessSecret:  "access-secret-at-least-thirty-two-characters",
			JWTRefreshSecret: "refresh-secret-at-least-thirty-two-characters",
			AccessTTLMinutes: 15, RefreshTTLHours: 24, Issuer: "test", Audience: "test",
		},
		Redis: RedisConfig{Address: "redis:6379"},
		Queue: QueueConfig{Concurrency: 1, ShutdownSeconds: 1, Queues: map[string]int{"default": 1}},
		Mail:  MailConfig{Encryption: MailEncryptionNone, FromAddress: "test@example.com"},
	}
}
