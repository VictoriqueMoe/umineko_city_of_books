package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/cors"
	"github.com/gofiber/fiber/v3/middleware/etag"
	"github.com/gofiber/fiber/v3/middleware/logger"
)

func Setup(app *fiber.App, baseURL string) {
	app.Use(etag.New())

	app.Use(func(ctx fiber.Ctx) error {
		if !strings.HasPrefix(ctx.Path(), "/api") {
			ctx.Set("Cache-Control", "no-cache")
		}
		return ctx.Next()
	})

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
		AllowOrigins:     []string{baseURL},
	}))

	app.Use(logger.New(logger.Config{
		Format:     "${time} | ${status} | ${latency} | ${method} ${path} ${queryParams}\n",
		TimeFormat: "2006-01-02 15:04:05",
	}))
}
