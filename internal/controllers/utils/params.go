package utils

import (
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

// ParseIDParam parses the named route param as a UUID. On failure it writes a
// 400 response and returns uuid.Nil + false; callers should then `return nil`.
func ParseIDParam(ctx fiber.Ctx, param string) (uuid.UUID, bool) {
	id, err := uuid.Parse(ctx.Params(param))
	if err != nil {
		_ = BadRequest(ctx, "invalid "+param)
		return uuid.Nil, false
	}
	return id, true
}

func ParseID(ctx fiber.Ctx) (uuid.UUID, bool) {
	return ParseIDParam(ctx, "id")
}

func UserID(ctx fiber.Ctx) uuid.UUID {
	id, _ := ctx.Locals("userID").(uuid.UUID)
	return id
}

func OptionalUserID(ctx fiber.Ctx) (uuid.UUID, bool) {
	id, ok := ctx.Locals("userID").(uuid.UUID)
	return id, ok
}

func ActorAndTarget(ctx fiber.Ctx) (uuid.UUID, uuid.UUID, bool) {
	targetID, ok := ParseID(ctx)
	if !ok {
		return uuid.Nil, uuid.Nil, false
	}
	return UserID(ctx), targetID, true
}
