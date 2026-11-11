package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterBurstAndRefill(t *testing.T) {
	l := New(2)
	now := time.Unix(1000, 0)
	l.now = func() time.Time { return now }

	for i := 0; i < 2; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d should pass", i)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait <= 0 {
		t.Fatalf("third request should be limited, got ok=%v wait=%v", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("other keys must not be affected")
	}
	now = now.Add(31 * time.Second) // 2/min => one token per 30s
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("token should have refilled")
	}
}

func TestLimiterDisabled(t *testing.T) {
	l := New(0)
	for i := 0; i < 100; i++ {
		if ok, _ := l.Allow("x"); !ok {
			t.Fatal("disabled limiter must always allow")
		}
	}
}
