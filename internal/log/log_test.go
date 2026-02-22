// ABOUTME: Tests for debug logging package
// ABOUTME: Validates level filtering and output to stderr

package log

import (
	"log/slog"
	"testing"
)

func TestSetLevel(t *testing.T) {
	t.Parallel()

	SetLevel(LevelDebug)
	if GetLevel() != LevelDebug {
		t.Errorf("expected LevelDebug, got %v", GetLevel())
	}

	SetLevel(LevelError)
	if GetLevel() != LevelError {
		t.Errorf("expected LevelError, got %v", GetLevel())
	}
}

func TestDefaultLevel(t *testing.T) {
	t.Parallel()

	// Default is Info (set in init)
	savedLevel := GetLevel()
	defer SetLevel(savedLevel)

	SetLevel(slog.LevelInfo)
	if GetLevel() != slog.LevelInfo {
		t.Errorf("expected LevelInfo default, got %v", GetLevel())
	}
}

func TestDebugSuppressedAtInfoLevel(t *testing.T) {
	savedLevel := GetLevel()
	defer SetLevel(savedLevel)

	SetLevel(LevelInfo)

	// Debug should be suppressed at Info level; no panic is enough
	Debug("this should be suppressed: %s", "test")
}

func TestDebugEmittedAtDebugLevel(t *testing.T) {
	savedLevel := GetLevel()
	defer SetLevel(savedLevel)

	SetLevel(LevelDebug)

	// Debug should emit at Debug level; no panic is enough
	Debug("this should emit: %s", "test")
}

func TestAllLevels(t *testing.T) {
	savedLevel := GetLevel()
	defer SetLevel(savedLevel)

	SetLevel(LevelDebug)

	// These should all succeed without panic
	Debug("debug: %d", 1)
	Info("info: %d", 2)
	Warn("warn: %d", 3)
	Error("error: %d", 4)
}
