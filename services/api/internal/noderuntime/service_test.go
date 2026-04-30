package noderuntime_test

import (
	"context"
	"testing"
	"time"

	"github.com/rayip/rayip/services/api/internal/noderuntime"
)

func TestSellableGateAllowsAcceptedAlignedCapableNode(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   12,
		CurrentRevision:    12,
		LastGoodRevision:   12,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-a",
		Capabilities:       []string{"PROXY_ACCOUNT", "SOCKS5", "HTTP"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"SOCKS5",
		},
		ManifestHash:  "manifest-a",
		BinaryHash:    "binary-a",
		ExtensionABI:  "rayip-runtime-v1",
		BundleChannel: "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if !status.Sellable || len(status.UnsellableReasons) != 0 {
		t.Fatalf("status = %#v", status)
	}
}

func TestSellableGateBlocksDigestMismatchAndRuntimeLag(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        true,
		RuntimeVerdict:     noderuntime.RuntimeVerdictAccepted,
		ExpectedRevision:   12,
		CurrentRevision:    10,
		LastGoodRevision:   10,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-b",
		Capabilities:       []string{"PROXY_ACCOUNT", "SOCKS5"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"SOCKS5",
		},
		ManifestHash:  "manifest-a",
		BinaryHash:    "binary-a",
		ExtensionABI:  "rayip-runtime-v1",
		BundleChannel: "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if status.Sellable {
		t.Fatalf("digest mismatch node should not be sellable: %#v", status)
	}
	assertHasReason(t, status.UnsellableReasons, noderuntime.UnsellableDigestMismatch)
	assertHasReason(t, status.UnsellableReasons, noderuntime.UnsellableRuntimeLagging)
}

func TestSellableGateBlocksOfflineUnsupportedCapabilityAndHolds(t *testing.T) {
	now := time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC)
	svc := noderuntime.NewService(noderuntime.NewMemoryRepository(), func() time.Time { return now })
	status, err := svc.UpsertStatus(context.Background(), noderuntime.StatusInput{
		NodeID:             "node-1",
		LeaseOnline:        false,
		RuntimeVerdict:     noderuntime.RuntimeVerdictDegraded,
		ExpectedRevision:   12,
		CurrentRevision:    12,
		LastGoodRevision:   12,
		ExpectedDigestHash: "digest-a",
		RuntimeDigestHash:  "digest-a",
		Capabilities:       []string{"PROXY_ACCOUNT"},
		RequiredCapabilities: []string{
			"PROXY_ACCOUNT",
			"HTTP",
		},
		ManualHold:     true,
		ComplianceHold: true,
		ManifestHash:   "manifest-a",
		BinaryHash:     "binary-a",
		ExtensionABI:   "rayip-runtime-v1",
		BundleChannel:  "stable",
	})
	if err != nil {
		t.Fatalf("UpsertStatus() error = %v", err)
	}
	if status.Sellable {
		t.Fatalf("blocked node should not be sellable: %#v", status)
	}
	for _, reason := range []noderuntime.UnsellableReason{
		noderuntime.UnsellableOffline,
		noderuntime.UnsellableDegraded,
		noderuntime.UnsellableUnsupportedCapability,
		noderuntime.UnsellableManualHold,
		noderuntime.UnsellableComplianceHold,
	} {
		assertHasReason(t, status.UnsellableReasons, reason)
	}
}

func assertHasReason(t *testing.T, reasons []noderuntime.UnsellableReason, expected noderuntime.UnsellableReason) {
	t.Helper()
	for _, reason := range reasons {
		if reason == expected {
			return
		}
	}
	t.Fatalf("reasons %v missing %s", reasons, expected)
}
