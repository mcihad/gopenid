package auth

import (
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Mount(app *fiber.App) {
	app.Post("/api/auth/login", h.login)
}

func (h *Handler) login(c fiber.Ctx) error {
	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.Bind().Body(&req); err != nil {
		return httpx.BadRequest(c, "invalid json")
	}
	user, err := h.svc.Authenticate(req.Email, req.Password)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, err.Error())
	}
	token, err := h.svc.AccessToken(user, "gopenid", "openid profile email")
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "token failed")
	}
	return c.JSON(fiber.Map{"access_token": token, "token_type": "Bearer", "expires_in": int(h.svc.cfg.TokenTTL.Seconds())})
}
