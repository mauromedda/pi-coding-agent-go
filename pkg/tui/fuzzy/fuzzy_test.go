// ABOUTME: Tests for the fuzzy matching wrapper
// ABOUTME: Verifies match ranking and filtering behavior

package fuzzy

import "testing"

func TestFind_BasicMatch(t *testing.T) {
	t.Parallel()

	items := []string{"apple", "application", "banana", "apricot"}
	matches := Find("app", items)

	if len(matches) == 0 {
		t.Fatal("expected matches for 'app'")
	}
	// "apple" and "application" should match
	found := false
	for _, m := range matches {
		if m.Str == "apple" || m.Str == "application" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'apple' or 'application' in results")
	}
}

func TestFind_NoMatch(t *testing.T) {
	t.Parallel()

	items := []string{"cat", "dog", "fish"}
	matches := Find("zzz", items)

	if len(matches) != 0 {
		t.Errorf("expected no matches, got %d", len(matches))
	}
}

func TestFind_Empty(t *testing.T) {
	t.Parallel()

	matches := Find("", []string{"a", "b"})
	// Empty pattern matches everything in sahilm/fuzzy
	_ = matches
}
