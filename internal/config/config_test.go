package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadDotEnvLoadsValues(t *testing.T) {
	const key = "CONFIG_TEST_DOTENV_VALUE"
	original, existed := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(key, original)
			return
		}
		_ = os.Unsetenv(key)
	})

	filename := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(filename, []byte(key+"=from-dotenv\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(filename); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(key); got != "from-dotenv" {
		t.Fatalf("%s = %q, want %q", key, got, "from-dotenv")
	}
}

func TestLoadDotEnvDoesNotOverrideEnvironment(t *testing.T) {
	const key = "CONFIG_TEST_DOTENV_PRECEDENCE"
	t.Setenv(key, "from-environment")
	filename := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(filename, []byte(key+"=from-dotenv\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := loadDotEnv(filename); err != nil {
		t.Fatal(err)
	}
	if got := os.Getenv(key); got != "from-environment" {
		t.Fatalf("%s = %q, want %q", key, got, "from-environment")
	}
}

func TestLoadDotEnvAllowsMissingFile(t *testing.T) {
	if err := loadDotEnv(filepath.Join(t.TempDir(), "missing.env")); err != nil {
		t.Fatalf("load missing dotenv file: %v", err)
	}
}

func TestLoadUsesDotEnvAndEnvironmentPrecedence(t *testing.T) {
	const redisPasswordKey = "REDIS_PASSWORD"
	originalRedisPassword, existed := os.LookupEnv(redisPasswordKey)
	if err := os.Unsetenv(redisPasswordKey); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if existed {
			_ = os.Setenv(redisPasswordKey, originalRedisPassword)
			return
		}
		_ = os.Unsetenv(redisPasswordKey)
	})
	t.Setenv("DATABASE_PASSWORD", "from-environment")

	directory := t.TempDir()
	if err := os.Mkdir(filepath.Join(directory, "configs"), 0o755); err != nil {
		t.Fatal(err)
	}
	configFile := `
app:
  port: 8080
  environment: development
database:
  host: db
  port: 3306
  user: app
  password: from-yaml
  name: app
redis:
  address: redis:6379
queue:
  queues:
    default: 1
`
	if err := os.WriteFile(filepath.Join(directory, "configs", "config.yaml"), []byte(configFile), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(
		filepath.Join(directory, ".env"),
		[]byte("DATABASE_PASSWORD=from-dotenv\nREDIS_PASSWORD=redis-from-dotenv\n"),
		0o600,
	); err != nil {
		t.Fatal(err)
	}
	originalDirectory, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(directory); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(originalDirectory) })

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Database.Password != "from-environment" {
		t.Fatalf("database password = %q, want environment value", cfg.Database.Password)
	}
	if cfg.Redis.Password != "redis-from-dotenv" {
		t.Fatalf("redis password = %q, want dotenv value", cfg.Redis.Password)
	}
}

func TestNormalizeQueueConfigDefaultsToRedis(t *testing.T) {
	cfg := QueueConfig{}
	if err := normalizeQueueConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Driver != QueueDriverRedis {
		t.Fatalf("driver = %q, want %q", cfg.Driver, QueueDriverRedis)
	}
	if cfg.Concurrency != 1 || cfg.ShutdownSeconds != 30 {
		t.Fatalf("unexpected worker defaults: %+v", cfg)
	}
	if cfg.Database.PollIntervalMilliseconds != 500 || cfg.Database.ReservationSeconds != 60 {
		t.Fatalf("unexpected database queue defaults: %+v", cfg.Database)
	}
}

func TestNormalizeQueueConfigKeepsDatabaseDriver(t *testing.T) {
	cfg := QueueConfig{Driver: QueueDriverDatabase}
	if err := normalizeQueueConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Driver != QueueDriverDatabase {
		t.Fatalf("driver = %q, want %q", cfg.Driver, QueueDriverDatabase)
	}
}

func TestNormalizeQueueConfigRejectsUnknownDriver(t *testing.T) {
	cfg := QueueConfig{Driver: "sqs"}
	if err := normalizeQueueConfig(&cfg); err == nil {
		t.Fatal("expected unsupported driver error")
	}
}

