package httpapi_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/httpapi"
)

func TestHealthAndVersionRoutes(t *testing.T) {
	app := fiber.New()
	httpapi.RegisterHealthRoutes(app, httpapi.HealthOptions{
		ServiceName: "rayip-api",
		Version:     "test-version",
		InstanceID:  "api-1",
		ReadyCheck: func() httpapi.ReadyReport {
			return httpapi.ReadyReport{
				Status: "ok",
				Checks: map[string]string{
					"postgres": "ok",
					"redis":    "ok",
					"nats":     "ok",
				},
			}
		},
	})

	healthResp, err := app.Test(httptestRequest(t, http.MethodGet, "/healthz"))
	if err != nil {
		t.Fatalf("GET /healthz error = %v", err)
	}
	if healthResp.StatusCode != http.StatusOK {
		t.Fatalf("GET /healthz status = %d, want 200", healthResp.StatusCode)
	}

	var health map[string]any
	if err := json.NewDecoder(healthResp.Body).Decode(&health); err != nil {
		t.Fatalf("decode health response: %v", err)
	}
	if health["service"] != "rayip-api" || health["status"] != "ok" {
		t.Fatalf("unexpected health response: %#v", health)
	}

	versionResp, err := app.Test(httptestRequest(t, http.MethodGet, "/version"))
	if err != nil {
		t.Fatalf("GET /version error = %v", err)
	}
	var version map[string]any
	if err := json.NewDecoder(versionResp.Body).Decode(&version); err != nil {
		t.Fatalf("decode version response: %v", err)
	}
	if version["version"] != "test-version" || version["instance_id"] != "api-1" {
		t.Fatalf("unexpected version response: %#v", version)
	}
}

func TestReadyRouteReportsUnavailableDependency(t *testing.T) {
	app := fiber.New()
	httpapi.RegisterHealthRoutes(app, httpapi.HealthOptions{
		ServiceName: "rayip-api",
		Version:     "test-version",
		InstanceID:  "api-1",
		ReadyCheck: func() httpapi.ReadyReport {
			return httpapi.ReadyReport{
				Status: "degraded",
				Checks: map[string]string{"postgres": "error"},
			}
		},
	})

	resp, err := app.Test(httptestRequest(t, http.MethodGet, "/readyz"))
	if err != nil {
		t.Fatalf("GET /readyz error = %v", err)
	}
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("GET /readyz status = %d, want 503", resp.StatusCode)
	}
}

func httptestRequest(t *testing.T, method string, target string) *http.Request {
	t.Helper()

	req, err := http.NewRequest(method, target, nil)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	return req
}
