package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/url"
	"strings"
	"time"

	"gopenid/internal/audit"
	"gopenid/internal/auth"
	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/keys"
	"gopenid/internal/policy"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	db       *store.Store
	cfg      config.Config
	svc      *auth.Service
	key      *keys.Manager
	policy   *policy.Engine
	recorder *audit.Recorder
}

func New(db *store.Store, cfg config.Config, svc *auth.Service, keyManager *keys.Manager, policyEngine *policy.Engine, recorder *audit.Recorder) *Handler {
	return &Handler{db: db, cfg: cfg, svc: svc, key: keyManager, policy: policyEngine, recorder: recorder}
}

func (h *Handler) Mount(app *fiber.App) {
	app.Get("/.well-known/openid-configuration", h.discovery)
	app.Get("/.well-known/jwks.json", h.jwks)
	app.Get("/oauth/jwks", h.jwks)
	app.Get("/oauth/authorize", h.authorizePage)
	app.Post("/oauth/authorize", h.authorize)
	app.Post("/oauth/token", h.token)
	app.Get("/oauth/userinfo", h.userinfo)
	app.Post("/oauth/userinfo", h.userinfo)
}

func (h *Handler) discovery(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"issuer":                                h.cfg.Issuer,
		"authorization_endpoint":                h.cfg.Issuer + "/oauth/authorize",
		"token_endpoint":                        h.cfg.Issuer + "/oauth/token",
		"userinfo_endpoint":                     h.cfg.Issuer + "/oauth/userinfo",
		"jwks_uri":                              h.cfg.Issuer + "/.well-known/jwks.json",
		"response_types_supported":              []string{"code"},
		"grant_types_supported":                 []string{"authorization_code", "password", "refresh_token"},
		"scopes_supported":                      []string{"openid", "profile", "email", "roles", "client_roles", "offline_access"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "jti", "email", "name", "roles", "department"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *Handler) jwks(c fiber.Ctx) error {
	return c.JSON(h.key.JWKS())
}

func (h *Handler) clientIP(c fiber.Ctx) net.IP {
	ipText, _ := audit.RequestContext(c)
	return net.ParseIP(ipText)
}

// authorizePage renders the branded login screen for an authorization request.
func (h *Handler) authorizePage(c fiber.Ctx) error {
	action := c.OriginalURL()
	clientID := c.Query("client_id")
	page := loginPage{Action: action, ShowForm: true}

	if clientID != "" {
		client, err := h.db.GetClientByClientID(context.Background(), clientID)
		if err != nil {
			page.ShowForm = false
			page.Notice = "Geçersiz uygulama (client_id). Lütfen bağlantıyı kontrol edin."
			page.NoticeError = true
			return c.Type("html").SendString(renderLoginPage(page))
		}
		page.ClientName = client.Name
		page.ClientLogo = client.LogoURL
		page.ClientHome = client.HomeURL

		// Application-level policies can be evaluated before authentication so a
		// time/IP restriction is shown immediately.
		decision, derr := h.policy.EvaluateClient(context.Background(), client.ID, h.clientIP(c), time.Now())
		if derr == nil && !decision.Allowed {
			page.ShowForm = false
			page.Notice = decision.Reason
			page.NoticeError = true
		}
	}
	return c.Type("html").SendString(renderLoginPage(page))
}

func (h *Handler) authorize(c fiber.Ctx) error {
	email := c.FormValue("email")
	action := c.OriginalURL()
	clientID := c.FormValue("client_id")

	var client domain.Client
	var clientErr error
	if clientID != "" {
		client, clientErr = h.db.GetClientByClientID(context.Background(), clientID)
	}

	render := func(message string, showForm bool) error {
		page := loginPage{Action: action, Email: email, Notice: message, NoticeError: true, ShowForm: showForm}
		if clientErr == nil && client.ID != 0 {
			page.ClientName = client.Name
			page.ClientLogo = client.LogoURL
			page.ClientHome = client.HomeURL
		}
		return c.Status(fiber.StatusUnauthorized).Type("html").SendString(renderLoginPage(page))
	}

	user, err := h.svc.Authenticate(email, c.FormValue("password"))
	if err != nil {
		h.recorder.Record(c, audit.Entry{Email: email, ClientID: clientID, Event: domain.EventLoginFailed, Success: false, Message: err.Error()})
		return render(authErrorMessage(err), err != auth.ErrUserBlocked)
	}
	if c.FormValue("response_type") != "" && c.FormValue("response_type") != "code" {
		return httpx.BadRequest(c, "unsupported response_type")
	}
	if scope := c.FormValue("scope"); scope != "" && !strings.Contains(" "+scope+" ", " openid ") {
		return httpx.BadRequest(c, "scope must include openid")
	}
	if clientErr != nil {
		return render("Geçersiz uygulama (client_id).", false)
	}

	if !redirectAllowed(client.RedirectURIs, c.FormValue("redirect_uri")) {
		return httpx.BadRequest(c, "redirect_uri not allowed")
	}

	ok, err := h.db.UserAuthorizedForClient(context.Background(), user.ID, client.ID)
	if err != nil || !ok {
		h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: "user not authorized for client"})
		return render("Bu uygulamaya erişim yetkiniz bulunmuyor. Lütfen yöneticinizle iletişime geçin.", false)
	}

	decision, derr := h.policy.EvaluateLogin(context.Background(), user, client.ID, h.clientIP(c), time.Now())
	if derr == nil && !decision.Allowed {
		h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: decision.Reason})
		return render(decision.Reason, false)
	}

	code, err := auth.RandomToken(32)
	if err != nil {
		return httpx.Error(c, 500, "code failed")
	}
	row := domain.AuthCode{
		Code: code, UserID: user.ID, ClientID: clientID,
		RedirectURI: c.FormValue("redirect_uri"), Scope: c.FormValue("scope"),
		Nonce: c.FormValue("nonce"), CodeChallenge: c.FormValue("code_challenge"),
		CodeChallengeMethod: c.FormValue("code_challenge_method"),
	}
	if err := h.db.CreateAuthCode(context.Background(), row); err != nil {
		return httpx.Error(c, 500, "code save failed")
	}
	_ = h.db.TouchLastLogin(context.Background(), user.ID)
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventLogin, Success: true, Message: "authorization_code"})

	target, _ := url.Parse(row.RedirectURI)
	q := target.Query()
	q.Set("code", code)
	if state := c.FormValue("state"); state != "" {
		q.Set("state", state)
	}
	target.RawQuery = q.Encode()
	return c.Redirect().To(target.String())
}

