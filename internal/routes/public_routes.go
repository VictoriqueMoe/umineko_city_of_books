package routes

import (
	"umineko_city_of_books/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func PublicRoutes(service controllers.Service, app *fiber.App) {
	apiRoutes := service.GetAPIRoutes()
	api := app.Group("/api/v1")
	for i := range apiRoutes {
		apiRoutes[i](api)
	}

	pageRoutes := service.GetPageRoutes()
	for i := range pageRoutes {
		pageRoutes[i](app)
	}
}
