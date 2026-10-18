package middleware

import (
	"errors"
	"fmt"

	"github.com/gofiber/fiber/v3"
)

func ErrorHandler(ctx fiber.Ctx, err error) error {
	fiberErr, ok := errors.AsType[*fiber.Error](err)
	if !ok || fiberErr.Code != fiber.StatusRequestEntityTooLarge {
		return fiber.DefaultErrorHandler(ctx, err)
	}

	limitMB := ctx.App().Server().MaxRequestBodySize / (1024 * 1024)

	return ctx.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{
		"error": fmt.Sprintf("This is more than the %d MB one request can carry. Every file is sent together with the post, so remove or shrink some files and try again.", limitMB),
	})
}
