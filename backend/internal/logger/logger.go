package logger

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// Init configures the default slog logger from LOG_LEVEL and LOG_FILE env vars.
// Supported LOG_LEVEL values: debug, info, warn, error (default: info).
// If LOG_FILE is set, logs are written to both stdout and that file.
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

	var w io.Writer = os.Stdout
	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		if err := os.MkdirAll(filepath.Dir(logFile), 0o755); err == nil {
			if f, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644); err == nil {
				w = io.MultiWriter(os.Stdout, f)
			}
		}
	}

	h := slog.NewTextHandler(w, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}
