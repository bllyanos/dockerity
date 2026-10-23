package ratelimit

import (
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(3, time.Minute)
	l.Now = func() time.Time { return now }

	for range 3 {
		if !l.Allowed("a") {
			t.Fatal("blocked too early")
		}
		l.Fail("a")
	}
	if l.Allowed("a") {
		t.Fatal("should be blocked after 3 failures")
	}
	if !l.Allowed("b") {
		t.Fatal("other keys must not be affected")
	}

	now = now.Add(time.Minute)
	if !l.Allowed("a") {
		t.Fatal("should be allowed after the window")
	}

	l.Fail("a")
	l.Fail("a")
	l.Reset("a")
	l.Fail("a")
	l.Fail("a")
	if !l.Allowed("a") {
		t.Fatal("reset should clear earlier failures")
	}
}

func TestLimiterPrunesExpired(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(1, time.Second)
	l.Now = func() time.Time { return now }
	for i := range 2000 {
		l.Fail(string(rune(i)))
	}
	now = now.Add(time.Second)
	l.Fail("fresh")
	if len(l.entries) != 1 {
		t.Fatalf("expected expired entries pruned, have %d", len(l.entries))
	}
}