func TestNormalizeMailConfigDefaults(t *testing.T) {
	cfg := MailConfig{}
	if err := normalizeMailConfig(&cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Host != "localhost" || cfg.Port != 1025 || cfg.Encryption != MailEncryptionNone {
		t.Fatalf("unexpected SMTP defaults: %+v", cfg)
	}
	if cfg.FromAddress != "hello@example.com" || cfg.FromName == "" {
		t.Fatalf("unexpected sender defaults: %+v", cfg)
	}
}

func TestNormalizeMailConfigRejectsInvalidValues(t *testing.T) {
	if err := normalizeMailConfig(&MailConfig{Encryption: "ssl"}); err == nil {
		t.Fatal("expected unsupported encryption error")
	}
	if err := normalizeMailConfig(&MailConfig{FromAddress: "not-an-email"}); err == nil {
		t.Fatal("expected invalid from address error")
	}
}

func TestNormalizeLoggingConfigDefaults(t *testing.T) {
	cfg := LoggingConfig{}
	normalizeLoggingConfig(&cfg)
	if cfg.Level != "info" || cfg.File != "logs/app.log" {
		t.Fatalf("unexpected logging defaults: %+v", cfg)
	}
}

func TestNormalizeAuthConfigRejectsUnsafeProductionSecrets(t *testing.T) {
	tests := []AuthConfig{
		{JWTAccessSecret: "short", JWTRefreshSecret: "also-short"},
		{JWTAccessSecret: "same-secret-that-is-at-least-32-bytes", JWTRefreshSecret: "same-secret-that-is-at-least-32-bytes"},
		{JWTAccessSecret: "super-secret-access-key-change-me", JWTRefreshSecret: "different-secret-that-is-at-least-32"},
		{JWTAccessSecret: "local-access-secret-change-me", JWTRefreshSecret: "local-refresh-secret-change-me-too"},
	}
	staging := AppConfig{Environment: "staging"}
	if err := normalizeAuthConfig(&staging, &AuthConfig{}); err == nil {
		t.Fatal("expected non-development environment to reject unsafe secrets")
	}
	for _, auth := range tests {
		app := AppConfig{Environment: "production"}
		if err := normalizeAuthConfig(&app, &auth); err == nil {
			t.Fatalf("expected rejection for %+v", auth)
		}
	}
}

func TestNormalizeAuthConfigDefaults(t *testing.T) {
	app := AppConfig{}
	auth := AuthConfig{}
	if err := normalizeAuthConfig(&app, &auth); err != nil {
		t.Fatal(err)
	}
	if app.Environment != "development" || auth.Issuer == "" || auth.Audience == "" ||
		auth.VerificationTTLHours != 24 || auth.AccessTTLMinutes != 15 || auth.RefreshTTLHours != 168 {
		t.Fatalf("unexpected defaults: app=%+v auth=%+v", app, auth)
	}
}

func TestValidateConfigRequiresProductionDatabasePassword(t *testing.T) {
	cfg := Config{
		App:      AppConfig{Port: "8080", Environment: "production"},
		Database: DatabaseConfig{Host: "db", Port: 3306, User: "app", Name: "app"},
		Redis:    RedisConfig{Address: "redis:6379"},
		Queue:    QueueConfig{Queues: map[string]int{"default": 1}},
	}
	err := validateConfig(&cfg)
	if err == nil || !strings.Contains(err.Error(), "DATABASE_PASSWORD") {
		t.Fatalf("error = %v, want missing DATABASE_PASSWORD error", err)
	}
}

func TestProductionSecretErrorsDoNotExposeValues(t *testing.T) {
	const secret = "placeholder-secret-value-that-must-not-leak"
	app := AppConfig{Environment: "production"}
	auth := AuthConfig{
		JWTAccessSecret:  secret,
		JWTRefreshSecret: "valid-refresh-secret-that-is-long-enough",
	}
	err := normalizeAuthConfig(&app, &auth)
	if err == nil {
		t.Fatal("expected unsafe production secret error")
	}
	if strings.Contains(err.Error(), secret) {
		t.Fatal("validation error exposed a secret value")
	}
}
