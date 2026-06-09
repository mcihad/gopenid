package app

import (
	"context"
	"log"
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
)

func Run() error {
	ctx := context.Background()
	cfg := config.Load()
	app, db, err := Build(ctx, cfg)
	if err != nil {
		return err
	}
	defer db.Pool.Close()
	return app.Listen(cfg.Addr)
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
	keyManager, err := keys.New(db, cfg)
	if err != nil {
		db.Pool.Close()
		return nil, nil, err
	}

	recorder := audit.New(db)
	policyEngine := policy.New(db)
	authSvc := auth.New(db, cfg, keyManager)
	limiter := ratelimit.New(cfg.RateLimitMax, cfg.RateLimitWindow)

	app := fiber.New(fiber.Config{AppName: "GOpenID", ProxyHeader: fiber.HeaderXForwardedFor})
	app.Use(recover.New())
	app.Use(limiter.Middleware())

	auth.NewHandler(authSvc, db, recorder).Mount(app)
	oidc.New(db, cfg, authSvc, keyManager, policyEngine, recorder).Mount(app)
	pages.Mount(app)
	app.Use("/api/admin", auth.RequireBearer(authSvc))
	admin.New(db).Mount(app)
	if err := web.Mount(app); err != nil {
		db.Pool.Close()
		return nil, nil, err
	}

	startMaintenance(ctx, db, limiter, cfg.CleanupInterval)
	return app, db, nil
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
				if removed, err := db.CleanupExpired(context.Background()); err != nil {
					log.Printf("cleanup error: %v", err)
				} else if removed > 0 {
					log.Printf("cleanup removed %d expired rows", removed)
				}
				limiter.Cleanup()
			}
		}
	}()
}
