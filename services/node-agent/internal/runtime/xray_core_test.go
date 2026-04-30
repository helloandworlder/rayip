package runtime_test

import (
	"context"
	"net"
	"testing"

	runtimev1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/runtime/v1"
	"github.com/rayip/rayip/services/node-agent/internal/runtime"
	handlercmd "github.com/xtls/xray-core/app/proxyman/command"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestXrayCoreMapsAccountPolicyAndDigest(t *testing.T) {
	fake := &fakeRuntimeServer{policies: map[string]*runtimev1.AccountPolicy{}}
	handler := &fakeHandlerServer{inbounds: map[string]*core.InboundHandlerConfig{}, users: map[string]*protocol.User{}}
	client, handlerClient, cleanup := runtimeClient(t, fake, handler)
	defer cleanup()

	core := runtime.NewXrayCoreWithClients(client, handlerClient)
	if err := core.UpsertAccount(context.Background(), runtime.Account{
		ProxyAccountID:    "acct-1",
		RuntimeEmail:      "email-1",
		Protocol:          runtime.ProtocolSOCKS5,
		ListenIP:          "127.0.0.1",
		Port:              18080,
		Username:          "customer",
		Password:          "secret",
		EgressLimitBPS:    1024,
		IngressLimitBPS:   2048,
		MaxConnections:    2,
		Priority:          3,
		AbuseBytesPerMin:  4096,
		AbuseAction:       runtime.AbuseActionDisableAndReport,
		DesiredGeneration: 7,
		Status:            runtime.AccountStatusEnabled,
	}); err != nil {
		t.Fatalf("UpsertAccount() error = %v", err)
	}

	if got := fake.policies["email-1"]; got.GetEgressLimitBps() != 1024 || got.GetMaxConnections() != 2 {
		t.Fatalf("unexpected policy: %#v", got)
	}
	if len(handler.inbounds) != 1 || handler.users["email-1"] == nil {
		t.Fatalf("handler state = inbounds %#v users %#v", handler.inbounds, handler.users)
	}

	usage, err := core.Usage(context.Background(), "acct-1")
	if err != nil {
		t.Fatalf("Usage() error = %v", err)
	}
	if usage.RuntimeEmail != "email-1" {
		t.Fatalf("usage runtime email = %q, want email-1", usage.RuntimeEmail)
	}

	digest, err := core.Digest(context.Background())
	if err != nil {
		t.Fatalf("Digest() error = %v", err)
	}
	if digest.AccountCount != 1 || digest.MaxGeneration != 7 {
		t.Fatalf("unexpected digest: %#v", digest)
	}

	if err := core.DeleteAccount(context.Background(), "acct-1"); err != nil {
		t.Fatalf("DeleteAccount() error = %v", err)
	}
	if got := fake.policies["email-1"]; got != nil {
		t.Fatalf("policy after delete = %#v, want removed", got)
	}
	if handler.users["email-1"] != nil {
		t.Fatalf("handler user after delete = %#v, want removed", handler.users["email-1"])
	}
}

type fakeRuntimeServer struct {
	runtimev1.UnimplementedRuntimeServiceServer
	policies map[string]*runtimev1.AccountPolicy
}

func (s *fakeRuntimeServer) GetCapabilities(context.Context, *runtimev1.GetCapabilitiesRequest) (*runtimev1.GetCapabilitiesResponse, error) {
	return &runtimev1.GetCapabilitiesResponse{ExtensionAbi: "rayip.runtime.v1"}, nil
}

func (s *fakeRuntimeServer) UpsertAccountPolicy(_ context.Context, request *runtimev1.UpsertAccountPolicyRequest) (*runtimev1.UpsertAccountPolicyResponse, error) {
	policy := request.GetPolicy()
	s.policies[policy.GetEmail()] = policy
	return &runtimev1.UpsertAccountPolicyResponse{Digest: s.digest()}, nil
}

func (s *fakeRuntimeServer) RemoveAccountPolicy(_ context.Context, request *runtimev1.RemoveAccountPolicyRequest) (*runtimev1.RemoveAccountPolicyResponse, error) {
	delete(s.policies, request.GetEmail())
	return &runtimev1.RemoveAccountPolicyResponse{Digest: s.digest()}, nil
}

func (s *fakeRuntimeServer) GetUserSpeed(_ context.Context, request *runtimev1.GetUserSpeedRequest) (*runtimev1.GetUserSpeedResponse, error) {
	return &runtimev1.GetUserSpeedResponse{Speed: &runtimev1.UserSpeed{Email: request.GetEmail()}}, nil
}

func (s *fakeRuntimeServer) GetDigest(context.Context, *runtimev1.GetDigestRequest) (*runtimev1.GetDigestResponse, error) {
	return &runtimev1.GetDigestResponse{Digest: s.digest()}, nil
}

func (s *fakeRuntimeServer) digest() *runtimev1.Digest {
	digest := &runtimev1.Digest{AccountCount: uint64(len(s.policies))}
	for _, policy := range s.policies {
		if policy.GetDisabled() {
			digest.DisabledCount++
		} else {
			digest.EnabledCount++
		}
		if policy.GetGeneration() > digest.MaxGeneration {
			digest.MaxGeneration = policy.GetGeneration()
		}
	}
	return digest
}

type fakeHandlerServer struct {
	handlercmd.UnimplementedHandlerServiceServer
	inbounds map[string]*core.InboundHandlerConfig
	users    map[string]*protocol.User
}

func (s *fakeHandlerServer) ListInbounds(context.Context, *handlercmd.ListInboundsRequest) (*handlercmd.ListInboundsResponse, error) {
	response := &handlercmd.ListInboundsResponse{}
	for _, inbound := range s.inbounds {
		response.Inbounds = append(response.Inbounds, inbound)
	}
	return response, nil
}

func (s *fakeHandlerServer) AddInbound(_ context.Context, request *handlercmd.AddInboundRequest) (*handlercmd.AddInboundResponse, error) {
	s.inbounds[request.GetInbound().GetTag()] = request.GetInbound()
	return &handlercmd.AddInboundResponse{}, nil
}

func (s *fakeHandlerServer) AlterInbound(_ context.Context, request *handlercmd.AlterInboundRequest) (*handlercmd.AlterInboundResponse, error) {
	instance, err := request.GetOperation().GetInstance()
	if err != nil {
		return nil, err
	}
	switch op := instance.(type) {
	case *handlercmd.AddUserOperation:
		s.users[op.GetUser().GetEmail()] = op.GetUser()
	case *handlercmd.RemoveUserOperation:
		delete(s.users, op.GetEmail())
	}
	return &handlercmd.AlterInboundResponse{}, nil
}

func runtimeClient(t *testing.T, server runtimev1.RuntimeServiceServer, handler handlercmd.HandlerServiceServer) (runtimev1.RuntimeServiceClient, handlercmd.HandlerServiceClient, func()) {
	t.Helper()
	listener := bufconn.Listen(1024 * 1024)
	grpcServer := grpc.NewServer()
	runtimev1.RegisterRuntimeServiceServer(grpcServer, server)
	handlercmd.RegisterHandlerServiceServer(grpcServer, handler)
	go func() {
		_ = grpcServer.Serve(listener)
	}()
	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		t.Fatalf("DialContext() error = %v", err)
	}
	return runtimev1.NewRuntimeServiceClient(conn), handlercmd.NewHandlerServiceClient(conn), func() {
		_ = conn.Close()
		grpcServer.Stop()
		_ = listener.Close()
	}
}
