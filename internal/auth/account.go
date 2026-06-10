package auth

import (
	"time"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

const (
	accountTokenPasswordReset = "password_reset"
	accountTokenEmailVerify   = "email_verify"
)

func (h *Handler) requestPasswordReset(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Email == "" {
		return httpx.BadRequest(c, "email is required")
	}
	token := ""
	if user, err := h.db.GetUserByEmail(c.Context(), req.Email); err == nil {
		token, _ = h.createAccountToken(c, user.ID, accountTokenPasswordReset, time.Hour)
	}
	return c.JSON(fiber.Map{"message": "password reset requested", "resetToken": token})
}

func (h *Handler) confirmPasswordReset(c fiber.Ctx) error {
	var req struct {
		Token       string `json:"token"`
		NewPassword string `json:"newPassword"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Token == "" || len(req.NewPassword) < 8 {
		return httpx.BadRequest(c, "token and newPassword are required")
	}
	row, err := h.db.ConsumeAccountToken(c.Context(), HashToken(req.Token), accountTokenPasswordReset)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid or expired token")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "password failed")
	}
	if err := h.db.ChangePassword(c.Context(), row.UserID, string(hash)); err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "password failed")
	}
	_ = h.db.RevokeAllUserRefreshTokens(c.Context(), row.UserID)
	return c.JSON(fiber.Map{"message": "password reset complete"})
}

func (h *Handler) requestEmailVerification(c fiber.Ctx) error {
	var req struct {
		Email string `json:"email"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Email == "" {
		return httpx.BadRequest(c, "email is required")
	}
	token := ""
	if user, err := h.db.GetUserByEmail(c.Context(), req.Email); err == nil {
		token, _ = h.createAccountToken(c, user.ID, accountTokenEmailVerify, 24*time.Hour)
	}
	return c.JSON(fiber.Map{"message": "email verification requested", "verificationToken": token})
}

func (h *Handler) confirmEmailVerification(c fiber.Ctx) error {
	var req struct {
		Token string `json:"token"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Token == "" {
		return httpx.BadRequest(c, "token is required")
	}
	row, err := h.db.ConsumeAccountToken(c.Context(), HashToken(req.Token), accountTokenEmailVerify)
	if err != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "invalid or expired token")
	}
	if err := h.db.SetEmailVerified(c.Context(), row.UserID, true); err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "verification failed")
	}
	return c.JSON(fiber.Map{"message": "email verified"})
}

func (h *Handler) createAccountToken(c fiber.Ctx, userID int64, tokenType string, ttl time.Duration) (string, error) {
	raw, err := RandomToken(32)
	if err != nil {
		return "", err
	}
	_, err = h.db.CreateAccountToken(c.Context(), domain.AccountToken{
		UserID: userID, TokenHash: HashToken(raw), Type: tokenType, ExpiresAt: time.Now().Add(ttl),
	})
	if err != nil && err != store.ErrNotFound {
		return "", err
	}
	return raw, err
}
