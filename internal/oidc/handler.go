package oidc

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"net/url"
	"strings"
	"time"

	"gopenid/internal/auth"
	"gopenid/internal/config"
	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/keys"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
)

type Handler struct {
	db  *store.Store
	cfg config.Config
	svc *auth.Service
	key *keys.Manager
}

func New(db *store.Store, cfg config.Config, svc *auth.Service, keyManager *keys.Manager) *Handler {
	return &Handler{db: db, cfg: cfg, svc: svc, key: keyManager}
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
		"grant_types_supported":                 []string{"authorization_code", "password"},
		"scopes_supported":                      []string{"openid", "profile", "email", "roles", "client_roles"},
		"claims_supported":                      []string{"sub", "iss", "aud", "exp", "iat", "email", "name", "roles", "department"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_post", "client_secret_basic", "none"},
		"code_challenge_methods_supported":      []string{"S256", "plain"},
	})
}

func (h *Handler) jwks(c fiber.Ctx) error {
	return c.JSON(h.key.JWKS())
}

func (h *Handler) authorizePage(c fiber.Ctx) error {
	return c.Type("html").SendString(authForm(c.OriginalURL()))
}

func (h *Handler) authorize(c fiber.Ctx) error {
	user, err := h.svc.Authenticate(c.FormValue("email"), c.FormValue("password"))
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	if c.FormValue("response_type") != "" && c.FormValue("response_type") != "code" {
		return httpx.BadRequest(c, "unsupported response_type")
	}
	if scope := c.FormValue("scope"); scope != "" && !strings.Contains(" "+scope+" ", " openid ") {
		return httpx.BadRequest(c, "scope must include openid")
	}

	clientID := c.FormValue("client_id")
	client, err := h.db.GetClientByClientID(context.Background(), clientID)
	if err != nil {
		return httpx.BadRequest(c, "invalid client_id")
	}

	// Verify redirect URI
	uriAllowed := false
	allowedURIs := strings.Split(client.RedirectURIs, ",")
	reqURI := c.FormValue("redirect_uri")
	for _, u := range allowedURIs {
		if strings.TrimSpace(u) == reqURI {
			uriAllowed = true
			break
		}
	}
	if !uriAllowed {
		return httpx.BadRequest(c, "redirect_uri not allowed")
	}

	// Verify user is authorized for this client
	ok, err := h.db.UserAuthorizedForClient(context.Background(), user.ID, client.ID)
	if err != nil || !ok {
		return httpx.Error(c, fiber.StatusForbidden, "user not authorized for this client")
	}

	code, err := auth.RandomToken(32)
	if err != nil {
		return httpx.Error(c, 500, "code failed")
	}
	row := domain.AuthCode{
		Code: code, UserID: user.ID, ClientID: clientID,
		RedirectURI: reqURI, Scope: c.FormValue("scope"),
		Nonce: c.FormValue("nonce"), CodeChallenge: c.FormValue("code_challenge"),
		CodeChallengeMethod: c.FormValue("code_challenge_method"),
	}
	if err := h.db.CreateAuthCode(context.Background(), row); err != nil {
		return httpx.Error(c, 500, "code save failed")
	}
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
	default:
		return httpx.BadRequest(c, "unsupported grant_type")
	}
}

func (h *Handler) passwordGrant(c fiber.Ctx) error {
	user, err := h.svc.Authenticate(c.FormValue("username"), c.FormValue("password"))
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	clientID := c.FormValue("client_id")
	if clientID != "" {
		client, err := h.db.GetClientByClientID(context.Background(), clientID)
		if err == nil {
			if clientSecret := c.FormValue("client_secret"); clientSecret != "" && client.ClientSecret != clientSecret {
				return httpx.Error(c, fiber.StatusUnauthorized, "invalid client credentials")
			}
			ok, err := h.db.UserAuthorizedForClient(context.Background(), user.ID, client.ID)
			if err != nil || !ok {
				return httpx.Error(c, fiber.StatusForbidden, "user not authorized for this client")
			}
		} else {
			return httpx.BadRequest(c, "invalid client_id")
		}
	}
	return h.tokenResponse(c, user, "", clientID, c.FormValue("scope"))
}

func (h *Handler) codeGrant(c fiber.Ctx) error {
	var code domain.AuthCode
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
	h.db.MarkAuthCodeUsed(context.Background(), code.ID)
	return h.tokenResponse(c, user, code.Nonce, code.ClientID, code.Scope)
}

func (h *Handler) tokenResponse(c fiber.Ctx, user domain.User, nonce string, clientID string, scopes ...string) error {
	scope := strings.Join(scopes, " ")
	accessToken, err := h.svc.AccessToken(user, clientID, scope)
	if err != nil {
		return httpx.Error(c, 500, "token failed")
	}
	idToken, err := h.svc.IDToken(user, clientID, scope, nonce)
	if err != nil {
		return httpx.Error(c, 500, "token failed")
	}
	return c.JSON(fiber.Map{
		"access_token": accessToken, "id_token": idToken,
		"token_type": "Bearer", "expires_in": int(h.cfg.TokenTTL.Seconds()),
		"scope": strings.TrimSpace(scope),
	})
}

func (h *Handler) userinfo(c fiber.Ctx) error {
	claims, err := h.parseBearer(c)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
	}
	return c.JSON(fiber.Map{
		"sub":        claims["sub"],
		"email":      claims["email"],
		"name":       claims["name"],
		"roles":      claims["roles"],
		"department": claims["department"],
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

func (h *Handler) parseBearer(c fiber.Ctx) (jwt.MapClaims, error) {
	raw := strings.TrimPrefix(c.Get("Authorization"), "Bearer ")
	token, err := jwt.ParseWithClaims(raw, jwt.MapClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodRS256 {
			return nil, jwt.ErrTokenSignatureInvalid
		}
		return h.key.PublicKey(), nil
	}, jwt.WithIssuer(h.cfg.Issuer))
	if err != nil || !token.Valid {
		return nil, jwt.ErrTokenInvalidClaims
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, jwt.ErrTokenInvalidClaims
	}
	return claims, nil
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
