package admin

import (
	"context"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listGroups(c fiber.Ctx) error {
	rows, err := h.db.ListGroups(context.Background())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createGroup(c fiber.Ctx) error {
	var req groupInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row, err := h.db.CreateGroup(context.Background(), domain.Group{Name: req.Name, Description: req.Description})
	if err != nil {
		return httpx.BadRequest(c, "group already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getGroup(c fiber.Ctx) error {
	row, err := h.db.GetGroup(context.Background(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateGroup(c fiber.Ctx) error {
	var req groupInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row, err := h.db.UpdateGroup(context.Background(), idParam(c), domain.Group{Name: req.Name, Description: req.Description})
	if err != nil {
		return httpx.BadRequest(c, "group already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteGroup(c fiber.Ctx) error {
	if err := h.db.DeleteGroup(context.Background(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

type groupInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
