package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Addr       string
	Issuer     string
	Database   Database
	KeyID      string
	TokenTTL   time.Duration
	DevSeed    bool
	AdminEmail string
	AdminPass  string
}

type Database struct {
	Host     string
	Port     string
	User     string
	Password string
	Name     string
	Schema   string
	SSLMode  string
	URL      string
}

func Load() Config {
	_ = godotenv.Load()
	db := Database{
		Host:     env("DB_HOST", "localhost"),
		Port:     env("DB_PORT", "5432"),
		User:     env("DB_USER", "postgres"),
		Password: env("DB_PASSWORD", "postgres"),
		Name:     env("DB_NAME", "postgres"),
		Schema:   env("DB_SCHEMA", "auth"),
		SSLMode:  env("DB_SSLMODE", "disable"),
		URL:      os.Getenv("DATABASE_URL"),
	}
	return Config{
		Addr:       env("GOPENID_ADDR", ":8080"),
		Issuer:     strings.TrimRight(env("GOPENID_ISSUER", "http://localhost:8080"), "/"),
		Database:   db,
		KeyID:      env("GOPENID_KEY_ID", "gopenid-rs256-1"),
		TokenTTL:   8 * time.Hour,
		DevSeed:    env("GOPENID_DEV_SEED", "true") == "true",
		AdminEmail: env("GOPENID_ADMIN_EMAIL", "admin@gopenid.local"),
		AdminPass:  env("GOPENID_ADMIN_PASS", "admin12345"),
	}
}

func (db Database) ConnString() string {
	if db.URL != "" {
		return withSchemaOption(db.URL, db.Schema)
	}
	user := url.QueryEscape(db.User)
	pass := url.QueryEscape(db.Password)
	name := url.PathEscape(db.Name)
	schema := url.QueryEscape("-c search_path=" + db.Schema)
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s&options=%s", user, pass, db.Host, db.Port, name, db.SSLMode, schema)
}

func withSchemaOption(rawURL string, schema string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return rawURL
	}
	query := parsed.Query()
	if query.Get("options") == "" {
		query.Set("options", "-c search_path="+schema)
	}
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
