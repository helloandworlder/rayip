package grpcapi

import (
	"fmt"
	"slices"

	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
)

type runtimeNegotiationPolicy struct {
	RequiredExtensionABI string
	RequiredCapabilities []string
	RequireBinaryHash    bool
}

func defaultRuntimeNegotiationPolicy() runtimeNegotiationPolicy {
	return runtimeNegotiationPolicy{
		RequiredExtensionABI: "rayip.runtime.v1",
		RequiredCapabilities: []string{
			"socks5",
			"http",
			"account-rate-limit",
			"connection-limit",
			"usage-stats",
			"runtime-digest",
		},
		RequireBinaryHash: true,
	}
}

func negotiateRuntime(policy runtimeNegotiationPolicy, observation *controlv1.RuntimeObservation) *controlv1.RuntimeVerdict {
	if observation == nil {
		return runtimeVerdict(
			controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_QUARANTINED,
			"MISSING_RUNTIME_OBSERVATION",
			"node did not report runtime observation",
			policy,
		)
	}
	if policy.RequiredExtensionABI != "" && observation.GetExtensionAbi() != policy.RequiredExtensionABI {
		return runtimeVerdict(
			controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_NEEDS_UPGRADE,
			"EXTENSION_ABI_MISMATCH",
			fmt.Sprintf("runtime extension ABI %q does not match required %q", observation.GetExtensionAbi(), policy.RequiredExtensionABI),
			policy,
		)
	}
	if policy.RequireBinaryHash && observation.GetBinarySha256() == "" {
		return runtimeVerdict(
			controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_QUARANTINED,
			"MISSING_BINARY_HASH",
			"runtime binary hash is required",
			policy,
		)
	}

	capabilities := append([]string(nil), observation.GetCapabilities()...)
	slices.Sort(capabilities)
	for _, required := range policy.RequiredCapabilities {
		if !slices.Contains(capabilities, required) {
			return runtimeVerdict(
				controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_UNSUPPORTED_CAPABILITY,
				"MISSING_CAPABILITY",
				fmt.Sprintf("runtime capability %q is required", required),
				policy,
			)
		}
	}

	return runtimeVerdict(
		controlv1.RuntimeVerdictStatus_RUNTIME_VERDICT_STATUS_ACCEPTED,
		"ACCEPTED",
		"runtime accepted",
		policy,
	)
}

func runtimeVerdict(status controlv1.RuntimeVerdictStatus, reason string, message string, policy runtimeNegotiationPolicy) *controlv1.RuntimeVerdict {
	return &controlv1.RuntimeVerdict{
		Status:               status,
		ReasonCode:           reason,
		Message:              message,
		RequiredCapabilities: append([]string(nil), policy.RequiredCapabilities...),
		RequiredExtensionAbi: policy.RequiredExtensionABI,
	}
}
