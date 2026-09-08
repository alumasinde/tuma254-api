package services

import (
	"context"
	"testing"
	"time"
)

func TestMemoryLoginLimiter(t *testing.T) {
	limiter := NewMemoryLoginLimiter(time.Minute, 2)
	ctx := context.Background()
	key := "account:test@example.com"
	if !limiter.Allow(ctx, key) { t.Fatal("expected initial request to be allowed") }
	limiter.RecordFailure(ctx, key)
	limiter.RecordFailure(ctx, key)
	if limiter.Allow(ctx, key) { t.Fatal("expected limiter to block after maximum failures") }
	limiter.Reset(ctx, key)
	if !limiter.Allow(ctx, key) { t.Fatal("expected reset to allow login") }
}
