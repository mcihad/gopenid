package app

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"gopenid/internal/config"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOpenIDFlows(t *testing.T) {
	ctx := context.Background()
	schema := fmt.Sprintf("auth_test_%d", time.Now().UnixNano())
	cfg := testConfig(schema)
	if !postgresAvailable(ctx, cfg) {
		t.Skip("postgres is not available for integration tests")
	}
	app, db, err := Build(ctx, cfg)
	if err != nil {
		t.Fatalf("build app: %v", err)
	}
	defer db.Pool.Close()
	defer dropSchema(ctx, t, cfg)

	loginBody := postJSON(t, app, "/api/auth/login", nil, map[string]string{
		"email": "admin@gopenid.local", "password": "admin12345",
	}, http.StatusOK)
	adminToken := stringField(t, loginBody, "access_token")
	assertJWTClaim(t, adminToken, "roles", []any{"admin"})

	getJSON(t, app, "/.well-known/openid-configuration", nil, http.StatusOK)
	jwks := getJSON(t, app, "/.well-known/jwks.json", nil, http.StatusOK)
	if len(jwks["keys"].([]any)) == 0 {
		t.Fatal("jwks keys empty")
	}
	doStatus(t, app, http.MethodGet, "/api/admin/users", bearer(adminToken), nil, http.StatusOK)
	doStatus(t, app, http.MethodGet, "/api/admin/users", nil, nil, http.StatusUnauthorized)
	userinfo := getJSON(t, app, "/oauth/userinfo", bearer(adminToken), http.StatusOK)
	if userinfo["email"] != "admin@gopenid.local" {
		t.Fatalf("unexpected userinfo: %#v", userinfo)
	}

	tempRole := postJSON(t, app, "/api/admin/roles", bearer(adminToken), map[string]string{
		"name": "temporary-role", "description": "deleted role reuse",
	}, http.StatusCreated)
	doStatus(t, app, http.MethodDelete, fmt.Sprintf("/api/admin/roles/%d", int64(tempRole["ID"].(float64))), bearer(adminToken), nil, http.StatusNoContent)
	reusedRole := postJSON(t, app, "/api/admin/roles", bearer(adminToken), map[string]string{
		"name": "temporary-role", "description": "reused after soft delete",
	}, http.StatusCreated)
	doStatus(t, app, http.MethodDelete, fmt.Sprintf("/api/admin/roles/%d", int64(reusedRole["ID"].(float64))), bearer(adminToken), nil, http.StatusNoContent)

	client := postJSON(t, app, "/api/admin/clients", bearer(adminToken), map[string]string{
		"clientId": "web", "clientSecret": "secret", "name": "Web", "redirectUris": "http://localhost:3000/callback",
	}, http.StatusCreated)
	clientDBID := int64(client["ID"].(float64))
	tempClientRole := postJSON(t, app, fmt.Sprintf("/api/admin/clients/%d/roles", clientDBID), bearer(adminToken), map[string]string{
		"name": "temporary-client-role", "description": "deleted client role reuse",
	}, http.StatusCreated)
	doStatus(t, app, http.MethodDelete, fmt.Sprintf("/api/admin/clients/%d/roles/%d", clientDBID, int64(tempClientRole["ID"].(float64))), bearer(adminToken), nil, http.StatusNoContent)
	reusedClientRole := postJSON(t, app, fmt.Sprintf("/api/admin/clients/%d/roles", clientDBID), bearer(adminToken), map[string]string{
		"name": "temporary-client-role", "description": "reused after soft delete",
	}, http.StatusCreated)
	doStatus(t, app, http.MethodDelete, fmt.Sprintf("/api/admin/clients/%d/roles/%d", clientDBID, int64(reusedClientRole["ID"].(float64))), bearer(adminToken), nil, http.StatusNoContent)
	role := postJSON(t, app, fmt.Sprintf("/api/admin/clients/%d/roles", clientDBID), bearer(adminToken), map[string]string{
		"name": "reader", "description": "Reader",
	}, http.StatusCreated)
	roleID := int64(role["ID"].(float64))

	putJSON(t, app, "/api/admin/users/1", bearer(adminToken), map[string]any{
		"email": "admin@gopenid.local", "name": "System Admin", "password": "", "active": true,
		"departmentId": 1, "roleIds": []int{1}, "clientIds": []int64{clientDBID}, "clientRoleIds": []int64{roleID},
	}, http.StatusOK)

	withoutClientRoles := tokenForm(t, app, url.Values{
		"grant_type": {"password"}, "username": {"admin@gopenid.local"}, "password": {"admin12345"},
		"client_id": {"web"}, "client_secret": {"secret"}, "scope": {"openid profile email roles"},
	})
	assertJWTMissingClaim(t, withoutClientRoles, "client_roles")
	assertJWTClaim(t, withoutClientRoles, "roles", []any{"admin"})

	withClientRoles := tokenForm(t, app, url.Values{
		"grant_type": {"password"}, "username": {"admin@gopenid.local"}, "password": {"admin12345"},
		"client_id": {"web"}, "client_secret": {"secret"}, "scope": {"openid profile email roles client_roles"},
	})
	assertJWTClaim(t, withClientRoles, "client_roles", []any{"reader"})

	code := authorizeCode(t, app)
	codeToken := tokenForm(t, app, url.Values{
		"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {"http://localhost:3000/callback"},
		"client_id": {"web"}, "client_secret": {"secret"}, "code_verifier": {"plain-verifier"},
	})
	assertJWTClaim(t, codeToken, "client_roles", []any{"reader"})
}

