package oidc

import (
	"crypto/sha256"
	"encoding/base64"
	"net"
	"net/url"
	"strconv"
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

const ssoCookieName = "gopenid_sso"

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
	app.Get("/oauth/logout", h.endSession)
	app.Post("/oauth/logout", h.endSession)
}

func (h *Handler) discovery(c fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"issuer":                                h.cfg.Issuer,
		"authorization_endpoint":                h.cfg.Issuer + "/oauth/authorize",
		"token_endpoint":                        h.cfg.Issuer + "/oauth/token",
		"userinfo_endpoint":                     h.cfg.Issuer + "/oauth/userinfo",
		"end_session_endpoint":                  h.cfg.Issuer + "/oauth/logout",
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
		client, err := h.db.GetClientByClientID(c.Context(), clientID)
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
		decision, derr := h.policy.EvaluateClient(c.Context(), client.ID, h.clientIP(c), time.Now())
		if derr == nil && !decision.Allowed {
			page.ShowForm = false
			page.Notice = decision.Reason
			page.NoticeError = true
		}
	}
	page.Email = c.Query("login_hint")
	if clientID != "" && c.Query("prompt") != "login" {
		if redirected, err := h.trySSOAuthorize(c, clientID); redirected || err != nil {
			return err
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
		client, clientErr = h.db.GetClientByClientID(c.Context(), clientID)
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

	user, err := h.svc.Authenticate(c.Context(), email, c.FormValue("password"), c.FormValue("totp_code"))
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

	ok, err := h.db.UserAuthorizedForClient(c.Context(), user.ID, client.ID)
	if err != nil || !ok {
		h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: "user not authorized for client"})
		return render("Bu uygulamaya erişim yetkiniz bulunmuyor. Lütfen yöneticinizle iletişime geçin.", false)
	}

	decision, derr := h.policy.EvaluateLogin(c.Context(), user, client.ID, h.clientIP(c), time.Now())
	if derr == nil && !decision.Allowed {
		h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: decision.Reason})
		return render(decision.Reason, false)
	}

	authTime := time.Now()
	if err := h.setSSOCookie(c, user.ID, authTime); err != nil {
		return httpx.Error(c, 500, "session failed")
	}
	_ = h.db.TouchLastLogin(c.Context(), user.ID)
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventLogin, Success: true, Message: "authorization_code"})
	return h.issueAuthCodeRedirect(c, user.ID, clientID, c.FormValue("redirect_uri"), c.FormValue("scope"), c.FormValue("nonce"), c.FormValue("code_challenge"), c.FormValue("code_challenge_method"), c.FormValue("state"))
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
	user, err := h.svc.Authenticate(c.Context(), c.FormValue("username"), c.FormValue("password"), c.FormValue("totp_code"))
	if err != nil {
		h.recorder.Record(c, audit.Entry{Email: c.FormValue("username"), ClientID: c.FormValue("client_id"), Event: domain.EventLoginFailed, Success: false, Message: err.Error()})
		return httpx.Error(c, fiber.StatusUnauthorized, authErrorMessage(err))
	}
	clientID := c.FormValue("client_id")
	var client domain.Client
	if clientID != "" {
		found, err := h.db.GetClientByClientID(c.Context(), clientID)
		if err != nil {
			return httpx.BadRequest(c, "invalid client_id")
		}
		client = found
		if !client.AllowPasswordGrant {
			return httpx.BadRequest(c, "password grant is disabled for this client")
		}
		if clientSecret := c.FormValue("client_secret"); clientSecret != "" && !store.VerifyClientSecret(client.ClientSecret, clientSecret) {
			return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
		}
		ok, err := h.db.UserAuthorizedForClient(c.Context(), user.ID, client.ID)
		if err != nil || !ok {
			h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: "user not authorized for client"})
			return httpx.Error(c, fiber.StatusForbidden, "Bu uygulamaya erişim yetkiniz bulunmuyor.")
		}
		decision, derr := h.policy.EvaluateLogin(c.Context(), user, client.ID, h.clientIP(c), time.Now())
		if derr == nil && !decision.Allowed {
			h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: clientID, Event: domain.EventAccessDenied, Success: false, Message: decision.Reason})
			return httpx.Error(c, fiber.StatusForbidden, decision.Reason)
		}
	}
	return h.tokenResponse(c, user, client, "", clientID, c.FormValue("scope"))
}

