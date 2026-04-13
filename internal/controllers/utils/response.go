package utils

import "github.com/gofiber/fiber/v3"

func BadRequest(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": msg})
}

func Unauthorized(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": msg})
}

func Forbidden(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": msg})
}

func NotFound(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": msg})
}

func Conflict(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusConflict).JSON(fiber.Map{"error": msg})
}

func TooLarge(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": msg})
}

func InternalError(ctx fiber.Ctx, msg string) error {
	return ctx.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": msg})
}

func OK(ctx fiber.Ctx) error {
	return ctx.JSON(fiber.Map{"status": "ok"})
}
