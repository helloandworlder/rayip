package httpapi

import (
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
)

type ReadyReport struct {
	Status string            `json:"status"`
	Checks map[string]string `json:"checks"`
}

type HealthOptions struct {
	ServiceName string
	Version     string
	InstanceID  string
	ReadyCheck  func() ReadyReport
}

func RegisterHealthRoutes(app *fiber.App, opts HealthOptions) {
	app.Get("/healthz", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":     "ok",
			"service":    opts.ServiceName,
			"checked_at": time.Now().UTC().Format(time.RFC3339),
		})
	})

	app.Get("/readyz", func(c fiber.Ctx) error {
		report := ReadyReport{Status: "ok", Checks: map[string]string{}}
		if opts.ReadyCheck != nil {
			report = opts.ReadyCheck()
		}
		status := http.StatusOK
		if report.Status != "ok" {
			status = http.StatusServiceUnavailable
		}
		return c.Status(status).JSON(report)
	})

	app.Get("/version", func(c fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"service":     opts.ServiceName,
			"version":     opts.Version,
			"instance_id": opts.InstanceID,
		})
	})
}