func (h *Handler) codeGrant(c fiber.Ctx) error {
	code, err := h.db.GetUnusedAuthCode(c.Context(), c.FormValue("code"))
	if err != nil || code.RedirectURI != c.FormValue("redirect_uri") || time.Since(code.CreatedAt) > 10*time.Minute {
		return httpx.BadRequest(c, "invalid code")
	}
	if clientID := c.FormValue("client_id"); clientID != "" && clientID != code.ClientID {
		return httpx.BadRequest(c, "client_id does not match authorization code")
	}
	if err := verifyPKCE(code, c.FormValue("code_verifier")); err != nil {
		return httpx.BadRequest(c, err.Error())
	}
	user, err := h.db.GetUser(c.Context(), code.UserID)
	if err != nil {
		return httpx.BadRequest(c, "invalid user")
	}
	var client domain.Client
	if code.ClientID != "" {
		if found, ferr := h.db.GetClientByClientID(c.Context(), code.ClientID); ferr == nil {
			client = found
		}
	}
	_ = h.db.MarkAuthCodeUsed(c.Context(), code.ID)
	return h.tokenResponse(c, user, client, code.Nonce, code.ClientID, code.Scope, auth.TokenOptions{Code: c.FormValue("code"), AuthTime: &code.CreatedAt})
}

func (h *Handler) refreshGrant(c fiber.Ctx) error {
	raw := c.FormValue("refresh_token")
	if raw == "" {
		return httpx.BadRequest(c, "refresh_token is required")
	}
	user, stored, err := h.svc.ConsumeRefreshToken(c.Context(), raw)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	var client domain.Client
	if stored.ClientID != "" && stored.ClientID != "gopenid" {
		if found, ferr := h.db.GetClientByClientID(c.Context(), stored.ClientID); ferr == nil {
			client = found
		}
	}
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: stored.ClientID, Event: domain.EventTokenRefresh, Success: true})
	return h.tokenResponse(c, user, client, "", stored.ClientID, stored.Scope)
}

func (h *Handler) tokenResponse(c fiber.Ctx, user domain.User, client domain.Client, nonce, clientID string, scope string, options ...auth.TokenOptions) error {
	tokens, err := h.svc.IssueTokens(c.Context(), user, client, clientID, scope, nonce, true, options...)
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
	claims, err := h.svc.Verify(c.Context(), auth.BearerToken(c))
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
	}
	scope, _ := claims["scope"].(string)
	resp := fiber.Map{"sub": claims["sub"]}
	if auth.ScopeContains(scope, "email") {
		resp["email"] = claims["email"]
	}
	if auth.ScopeContains(scope, "profile") {
		resp["name"] = claims["name"]
		resp["department"] = claims["department"]
	}
	if auth.ScopeContains(scope, "roles") {
		resp["roles"] = claims["roles"]
	}
	if auth.ScopeContains(scope, "client_roles") {
		resp["client_roles"] = claims["client_roles"]
		resp["resource_access"] = claims["resource_access"]
	}
	return c.JSON(resp)
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
	client, err := h.db.GetClientByClientID(c.Context(), clientID)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
	}
	if client.ClientSecret != "" && !store.VerifyClientSecret(client.ClientSecret, clientSecret) {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
	}
	return nil
}

func (h *Handler) endSession(c fiber.Ctx) error {
	if raw := c.Cookies(ssoCookieName); raw != "" {
		_ = h.db.RevokeBrowserSession(c.Context(), auth.HashToken(raw))
		c.Cookie(&fiber.Cookie{Name: ssoCookieName, Value: "", Path: "/", HTTPOnly: true, SameSite: "Lax", Expires: time.Unix(0, 0), MaxAge: -1})
	}
	tokenText := auth.BearerToken(c)
	if tokenText != "" {
		if claims, err := h.svc.Verify(c.Context(), tokenText); err == nil {
			_ = h.svc.RevokeAccessClaims(c.Context(), claims, "rp_logout")
			email, _ := claims["email"].(string)
			var uid *int64
			if raw, ok := claims["uid"].(float64); ok {
				id := int64(raw)
				uid = &id
			}
			clientID, _ := claims["aud"].(string)
			h.recorder.Record(c, audit.Entry{UserID: uid, Email: email, ClientID: clientID, Event: domain.EventLogout, Success: true, Message: "rp-initiated logout"})
		}
	}
	redirectURI := c.Query("post_logout_redirect_uri")
	if redirectURI == "" {
		redirectURI = c.FormValue("post_logout_redirect_uri")
	}
	if redirectURI == "" {
		return c.JSON(fiber.Map{"message": "logged out"})
	}
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = c.FormValue("client_id")
	}
	if clientID == "" {
		return httpx.BadRequest(c, "client_id is required for post_logout_redirect_uri")
	}
	client, err := h.db.GetClientByClientID(c.Context(), clientID)
	if err != nil || !redirectAllowed(client.RedirectURIs, redirectURI) {
		return httpx.BadRequest(c, "post_logout_redirect_uri not allowed")
	}
	target, _ := url.Parse(redirectURI)
	q := target.Query()
	if state := c.Query("state"); state != "" {
		q.Set("state", state)
	}
	target.RawQuery = q.Encode()
	return c.Redirect().To(target.String())
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

