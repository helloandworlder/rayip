package httpapi

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
)

func RegisterRuntimeLabRoutes(app *fiber.App, lab *runtimelab.Service) {
	app.Get("/api/admin/runtime-lab/accounts", func(c fiber.Ctx) error {
		items, err := lab.ListAccounts(c.Context())
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	})

	app.Post("/api/admin/runtime-lab/accounts", func(c fiber.Ctx) error {
		var input runtimelab.CreateAccountInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		account, result, err := lab.CreateAccount(c.Context(), input)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"account": account, "result": result})
	})

	app.Patch("/api/admin/runtime-lab/accounts/:id/policy", func(c fiber.Ctx) error {
		var input runtimelab.PolicyInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		account, result, err := lab.UpsertAccountPolicy(c.Context(), c.Params("id"), input)
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"account": account, "result": result})
	})

	app.Post("/api/admin/runtime-lab/accounts/:id/disable", func(c fiber.Ctx) error {
		account, result, err := lab.DisableAccount(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"account": account, "result": result})
	})

	app.Delete("/api/admin/runtime-lab/accounts/:id", func(c fiber.Ctx) error {
		result, err := lab.DeleteAccount(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"result": result})
	})

	app.Get("/api/admin/runtime-lab/accounts/:id/usage", func(c fiber.Ctx) error {
		result, err := lab.GetUsage(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"result": result})
	})

	app.Post("/api/admin/runtime-lab/accounts/:id/probe", func(c fiber.Ctx) error {
		result, err := lab.ProbeAccount(c.Context(), c.Params("id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"result": result})
	})

	app.Get("/api/admin/runtime-lab/accounts/:id/results", func(c fiber.Ctx) error {
		limit, _ := strconv.Atoi(c.Query("limit", "20"))
		items, err := lab.ListApplyResults(c.Context(), c.Params("id"), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	})

	app.Get("/api/admin/runtime-lab/nodes/:node_id/digest", func(c fiber.Ctx) error {
		result, err := lab.GetDigest(c.Context(), c.Params("node_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		return c.JSON(fiber.Map{"result": result})
	})
}
