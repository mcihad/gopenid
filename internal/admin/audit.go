package admin

import (
	"context"
	"strconv"

	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listAuditLogs(c fiber.Ctx) error {
	userID, _ := strconv.ParseInt(c.Query("userId"), 10, 64)
	limit, _ := strconv.Atoi(c.Query("limit"))
	event := c.Query("event")
	rows, err := h.db.ListAuditLogs(context.Background(), userID, event, limit)
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}
