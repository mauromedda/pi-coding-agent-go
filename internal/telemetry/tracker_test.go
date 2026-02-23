// ABOUTME: Tests for cumulative token/cost tracker with budget alerts
// ABOUTME: Covers recording, accumulation, budget thresholds, callbacks, reset, concurrency

package telemetry

import (
	"math"
	"sync"
	"testing"
)

func TestTracker_RecordSingle(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0, 80)
	alerts := tr.Record("claude-sonnet-4", 1000, 500)

	if len(alerts) != 0 {
		t.Errorf("expected no alerts with no budget, got %d", len(alerts))
	}

	s := tr.Summary()
	if s.TotalInputTokens != 1000 {
		t.Errorf("TotalInputTokens = %d, want 1000", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 500 {
		t.Errorf("TotalOutputTokens = %d, want 500", s.TotalOutputTokens)
	}
	if s.CallCount != 1 {
		t.Errorf("CallCount = %d, want 1", s.CallCount)
	}

	wantCost := EstimateCost("claude-sonnet-4", 1000, 500)
	if math.Abs(s.TotalCostUSD-wantCost) > 1e-9 {
		t.Errorf("TotalCostUSD = %v, want %v", s.TotalCostUSD, wantCost)
	}
}

func TestTracker_RecordMultiple_Accumulation(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0, 80)
	tr.Record("claude-sonnet-4", 1000, 500)
	tr.Record("gpt-4o", 2000, 1000)
	tr.Record("gemini-2.0-flash", 5000, 2000)

	s := tr.Summary()
	if s.TotalInputTokens != 8000 {
		t.Errorf("TotalInputTokens = %d, want 8000", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 3500 {
		t.Errorf("TotalOutputTokens = %d, want 3500", s.TotalOutputTokens)
	}
	if s.CallCount != 3 {
		t.Errorf("CallCount = %d, want 3", s.CallCount)
	}

	wantCost := EstimateCost("claude-sonnet-4", 1000, 500) +
		EstimateCost("gpt-4o", 2000, 1000) +
		EstimateCost("gemini-2.0-flash", 5000, 2000)
	if math.Abs(s.TotalCostUSD-wantCost) > 1e-9 {
		t.Errorf("TotalCostUSD = %v, want %v", s.TotalCostUSD, wantCost)
	}
}

func TestTracker_BudgetWarning(t *testing.T) {
	t.Parallel()

	// Budget $1.00, warn at 80% ($0.80).
	// claude-opus-4: 1M input = $15. Use tokens to cross 80%.
	// 50k input tokens = 50000/1M * 15 = $0.75 (under 80%).
	// Then 5k more input = 55000/1M * 15 = $0.825 (over 80%).
	tr := NewTracker(1.0, 80)

	alerts := tr.Record("claude-opus-4", 50000, 0)
	if len(alerts) != 0 {
		t.Errorf("expected no alerts at 75%%, got %d", len(alerts))
	}

	alerts = tr.Record("claude-opus-4", 5000, 0)
	if len(alerts) != 1 {
		t.Fatalf("expected 1 warning alert at ~82.5%%, got %d", len(alerts))
	}
	if alerts[0].Type != "warning" {
		t.Errorf("alert type = %q, want %q", alerts[0].Type, "warning")
	}
}

func TestTracker_BudgetLimit(t *testing.T) {
	t.Parallel()

	// Budget $0.10. claude-sonnet-4: input $3/M.
	// 33334 tokens = ~$0.10. Push over with one more call.
	tr := NewTracker(0.10, 80)

	// First call: ~$0.09 (30000 tokens * $3/M = $0.09), triggers warning at 80% ($0.08)
	alerts := tr.Record("claude-sonnet-4", 30000, 0)
	hasWarning := false
	for _, a := range alerts {
		if a.Type == "warning" {
			hasWarning = true
		}
	}
	if !hasWarning {
		t.Errorf("expected warning alert at 90%%, got alerts: %v", alerts)
	}

	// Second call: push over $0.10 limit.
	alerts = tr.Record("claude-sonnet-4", 10000, 0)
	hasLimit := false
	for _, a := range alerts {
		if a.Type == "limit" {
			hasLimit = true
		}
	}
	if !hasLimit {
		t.Errorf("expected limit alert when over budget, got alerts: %v", alerts)
	}
}

func TestTracker_NoBudget(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0, 80)

	// Record large amounts; should never get alerts.
	for i := range 100 {
		_ = i
		alerts := tr.Record("claude-opus-4", 1_000_000, 1_000_000)
		if len(alerts) != 0 {
			t.Fatalf("expected no alerts with budget=0, got %d at iteration %d", len(alerts), i)
		}
	}
}

func TestTracker_AlertCallback(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0.01, 80)

	var received []Alert
	tr.SetAlertCallback(func(a Alert) {
		received = append(received, a)
	})

	// Exceed budget immediately: claude-opus-4 at 1000 input = $0.015 > $0.01.
	tr.Record("claude-opus-4", 1000, 0)

	if len(received) == 0 {
		t.Fatal("expected callback to be called")
	}
}

