package web

import (
	"embed"
	"io/fs"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/static"
)

//go:embed dist
var files embed.FS

func Mount(app *fiber.App) error {
	dist, err := fs.Sub(files, "dist")
	if err != nil {
		return err
	}
	app.Use("/", static.New("", static.Config{FS: dist}))
	app.Get("/*", func(c fiber.Ctx) error {
		return c.SendFile("index.html", fiber.SendFile{FS: dist})
	})
	return nil
}
