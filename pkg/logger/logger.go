// Package logger provides a factory for the application's shared slog.Logger.
// It writes plain-text to both stdout and a rotating log file via lumberjack.
// verbose=true enables DEBUG level; false emits INFO and above only.
package logger

import (
	"io"
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

// Config mirrors the log: block in config.yaml.
type Config struct {
	Verbose    bool
	File       string
	MaxSizeMB  int
	MaxBackups int
	MaxAgeDays int
}

// New returns a slog.Logger that writes to stdout and a rotating file.
func New(cfg Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.Verbose {
		level = slog.LevelDebug
	}

	rotation := &lumberjack.Logger{
		Filename:   cfg.File,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAgeDays,
		Compress:   true,
	}

	w := io.MultiWriter(os.Stdout, rotation)
	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}
