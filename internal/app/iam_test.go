package app

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
)

// TestIAMFeatures exercises the extended identity-management capabilities:
// refresh tokens, revocation/logout, blocking, self-service, groups, multiple
// departments, client access control, login policies and audit logging.
func TestIAMFeatures(t *testing.T) {
	ctx := context.Background()
	schema := fmt.Sprintf("auth_iam_%d", time.Now().UnixNano())
	cfg, cleanupPostgres := testConfigWithPostgres(t, ctx, schema)
	defer cleanupPostgres()
	app, db, err := Build(ctx, cfg)
	if err != nil {
		t.Fatalf("build app: %v", err)
	}
	defer db.Pool.Close()
	defer dropSchema(ctx, t, cfg)

	doStatus(t, app, http.MethodGet, "/healthz", nil, nil, http.StatusOK)
	doStatus(t, app, http.MethodGet, "/readyz", nil, nil, http.StatusOK)
	doStatus(t, app, http.MethodGet, "/metrics", nil, nil, http.StatusOK)

	adminCreds := map[string]string{"email": "admin@gopenid.local", "password": "admin12345"}

	// --- Refresh token rotation ----------------------------------------------
	login := postJSON(t, app, "/api/auth/login", nil, adminCreds, http.StatusOK)
	adminToken := stringField(t, login, "access_token")
	adminRefresh := stringField(t, login, "refresh_token")

	refreshed := postJSON(t, app, "/api/auth/refresh", nil, map[string]string{"refresh_token": adminRefresh}, http.StatusOK)
	stringField(t, refreshed, "access_token")
	// Old refresh token is rotated out and must be rejected on reuse.
	postJSON(t, app, "/api/auth/refresh", nil, map[string]string{"refresh_token": adminRefresh}, http.StatusUnauthorized)

	// --- Self-service ---------------------------------------------------------
	me := getJSON(t, app, "/api/me", bearer(adminToken), http.StatusOK)
	if me["email"] != "admin@gopenid.local" {
		t.Fatalf("unexpected /api/me: %#v", me)
	}
	roles := getArray(t, app, "/api/me/roles", bearer(adminToken), http.StatusOK)
	if !arrayHasName(roles, "admin") {
		t.Fatalf("admin should have admin role, got %#v", roles)
	}
	getArray(t, app, "/api/me/departments", bearer(adminToken), http.StatusOK)
	getArray(t, app, "/api/me/groups", bearer(adminToken), http.StatusOK)
	updated := putJSON(t, app, "/api/me", bearer(adminToken), map[string]any{
		"name": "System Admin", "phone": "+90 555 000 0000", "title": "Yönetici",
	}, http.StatusOK)
	if updated["phone"] != "+90 555 000 0000" {
		t.Fatalf("profile phone not updated: %#v", updated)
	}

	// --- Logout / access-token revocation ------------------------------------
	fresh := postJSON(t, app, "/api/auth/login", nil, adminCreds, http.StatusOK)
	freshToken := stringField(t, fresh, "access_token")
	doStatus(t, app, http.MethodGet, "/api/me", bearer(freshToken), nil, http.StatusOK)
	doStatus(t, app, http.MethodPost, "/api/auth/logout", bearer(freshToken), nil, http.StatusOK)
	// Token is now revoked and must fail validation.
	doStatus(t, app, http.MethodGet, "/api/me", bearer(freshToken), nil, http.StatusUnauthorized)

	// --- Blocking -------------------------------------------------------------
	created := postJSON(t, app, "/api/admin/users", bearer(adminToken), map[string]any{
		"email": "worker@gopenid.local", "name": "Worker", "password": "worker12345", "active": true,
	}, http.StatusCreated)
	workerID := int64(created["ID"].(float64))
	workerCreds := map[string]string{"email": "worker@gopenid.local", "password": "worker12345"}

	postJSON(t, app, "/api/auth/login", nil, workerCreds, http.StatusOK)
	postJSON(t, app, fmt.Sprintf("/api/admin/users/%d/block", workerID), bearer(adminToken), map[string]string{"reason": "test"}, http.StatusOK)
	postJSON(t, app, "/api/auth/login", nil, workerCreds, http.StatusUnauthorized)
	postJSON(t, app, fmt.Sprintf("/api/admin/users/%d/unblock", workerID), bearer(adminToken), nil, http.StatusOK)
	postJSON(t, app, "/api/auth/login", nil, workerCreds, http.StatusOK)

	// --- Password self-service ------------------------------------------------
	workerLogin := postJSON(t, app, "/api/auth/login", nil, workerCreds, http.StatusOK)
	workerToken := stringField(t, workerLogin, "access_token")
	doStatus(t, app, http.MethodGet, "/api/admin/users", bearer(workerToken), nil, http.StatusForbidden)
	postJSON(t, app, "/api/me/password", bearer(workerToken), map[string]string{
		"currentPassword": "worker12345", "newPassword": "worker54321",
	}, http.StatusOK)
	postJSON(t, app, "/api/auth/login", nil, workerCreds, http.StatusUnauthorized)
	postJSON(t, app, "/api/auth/login", nil, map[string]string{"email": "worker@gopenid.local", "password": "worker54321"}, http.StatusOK)

	resetReq := postJSON(t, app, "/api/auth/password-reset/request", nil, map[string]string{"email": "worker@gopenid.local"}, http.StatusOK)
	resetToken := stringField(t, resetReq, "resetToken")
	postJSON(t, app, "/api/auth/password-reset/confirm", nil, map[string]string{"token": resetToken, "newPassword": "worker67890"}, http.StatusOK)
	postJSON(t, app, "/api/auth/login", nil, map[string]string{"email": "worker@gopenid.local", "password": "worker54321"}, http.StatusUnauthorized)
	postJSON(t, app, "/api/auth/login", nil, map[string]string{"email": "worker@gopenid.local", "password": "worker67890"}, http.StatusOK)
	verifyReq := postJSON(t, app, "/api/auth/email-verification/request", nil, map[string]string{"email": "worker@gopenid.local"}, http.StatusOK)
	verifyToken := stringField(t, verifyReq, "verificationToken")
	postJSON(t, app, "/api/auth/email-verification/confirm", nil, map[string]string{"token": verifyToken}, http.StatusOK)
	workerVerified := getJSON(t, app, fmt.Sprintf("/api/admin/users/%d", workerID), bearer(adminToken), http.StatusOK)
	if workerVerified["emailVerified"] != true {
		t.Fatalf("worker email should be verified: %#v", workerVerified)
	}

	// --- Groups + multiple departments ---------------------------------------
	group := postJSON(t, app, "/api/admin/groups", bearer(adminToken), map[string]string{"name": "operators", "description": "Ops"}, http.StatusCreated)
	groupID := int64(group["ID"].(float64))
	dept2 := postJSON(t, app, "/api/admin/departments", bearer(adminToken), map[string]string{"name": "Support", "description": "Support desk"}, http.StatusCreated)
	dept2ID := int64(dept2["ID"].(float64))

	putJSON(t, app, fmt.Sprintf("/api/admin/users/%d", workerID), bearer(adminToken), map[string]any{
		"email": "worker@gopenid.local", "name": "Worker", "password": "", "active": true,
		"groupIds": []int64{groupID}, "departmentIds": []int64{1, dept2ID},
	}, http.StatusOK)
	workerRow := getJSON(t, app, fmt.Sprintf("/api/admin/users/%d", workerID), bearer(adminToken), http.StatusOK)
	if got := len(workerRow["departments"].([]any)); got != 2 {
		t.Fatalf("worker should have 2 departments, got %d", got)
	}
	if got := len(workerRow["groups"].([]any)); got != 1 {
		t.Fatalf("worker should have 1 group, got %d", got)
	}

	// --- Client access control ------------------------------------------------
	client := postJSON(t, app, "/api/admin/clients", bearer(adminToken), map[string]any{
		"clientId": "portal", "clientSecret": "portal-secret", "name": "Portal",
		"redirectUris": "http://localhost:3000/callback", "homeUrl": "http://localhost:3000", "tokenTtlSeconds": 3600,
		"allowPasswordGrant": true,
	}, http.StatusCreated)
	if client["clientSecret"] != "portal-secret" {
		t.Fatalf("new client should return secret once, got %#v", client["clientSecret"])
	}
	clientDBID := int64(client["ID"].(float64))
	for _, item := range getArray(t, app, "/api/admin/clients", bearer(adminToken), http.StatusOK) {
		row := item.(map[string]any)
		if row["clientId"] == "portal" {
			if _, ok := row["clientSecret"]; ok {
				t.Fatalf("client list must not return secret: %#v", row)
			}
		}
	}

	tokenValues := url.Values{
		"grant_type": {"password"}, "username": {"worker@gopenid.local"}, "password": {"worker67890"},
		"client_id": {"portal"}, "client_secret": {"portal-secret"}, "scope": {"openid profile"},
	}
	// Not authorized for this client yet -> forbidden.
	formStatus(t, app, "/oauth/token", tokenValues, http.StatusForbidden)
	// Authorize the user for the client.
	putJSON(t, app, fmt.Sprintf("/api/admin/users/%d", workerID), bearer(adminToken), map[string]any{
		"email": "worker@gopenid.local", "name": "Worker", "password": "", "active": true,
		"groupIds": []int64{groupID}, "departmentIds": []int64{1, dept2ID}, "clientIds": []int64{clientDBID},
	}, http.StatusOK)
	formStatus(t, app, "/oauth/token", tokenValues, http.StatusOK)

	// --- Login policies (hierarchy) ------------------------------------------
	allDays := []int{0, 1, 2, 3, 4, 5, 6}
	denyPolicy := postJSON(t, app, "/api/admin/policies", bearer(adminToken), map[string]any{
		"name": "client-curfew", "type": "time", "effect": "deny",
		"daysOfWeek": allDays, "startTime": "00:00", "endTime": "23:59",
	}, http.StatusCreated)
	denyID := int64(denyPolicy["ID"].(float64))
	postJSON(t, app, fmt.Sprintf("/api/admin/policies/%d/assignments", denyID), bearer(adminToken), map[string]any{
		"subjectType": "client", "subjectId": clientDBID,
	}, http.StatusCreated)
	// Client-level deny policy now blocks the token request.
	formStatus(t, app, "/oauth/token", tokenValues, http.StatusForbidden)

	allowPolicy := postJSON(t, app, "/api/admin/policies", bearer(adminToken), map[string]any{
		"name": "user-allow", "type": "time", "effect": "allow",
		"daysOfWeek": allDays, "startTime": "00:00", "endTime": "23:59",
	}, http.StatusCreated)
	allowID := int64(allowPolicy["ID"].(float64))
	postJSON(t, app, fmt.Sprintf("/api/admin/policies/%d/assignments", allowID), bearer(adminToken), map[string]any{
		"subjectType": "user", "subjectId": workerID,
	}, http.StatusCreated)
	// User-level allow overrides the client deny.
	formStatus(t, app, "/oauth/token", tokenValues, http.StatusOK)

	// --- IP policy ------------------------------------------------------------
	ipDeny := postJSON(t, app, "/api/admin/policies", bearer(adminToken), map[string]any{
		"name": "ip-block", "type": "ip", "effect": "deny", "ipCidrs": "203.0.113.0/24",
	}, http.StatusCreated)
	ipDenyID := int64(ipDeny["ID"].(float64))
	postJSON(t, app, fmt.Sprintf("/api/admin/policies/%d/assignments", ipDenyID), bearer(adminToken), map[string]any{
		"subjectType": "user", "subjectId": workerID,
	}, http.StatusCreated)
	// Spoofed forwarding headers are ignored unless the proxy is trusted.
	formStatusWithHeaders(t, app, "/oauth/token", tokenValues, map[string]string{"X-Forwarded-For": "203.0.113.10"}, http.StatusOK)

	// --- Audit logging --------------------------------------------------------
	auditPage := getJSON(t, app, "/api/admin/audit-logs?pageSize=100", bearer(adminToken), http.StatusOK)
	logs := auditPage["items"].([]any)
	if len(logs) == 0 {
		t.Fatal("expected audit logs to be recorded")
	}
	if !auditHasEvent(logs, "login") || !auditHasEvent(logs, "access_denied") {
		t.Fatalf("expected login and access_denied audit events, got %d entries", len(logs))
	}
	if !auditHasEvent(logs, "logout") {
		t.Fatalf("expected logout audit event, got %d entries", len(logs))
	}
	firstPage := getJSON(t, app, "/api/admin/audit-logs?page=1&pageSize=2&event=login", bearer(adminToken), http.StatusOK)
	if int(firstPage["pageSize"].(float64)) != 2 || len(firstPage["items"].([]any)) > 2 || firstPage["total"].(float64) < 1 {
		t.Fatalf("audit pagination response invalid: %#v", firstPage)
	}

	locked := postJSON(t, app, "/api/admin/users", bearer(adminToken), map[string]any{
		"email": "locked@gopenid.local", "name": "Locked", "password": "locked12345", "active": true,
	}, http.StatusCreated)
	_ = locked
	for range 5 {
		postJSON(t, app, "/api/auth/login", nil, map[string]string{"email": "locked@gopenid.local", "password": "bad"}, http.StatusUnauthorized)
	}
	postJSON(t, app, "/api/auth/login", nil, map[string]string{"email": "locked@gopenid.local", "password": "locked12345"}, http.StatusUnauthorized)
}

