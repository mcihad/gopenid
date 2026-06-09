package auth

import (
	"context"

	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

// currentUser loads the authenticated user from the request claims.
func (h *Handler) currentUser(c fiber.Ctx) (int64, error) {
	id, ok := UserIDFromCtx(c)
	if !ok || id == 0 {
		return 0, httpx.Error(c, fiber.StatusUnauthorized, "invalid bearer token")
	}
	return id, nil
}

func (h *Handler) profile(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user)
}

func (h *Handler) updateProfile(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	var req struct {
		Name      string `json:"name"`
		Phone     string `json:"phone"`
		Title     string `json:"title"`
		AvatarURL string `json:"avatarUrl"`
	}
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	user, err := h.db.UpdateProfile(context.Background(), id, req.Name, req.Phone, req.Title, req.AvatarURL)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "update failed")
	}
	return c.JSON(user)
}

func (h *Handler) changePassword(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	var req struct {
		CurrentPassword string `json:"currentPassword"`
		NewPassword     string `json:"newPassword"`
	}
	if err := c.Bind().Body(&req); err != nil || req.NewPassword == "" {
		return httpx.BadRequest(c, "newPassword is required")
	}
	if len(req.NewPassword) < 8 {
		return httpx.BadRequest(c, "newPassword must be at least 8 characters")
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.CurrentPassword)) != nil {
		return httpx.Error(c, fiber.StatusUnauthorized, "Mevcut parola hatalı.")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "password failed")
	}
	if err := h.db.ChangePassword(context.Background(), id, string(hash)); err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "update failed")
	}
	// Invalidate existing refresh tokens so other sessions must re-authenticate.
	_ = h.db.RevokeAllUserRefreshTokens(context.Background(), id)
	return c.JSON(fiber.Map{"message": "password changed"})
}

func (h *Handler) myRoles(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user.Roles)
}

func (h *Handler) myDepartments(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user.Departments)
}

func (h *Handler) myGroups(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user.Groups)
}

func (h *Handler) myClients(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	user, err := h.db.GetUser(context.Background(), id)
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user.AuthorizedClients)
}

func (h *Handler) mySessions(c fiber.Ctx) error {
	id, err := h.currentUser(c)
	if err != nil {
		return err
	}
	sessions, err := h.db.ListRefreshTokensForUser(context.Background(), id)
	if err != nil {
		return httpx.Error(c, fiber.StatusInternalServerError, "list failed")
	}
	return c.JSON(sessions)
}
