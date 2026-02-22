// ABOUTME: Debug logging wrapper around slog for verbose mode output
// ABOUTME: Global level via SetLevel; writes to stderr to avoid mixing with TUI

package log

import (
	"fmt"
	"log/slog"
	"os"
	"sync/atomic"
)

// Level constants matching slog levels.
const (
	LevelDebug = slog.LevelDebug
	LevelInfo  = slog.LevelInfo
	LevelWarn  = slog.LevelWarn
	LevelError = slog.LevelError
)

var level atomic.Int64

func init() {
	level.Store(int64(LevelInfo))
}

// SetLevel sets the global log level.
func SetLevel(l slog.Level) {
	level.Store(int64(l))
}

// Level returns the current log level.
func GetLevel() slog.Level {
	return slog.Level(level.Load())
}

// Debug logs a debug message if the level allows it.
func Debug(format string, args ...any) {
	if slog.Level(level.Load()) > LevelDebug {
		return
	}
	fmt.Fprintf(os.Stderr, "[DEBUG] "+format+"\n", args...)
}

// Info logs an info message if the level allows it.
func Info(format string, args ...any) {
	if slog.Level(level.Load()) > LevelInfo {
		return
	}
	fmt.Fprintf(os.Stderr, "[INFO] "+format+"\n", args...)
}

// Warn logs a warning message if the level allows it.
func Warn(format string, args ...any) {
	if slog.Level(level.Load()) > LevelWarn {
		return
	}
	fmt.Fprintf(os.Stderr, "[WARN] "+format+"\n", args...)
}

// Error logs an error message (always emitted).
func Error(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "[ERROR] "+format+"\n", args...)
}
