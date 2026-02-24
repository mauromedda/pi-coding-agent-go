// ABOUTME: Tests for polling-based file watcher
// ABOUTME: Validates mtime change detection, stop behavior, and force check

package config

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestWatcher_DetectsChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var called atomic.Int32
	w := NewWatcher([]string{path}, func() {
		called.Add(1)
	})
	w.SetInterval(50 * time.Millisecond)
	w.Start()
	defer w.Stop()

	// Wait for initial snapshot
	time.Sleep(100 * time.Millisecond)

	// Modify the file (ensure mtime changes)
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"changed": true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	// Wait for detection
	time.Sleep(200 * time.Millisecond)

	if called.Load() == 0 {
		t.Error("expected onChange to be called after file modification")
	}
}

func TestWatcher_NoChangeNoCallback(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var called atomic.Int32
	w := NewWatcher([]string{path}, func() {
		called.Add(1)
	})
	w.SetInterval(50 * time.Millisecond)
	w.Start()
	defer w.Stop()

	time.Sleep(200 * time.Millisecond)

	if called.Load() != 0 {
		t.Errorf("expected no onChange calls without modification, got %d", called.Load())
	}
}

func TestWatcher_ForceCheck(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var called atomic.Int32
	w := NewWatcher([]string{path}, func() {
		called.Add(1)
	})
	// Don't start polling; just use ForceCheck
	w.snapshotLocked()

	// Modify file
	time.Sleep(50 * time.Millisecond)
	if err := os.WriteFile(path, []byte(`{"force": true}`), 0o600); err != nil {
		t.Fatal(err)
	}

	w.ForceCheck()
	time.Sleep(50 * time.Millisecond) // onChange runs in goroutine

	if called.Load() == 0 {
		t.Error("expected onChange to be called after ForceCheck")
	}
}

func TestWatcher_StopIsIdempotent(t *testing.T) {
	w := NewWatcher(nil, func() {})
	w.Start()
	w.Stop()
	w.Stop() // should not panic
}

func TestWatcher_ConcurrentStop(t *testing.T) {
	w := NewWatcher(nil, func() {})
	w.Start()

	// Multiple goroutines calling Stop concurrently must not panic.
	var wg sync.WaitGroup
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w.Stop()
		}()
	}
	wg.Wait()
}

func TestWatcher_StartIsIdempotent(t *testing.T) {
	w := NewWatcher(nil, func() {})
	w.Start()
	w.Start() // should not panic or start second goroutine
	w.Stop()
}

func TestWatcher_DetectsFileRemoval(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}

	var called atomic.Int32
	w := NewWatcher([]string{path}, func() {
		called.Add(1)
	})
	w.SetInterval(50 * time.Millisecond)
	w.Start()
	defer w.Stop()

	time.Sleep(100 * time.Millisecond)

	// Remove the file
	os.Remove(path)

	time.Sleep(200 * time.Millisecond)

	if called.Load() == 0 {
		t.Error("expected onChange to be called after file removal")
	}
}

func TestWatcher_MissingFileNoError(t *testing.T) {
	w := NewWatcher([]string{"/nonexistent/file.json"}, func() {})
	w.SetInterval(50 * time.Millisecond)
	w.Start()
	time.Sleep(100 * time.Millisecond)
	w.Stop() // should not panic
}
