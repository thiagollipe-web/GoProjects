package upstream

import (
	"testing"
	"time"
)

func TestBreakerLifecycle(t *testing.T) {
	b := newBreaker(2, 10*time.Second)
	now := time.Unix(0, 0)
	b.now = func() time.Time { return now }

	b.failure()
	if !b.allow() || b.state() != "closed" {
		t.Fatal("should stay closed below threshold")
	}
	b.failure()
	if b.allow() || b.state() != "open" {
		t.Fatal("should open at threshold")
	}
	now = now.Add(11 * time.Second)
	if b.state() != "half-open" || !b.allow() {
		t.Fatal("should allow one probe after cooldown")
	}
	if b.allow() {
		t.Fatal("only one probe at a time")
	}
	b.success()
	if b.state() != "closed" || !b.allow() {
		t.Fatal("success should close the breaker")
	}
}
