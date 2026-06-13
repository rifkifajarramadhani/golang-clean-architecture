package config

import "testing"

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
