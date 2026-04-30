package commercial

import (
	"context"

	"github.com/rayip/rayip/services/api/internal/runtimecontrol"
)

type RuntimeControlAdapter struct {
	service *runtimecontrol.Service
}

func NewRuntimeControlAdapter(service *runtimecontrol.Service) *RuntimeControlAdapter {
	return &RuntimeControlAdapter{service: service}
}

func (a *RuntimeControlAdapter) UpsertProxyAccount(ctx context.Context, input RuntimeProxyAccountInput) (RuntimeMutationResult, error) {
	result, err := a.service.UpsertProxyAccount(ctx, runtimecontrol.ResourceInput{
		ProxyAccountID:  input.ProxyAccountID,
		NodeID:          input.NodeID,
		RuntimeEmail:    input.RuntimeEmail,
		Protocol:        runtimecontrol.Protocol(input.Protocol),
		ListenIP:        input.ListenIP,
		Port:            input.Port,
		Username:        input.Username,
		Password:        input.Password,
		EgressLimitBPS:  input.EgressLimitBPS,
		IngressLimitBPS: input.IngressLimitBPS,
		MaxConnections:  input.MaxConnections,
		ExpiresAt:       input.ExpiresAt,
	})
	if err != nil {
		return RuntimeMutationResult{}, err
	}
	return RuntimeMutationResult{ProxyAccountID: result.State.ProxyAccountID, NodeID: result.State.NodeID}, nil
}

func (a *RuntimeControlAdapter) RemoveProxyAccount(ctx context.Context, proxyAccountID string) (RuntimeMutationResult, error) {
	result, err := a.service.RemoveProxyAccount(ctx, proxyAccountID)
	if err != nil {
		return RuntimeMutationResult{}, err
	}
	return RuntimeMutationResult{ProxyAccountID: result.State.ProxyAccountID, NodeID: result.State.NodeID}, nil
}

var _ RuntimeWriter = (*RuntimeControlAdapter)(nil)
