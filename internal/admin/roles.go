package admin

import (
	"gopenid/internal/domain"
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listRoles(c fiber.Ctx) error {
	var rows []domain.Role
	rows, err := h.db.ListRoles(c.Context())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createRole(c fiber.Ctx) error {
	var req roleInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row := domain.Role{Name: req.Name, Description: req.Description}
	row, err := h.db.CreateRole(c.Context(), row)
	if err != nil {
		return httpx.BadRequest(c, "role already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getRole(c fiber.Ctx) error {
	var row domain.Role
	row, err := h.db.GetRole(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateRole(c fiber.Ctx) error {
	var row domain.Role
	row, err := h.db.GetRole(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	var req roleInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row.Name, row.Description = req.Name, req.Description
	row, err = h.db.UpdateRole(c.Context(), row.ID, row)
	if err != nil {
		return httpx.BadRequest(c, "role already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteRole(c fiber.Ctx) error {
	if err := h.db.DeleteRole(c.Context(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

type roleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
