package utils

import (
	kernel "umineko_city_of_books/internal/utils"

	"github.com/gofiber/fiber/v3"
)

func BindJSON[T any](ctx fiber.Ctx) (T, bool) {
	var req T
	if kernel.IsMultipart(ctx) {
		_ = BadRequest(ctx, "invalid request body")
		return req, false
	}

	if err := ctx.Bind().JSON(&req); err != nil {
		_ = BadRequest(ctx, "invalid request body")
		return req, false
	}
	return req, true
}
