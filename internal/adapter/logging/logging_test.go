package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rifkifajarramadhani/golang-clean-architecture/internal/config"
)

func TestLoggerWritesAndClosesFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "app.log")
	logger, err := New(config.LoggingConfig{Level: "info", File: path})
	if err != nil {
		t.Fatal(err)
	}
	logger.Info("hello", "component", "test")
	if err := logger.Close(); err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `"msg":"hello"`) {
		t.Fatalf("log content = %s", content)
	}
}
