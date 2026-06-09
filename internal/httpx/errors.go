package httpx

import "github.com/gofiber/fiber/v3"

func Error(c fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(fiber.Map{"error": message})
}

func BadRequest(c fiber.Ctx, message string) error {
	return Error(c, fiber.StatusBadRequest, message)
}

func NotFound(c fiber.Ctx) error {
	return Error(c, fiber.StatusNotFound, "not found")
}
