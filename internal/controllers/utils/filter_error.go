package utils

import (
	"errors"

	"umineko_city_of_books/internal/contentfilter"

	"github.com/gofiber/fiber/v3"
)

func MapFilterError(ctx fiber.Ctx, err error) bool {
	var rej *contentfilter.RejectedError
	if !errors.As(err, &rej) {
		return false
	}
	_ = ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{
		"error":  rej.Rejection.Reason,
		"code":   "content_rejected",
		"rule":   string(rej.Rejection.Rule),
		"detail": rej.Rejection.Detail,
	})
	return true
}
