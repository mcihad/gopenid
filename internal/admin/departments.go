package admin

import (
	"context"
	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listDepartments(c fiber.Ctx) error {
	var rows []domain.Department
	rows, err := h.db.ListDepartments(context.Background())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createDepartment(c fiber.Ctx) error {
	var req departmentInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row := domain.Department{Name: req.Name, Description: req.Description}
	row, err := h.db.CreateDepartment(context.Background(), row)
	if err != nil {
		return httpx.BadRequest(c, "department already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getDepartment(c fiber.Ctx) error {
	var row domain.Department
	row, err := h.db.GetDepartment(context.Background(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateDepartment(c fiber.Ctx) error {
	var row domain.Department
	row, err := h.db.GetDepartment(context.Background(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	var req departmentInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row.Name, row.Description = req.Name, req.Description
	row, err = h.db.UpdateDepartment(context.Background(), row.ID, row)
	if err != nil {
		return httpx.BadRequest(c, "department already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteDepartment(c fiber.Ctx) error {
	if err := h.db.DeleteDepartment(context.Background(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func idParam(c fiber.Ctx) int64 {
	id, _ := strconv.ParseInt(c.Params("id"), 10, 64)
	return id
}

type departmentInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
