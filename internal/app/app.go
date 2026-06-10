package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"sync/atomic"
	"syscall"
	"time"

	"gopenid/internal/admin"
	"gopenid/internal/audit"
	"gopenid/internal/auth"
	"gopenid/internal/config"
	"gopenid/internal/database"
	"gopenid/internal/keys"
	"gopenid/internal/oidc"
	"gopenid/internal/pages"
	"gopenid/internal/policy"
	"gopenid/internal/ratelimit"
	"gopenid/internal/store"
	"gopenid/internal/web"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
)

func Run() error {
	ctx := context.Background()
	cfg := config.Load()
	app, db, err := Build(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Pool.Close()
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.Listen(cfg.Addr)
	}()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	select {
	case err := <-errCh:
		return err
	case <-stop:
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return app.ShutdownWithContext(shutdownCtx)
	}
}

func Build(ctx context.Context, cfg config.Config) (*fiber.App, *store.Store, error) {
	db, err := database.Open(ctx, cfg)
	if err != nil {
		return nil, nil, err
	}
	if err := db.Seed(ctx, cfg); err != nil {
		db.Pool.Close()
		return nil, nil, err
	}
	keyManager, err := keys.New(ctx, db, cfg)
	if err != nil {
		db.Pool.Close()
		return nil, nil, err
	}

	recorder := audit.New(db, cfg.WebhookURL)
	policyEngine := policy.New(db)
	authSvc := auth.New(db, cfg, keyManager)
	limiter := ratelimit.New(cfg.RateLimitMax, cfg.RateLimitWindow)
	metrics := &appMetrics{startedAt: time.Now()}

	app := fiber.New(fiber.Config{
		AppName:     "GOpenID",
		ProxyHeader: fiber.HeaderXForwardedFor,
		TrustProxy:  true,
		TrustProxyConfig: fiber.TrustProxyConfig{
			Loopback: true,
			Private:  true,
		},
	})
	app.Use(recover.New())
	app.Use(func(c fiber.Ctx) error {
		requestID := c.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("X-Request-ID", requestID)
		start := time.Now()
		err := c.Next()
		status := c.Response().StatusCode()
		metrics.record(status)
		slog.Info("request", "request_id", requestID, "method", c.Method(), "path", c.Path(), "status", status, "duration_ms", time.Since(start).Milliseconds())
		return err
	})
	app.Use(limiter.Middleware())
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})
	app.Get("/readyz", func(c fiber.Ctx) error {
		if err := db.Pool.Ping(c.Context()); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"status": "unready"})
		}
		return c.JSON(fiber.Map{"status": "ready"})
	})
	app.Get("/metrics", func(c fiber.Ctx) error {
		return c.Type("text/plain; version=0.0.4").SendString(metrics.prometheus())
	})

	auth.NewHandler(authSvc, db, recorder).Mount(app)
	oidc.New(db, cfg, authSvc, keyManager, policyEngine, recorder).Mount(app)
	pages.Mount(app)
	app.Use("/api/admin", auth.RequireBearer(authSvc), auth.RequireRole("admin"))
	app.Post("/api/admin/signing-keys/rotate", func(c fiber.Ctx) error {
		kid, err := keyManager.Rotate(c.Context())
		if err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "key rotation failed"})
		}
		return c.JSON(fiber.Map{"keyId": kid})
	})
	admin.New(db, recorder).Mount(app)
	if err := web.Mount(app); err != nil {
		db.Pool.Close()
		return nil, nil, err
	}

	startMaintenance(ctx, db, limiter, cfg.CleanupInterval)
	return app, db, nil
}

type appMetrics struct {
	startedAt time.Time
	total     atomic.Uint64
	status    [6]atomic.Uint64
}

func (m *appMetrics) record(status int) {
	m.total.Add(1)
	class := status / 100
	if class >= 1 && class <= 5 {
		m.status[class].Add(1)
	}
}

func (m *appMetrics) prometheus() string {
	out := "# HELP gopenid_http_requests_total Total HTTP requests.\n# TYPE gopenid_http_requests_total counter\n"
	out += "gopenid_http_requests_total " + strconv.FormatUint(m.total.Load(), 10) + "\n"
	out += "# HELP gopenid_http_responses_total HTTP responses by status class.\n# TYPE gopenid_http_responses_total counter\n"
	for class := 1; class <= 5; class++ {
		out += `gopenid_http_responses_total{class="` + strconv.Itoa(class) + `xx"} ` + strconv.FormatUint(m.status[class].Load(), 10) + "\n"
	}
	out += "# HELP gopenid_uptime_seconds Process uptime in seconds.\n# TYPE gopenid_uptime_seconds gauge\n"
	out += "gopenid_uptime_seconds " + strconv.FormatInt(int64(time.Since(m.startedAt).Seconds()), 10) + "\n"
	return out
}

// startMaintenance periodically removes expired tokens/codes and prunes stale
// rate-limit windows. It stops when the provided context is cancelled.
func startMaintenance(ctx context.Context, db *store.Store, limiter *ratelimit.Limiter, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if removed, err := db.CleanupExpired(ctx); err != nil {
					slog.Warn("cleanup error", "error", err)
				} else if removed > 0 {
					slog.Info("cleanup removed expired rows", "count", removed)
				}
				limiter.Cleanup()
			}
		}
	}()
}
