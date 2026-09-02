package ratelimit

import (
	"context"
	"testing"
)

func TestLocalLimiterAllowsConfiguredWindow(t *testing.T) {
	limiter := NewLocal(2, 2)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		allowed, err := limiter.Allow(ctx, "consumer:1")
		if err != nil || !allowed {
			t.Fatalf("request %d allowed=%v err=%v", i, allowed, err)
		}
	}
	allowed, err := limiter.Allow(ctx, "consumer:1")
	if err != nil {
		t.Fatalf("third request error: %v", err)
	}
	if allowed {
		t.Fatal("third request should be rate limited")
	}
}
