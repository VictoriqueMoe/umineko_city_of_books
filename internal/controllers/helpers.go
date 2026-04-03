package controllers

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

func parseUUIDParam(ctx fiber.Ctx, param string) (uuid.UUID, error) {
	id, err := uuid.Parse(ctx.Params(param))
	if err != nil {
		return uuid.Nil, ctx.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid " + param})
	}
	return id, nil
}

func actorAndTarget(ctx fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	actorID := ctx.Locals("userID").(uuid.UUID)
	targetID, err := parseUUIDParam(ctx, "id")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return actorID, targetID, nil
}
