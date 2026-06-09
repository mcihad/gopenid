// Package audit records authentication lifecycle events (logins, logouts,
// token refreshes and access denials) together with the originating client
// context (IP address, user agent, device, browser and OS).
package audit

import (
	"context"
	"strings"

	"gopenid/internal/domain"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

type Recorder struct {
	db *store.Store
}

func New(db *store.Store) *Recorder {
	return &Recorder{db: db}
}

// Entry describes a single auditable event.
type Entry struct {
	UserID   *int64
	Email    string
	ClientID string
	Event    string
	Success  bool
	Message  string
}

// Record persists an audit entry, enriching it with request context. Failures
// are swallowed so auditing never breaks the primary request flow.
func (r *Recorder) Record(c fiber.Ctx, e Entry) {
	if r == nil || r.db == nil {
		return
	}
	ip, ua := RequestContext(c)
	device, browser, os := ParseUserAgent(ua)
	_ = r.db.WriteAudit(context.Background(), domain.AuditLog{
		UserID:    e.UserID,
		Email:     e.Email,
		ClientID:  e.ClientID,
		Event:     e.Event,
		Success:   e.Success,
		Message:   truncate(e.Message, 500),
		IP:        ip,
		UserAgent: truncate(ua, 500),
		Device:    device,
		Browser:   browser,
		OS:        os,
	})
}

// RequestContext extracts the client IP (honoring common proxy headers) and the
// User-Agent string from a request.
func RequestContext(c fiber.Ctx) (ip string, userAgent string) {
	userAgent = c.Get("User-Agent")
	if fwd := c.Get("X-Forwarded-For"); fwd != "" {
		ip = strings.TrimSpace(strings.Split(fwd, ",")[0])
		return ip, userAgent
	}
	if real := c.Get("X-Real-IP"); real != "" {
		return strings.TrimSpace(real), userAgent
	}
	return c.IP(), userAgent
}

// ParseUserAgent performs a lightweight, dependency-free classification of a
// user agent string into device type, browser and operating system.
func ParseUserAgent(ua string) (device, browser, os string) {
	if ua == "" {
		return "unknown", "unknown", "unknown"
	}
	lower := strings.ToLower(ua)

	switch {
	case strings.Contains(lower, "windows"):
		os = "Windows"
	case strings.Contains(lower, "android"):
		os = "Android"
	case strings.Contains(lower, "iphone"), strings.Contains(lower, "ipad"), strings.Contains(lower, "ipod"):
		os = "iOS"
	case strings.Contains(lower, "mac os"), strings.Contains(lower, "macintosh"):
		os = "macOS"
	case strings.Contains(lower, "linux"):
		os = "Linux"
	default:
		os = "Other"
	}

	switch {
	case strings.Contains(lower, "edg/"), strings.Contains(lower, "edge"):
		browser = "Edge"
	case strings.Contains(lower, "opr/"), strings.Contains(lower, "opera"):
		browser = "Opera"
	case strings.Contains(lower, "firefox"):
		browser = "Firefox"
	case strings.Contains(lower, "chrome"), strings.Contains(lower, "crios"):
		browser = "Chrome"
	case strings.Contains(lower, "safari"):
		browser = "Safari"
	case strings.Contains(lower, "curl"):
		browser = "curl"
	case strings.Contains(lower, "postman"):
		browser = "Postman"
	default:
		browser = "Other"
	}

	switch {
	case strings.Contains(lower, "mobile"), strings.Contains(lower, "iphone"), strings.Contains(lower, "android"):
		device = "Mobile"
	case strings.Contains(lower, "ipad"), strings.Contains(lower, "tablet"):
		device = "Tablet"
	default:
		device = "Desktop"
	}
	return device, browser, os
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}