func TestTracker_Summary(t *testing.T) {
	t.Parallel()

	tr := NewTracker(1.0, 80)
	tr.Record("claude-sonnet-4", 1000, 500)

	s := tr.Summary()
	if s.BudgetUSD != 1.0 {
		t.Errorf("BudgetUSD = %v, want 1.0", s.BudgetUSD)
	}

	wantPct := (s.TotalCostUSD / 1.0) * 100
	if math.Abs(s.BudgetUsedPct-wantPct) > 1e-9 {
		t.Errorf("BudgetUsedPct = %v, want %v", s.BudgetUsedPct, wantPct)
	}

	if len(s.Alerts) != 0 {
		t.Errorf("expected 0 alerts in summary, got %d", len(s.Alerts))
	}
}

func TestTracker_Summary_NoBudget(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0, 80)
	tr.Record("claude-sonnet-4", 1000, 500)

	s := tr.Summary()
	if s.BudgetUSD != 0 {
		t.Errorf("BudgetUSD = %v, want 0", s.BudgetUSD)
	}
	if s.BudgetUsedPct != 0 {
		t.Errorf("BudgetUsedPct = %v, want 0 when no budget set", s.BudgetUsedPct)
	}
}

func TestTracker_Reset(t *testing.T) {
	t.Parallel()

	tr := NewTracker(1.0, 80)
	tr.Record("claude-sonnet-4", 1000, 500)
	tr.Record("gpt-4o", 2000, 1000)

	tr.Reset()

	s := tr.Summary()
	if s.TotalInputTokens != 0 {
		t.Errorf("after Reset, TotalInputTokens = %d, want 0", s.TotalInputTokens)
	}
	if s.TotalOutputTokens != 0 {
		t.Errorf("after Reset, TotalOutputTokens = %d, want 0", s.TotalOutputTokens)
	}
	if s.TotalCostUSD != 0 {
		t.Errorf("after Reset, TotalCostUSD = %v, want 0", s.TotalCostUSD)
	}
	if s.CallCount != 0 {
		t.Errorf("after Reset, CallCount = %d, want 0", s.CallCount)
	}
	if len(s.Alerts) != 0 {
		t.Errorf("after Reset, Alerts count = %d, want 0", len(s.Alerts))
	}
	// Budget should be preserved.
	if s.BudgetUSD != 1.0 {
		t.Errorf("after Reset, BudgetUSD = %v, want 1.0 (preserved)", s.BudgetUSD)
	}
}

func TestTracker_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0, 80)
	const goroutines = 50
	const callsPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for range goroutines {
		go func() {
			defer wg.Done()
			for range callsPerGoroutine {
				tr.Record("claude-sonnet-4", 100, 50)
			}
		}()
	}

	wg.Wait()

	s := tr.Summary()
	wantCalls := goroutines * callsPerGoroutine
	if s.CallCount != wantCalls {
		t.Errorf("CallCount = %d, want %d", s.CallCount, wantCalls)
	}

	wantInput := goroutines * callsPerGoroutine * 100
	if s.TotalInputTokens != wantInput {
		t.Errorf("TotalInputTokens = %d, want %d", s.TotalInputTokens, wantInput)
	}

	wantOutput := goroutines * callsPerGoroutine * 50
	if s.TotalOutputTokens != wantOutput {
		t.Errorf("TotalOutputTokens = %d, want %d", s.TotalOutputTokens, wantOutput)
	}
}

func TestTracker_AlertCallback_NoDeadlock(t *testing.T) {
	t.Parallel()

	tr := NewTracker(0.01, 80)

	// Callback calls Summary() which acquires the lock.
	// Before the fix, this would deadlock because Record held the lock.
	tr.SetAlertCallback(func(a Alert) {
		s := tr.Summary()
		if s.CallCount == 0 {
			t.Error("expected CallCount > 0 in callback")
		}
	})

	// Exceed budget: should fire callback without deadlocking.
	// Test will timeout if deadlock occurs.
	tr.Record("claude-opus-4", 1000, 0)
}

func TestTracker_WarningOnlyOnce(t *testing.T) {
	t.Parallel()

	// Budget $1.00, warn at 80%.
	// First call crosses 80%, second call stays above 80% but below 100%.
	// Warning should fire only once.
	tr := NewTracker(1.0, 80)

	// claude-sonnet-4: $3/M input. 300000 tokens = $0.90 (90%).
	alerts1 := tr.Record("claude-sonnet-4", 300000, 0)
	warnCount := 0
	for _, a := range alerts1 {
		if a.Type == "warning" {
			warnCount++
		}
	}
	if warnCount != 1 {
		t.Errorf("first call: expected 1 warning, got %d", warnCount)
	}

	// Another small call, still under 100%.
	alerts2 := tr.Record("claude-sonnet-4", 10000, 0)
	for _, a := range alerts2 {
		if a.Type == "warning" {
			t.Error("second call should not re-trigger warning")
		}
	}
}
