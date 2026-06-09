// Package ratelimit provides a small, dependency-free, in-memory IP based rate
// limiter implemented as a fixed-window counter. It is suitable for a single
// instance deployment; behind a load balancer use a shared store instead.
package ratelimit

import (
	"strconv"
	"sync"
	"time"

	"gopenid/internal/audit"
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

type window struct {
	count int
	reset time.Time
}

type Limiter struct {
	mu      sync.Mutex
	clients map[string]*window
	max     int
	window  time.Duration
}

func New(max int, win time.Duration) *Limiter {
	if max <= 0 {
		max = 120
	}
	if win <= 0 {
		win = time.Minute
	}
	l := &Limiter{clients: make(map[string]*window), max: max, window: win}
	return l
}

// allow registers a hit for key and reports whether it is within the limit,
// along with the seconds until the current window resets.
func (l *Limiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.clients[key]
	if !ok || now.After(w.reset) {
		l.clients[key] = &window{count: 1, reset: now.Add(l.window)}
		return true, int(l.window.Seconds())
	}
	w.count++
	retry := int(time.Until(w.reset).Seconds())
	if retry < 0 {
		retry = 0
	}
	return w.count <= l.max, retry
}

// Middleware returns a Fiber handler enforcing the IP based limit.
func (l *Limiter) Middleware() fiber.Handler {
	return func(c fiber.Ctx) error {
		ip, _ := audit.RequestContext(c)
		if ip == "" {
			ip = c.IP()
		}
		ok, retry := l.allow(ip, time.Now())
		if !ok {
			c.Set("Retry-After", strconv.Itoa(retry))
			return httpx.Error(c, fiber.StatusTooManyRequests, "rate limit exceeded")
		}
		return c.Next()
	}
}

// Cleanup removes expired windows. Safe to call periodically.
func (l *Limiter) Cleanup() {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	for key, w := range l.clients {
		if now.After(w.reset) {
			delete(l.clients, key)
		}
	}
}
