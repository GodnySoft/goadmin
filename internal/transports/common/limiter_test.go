package common

import (
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	l := NewRateLimiter(2, time.Second)
	now := time.Now()
	if !l.Allow("u1", now) {
		t.Fatalf("first should pass")
	}
	if !l.Allow("u1", now.Add(100*time.Millisecond)) {
		t.Fatalf("second should pass")
	}
	if l.Allow("u1", now.Add(200*time.Millisecond)) {
		t.Fatalf("third should be blocked")
	}
	if !l.Allow("u1", now.Add(2*time.Second)) {
		t.Fatalf("should pass after window")
	}
}
