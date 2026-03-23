package rate

import (
	"context"
	"testing"
	"time"
)

func TestInMemoryLimiter(t *testing.T) {
	l := NewInMemoryLimiter()
	ctx := context.Background()
	key := "k1"
	limit := 3
	window := 200 * time.Millisecond
	for i := 0; i < limit; i++ {
		ok, err := l.Allow(ctx, key, limit, window)
		if err != nil || !ok {
			t.Fatalf("expected allow #%d", i+1)
		}
	}
	if ok, _ := l.Allow(ctx, key, limit, window); ok {
		t.Fatalf("expected deny after limit exceeded")
	}
	time.Sleep(window + 10*time.Millisecond)
	if ok, _ := l.Allow(ctx, key, limit, window); !ok {
		t.Fatalf("expected allow after window reset")
	}
}
