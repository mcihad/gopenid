package admin

import (
	"context"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"

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
	user := domain.User{Email: req.Email, Name: req.Name, PasswordHash: string(hash), Active: req.Active, DepartmentID: req.DepartmentID}
	user, err = h.db.CreateUser(context.Background(), user, req.RoleIDs, req.ClientIDs, req.ClientRoleIDs)
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
	user := domain.User{Email: req.Email, Name: req.Name, PasswordHash: passwordHash, Active: req.Active, DepartmentID: req.DepartmentID}
	user, err := h.db.UpdateUser(context.Background(), idParam(c), user, req.RoleIDs, req.ClientIDs, req.ClientRoleIDs)
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

type userInput struct {
	Email         string  `json:"email"`
	Name          string  `json:"name"`
	Password      string  `json:"password"`
	Active        bool    `json:"active"`
	DepartmentID  *int64  `json:"departmentId"`
	RoleIDs       []int64 `json:"roleIds"`
	ClientIDs     []int64 `json:"clientIds"`
	ClientRoleIDs []int64 `json:"clientRoleIds"`
}
