package admin

import (
	"context"
	"strconv"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listClients(c fiber.Ctx) error {
	rows, err := h.db.ListClients(context.Background())
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createClient(c fiber.Ctx) error {
	var req clientInput
	if err := c.Bind().Body(&req); err != nil || req.ClientID == "" || req.ClientSecret == "" || req.Name == "" {
		return httpx.BadRequest(c, "clientId, clientSecret and name are required")
	}
	row, err := h.db.CreateClient(context.Background(), domain.Client{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret, Name: req.Name, RedirectURIs: req.RedirectURIs,
	})
	if err != nil {
		return httpx.BadRequest(c, "client already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) getClient(c fiber.Ctx) error {
	row, err := h.db.GetClient(context.Background(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateClient(c fiber.Ctx) error {
	var req clientInput
	if err := c.Bind().Body(&req); err != nil || req.ClientID == "" || req.ClientSecret == "" || req.Name == "" {
		return httpx.BadRequest(c, "clientId, clientSecret and name are required")
	}
	row, err := h.db.UpdateClient(context.Background(), idParam(c), domain.Client{
		ClientID: req.ClientID, ClientSecret: req.ClientSecret, Name: req.Name, RedirectURIs: req.RedirectURIs,
	})
	if err != nil {
		return httpx.BadRequest(c, "client already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteClient(c fiber.Ctx) error {
	if err := h.db.DeleteClient(context.Background(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func (h *Handler) listClientRoles(c fiber.Ctx) error {
	rows, err := h.db.ListClientRoles(context.Background(), idParam(c))
	if err != nil {
		return httpx.Error(c, 500, "list failed")
	}
	return c.JSON(rows)
}

func (h *Handler) createClientRole(c fiber.Ctx) error {
	var req clientRoleInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row, err := h.db.CreateClientRole(context.Background(), idParam(c), domain.ClientRole{Name: req.Name, Description: req.Description})
	if err != nil {
		return httpx.BadRequest(c, "role already exists or invalid")
	}
	return c.Status(201).JSON(row)
}

func (h *Handler) updateClientRole(c fiber.Ctx) error {
	var req clientRoleInput
	if err := c.Bind().Body(&req); err != nil || req.Name == "" {
		return httpx.BadRequest(c, "name is required")
	}
	row, err := h.db.UpdateClientRole(context.Background(), idParam(c), roleIDParam(c), domain.ClientRole{Name: req.Name, Description: req.Description})
	if err != nil {
		return httpx.BadRequest(c, "role already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteClientRole(c fiber.Ctx) error {
	if err := h.db.DeleteClientRole(context.Background(), idParam(c), roleIDParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func roleIDParam(c fiber.Ctx) int64 {
	id, _ := strconv.ParseInt(c.Params("roleId"), 10, 64)
	return id
}

type clientInput struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	Name         string `json:"name"`
	RedirectURIs string `json:"redirectUris"`
}

type clientRoleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
