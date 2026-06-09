package app

import (
	"context"

	"gopenid/internal/admin"
	"gopenid/internal/auth"
	"gopenid/internal/config"
	"gopenid/internal/database"
	"gopenid/internal/keys"
	"gopenid/internal/oidc"
	"gopenid/internal/pages"
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

	app := fiber.New(fiber.Config{AppName: "GOpenID"})
	app.Use(recover.New())
	authSvc := auth.New(db, cfg, keyManager)
	auth.NewHandler(authSvc).Mount(app)
	oidc.New(db, cfg, authSvc, keyManager).Mount(app)
	pages.Mount(app)
	app.Use("/api/admin", auth.RequireBearer(cfg, keyManager))
	admin.New(db).Mount(app)
	if err := web.Mount(app); err != nil {
		db.Pool.Close()
		return nil, nil, err
	}
	return app, db, nil
}
