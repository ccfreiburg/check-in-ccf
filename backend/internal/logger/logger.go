package logger

import (
	"log/slog"
	"os"
	"strings"
)

// Init configures the default slog logger from LOG_LEVEL env var.
// Supported values: debug, info, warn, error (default: info).
func Init() {
	level := slog.LevelInfo
	switch strings.ToLower(os.Getenv("LOG_LEVEL")) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}
	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}
