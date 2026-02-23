// ABOUTME: Tests for SHA256-keyed prompt cache with concurrent access
// ABOUTME: Validates get/set, key uniqueness, invalidation, and thread safety

package prompts

import (
	"sync"
	"testing"
)

func TestCache_GetSet(t *testing.T) {
	t.Parallel()

	c := NewCache()
	vars := map[string]string{"MODE": "execute"}
	c.Set("v1.0.0", vars, "composed prompt")

	got, ok := c.Get("v1.0.0", vars)
	if !ok {
		t.Fatal("Cache.Get() returned false; want true")
	}
	if got != "composed prompt" {
		t.Errorf("Cache.Get() = %q; want %q", got, "composed prompt")
	}
}

func TestCache_Miss(t *testing.T) {
	t.Parallel()

	c := NewCache()
	_, ok := c.Get("v1.0.0", map[string]string{"MODE": "plan"})
	if ok {
		t.Fatal("Cache.Get() returned true for missing entry; want false")
	}
}

func TestCache_DifferentVarsProduceDifferentKeys(t *testing.T) {
	t.Parallel()

	c := NewCache()
	vars1 := map[string]string{"MODE": "execute"}
	vars2 := map[string]string{"MODE": "plan"}

	c.Set("v1.0.0", vars1, "prompt-execute")
	c.Set("v1.0.0", vars2, "prompt-plan")

	got1, ok1 := c.Get("v1.0.0", vars1)
	got2, ok2 := c.Get("v1.0.0", vars2)

	if !ok1 || !ok2 {
		t.Fatal("expected both cache entries to exist")
	}
	if got1 == got2 {
		t.Errorf("different vars should produce different entries; both = %q", got1)
	}
	if got1 != "prompt-execute" {
		t.Errorf("vars1 entry = %q; want %q", got1, "prompt-execute")
	}
	if got2 != "prompt-plan" {
		t.Errorf("vars2 entry = %q; want %q", got2, "prompt-plan")
	}
}

func TestCache_Invalidate(t *testing.T) {
	t.Parallel()

	c := NewCache()
	c.Set("v1.0.0", map[string]string{"MODE": "execute"}, "prompt")

	if c.Size() != 1 {
		t.Fatalf("Cache.Size() = %d; want 1", c.Size())
	}

	c.Invalidate()

	if c.Size() != 0 {
		t.Errorf("Cache.Size() after Invalidate() = %d; want 0", c.Size())
	}
	_, ok := c.Get("v1.0.0", map[string]string{"MODE": "execute"})
	if ok {
		t.Error("Cache.Get() returned true after Invalidate(); want false")
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	c := NewCache()
	var wg sync.WaitGroup

	for i := range 100 {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			vars := map[string]string{"N": string(rune('A' + n%26))}
			c.Set("v1.0.0", vars, "value")
			c.Get("v1.0.0", vars)
		}(i)
	}

	wg.Wait()

	// No panic or race = pass. Size should be <= 26 (26 unique keys).
	if c.Size() > 26 {
		t.Errorf("Cache.Size() = %d; want <= 26", c.Size())
	}
}
