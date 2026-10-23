// Package ratelimit counts failures per key in fixed windows, used to slow
// down password guessing. State is in memory, which is enough for one instance.
package ratelimit

import (
	"sync"
	"time"
)

// Limiter blocks a key after Max failures within Window.
type Limiter struct {
	Max    int
	Window time.Duration
	Now    func() time.Time

	mu      sync.Mutex
	entries map[string]entry
}

type entry struct {
	failures int
	start    time.Time
}

// New returns a limiter allowing max failures per window.
func New(maxFailures int, window time.Duration) *Limiter {
	return &Limiter{Max: maxFailures, Window: window, Now: time.Now}
}

// Allowed reports whether key may make another attempt.
func (l *Limiter) Allowed(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	e, ok := l.current(key)
	return !ok || e.failures < l.Max
}

// Fail records a failed attempt for key.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.entries == nil {
		l.entries = map[string]entry{}
	}
	e, ok := l.current(key)
	if !ok {
		e = entry{start: l.Now()}
	}
	e.failures++
	l.entries[key] = e
	l.prune()
}

// Reset clears failures for key, e.g. after a successful login.
func (l *Limiter) Reset(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.entries, key)
}

// current returns the live entry for key, dropping an expired one.
func (l *Limiter) current(key string) (entry, bool) {
	e, ok := l.entries[key]
	if ok && l.Now().Sub(e.start) >= l.Window {
		delete(l.entries, key)
		return entry{}, false
	}
	return e, ok
}

// prune bounds memory by dropping expired entries once the map grows.
func (l *Limiter) prune() {
	if len(l.entries) < 1024 {
		return
	}
	now := l.Now()
	for k, e := range l.entries {
		if now.Sub(e.start) >= l.Window {
			delete(l.entries, k)
		}
	}
}
