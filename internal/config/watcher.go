// ABOUTME: Polling-based file watcher for config hot-reload
// ABOUTME: Monitors file mtime at configurable intervals; no external dependencies

package config

import (
	"os"
	"sync"
	"time"
)

// Watcher monitors files for changes by polling mtime at regular intervals.
type Watcher struct {
	paths    []string
	onChange func()
	interval time.Duration
	mtimes   map[string]time.Time
	stopCh   chan struct{}
	mu       sync.Mutex
	running  bool
	stopOnce sync.Once
}

// NewWatcher creates a watcher that calls onChange when any monitored file changes.
func NewWatcher(paths []string, onChange func()) *Watcher {
	return &Watcher{
		paths:    paths,
		onChange: onChange,
		interval: 2 * time.Second,
		mtimes:   make(map[string]time.Time),
		stopCh:   make(chan struct{}),
	}
}

// SetInterval overrides the default polling interval (2s).
func (w *Watcher) SetInterval(d time.Duration) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.interval = d
}

// Start begins polling in a goroutine. Safe to call multiple times; subsequent calls are no-ops.
func (w *Watcher) Start() {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return
	}
	w.running = true
	w.snapshotLocked()
	w.mu.Unlock()

	go w.loop()
}

// Stop halts the polling goroutine. Safe to call multiple times and concurrently.
func (w *Watcher) Stop() {
	w.stopOnce.Do(func() {
		w.mu.Lock()
		w.running = false
		w.mu.Unlock()
		close(w.stopCh)
	})
}

// ForceCheck triggers an immediate check outside the polling cycle.
func (w *Watcher) ForceCheck() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.checkLocked() {
		w.snapshotLocked()
		go w.onChange()
	}
}

func (w *Watcher) loop() {
	w.mu.Lock()
	interval := w.interval
	w.mu.Unlock()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.mu.Lock()
			changed := w.checkLocked()
			if changed {
				w.snapshotLocked()
			}
			w.mu.Unlock()

			if changed {
				w.onChange()
			}
		}
	}
}

// checkLocked compares current mtimes with stored snapshots. Must hold mu.
func (w *Watcher) checkLocked() bool {
	for _, path := range w.paths {
		info, err := os.Stat(path)
		if err != nil {
			// File removed or inaccessible: check if it existed before
			if _, existed := w.mtimes[path]; existed {
				return true
			}
			continue
		}
		prev, ok := w.mtimes[path]
		if !ok || !info.ModTime().Equal(prev) {
			return true
		}
	}
	return false
}

// snapshotLocked records current mtimes. Must hold mu.
func (w *Watcher) snapshotLocked() {
	for _, path := range w.paths {
		info, err := os.Stat(path)
		if err != nil {
			delete(w.mtimes, path)
			continue
		}
		w.mtimes[path] = info.ModTime()
	}
}
