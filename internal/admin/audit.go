package admin

import (
	"strconv"
	"time"

	"gopenid/internal/httpx"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listAuditLogs(c fiber.Ctx) error {
	userID, _ := strconv.ParseInt(c.Query("userId"), 10, 64)
	page, _ := strconv.Atoi(c.Query("page"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize"))
	if pageSize == 0 {
		pageSize, _ = strconv.Atoi(c.Query("limit"))
	}
	var success *bool
	if raw := c.Query("success"); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err == nil {
			success = &parsed
		}
	}
	filter := store.AuditFilter{
		UserID: userID, Event: c.Query("event"), Email: c.Query("email"),
		ClientID: c.Query("clientId"), IP: c.Query("ip"), Success: success,
		From: parseAuditTime(c.Query("from")), To: parseAuditTime(c.Query("to")),
		Page: page, PageSize: pageSize,
	}
	rows, err := h.db.ListAuditLogs(c.Context(), filter)
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func parseAuditTime(raw string) *time.Time {
	if raw == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return &parsed
	}
	return nil
}
