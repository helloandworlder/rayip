package runtimelab

import controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"

func CommandToProto(cmd RuntimeCommand) *controlv1.RuntimeCommand {
	return &controlv1.RuntimeCommand{
		CommandId:         cmd.CommandID,
		NodeId:            cmd.NodeID,
		Operation:         operationToProto(cmd.Operation),
		Account:           accountToProto(cmd.Account),
		DesiredGeneration: cmd.DesiredGeneration,
		DeadlineUnixMs:    cmd.DeadlineUnixMS,
	}
}

func ResultFromProto(result *controlv1.RuntimeResult) ApplyResult {
	if result == nil {
		return ApplyResult{}
	}
	return ApplyResult{
		CommandID:         result.GetCommandId(),
		Status:            statusFromProto(result.GetStatus()),
		ErrorCode:         result.GetErrorCode(),
		ErrorMessage:      result.GetErrorMessage(),
		AppliedGeneration: result.GetAppliedGeneration(),
		Usage: Usage{
			ProxyAccountID:    result.GetUsage().GetProxyAccountId(),
			RuntimeEmail:      result.GetUsage().GetRuntimeEmail(),
			RxBytes:           result.GetUsage().GetRxBytes(),
			TxBytes:           result.GetUsage().GetTxBytes(),
			ActiveConnections: result.GetUsage().GetActiveConnections(),
			RxBytesPerSecond:  result.GetUsage().GetRxBytesPerSecond(),
			TxBytesPerSecond:  result.GetUsage().GetTxBytesPerSecond(),
		},
		Digest: Digest{
			AccountCount:  result.GetDigest().GetAccountCount(),
			EnabledCount:  result.GetDigest().GetEnabledCount(),
			DisabledCount: result.GetDigest().GetDisabledCount(),
			MaxGeneration: result.GetDigest().GetMaxGeneration(),
			Hash:          result.GetDigest().GetHash(),
		},
	}
}

func accountToProto(account Account) *controlv1.RuntimeAccount {
	return &controlv1.RuntimeAccount{
		ProxyAccountId:    account.ProxyAccountID,
		RuntimeEmail:      account.RuntimeEmail,
		Protocol:          protocolToProto(account.Protocol),
		ListenIp:          account.ListenIP,
		Port:              account.Port,
		Username:          account.Username,
		Password:          account.Password,
		ExpiresAtUnixMs:   account.ExpiresAt.UnixMilli(),
		EgressLimitBps:    account.EgressLimitBPS,
		IngressLimitBps:   account.IngressLimitBPS,
		MaxConnections:    account.MaxConnections,
		Status:            accountStatusToProto(account.Status),
		PolicyVersion:     account.PolicyVersion,
		DesiredGeneration: account.DesiredGeneration,
	}
}

func operationToProto(operation Operation) controlv1.RuntimeOperation {
	switch operation {
	case OperationUpsert:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_UPSERT
	case OperationDelete:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_DELETE
	case OperationDisable:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_DISABLE
	case OperationUpdatePolicy:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_UPDATE_POLICY
	case OperationGetUsage:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_GET_USAGE
	case OperationGetDigest:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_GET_DIGEST
	case OperationProbe:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_PROBE
	default:
		return controlv1.RuntimeOperation_RUNTIME_OPERATION_UNSPECIFIED
	}
}

func protocolToProto(protocol Protocol) controlv1.RuntimeProtocol {
	switch protocol {
	case ProtocolSOCKS5:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_SOCKS5
	case ProtocolHTTP:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_HTTP
	default:
		return controlv1.RuntimeProtocol_RUNTIME_PROTOCOL_UNSPECIFIED
	}
}

func accountStatusToProto(status AccountStatus) controlv1.RuntimeAccountStatus {
	switch status {
	case AccountStatusEnabled:
		return controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_ENABLED
	case AccountStatusDisabled:
		return controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_DISABLED
	case AccountStatusDeleted:
		return controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_DELETED
	default:
		return controlv1.RuntimeAccountStatus_RUNTIME_ACCOUNT_STATUS_UNSPECIFIED
	}
}

func statusFromProto(status controlv1.RuntimeResultStatus) ApplyStatus {
	switch status {
	case controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_SUCCESS:
		return ApplyStatusSuccess
	case controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_FAILED:
		return ApplyStatusFailed
	case controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_SKIPPED:
		return ApplyStatusSkipped
	case controlv1.RuntimeResultStatus_RUNTIME_RESULT_STATUS_DUPLICATE:
		return ApplyStatusDuplicate
	default:
		return ApplyStatusFailed
	}
}
