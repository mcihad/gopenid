package web

import (
	"embed"
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed dist
var files embed.FS

//go:embed llms.txt
var llmsTxt []byte

func Mount(app *fiber.App) error {
	dist, err := fs.Sub(files, "dist")
	if err != nil {
		return err
	}
	// LLM-friendly usage guide for external apps and the admin API.
	app.Get("/llms.txt", func(c fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "text/plain; charset=utf-8")
		return c.Send(llmsTxt)
	})
	app.Use("/", static.New("", static.Config{FS: dist}))
	app.Get("/*", func(c fiber.Ctx) error {
		return c.SendFile("index.html", fiber.SendFile{FS: dist})
	})
	return nil
}
