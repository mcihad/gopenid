package auth

import (
	"gopenid/internal/audit"
	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	svc      *Service
	db       *store.Store
	recorder *audit.Recorder
}

func NewHandler(svc *Service, db *store.Store, recorder *audit.Recorder) *Handler {
	return &Handler{svc: svc, db: db, recorder: recorder}
}

func (h *Handler) Mount(app *fiber.App) {
	app.Post("/api/auth/login", h.login)
	app.Post("/api/auth/refresh", h.refresh)
	app.Post("/api/auth/logout", RequireBearer(h.svc), h.logout)
	app.Post("/api/auth/password-reset/request", h.requestPasswordReset)
	app.Post("/api/auth/password-reset/confirm", h.confirmPasswordReset)
	app.Post("/api/auth/email-verification/request", h.requestEmailVerification)
	app.Post("/api/auth/email-verification/confirm", h.confirmEmailVerification)

	me := app.Group("/api/me", RequireBearer(h.svc))
	me.Get("/", h.profile)
	me.Put("/", h.updateProfile)
	me.Post("/password", h.changePassword)
	me.Post("/mfa/setup", h.setupMFA)
	me.Post("/mfa/enable", h.enableMFA)
	me.Post("/mfa/disable", h.disableMFA)
	me.Get("/roles", h.myRoles)
	me.Get("/departments", h.myDepartments)
	me.Get("/groups", h.myGroups)
	me.Get("/clients", h.myClients)
	me.Get("/sessions", h.mySessions)
}

func (h *Handler) login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		TOTPCode string `json:"totpCode"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(c, "invalid json")
	}
	user, err := h.svc.Authenticate(c.Context(), req.Email, req.Password, req.TOTPCode)
	if err != nil {
		h.recorder.Record(c, audit.Entry{Email: req.Email, Event: domain.EventLoginFailed, Success: false, Message: err.Error()})
		return httpx.Error(c, fiber.StatusUnauthorized, loginErrorMessage(err))
	}
	tokens, err := h.svc.IssueTokens(c.Context(), user, domain.Client{}, "gopenid", "openid profile email", "", true)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "token failed")
	}
	_ = h.db.TouchLastLogin(c.Context(), user.ID)
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, Event: domain.EventLogin, Success: true, Message: "first-party login"})
	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"refresh_token": tokens.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokens.ExpiresIn,
		"user":          userSummary(user),
	})
}

func (h *Handler) refresh(c fiber.Ctx) error {
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.Bind().Body(&req); err != nil || req.RefreshToken == "" {
		// Also accept form encoded refresh_token for OAuth-style clients.
		req.RefreshToken = c.FormValue("refresh_token")
	}
	if req.RefreshToken == "" {
		return httpx.BadRequest(c, "refresh_token is required")
	}
	user, stored, err := h.svc.ConsumeRefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	client := domain.Client{}
	if stored.ClientID != "" && stored.ClientID != "gopenid" {
		if found, ferr := h.db.GetClientByClientID(c.Context(), stored.ClientID); ferr == nil {
			client = found
		}
	}
	tokens, err := h.svc.IssueTokens(c.Context(), user, client, stored.ClientID, stored.Scope, "", true)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "token failed")
	}
	h.recorder.Record(c, audit.Entry{UserID: &user.ID, Email: user.Email, ClientID: stored.ClientID, Event: domain.EventTokenRefresh, Success: true})
	return c.JSON(fiber.Map{
		"access_token":  tokens.AccessToken,
		"id_token":      tokens.IDToken,
		"refresh_token": tokens.RefreshToken,
		"token_type":    "Bearer",
		"expires_in":    tokens.ExpiresIn,
		"scope":         tokens.Scope,
	})
}

func (h *Handler) logout(c fiber.Ctx) error {
	claims, ok := ClaimsFromCtx(c)
	if !ok {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
	}
	if err := h.svc.RevokeAccessClaims(c.Context(), claims, "logout"); err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "logout failed")
	}
	email, _ := claims["email"].(string)
	var uid *int64
	if id, ok := UserIDFromCtx(c); ok {
		uid = &id
	}
	h.recorder.Record(c, audit.Entry{UserID: uid, Email: email, Event: domain.EventLogout, Success: true})
	return c.JSON(fiber.Map{"message": "logged out"})
}

func loginErrorMessage(err error) string {
	switch err {
	case ErrUserInactive:
		return "Hesabınız pasif durumda. Lütfen yöneticinizle iletişime geçin."
	case ErrUserBlocked:
		return "Hesabınız engellenmiş. Lütfen yöneticinizle iletişime geçin."
	case ErrUserLocked:
		return "Çok fazla başarısız deneme yapıldı. Lütfen bir süre sonra tekrar deneyin."
	case ErrMFACodeRequired:
		return "Doğrulama kodu gerekli veya hatalı."
	default:
		return "E-posta veya parola hatalı."
	}
}

func userSummary(user domain.User) fiber.Map {
	roles := make([]string, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, r.Name)
	}
	return fiber.Map{
		"id":    user.ID,
		"email": user.Email,
		"name":  user.Name,
		"title": user.Title,
		"roles": roles,
	}
}
