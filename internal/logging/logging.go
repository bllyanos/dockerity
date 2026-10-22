// Package logging builds the process-wide structured logger.
package logging

import (
	"io"
	"log/slog"
)

// New returns a JSON logger writing to w at the given level.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level}))
}
