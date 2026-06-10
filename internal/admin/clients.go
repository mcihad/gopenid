package admin

import (
	"strconv"

	"gopenid/internal/domain"
	"gopenid/internal/httpx"
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

func (h *Handler) listClients(c fiber.Ctx) error {
	rows, err := h.db.ListClients(c.Context())
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
	client := req.toClient()
	plainSecret := req.ClientSecret
	hash, err := store.HashClientSecret(plainSecret)
	if err != nil {
		return httpx.Error(c, 500, "secret failed")
	}
	client.ClientSecret = hash
	row, err := h.db.CreateClient(c.Context(), client)
	if err != nil {
		return httpx.BadRequest(c, "client already exists or invalid")
	}
	row.ClientSecretPlain = plainSecret
	return c.Status(201).JSON(row)
}

func (h *Handler) getClient(c fiber.Ctx) error {
	row, err := h.db.GetClient(c.Context(), idParam(c))
	if err != nil {
		return httpx.NotFound(c)
	}
	return c.JSON(row)
}

func (h *Handler) updateClient(c fiber.Ctx) error {
	var req clientInput
	if err := c.Bind().Body(&req); err != nil || req.ClientID == "" || req.Name == "" {
		return httpx.BadRequest(c, "clientId and name are required")
	}
	client := req.toClient()
	plainSecret := req.ClientSecret
	if plainSecret != "" {
		hash, err := store.HashClientSecret(plainSecret)
		if err != nil {
			return httpx.Error(c, 500, "secret failed")
		}
		client.ClientSecret = hash
	}
	row, err := h.db.UpdateClient(c.Context(), idParam(c), client)
	if err != nil {
		return httpx.BadRequest(c, "client already exists or invalid")
	}
	if plainSecret != "" {
		row.ClientSecretPlain = plainSecret
	}
	return c.JSON(row)
}

func (h *Handler) deleteClient(c fiber.Ctx) error {
	if err := h.db.DeleteClient(c.Context(), idParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func (h *Handler) listClientRoles(c fiber.Ctx) error {
	rows, err := h.db.ListClientRoles(c.Context(), idParam(c))
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
	row, err := h.db.CreateClientRole(c.Context(), idParam(c), domain.ClientRole{Name: req.Name, Description: req.Description})
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
	row, err := h.db.UpdateClientRole(c.Context(), idParam(c), roleIDParam(c), domain.ClientRole{Name: req.Name, Description: req.Description})
	if err != nil {
		return httpx.BadRequest(c, "role already exists or invalid")
	}
	return c.JSON(row)
}

func (h *Handler) deleteClientRole(c fiber.Ctx) error {
	if err := h.db.DeleteClientRole(c.Context(), idParam(c), roleIDParam(c)); err != nil {
		return httpx.Error(c, 500, "delete failed")
	}
	return c.SendStatus(204)
}

func roleIDParam(c fiber.Ctx) int64 {
	id, _ := strconv.ParseInt(c.Params("roleId"), 10, 64)
	return id
}

type clientInput struct {
	ClientID           string `json:"clientId"`
	ClientSecret       string `json:"clientSecret"`
	Name               string `json:"name"`
	Description        string `json:"description"`
	HomeURL            string `json:"homeUrl"`
	LogoURL            string `json:"logoUrl"`
	RedirectURIs       string `json:"redirectUris"`
	TokenTTLSeconds    int    `json:"tokenTtlSeconds"`
	RefreshTTLSeconds  int    `json:"refreshTtlSeconds"`
	AllowPasswordGrant bool   `json:"allowPasswordGrant"`
}

func (req clientInput) toClient() domain.Client {
	return domain.Client{
		ClientID:           req.ClientID,
		ClientSecret:       req.ClientSecret,
		Name:               req.Name,
		Description:        req.Description,
		HomeURL:            req.HomeURL,
		LogoURL:            req.LogoURL,
		RedirectURIs:       req.RedirectURIs,
		TokenTTLSeconds:    req.TokenTTLSeconds,
		RefreshTTLSeconds:  req.RefreshTTLSeconds,
		AllowPasswordGrant: req.AllowPasswordGrant,
	}
}

type clientRoleInput struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
