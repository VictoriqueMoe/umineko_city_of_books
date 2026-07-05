package routes

import (
	"umineko_city_of_books/internal/controllers"

	"github.com/gofiber/fiber/v3"
)

func PublicRoutes(service controllers.Service, app *fiber.App) {
	apiRoutes := service.GetAPIRoutes()
	api := app.Group("/api/v1")
	for i := 0; i < len(apiRoutes); i++ {
		apiRoutes[i](api)
	}

	pageRoutes := service.GetPageRoutes()
	for i := 0; i < len(pageRoutes); i++ {
		pageRoutes[i](app)
	}
}