func (h *Handler) trySSOAuthorize(c fiber.Ctx, clientID string) (bool, error) {
	raw := c.Cookies(ssoCookieName)
	if raw == "" {
		if c.Query("prompt") == "none" {
			return true, httpx.Error(c, fiber.StatusUnauthorized, "login_required")
		}
		return false, nil
	}
	session, err := h.db.GetBrowserSession(c.Context(), auth.HashToken(raw))
	if err != nil {
		return false, nil
	}
	if rawMaxAge := c.Query("max_age"); rawMaxAge != "" {
		maxAge, err := strconv.Atoi(rawMaxAge)
		if err == nil && maxAge >= 0 && time.Since(session.AuthTime) > time.Duration(maxAge)*time.Second {
			return false, nil
		}
	}
	user, err := h.db.GetUser(c.Context(), session.UserID)
	if err != nil || !user.Active || user.Blocked {
		return false, nil
	}
	client, err := h.db.GetClientByClientID(c.Context(), clientID)
	if err != nil || !redirectAllowed(client.RedirectURIs, c.Query("redirect_uri")) {
		return false, nil
	}
	ok, err := h.db.UserAuthorizedForClient(c.Context(), user.ID, client.ID)
	if err != nil || !ok {
		return false, nil
	}
	decision, derr := h.policy.EvaluateLogin(c.Context(), user, client.ID, h.clientIP(c), time.Now())
	if derr == nil && !decision.Allowed {
		return false, nil
	}
	return true, h.issueAuthCodeRedirect(c, user.ID, clientID, c.Query("redirect_uri"), c.Query("scope"), c.Query("nonce"), c.Query("code_challenge"), c.Query("code_challenge_method"), c.Query("state"))
}

func (h *Handler) setSSOCookie(c fiber.Ctx, userID int64, authTime time.Time) error {
	raw, err := auth.RandomToken(32)
	if err != nil {
		return err
	}
	expires := time.Now().Add(12 * time.Hour)
	if err := h.db.CreateBrowserSession(c.Context(), domain.BrowserSession{
		TokenHash: auth.HashToken(raw), UserID: userID, AuthTime: authTime, ExpiresAt: expires,
	}); err != nil {
		return err
	}
	c.Cookie(&fiber.Cookie{Name: ssoCookieName, Value: raw, Path: "/", HTTPOnly: true, SameSite: "Lax", Expires: expires})
	return nil
}

func (h *Handler) issueAuthCodeRedirect(c fiber.Ctx, userID int64, clientID, redirectURI, scope, nonce, challenge, challengeMethod, state string) error {
	code, err := auth.RandomToken(32)
	if err != nil {
		return httpx.Error(c, 500, "code failed")
	}
	row := domain.AuthCode{
		Code: code, UserID: userID, ClientID: clientID,
		RedirectURI: redirectURI, Scope: scope,
		Nonce: nonce, CodeChallenge: challenge, CodeChallengeMethod: challengeMethod,
	}
	if err := h.db.CreateAuthCode(c.Context(), row); err != nil {
		return httpx.Error(c, 500, "code save failed")
	}
	target, _ := url.Parse(row.RedirectURI)
	q := target.Query()
	q.Set("code", code)
	if state != "" {
		q.Set("state", state)
	}
	target.RawQuery = q.Encode()
	return c.Redirect().To(target.String())
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
