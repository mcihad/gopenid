package admin

import (
	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"strconv"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listDepartments(c fiber.Ctx) error {
	var rows []domain.Department
	rows, err := h.db.ListDepartments(c.Context())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(withDepartmentChildren(rows))
}

func (h *Handler) createDepartment(c fiber.Ctx) error {
	var req departmentInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row := domain.Department{Name: req.Name, Description: req.Description, ParentID: req.ParentID}
	row, err := h.db.CreateDepartment(c.Context(), row)
	if err != nil {
		return httpx.BadRequest(c, "department already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getDepartment(c fiber.Ctx) error {
	var row domain.Department
	row, err := h.db.GetDepartment(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateDepartment(c fiber.Ctx) error {
	var row domain.Department
	row, err := h.db.GetDepartment(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	var req departmentInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	if req.ParentID != nil && *req.ParentID == row.ID {
		return httpx.BadRequest(c, "department cannot be its own parent")
	}
	row.Name, row.Description, row.ParentID = req.Name, req.Description, req.ParentID
	row, err = h.db.UpdateDepartment(c.Context(), row.ID, row)
	if err != nil {
		return httpx.BadRequest(c, "department already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteDepartment(c fiber.Ctx) error {
	if err := h.db.DeleteDepartment(c.Context(), idParam(c)); err != nil {
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
	ParentID    *int64 `json:"parentId"`
}

func withDepartmentChildren(rows []domain.Department) []domain.Department {
	byParent := make(map[int64][]domain.Department)
	for _, row := range rows {
		if row.ParentID != nil {
			byParent[*row.ParentID] = append(byParent[*row.ParentID], row)
		}
	}
	for i := range rows {
		rows[i].Children = byParent[rows[i].ID]
	}
	return rows
}
