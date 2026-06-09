package ratelimit

import (
	"testing"
	"time"
)

func TestLimiterAllowsWithinWindow(t *testing.T) {
	l := New(3, time.Minute)
	now := time.Now()
	for i := 0; i < 3; i++ {
		ok, _ := l.allow("1.1.1.1", now)
		if !ok {
			t.Fatalf("request %d should be allowed", i+1)
		}
	}
	if ok, retry := l.allow("1.1.1.1", now); ok || retry < 0 {
		t.Fatalf("4th request should be blocked, retry=%d", retry)
	}
}

func TestLimiterResetsAfterWindow(t *testing.T) {
	l := New(1, time.Minute)
	now := time.Now()
	if ok, _ := l.allow("2.2.2.2", now); !ok {
		t.Fatal("first request should pass")
	}
	if ok, _ := l.allow("2.2.2.2", now); ok {
		t.Fatal("second request should be blocked")
	}
	if ok, _ := l.allow("2.2.2.2", now.Add(2*time.Minute)); !ok {
		t.Fatal("request after window should pass")
	}
}

func TestLimiterIsolatesKeys(t *testing.T) {
	l := New(1, time.Minute)
	now := time.Now()
	if ok, _ := l.allow("a", now); !ok {
		t.Fatal("key a first request should pass")
	}
	if ok, _ := l.allow("b", now); !ok {
		t.Fatal("key b should not be affected by key a")
	}
}
