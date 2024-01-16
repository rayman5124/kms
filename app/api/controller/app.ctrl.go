package controller

import (
	"fmt"

	"github.com/gofiber/fiber/v2"
)

type appCtrl struct {
}

func NewAppCtrl() *appCtrl {
	c := &appCtrl{}

	return c
}

func (c *appCtrl) BootStrap(router fiber.Router) {
	router.Get("/health", c.HealthCheck)
	router.Get("/error", c.Error)
}

// @tags Health
// @success 200
// @router /api/health [get]
func (c *appCtrl) HealthCheck(ctx *fiber.Ctx) error {
	return ctx.JSON(fiber.Map{
		"success": true,
	})
}

// @tags Health
// @success 200
// @router /api/error [get]
func (c *appCtrl) Error(ctx *fiber.Ctx) error {
	// return fiber.ErrInternalServerError
	return fmt.Errorf("sdfdsf")
}
