package logger

import (
	"log/slog"
	"os"
)

// New returns JSON logger with level taken from LOG_LEVEL (default info).
func New() *slog.Logger {
	level := slog.LevelInfo
	if env := os.Getenv("LOG_LEVEL"); env != "" {
		var parsed slog.Level
		if err := parsed.UnmarshalText([]byte(env)); err == nil {
			level = parsed
		}
	}
	h := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	return slog.New(h)
}
