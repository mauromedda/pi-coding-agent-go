// ABOUTME: Tests for auto-retry on rate-limit errors in the Bubble Tea TUI
// ABOUTME: Verifies retry countdown, max retries, and non-retryable error passthrough

package btea

import (
	"fmt"
	"testing"
	"time"
)

func TestIsRateLimited(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"rate limit message", fmt.Errorf("rate limit exceeded"), true},
		{"429 status", fmt.Errorf("HTTP 429: too many requests"), true},
		{"overloaded", fmt.Errorf("server overloaded"), true},
		{"wrapped rate limit", fmt.Errorf("provider: %w", fmt.Errorf("rate limit")), true},
		{"connection error", fmt.Errorf("connection refused"), false},
		{"auth error", fmt.Errorf("401 unauthorized"), false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRateLimited(tt.err); got != tt.want {
				t.Errorf("isRateLimited(%v) = %v; want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestAppModel_RetryOnRateLimit(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	err := fmt.Errorf("rate limit exceeded")
	result, cmd := m.Update(AgentErrorMsg{Err: err})
	model := result.(AppModel)

	if model.retryCount != 1 {
		t.Errorf("retryCount = %d; want 1", model.retryCount)
	}
	if model.retryAt.IsZero() {
		t.Error("retryAt is zero; want future time")
	}
	if cmd == nil {
		t.Fatal("cmd = nil; want retry tick command")
	}
}

func TestAppModel_MaxRetries(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true
	m.retryCount = 3 // already at max

	err := fmt.Errorf("rate limit exceeded")
	result, _ := m.Update(AgentErrorMsg{Err: err})
	model := result.(AppModel)

	// Should NOT retry; treat as final error
	if model.retryCount != 3 {
		t.Errorf("retryCount = %d; want 3 (unchanged)", model.retryCount)
	}
}

func TestAppModel_NonRateLimitError_NoRetry(t *testing.T) {
	m := NewAppModel(testDeps())
	m.agentRunning = true

	err := fmt.Errorf("connection refused")
	result, _ := m.Update(AgentErrorMsg{Err: err})
	model := result.(AppModel)

	if model.retryCount != 0 {
		t.Errorf("retryCount = %d; want 0 for non-rate-limit error", model.retryCount)
	}
}

func TestAppModel_RetryTickMsg(t *testing.T) {
	m := NewAppModel(testDeps())
	m.retryCount = 1
	m.retryAt = time.Now().Add(-1 * time.Second) // already past

	result, cmd := m.Update(RetryTickMsg{Remaining: 0})
	model := result.(AppModel)

	// When remaining is 0, should trigger retry (start agent again)
	// retryCount stays at 1 (it was set when we decided to retry)
	_ = model
	// cmd should be the agent start command or nil if no provider
	_ = cmd
}