func testConfig(schema string) config.Config {
	cfg := config.Load()
	cfg.Database.Schema = schema
	cfg.DevSeed = true
	cfg.AdminEmail = "admin@gopenid.local"
	cfg.AdminPass = "admin12345"
	return cfg
}

func postgresAvailable(ctx context.Context, cfg config.Config) bool {
	pool, err := pgxpool.New(ctx, cfg.Database.ConnString())
	if err != nil {
		return false
	}
	defer pool.Close()
	return pool.Ping(ctx) == nil
}

func dropSchema(ctx context.Context, t *testing.T, cfg config.Config) {
	t.Helper()
	pool, err := pgxpool.New(ctx, cfg.Database.ConnString())
	if err != nil {
		t.Logf("cleanup connect: %v", err)
		return
	}
	defer pool.Close()
	_, _ = pool.Exec(ctx, `DROP SCHEMA IF EXISTS `+cfg.Database.Schema+` CASCADE`)
}

func postJSON(t *testing.T, app *fiber.App, path string, headers map[string]string, body any, want int) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	return doJSON(t, app, http.MethodPost, path, headers, "application/json", bytes.NewReader(raw), want)
}

func putJSON(t *testing.T, app *fiber.App, path string, headers map[string]string, body any, want int) map[string]any {
	t.Helper()
	raw, _ := json.Marshal(body)
	return doJSON(t, app, http.MethodPut, path, headers, "application/json", bytes.NewReader(raw), want)
}

func getJSON(t *testing.T, app *fiber.App, path string, headers map[string]string, want int) map[string]any {
	t.Helper()
	return doJSON(t, app, http.MethodGet, path, headers, "", nil, want)
}

func doJSON(t *testing.T, app *fiber.App, method string, path string, headers map[string]string, contentType string, body io.Reader, want int) map[string]any {
	t.Helper()
	req, _ := http.NewRequest(method, path, body)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, res.StatusCode, want, string(raw))
	}
	if len(raw) == 0 {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode json: %v body=%s", err, string(raw))
	}
	return out
}

func doStatus(t *testing.T, app *fiber.App, method string, path string, headers map[string]string, body io.Reader, want int) {
	t.Helper()
	req, _ := http.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("%s %s status=%d want=%d body=%s", method, path, res.StatusCode, want, string(raw))
	}
}

func tokenForm(t *testing.T, app *fiber.App, values url.Values) string {
	t.Helper()
	body := doJSON(t, app, http.MethodPost, "/oauth/token", nil, "application/x-www-form-urlencoded", strings.NewReader(values.Encode()), http.StatusOK)
	return stringField(t, body, "access_token")
}

func authorizeCode(t *testing.T, app *fiber.App) string {
	t.Helper()
	values := url.Values{
		"email": {"admin@gopenid.local"}, "password": {"admin12345"},
		"response_type": {"code"}, "client_id": {"web"}, "redirect_uri": {"http://localhost:3000/callback"},
		"scope": {"openid profile email roles client_roles"}, "state": {"state-1"}, "nonce": {"nonce-1"},
		"code_challenge": {"plain-verifier"}, "code_challenge_method": {"plain"},
	}
	req, _ := http.NewRequest(http.MethodPost, "/oauth/authorize", strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("authorize: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusFound && res.StatusCode != http.StatusSeeOther {
		raw, _ := io.ReadAll(res.Body)
		t.Fatalf("authorize status=%d body=%s", res.StatusCode, string(raw))
	}
	location := res.Header.Get("Location")
	u, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse redirect: %v", err)
	}
	return u.Query().Get("code")
}

func bearer(token string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + token}
}

func stringField(t *testing.T, body map[string]any, key string) string {
	t.Helper()
	value, ok := body[key].(string)
	if !ok || value == "" {
		t.Fatalf("missing string field %s in %#v", key, body)
	}
	return value
}

func assertJWTClaim(t *testing.T, token string, key string, want []any) {
	t.Helper()
	claims := jwtPayload(t, token)
	got, ok := claims[key].([]any)
	if !ok {
		t.Fatalf("missing jwt array claim %s in %#v", key, claims)
	}
	if fmt.Sprint(got) != fmt.Sprint(want) {
		t.Fatalf("claim %s=%v want=%v", key, got, want)
	}
}

func assertJWTMissingClaim(t *testing.T, token string, key string) {
	t.Helper()
	claims := jwtPayload(t, token)
	if _, ok := claims[key]; ok {
		t.Fatalf("claim %s should be absent in %#v", key, claims)
	}
}

func jwtPayload(t *testing.T, token string) map[string]any {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("invalid jwt: %s", token)
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode jwt payload: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("unmarshal jwt payload: %v", err)
	}
	return out
}