func (h *Handler) token(c fiber.Ctx) error {
	if err := h.authenticateClient(c); err != nil {
		return err
	}
	switch c.FormValue("grant_type") {
	case "password":
		return h.passwordGrant(c)
	case "authorization_code":
		return h.codeGrant(c)
	case "refresh_token":
		return h.refreshGrant(c)
	default:
		return httpx.BadRequest(c, "unsupported grant_type")
	}
}

func (h *Handler) passwordGrant(c fiber.Ctx) error {
	user, err := h.svc.Authenticate(c.FormValue("username"), c.FormValue("password"))
	if err != nil {
		h.recorder.Record(c, audit.Entry{Email: c.FormValue("username"), ClientID: c.FormValue("client_id"), Event: domain.EventLoginFailed, Success: false, Message: err.Error()})
		return httpx.Error(c, fiber.StatusUnauthorized, authErrorMessage(err))
	}
	clientID := c.FormValue("client_id")
	var client domain.Client
	if clientID != "" {
		found, err := h.db.GetClientByClientID(context.Background(), clientID)
		if err != nil {
			return httpx.BadRequest(c, "invalid client_id")
		}
		client = found
		if clientSecret := c.FormValue("client_secret"); clientSecret != "" && client.ClientSecret != clientSecret {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
		}
		ok, err := h.db.UserAuthorizedForClient(context.Background(), user.ID, client.ID)
		if err != nil || !ok {
			h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: "user not authorized for client"})
			return httpx.Error(c, fiber.StatusForbidden, "Bu uygulamaya erişim yetkiniz bulunmuyor.")
		}
		decision, derr := h.policy.EvaluateLogin(context.Background(), user, client.ID, h.clientIP(c), time.Now())
		if derr == nil && !decision.Allowed {
			h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: decision.Reason})
			return httpx.Error(c, fiber.StatusForbidden, decision.Reason)
		}
	}
	return h.tokenResponse(c, user, client, "", clientID, c.FormValue("scope"))
}

