package grpcapi

import (
	"testing"

	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
)

func TestNegotiateRuntimeAcceptsMatchingObservation(t *testing.T) {
	verdict := negotiateRuntime(runtimeNegotiationPolicy{
		RequiredExtensionABI: "rayip.runtime.v1",
		RequiredCapabilities: []string{"socks5", "http", "account-rate-limit"},
	}, &controlv1.RuntimeObservation{
		ExtensionAbi: "rayip.runtime.v1",
		Capabilities: []string{"http", "socks5", "account-rate-limit", "runtime-digest"},
		BinarySha256: "sha256:binary",
	})

	if verdict.GetStatus() != controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_ACCEPTED {
		t.Fatalf("status = %s, want ACCEPTED: %#v", verdict.GetStatus(), verdict)
	}
}

func TestNegotiateRuntimeRejectsMissingCapability(t *testing.T) {
	verdict := negotiateRuntime(runtimeNegotiationPolicy{
		RequiredExtensionABI: "rayip.runtime.v1",
		RequiredCapabilities: []string{"socks5", "http", "account-rate-limit"},
	}, &controlv1.RuntimeObservation{
		ExtensionAbi: "rayip.runtime.v1",
		Capabilities: []string{"socks5", "http"},
		BinarySha256: "sha256:binary",
	})

	if verdict.GetStatus() != controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_UNSUPPORTED_CAPABILITY {
		t.Fatalf("status = %s, want UNSUPPORTED_CAPABILITY: %#v", verdict.GetStatus(), verdict)
	}
	if verdict.GetReasonCode() != "MISSING_CAPABILITY" {
		t.Fatalf("reason = %q, want MISSING_CAPABILITY", verdict.GetReasonCode())
	}
}

func TestNegotiateRuntimeRejectsABIAndMissingHash(t *testing.T) {
	verdict := negotiateRuntime(runtimeNegotiationPolicy{
		RequiredExtensionABI: "rayip.runtime.v1",
		RequiredCapabilities: []string{"socks5"},
		RequireBinaryHash:    true,
	}, &controlv1.RuntimeObservation{
		ExtensionAbi: "rayip.runtime.v0",
		Capabilities: []string{"socks5"},
	})

	if verdict.GetStatus() != controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_NEEDS_UPGRADE {
		t.Fatalf("status = %s, want NEEDS_UPGRADE: %#v", verdict.GetStatus(), verdict)
	}
	if verdict.GetReasonCode() != "EXTENSION_ABI_MISMATCH" {
		t.Fatalf("reason = %q, want EXTENSION_ABI_MISMATCH", verdict.GetReasonCode())
	}

	verdict = negotiateRuntime(runtimeNegotiationPolicy{
		RequiredCapabilities: []string{"socks5"},
		RequireBinaryHash:    true,
	}, &controlv1.RuntimeObservation{
		Capabilities: []string{"socks5"},
	})
	if verdict.GetStatus() != controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_QUARANTINED {
		t.Fatalf("status = %s, want QUARANTINED: %#v", verdict.GetStatus(), verdict)
	}
}
