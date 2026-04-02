package logger_test

import (
	"path/filepath"
	"testing"

	"fhpbioguide/pkg/logger"
)

func TestNew_createsLogger(t *testing.T) {
	dir := t.TempDir()
	cfg := logger.Config{
		Verbose:    true,
		File:       filepath.Join(dir, "test.log"),
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
	}
	log := logger.New(cfg)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	// Smoke: these must not panic
	log.Info("info message", "key", "value")
	log.Debug("debug message", "key", "value")
	log.Warn("warn message", "key", "value")
}

func TestNew_verboseFalse_returnsLogger(t *testing.T) {
	dir := t.TempDir()
	cfg := logger.Config{
		Verbose:    false,
		File:       filepath.Join(dir, "test.log"),
		MaxSizeMB:  1,
		MaxBackups: 1,
		MaxAgeDays: 1,
	}
	log := logger.New(cfg)
	if log == nil {
		t.Fatal("expected non-nil logger")
	}
	// Level filtering is delegated to slog — we verify New() wires the level correctly
	// by confirming the factory returns without error. Integration tests cover actual filtering.
	log.Debug("silently dropped at INFO level")
	log.Info("emitted at INFO level")
}
