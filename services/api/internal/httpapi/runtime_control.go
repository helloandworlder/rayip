package httpapi

import (
	"strconv"

	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/noderuntime"
	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
)

func RegisterRuntimeControlRoutes(app *fiber.App, runtimeControl *runtimecontrol.Service, runtimeWorker *runtimecontrol.Worker, reconcilePlanner *runtimecontrol.ReconcilePlanner, nodeRuntime *noderuntime.Service) {
	app.Get("/api/admin/runtime-control/outbox", func(c fiber.Ctx) error {
		limit, _ := strconv.Atoi(c.Query("limit", "100"))
		items, err := runtimeControl.ListOutbox(c.Context(), limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	})

	app.Get("/api/admin/runtime-control/nodes/:node_id/changes", func(c fiber.Ctx) error {
		afterSeq, _ := strconv.ParseUint(c.Query("after_seq", "0"), 10, 64)
		limit, _ := strconv.Atoi(c.Query("limit", "100"))
		items, err := runtimeControl.ListChanges(c.Context(), c.Params("node_id"), afterSeq, limit)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	})

	app.Post("/api/admin/runtime-control/nodes/:node_id/process", func(c fiber.Ctx) error {
		afterSeq, _ := strconv.ParseUint(c.Query("after_seq", "0"), 10, 64)
		limit, _ := strconv.Atoi(c.Query("limit", "100"))
		result, err := runtimeWorker.ProcessNodeChanges(c.Context(), c.Params("node_id"), afterSeq, limit)
		if err != nil {
			return c.Status(fiber.StatusAccepted).JSON(fiber.Map{"result": result, "error": err.Error()})
		}
		return c.JSON(fiber.Map{"result": result})
	})

	app.Get("/api/admin/runtime-control/nodes/:node_id/snapshot", func(c fiber.Ctx) error {
		offset, _ := strconv.Atoi(c.Query("offset", "0"))
		limit, _ := strconv.Atoi(c.Query("limit", "500"))
		baseRevision, _ := strconv.ParseUint(c.Query("base_revision", "0"), 10, 64)
		targetRevision, _ := strconv.ParseUint(c.Query("target_revision", "0"), 10, 64)
		apply, err := reconcilePlanner.BuildSnapshotApply(c.Context(), c.Params("node_id"), offset, limit, baseRevision, targetRevision)
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		return c.JSON(fiber.Map{"apply": apply, "resource_count": len(apply.Resources)})
	})

	app.Get("/api/admin/nodes/:node_id/runtime-status", func(c fiber.Ctx) error {
		status, ok, err := nodeRuntime.GetStatus(c.Context(), c.Params("node_id"))
		if err != nil {
			return fiber.NewError(fiber.StatusInternalServerError, err.Error())
		}
		if !ok {
			return fiber.NewError(fiber.StatusNotFound, "node runtime status not found")
		}
		return c.JSON(fiber.Map{"status": status})
	})
}
