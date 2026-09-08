package services

import (
	"context"
	"sync"
	"time"
)

type LoginLimiter interface {
	Allow(ctx context.Context, key string) bool
	RecordFailure(ctx context.Context, key string)
	Reset(ctx context.Context, key string)
}

type loginAttemptWindow struct {
	started time.Time
	failures int
}

type MemoryLoginLimiter struct {
	mu sync.Mutex
	window time.Duration
	maxFailures int
	attempts map[string]loginAttemptWindow
}

func NewMemoryLoginLimiter(window time.Duration, maxFailures int) *MemoryLoginLimiter {
	return &MemoryLoginLimiter{window: window, maxFailures: maxFailures, attempts: make(map[string]loginAttemptWindow)}
}

func (l *MemoryLoginLimiter) Allow(_ context.Context, key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	state, ok := l.attempts[key]
	if !ok || time.Since(state.started) >= l.window {
		return true
	}
	return state.failures < l.maxFailures
}

func (l *MemoryLoginLimiter) RecordFailure(_ context.Context, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	state, ok := l.attempts[key]
	if !ok || time.Since(state.started) >= l.window {
		l.attempts[key] = loginAttemptWindow{started: time.Now(), failures: 1}
		return
	}
	state.failures++
	l.attempts[key] = state
}

func (l *MemoryLoginLimiter) Reset(_ context.Context, key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.attempts, key)
}
