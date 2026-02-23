// ABOUTME: Tests for dual-limit truncation: line limits, byte limits, UTF-8 safety
// ABOUTME: Covers head/tail modes, empty content, within-limits, multi-byte chars

package tools

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestTruncateHead_WithinLimits(t *testing.T) {
	t.Parallel()

	content := "line1\nline2\nline3\n"
	got := TruncateHead(content, 100, 50*1024)

	if got.Truncated {
		t.Error("expected Truncated = false for content within limits")
	}
	if got.Content != content {
		t.Errorf("content changed: got %q; want %q", got.Content, content)
	}
	if got.Reason != "" {
		t.Errorf("reason = %q; want empty", got.Reason)
	}
	if got.TotalLines != 4 {
		t.Errorf("TotalLines = %d; want 4", got.TotalLines)
	}
	if got.TotalBytes != len(content) {
		t.Errorf("TotalBytes = %d; want %d", got.TotalBytes, len(content))
	}
}

func TestTruncateHead_LineLimit(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("line\n", 2500)
	got := TruncateHead(content, 2000, 50*1024)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if got.Reason != "line_limit" {
		t.Errorf("reason = %q; want %q", got.Reason, "line_limit")
	}
	// Count lines in truncated content.
	// strings.Split on 2000 elements joined by "\n" yields 2000 elements.
	resultLines := strings.Split(got.Content, "\n")
	if len(resultLines) != 2000 {
		t.Errorf("truncated line count = %d; want 2000", len(resultLines))
	}
	if got.TotalLines != 2501 {
		t.Errorf("TotalLines = %d; want 2501", got.TotalLines)
	}
}

func TestTruncateHead_ByteLimit(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("abcdefghij", 20) // 200 bytes
	got := TruncateHead(content, 10000, 100)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if got.Reason != "byte_limit" {
		t.Errorf("reason = %q; want %q", got.Reason, "byte_limit")
	}
	if len(got.Content) > 100 {
		t.Errorf("content length = %d; want <= 100", len(got.Content))
	}
}

func TestTruncateHead_UTF8MultiByte(t *testing.T) {
	t.Parallel()

	// Each emoji is 4 bytes. "🎉" = 4 bytes.
	content := strings.Repeat("🎉", 30) // 120 bytes
	got := TruncateHead(content, 10000, 50)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if !utf8.ValidString(got.Content) {
		t.Error("truncated content is not valid UTF-8")
	}
	if len(got.Content) > 50 {
		t.Errorf("content length = %d; want <= 50", len(got.Content))
	}
	// 50 bytes / 4 bytes per emoji = 12 emojis = 48 bytes (must not split).
	if len(got.Content) != 48 {
		t.Errorf("content length = %d; want 48 (12 complete emojis)", len(got.Content))
	}
}

func TestTruncateHead_Empty(t *testing.T) {
	t.Parallel()

	got := TruncateHead("", 2000, 50*1024)

	if got.Truncated {
		t.Error("expected Truncated = false for empty content")
	}
	if got.Content != "" {
		t.Errorf("content = %q; want empty", got.Content)
	}
	if got.Reason != "" {
		t.Errorf("reason = %q; want empty", got.Reason)
	}
}

func TestTruncateHead_BothLimitsExceeded_ByteWins(t *testing.T) {
	t.Parallel()

	// 3000 lines of 10 bytes each = 30000 bytes total.
	content := strings.Repeat("123456789\n", 3000)
	// maxLines=2000 would keep 20000 bytes; maxBytes=100 is tighter.
	got := TruncateHead(content, 2000, 100)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if got.Reason != "byte_limit" {
		t.Errorf("reason = %q; want %q", got.Reason, "byte_limit")
	}
	if len(got.Content) > 100 {
		t.Errorf("content length = %d; want <= 100", len(got.Content))
	}
}

func TestTruncateTail_WithinLimits(t *testing.T) {
	t.Parallel()

	content := "line1\nline2\nline3\n"
	got := TruncateTail(content, 100, 50*1024)

	if got.Truncated {
		t.Error("expected Truncated = false")
	}
	if got.Content != content {
		t.Errorf("content changed: got %q; want %q", got.Content, content)
	}
	if got.Reason != "" {
		t.Errorf("reason = %q; want empty", got.Reason)
	}
}

func TestTruncateTail_LineLimit(t *testing.T) {
	t.Parallel()

	lines := make([]string, 2500)
	for i := range lines {
		lines[i] = "line"
	}
	content := strings.Join(lines, "\n") + "\n"
	got := TruncateTail(content, 2000, 50*1024)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if got.Reason != "line_limit" {
		t.Errorf("reason = %q; want %q", got.Reason, "line_limit")
	}
	// Should keep last 2000 lines.
	resultLines := strings.Split(got.Content, "\n")
	if len(resultLines) != 2000 {
		t.Errorf("truncated line count = %d; want 2000", len(resultLines))
	}
}

func TestTruncateTail_ByteLimit(t *testing.T) {
	t.Parallel()

	content := strings.Repeat("abcdefghij", 20) // 200 bytes
	got := TruncateTail(content, 10000, 100)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if got.Reason != "byte_limit" {
		t.Errorf("reason = %q; want %q", got.Reason, "byte_limit")
	}
	if len(got.Content) > 100 {
		t.Errorf("content length = %d; want <= 100", len(got.Content))
	}
	// Should keep the last 100 bytes.
	want := strings.Repeat("abcdefghij", 10)
	if got.Content != want {
		t.Errorf("content = %q; want %q", got.Content, want)
	}
}

func TestTruncateTail_UTF8MultiByte(t *testing.T) {
	t.Parallel()

	// CJK character "你" = 3 bytes.
	content := strings.Repeat("你", 30) // 90 bytes
	got := TruncateTail(content, 10000, 50)

	if !got.Truncated {
		t.Error("expected Truncated = true")
	}
	if !utf8.ValidString(got.Content) {
		t.Error("truncated content is not valid UTF-8")
	}
	if len(got.Content) > 50 {
		t.Errorf("content length = %d; want <= 50", len(got.Content))
	}
	// 50 bytes / 3 bytes per char = 16 chars = 48 bytes.
	if len(got.Content) != 48 {
		t.Errorf("content length = %d; want 48 (16 complete CJK chars)", len(got.Content))
	}
}

func TestTruncateTail_Empty(t *testing.T) {
	t.Parallel()

	got := TruncateTail("", 2000, 50*1024)

	if got.Truncated {
		t.Error("expected Truncated = false for empty content")
	}
	if got.Content != "" {
		t.Errorf("content = %q; want empty", got.Content)
	}
}
