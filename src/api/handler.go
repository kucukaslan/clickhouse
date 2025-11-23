package api

import (
	"github.com/gofiber/fiber/v2"
)

type EventHandler interface {
	PostEvent(ctx *fiber.Ctx) error
	PostEventsBulk(ctx *fiber.Ctx) error
	GetMetrics(ctx *fiber.Ctx) error
}