func (h *Handler) codeGrant(c fiber.Ctx) error {
	code, err := h.db.GetUnusedAuthCode(context.Background(), c.FormValue("code"))
	if err != nil || code.RedirectURI != c.FormValue("redirect_uri") || time.Since(code.CreatedAt) > 10*time.Minute {
		return httpx.BadRequest(c, "invalid code")
	}
	if clientID := c.FormValue("client_id"); clientID != "" && clientID != code.ClientID {
		return httpx.BadRequest(c, "client_id does not match authorization code")
	}
	if err := verifyPKCE(code, c.FormValue("code_verifier")); err != nil {
		return httpx.BadRequest(c, err.Error())
	}
	user, err := h.db.GetUser(context.Background(), code.UserID)
	if err != nil {
		return httpx.BadRequest(c, "invalid user")
	}
	var client domain.Client
	if code.ClientID != "" {
		if found, ferr := h.db.GetClientByClientID(context.Background(), code.ClientID); ferr == nil {
			client = found
		}
	}
	_ = h.db.MarkAuthCodeUsed(context.Background(), code.ID)
	return h.tokenResponse(c, user, client, code.Nonce, code.ClientID, code.Scope)
}

func (h *Handler) refreshGrant(c fiber.Ctx) error {
	raw := c.FormValue("refresh_token")
	if raw == "" {
		return httpx.BadRequest(c, "refresh_token is required")
	}
	user, stored, err := h.svc.ConsumeRefreshToken(context.Background(), raw)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	var client domain.Client
	if stored.ClientID != "" && stored.ClientID != "gopenid" {
		if found, ferr := h.db.GetClientByClientID(context.Background(), stored.ClientID); ferr == nil {
			client = found
		}
	}
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: stored.ClientID, Event: domain.EventTokenRefresh, Success: true})
	return h.tokenResponse(c, user, client, "", stored.ClientID, stored.Scope)
}

func (h *Handler) tokenResponse(c fiber.Ctx, user domain.User, client domain.Client, nonce, clientID string, scopes ...string) error {
	scope := strings.Join(scopes, " ")
	tokens, err := h.svc.IssueTokens(context.Background(), user, client, clientID, scope, nonce, true)
	if err != nil {
		return httpx.Error(c, 500, "token failed")
	}
	resp := fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokens.ExpiresIn,
		"scope":         tokens.Scope,
	}
	if tokens.IDToken != "" {
		resp["id_token"] = tokens.IDToken
	}
	return c.JSON(resp)
}

func (h *Handler) userinfo(c fiber.Ctx) error {
	claims, err := h.svc.Verify(context.Background(), auth.BearerToken(c))
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
	}
	return c.JSON(fiber.Map{
		"sub":             claims["sub"],
		"email":           claims["email"],
		"name":            claims["name"],
		"roles":           claims["roles"],
		"department":      claims["department"],
		"client_roles":    claims["client_roles"],
		"resource_access": claims["resource_access"],
	})
}

func (h *Handler) authenticateClient(c fiber.Ctx) error {
	clientID := c.FormValue("client_id")
	clientSecret := c.FormValue("client_secret")
	if clientID == "" {
		clientID, clientSecret = basicAuth(c.Get("Authorization"))
	}
	if clientID == "" {
		return nil
	}
	client, err := h.db.GetClientByClientID(context.Background(), clientID)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
	}
	if client.ClientSecret != "" && client.ClientSecret != clientSecret {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
	}
	return nil
}

func basicAuth(header string) (string, string) {
	if !strings.HasPrefix(header, "Basic ") {
		return "", ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(header, "Basic "))
	if err != nil {
		return "", ""
	}
	user, pass, ok := strings.Cut(string(decoded), ":")
	if !ok {
		return "", ""
	}
	return user, pass
}

func redirectAllowed(allowed, requested string) bool {
	for _, u := range strings.Split(allowed, ",") {
		if strings.TrimSpace(u) == requested {
			return true
		}
	}
	return false
}

func verifyPKCE(code domain.AuthCode, verifier string) error {
	if code.CodeChallenge == "" {
		return nil
	}
	if verifier == "" {
		return fiber.NewError(fiber.StatusBadRequest, "code_verifier is required")
	}
	method := code.CodeChallengeMethod
	if method == "" {
		method = "plain"
	}
	switch method {
	case "plain":
		if verifier != code.CodeChallenge {
			return fiber.NewError(fiber.StatusBadRequest, "invalid code_verifier")
		}
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		if base64.RawURLEncoding.EncodeToString(sum[:]) != code.CodeChallenge {
			return fiber.NewError(fiber.StatusBadRequest, "invalid code_verifier")
		}
	default:
		return fiber.NewError(fiber.StatusBadRequest, "unsupported code_challenge_method")
	}
	return nil
}

func authErrorMessage(err error) string {
	switch err {
	case auth.ErrUserInactive:
		return "Hesabınız pasif durumda. Lütfen yöneticinizle iletişime geçin."
	case auth.ErrUserBlocked:
		return "Hesabınız engellenmiş. Lütfen yöneticinizle iletişime geçin."
	default:
		return "E-posta veya parola hatalı."
	}
}
