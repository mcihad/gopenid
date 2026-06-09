package admin

import (
	"gopenid/internal/store"

	"github.com/gofiber/fiber/v3"
)

type Handler struct {
	db *store.Store
}

func New(db *store.Store) *Handler {
	return &Handler{db: db}
}

func (h *Handler) Mount(app *fiber.App) {
	api := app.Group("/api/admin")
	api.Get("/departments", h.listDepartments)
	api.Post("/departments", h.createDepartment)
	api.Get("/departments/:id", h.getDepartment)
	api.Put("/departments/:id", h.updateDepartment)
	api.Delete("/departments/:id", h.deleteDepartment)
	api.Get("/roles", h.listRoles)
	api.Post("/roles", h.createRole)
	api.Get("/roles/:id", h.getRole)
	api.Put("/roles/:id", h.updateRole)
	api.Delete("/roles/:id", h.deleteRole)
	api.Get("/users", h.listUsers)
	api.Post("/users", h.createUser)
	api.Get("/users/:id", h.getUser)
	api.Put("/users/:id", h.updateUser)
	api.Delete("/users/:id", h.deleteUser)

	api.Get("/clients", h.listClients)
	api.Post("/clients", h.createClient)
	api.Get("/clients/:id", h.getClient)
	api.Put("/clients/:id", h.updateClient)
	api.Delete("/clients/:id", h.deleteClient)
	api.Get("/clients/:id/roles", h.listClientRoles)
	api.Post("/clients/:id/roles", h.createClientRole)
	api.Put("/clients/:id/roles/:roleId", h.updateClientRole)
	api.Delete("/clients/:id/roles/:roleId", h.deleteClientRole)
}
