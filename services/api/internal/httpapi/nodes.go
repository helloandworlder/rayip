package httpapi

import (
	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/node"
)

func RegisterNodeRoutes(app *fiber.App, nodes *node.Service) {
	app.Get("/api/admin/nodes", func(c fiber.Ctx) error {
		items, err := nodes.ListNodes(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{
			"items": items,
			"total": len(items),
		})
	})
}
