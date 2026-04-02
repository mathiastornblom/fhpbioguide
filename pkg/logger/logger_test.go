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

func TestNew_verboseFalse_doesNotPanic(t *testing.T) {
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
	log.Debug("this should be silently dropped")
	log.Info("this should be emitted")
}
