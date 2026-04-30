package runtime

import controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"

func ObservationToProto(info DiscoveryInfo) *controlv1.RuntimeObservation {
	return &controlv1.RuntimeObservation{
		AgentVersion:     info.AgentVersion,
		XrayVersion:      info.XrayVersion,
		BundleVersion:    info.BundleVersion,
		ExtensionAbi:     info.ExtensionABI,
		Capabilities:     append([]string(nil), info.Capabilities...),
		BinarySha256:     info.BinarySHA256,
		ManifestSha256:   info.ManifestSHA256,
		RuntimeDigest:    info.RuntimeDigest,
		LastGoodRevision: info.LastGoodGeneration,
	}
}

func ApplyFromProto(apply *controlv1.RuntimeApply) Apply {
	if apply == nil {
		return Apply{}
	}
	resources := make([]Resource, 0, len(apply.GetResources()))
	for _, resource := range apply.GetResources() {
		resources = append(resources, resourceFromProto(resource))
	}
	return Apply{
		ApplyID:              apply.GetApplyId(),
		NodeID:               apply.GetNodeId(),
		Mode:                 applyModeFromProto(apply.GetMode()),
		VersionInfo:          apply.GetVersionInfo(),
		Nonce:                apply.GetNonce(),
		BaseRevision:         apply.GetBaseRevision(),
		TargetRevision:       apply.GetTargetRevision(),
		DeadlineUnixMS:       apply.GetDeadlineUnixMs(),
		Resources:            resources,
		RemovedResourceNames: append([]string(nil), apply.GetRemovedResourceNames()...),
	}
}

func AckToProto(ack ApplyAck) *controlv1.RuntimeApplyAck {
	results := make([]*controlv1.RuntimeResourceResult, 0, len(ack.ResourceResults))
	for _, result := range ack.ResourceResults {
		results = append(results, &controlv1.RuntimeResourceResult{
			Name:        result.Name,
			Status:      resourceResultStatusToProto(result.Status),
			ErrorDetail: result.ErrorDetail,
		})
	}
	return &controlv1.RuntimeApplyAck{
		ApplyId:          ack.ApplyID,
		NodeId:           ack.NodeID,
		VersionInfo:      ack.VersionInfo,
		Nonce:            ack.Nonce,
		Status:           ackStatusToProto(ack.Status),
		AppliedRevision:  ack.AppliedRevision,
		LastGoodRevision: ack.LastGoodRevision,
		ResourceResults:  results,
		Digest: &controlv1.RuntimeDigest{
			AccountCount:  ack.Digest.AccountCount,
			EnabledCount:  ack.Digest.EnabledCount,
			DisabledCount: ack.Digest.DisabledCount,
			MaxGeneration: ack.Digest.MaxGeneration,
			Hash:          ack.Digest.Hash,
		},
		ErrorDetail: ack.ErrorDetail,
	}
}

func resourceFromProto(resource *controlv1.RuntimeResource) Resource {
	if resource == nil {
		return Resource{}
	}
	return Resource{
		Name:              resource.GetName(),
		Kind:              resourceKindFromProto(resource.GetKind()),
		ResourceVersion:   resource.GetResourceVersion(),
		RuntimeEmail:      resource.GetRuntimeEmail(),
		Protocol:          protocolFromProto(resource.GetProtocol()),
		ListenIP:          resource.GetListenIp(),
		Port:              resource.GetPort(),
		Username:          resource.GetUsername(),
		Password:          resource.GetPassword(),
		EgressLimitBPS:    resource.GetEgressLimitBps(),
		IngressLimitBPS:   resource.GetIngressLimitBps(),
		MaxConnections:    resource.GetMaxConnections(),
		Priority:          resource.GetPriority(),
		AbuseReportPolicy: resource.GetAbuseReportPolicy(),
		ExpiresAtUnixMS:   resource.GetExpiresAtUnixMs(),
	}
}

func applyModeFromProto(mode controlv1.RuntimeApplyMode) ApplyMode {
	switch mode {
	case controlv1.RuntimeApplyMode_RUNTIME_APPLY_MODE_SNAPSHOT:
		return ApplyModeSnapshot
	case controlv1.RuntimeApplyMode_RUNTIME_APPLY_MODE_DELTA:
		return ApplyModeDelta
	default:
		return ""
	}
}

func resourceKindFromProto(kind controlv1.RuntimeResourceKind) ResourceKind {
	switch kind {
	case controlv1.RuntimeResourceKind_RUNTIME_RESOURCE_KIND_PROXY_ACCOUNT:
		return ResourceKindProxyAccount
	default:
		return ""
	}
}

func protocolFromProto(protocol controlv1.RuntimeProtocol) Protocol {
	switch protocol {
	case controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_SOCKS5:
		return ProtocolSOCKS5
	case controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_HTTP:
		return ProtocolHTTP
	default:
		return ""
	}
}

func ackStatusToProto(status AckStatus) controlv1.RuntimeApplyStatus {
	switch status {
	case AckStatusACK:
		return controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_ACK
	case AckStatusNACK:
		return controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_NACK
	case AckStatusPartial:
		return controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_PARTIAL
	default:
		return controlv1.RuntimeApplyStatus_RUNTIME_APPLY_STATUS_UNSPECIFIED
	}
}

func resourceResultStatusToProto(status ResourceResultStatus) controlv1.RuntimeResourceApplyStatus {
	switch status {
	case ResourceResultApplied:
		return controlv1.RuntimeResourceApplyStatus_RUNTIME_RESOURCE_APPLY_STATUS_APPLIED
	case ResourceResultRemoved:
		return controlv1.RuntimeResourceApplyStatus_RUNTIME_RESOURCE_APPLY_STATUS_REMOVED
	case ResourceResultFailed:
		return controlv1.RuntimeResourceApplyStatus_RUNTIME_RESOURCE_APPLY_STATUS_FAILED
	default:
		return controlv1.RuntimeResourceApplyStatus_RUNTIME_RESOURCE_APPLY_STATUS_UNSPECIFIED
	}
}