// --- Test helpers ------------------------------------------------------------

func getArray(t *testing.T, app *fiber.App, path string, headers map[string]string, want int) []any {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("GET %s: %v", path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("GET %s status=%d want=%d body=%s", path, res.StatusCode, want, string(raw))
	}
	return decodeArray(t, raw)
}

func formStatus(t *testing.T, app *fiber.App, path string, values url.Values, want int) {
	t.Helper()
	formStatusWithHeaders(t, app, path, values, nil, want)
}

func formStatusWithHeaders(t *testing.T, app *fiber.App, path string, values url.Values, headers map[string]string, want int) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(values.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req)
	if err != nil {
		t.Fatalf("POST %s: %v", path, err)
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(res.Body)
	if res.StatusCode != want {
		t.Fatalf("POST %s status=%d want=%d body=%s", path, res.StatusCode, want, string(raw))
	}
}

func arrayHasName(items []any, name string) bool {
	for _, item := range items {
		if m, ok := item.(map[string]any); ok && m["name"] == name {
			return true
		}
	}
	return false
}

func auditHasEvent(items []any, event string) bool {
	for _, item := range items {
		if m, ok := item.(map[string]any); ok && m["event"] == event {
			return true
		}
	}
	return false
}

func decodeArray(t *testing.T, raw []byte) []any {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}
	var out []any
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("decode json array: %v body=%s", err, string(raw))
	}
	return out
}
