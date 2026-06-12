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
