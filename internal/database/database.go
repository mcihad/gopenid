package database

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopenid/internal/config"
	"gopenid/internal/store"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mcihad/pgxmigrate/migrator"
)

var schemaNamePattern = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func Open(ctx context.Context, cfg config.Config) (*store.Store, error) {
	if !schemaNamePattern.MatchString(cfg.Database.Schema) {
		return nil, fmt.Errorf("invalid db schema: %s", cfg.Database.Schema)
	}
	adminPool, err := pgxpool.New(ctx, cfg.Database.ConnString())
	if err != nil {
		return nil, err
	}
	if cfg.DBReset {
		if _, err := adminPool.Exec(ctx, `DROP SCHEMA IF EXISTS `+cfg.Database.Schema+` CASCADE`); err != nil {
			adminPool.Close()
			return nil, err
		}
	}
	if _, err := adminPool.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+cfg.Database.Schema); err != nil {
		adminPool.Close()
		return nil, err
	}
	adminPool.Close()

	poolCfg, err := pgxpool.ParseConfig(cfg.Database.ConnString())
	if err != nil {
		return nil, err
	}
	poolCfg.ConnConfig.RuntimeParams["search_path"] = cfg.Database.Schema
	poolCfg.AfterConnect = func(ctx context.Context, conn *pgx.Conn) error {
		_, err := conn.Exec(ctx, `SET search_path TO `+cfg.Database.Schema)
		return err
	}
	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, err
	}
	dir, err := migrationsDir()
	if err != nil {
		pool.Close()
		return nil, err
	}
	m := migrator.New(pool, dir)
	if err := m.Ensure(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	pending, err := m.Pending(ctx)
	if err != nil {
		pool.Close()
		return nil, err
	}
	applied, err := m.Up(ctx, 0)
	if err != nil {
		pool.Close()
		return nil, err
	}
	if err := ensureMigrated(ctx, pool, dir, len(applied), len(pending)); err != nil {
		pool.Close()
		return nil, err
	}
	return store.New(pool), nil
}

func ensureMigrated(ctx context.Context, pool *pgxpool.Pool, dir string, applied int, pending int) error {
	var exists bool
	if err := pool.QueryRow(ctx, `SELECT to_regclass('users') IS NOT NULL`).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("migration did not create users table; dir: %s; pending migrations: %d; applied migrations: %d", dir, pending, applied)
	}
	return nil
}

func migrationsDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() && hasSQLMigration(candidate) {
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("migrations directory not found")
		}
		dir = parent
	}
}

func hasSQLMigration(dir string) bool {
	matches, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	return err == nil && len(matches) > 0
}
