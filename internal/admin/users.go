package admin

import (
	"context"
	"errors"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
)

func (h *Handler) listUsers(c fiber.Ctx) error {
	rows, err := h.db.ListUsers(context.Background())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createUser(c fiber.Ctx) error {
	var req userInput
	if err := c.Bind().Body(&req); err != nil || req.Email == "" || req.Name == "" || req.Password == "" {
		return httpx.BadRequest(c, "email, name and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return httpx.Error(c, 500, "password failed")
	}
	user := req.toUser(string(hash))
	user, err = h.db.CreateUser(context.Background(), user, req.RoleIDs, req.ClientIDs, req.ClientRoleIDs, req.DepartmentIDs, req.GroupIDs)
	if err != nil {
		return httpx.BadRequest(c, "user already exists or invalid")
	}
	return c.Status(201).JSON(user)
}

func (h *Handler) getUser(c fiber.Ctx) error {
	user, err := h.db.GetUser(context.Background(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(user)
}

func (h *Handler) updateUser(c fiber.Ctx) error {
	var req userInput
	if err := c.Bind().Body(&req); err != nil || req.Email == "" || req.Name == "" {
		return httpx.BadRequest(c, "email and name are required")
	}
	passwordHash := ""
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			return httpx.Error(c, 500, "password failed")
		}
		passwordHash = string(hash)
	}
	user := req.toUser(passwordHash)
	user, err := h.db.UpdateUser(context.Background(), idParam(c), user, req.RoleIDs, req.ClientIDs, req.ClientRoleIDs, req.DepartmentIDs, req.GroupIDs)
	if err != nil {
		return httpx.BadRequest(c, "user already exists or invalid")
	}
	return c.JSON(user)
}

func (h *Handler) deleteUser(c fiber.Ctx) error {
	if err := h.db.DeleteUser(context.Background(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func (h *Handler) blockUser(c fiber.Ctx) error {
	var req struct {
		Reason string `json:"reason"`
	}
	_ = c.Bind().Body(&req)
	if err := h.db.SetBlocked(context.Background(), idParam(c), true, req.Reason); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.NotFound(c)
		}
		return httpx.Error(c, 500, "block failed")
	}
	// Blocking should also invalidate active refresh tokens.
	_ = h.db.RevokeAllUserRefreshTokens(context.Background(), idParam(c))
	return c.JSON(fiber.Map{"message": "user blocked"})
}

func (h *Handler) unblockUser(c fiber.Ctx) error {
	if err := h.db.SetBlocked(context.Background(), idParam(c), false, ""); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return httpx.NotFound(c)
		}
		return httpx.Error(c, 500, "unblock failed")
	}
	return c.JSON(fiber.Map{"message": "user unblocked"})
}

func (h *Handler) revokeUserSessions(c fiber.Ctx) error {
	if err := h.db.RevokeAllUserRefreshTokens(context.Background(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "revoke failed")
	}
	return c.JSON(fiber.Map{"message": "sessions revoked"})
}

type userInput struct {
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	Password      string  `json:"password"`
	Active        bool    `json:"active"`
	Blocked       bool    `json:"blocked"`
	BlockedReason string  `json:"blockedReason"`
	Phone         string  `json:"phone"`
	Title         string  `json:"title"`
	AvatarURL     string  `json:"avatarUrl"`
	DepartmentID  *int64  `json:"departmentId"`
	RoleIDs       []int64 `json:"roleIds"`
	ClientIDs     []int64 `json:"clientIds"`
	ClientRoleIDs []int64 `json:"clientRoleIds"`
	DepartmentIDs []int64 `json:"departmentIds"`
	GroupIDs      []int64 `json:"groupIds"`
}

func (req userInput) toUser(passwordHash string) domain.User {
	user := domain.User{
		Email:         req.Email,
		Name:          req.Name,
		PasswordHash:  passwordHash,
		Active:        req.Active,
		Blocked:       req.Blocked,
		BlockedReason: req.BlockedReason,
		Phone:         req.Phone,
		Title:         req.Title,
		AvatarURL:     req.AvatarURL,
		DepartmentID:  req.DepartmentID,
	}
	return user
}
