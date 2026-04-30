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

	app.Post("/api/admin/nodes/:id/scan", func(c fiber.Ctx) error {
		result, err := nodes.ScanNode(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"scan": result})
	})
}
