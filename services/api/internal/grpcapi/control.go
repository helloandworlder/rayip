package grpcapi

import (
	"context"
	"errors"
	"io"
	"net"

	"github.com/google/uuid"
	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
	"github.com/rayip/rayip/services/api/internal/config"
	"github.com/rayip/rayip/services/api/internal/node"
	"github.com/rayip/rayip/services/api/internal/runtimelab"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type ControlServer struct {
	controlv1.UnimplementedNodeControlServiceServer

	cfg     config.Config
	nodes   *node.Service
	runtime *RuntimeDispatcher
	log     *zap.Logger
}

func NewControlServer(cfg config.Config, nodes *node.Service, runtime *RuntimeDispatcher, log *zap.Logger) *ControlServer {
	return &ControlServer{cfg: cfg, nodes: nodes, runtime: runtime, log: log}
}

func (s *ControlServer) Connect(stream grpc.BidiStreamingServer[controlv1.AgentEnvelope, controlv1.ControlEnvelope]) error {
	ctx := stream.Context()
	first, err := stream.Recv()
	if err != nil {
		return err
	}

	hello := first.GetHello()
	if hello == nil {
		return errors.New("first control envelope must contain AgentHello")
	}
	if err := validateEnrollmentToken(s.cfg.Node.EnrollmentToken, hello.GetEnrollmentToken()); err != nil {
		return err
	}
	observation := hello.GetRuntime()
	if observation == nil {
		observation = &controlv1.RuntimeObservation{
			AgentVersion:  hello.GetAgentVersion(),
			XrayVersion:   hello.GetXrayVersion(),
			BundleVersion: hello.GetBundleVersion(),
			Capabilities:  hello.GetCapabilities(),
		}
	}
	verdict := negotiateRuntime(defaultRuntimeNegotiationPolicy(), observation)

	sessionID := uuid.NewString()
	leaseTTL := int(hello.GetLeaseTtlSeconds())
	if leaseTTL <= 0 {
		leaseTTL = s.cfg.Node.LeaseTTLSeconds
	}
	summary, err := s.nodes.RegisterLease(ctx, node.LeaseInput{
		NodeCode:        hello.GetNodeCode(),
		SessionID:       sessionID,
		APIInstanceID:   s.cfg.Service.InstanceID,
		BundleVersion:   observation.GetBundleVersion(),
		AgentVersion:    observation.GetAgentVersion(),
		XrayVersion:     observation.GetXrayVersion(),
		Capabilities:    observation.GetCapabilities(),
		Sequence:        hello.GetSequence(),
		LeaseTTLSeconds: leaseTTL,
	})
	if err != nil {
		return err
	}
	unregister := s.runtime.Register(summary.ID, stream)
	defer unregister()

	if err := stream.Send(&controlv1.ControlEnvelope{
		RequestId: first.GetRequestId(),
		Payload: &controlv1.ControlEnvelope_Welcome{Welcome: &controlv1.AgentWelcome{
			NodeId:          summary.ID,
			SessionId:       sessionID,
			LeaseTtlSeconds: int64(leaseTTL),
			ApiInstanceId:   s.cfg.Service.InstanceID,
			RuntimeVerdict:  verdict,
		}},
	}); err != nil {
		return err
	}

	s.log.Info("node connected", zap.String("node_code", hello.GetNodeCode()), zap.String("node_id", summary.ID))
	for {
		envelope, err := stream.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if lease := envelope.GetLease(); lease != nil {
			observation := lease.GetRuntime()
			if observation == nil {
				observation = &controlv1.RuntimeObservation{
					AgentVersion:  lease.GetAgentVersion(),
					XrayVersion:   lease.GetXrayVersion(),
					BundleVersion: lease.GetBundleVersion(),
					Capabilities:  lease.GetCapabilities(),
				}
			}
			_, err := s.nodes.RegisterLease(ctx, node.LeaseInput{
				NodeID:          lease.GetNodeId(),
				NodeCode:        lease.GetNodeCode(),
				SessionID:       lease.GetSessionId(),
				APIInstanceID:   s.cfg.Service.InstanceID,
				BundleVersion:   observation.GetBundleVersion(),
				AgentVersion:    observation.GetAgentVersion(),
				XrayVersion:     observation.GetXrayVersion(),
				Capabilities:    observation.GetCapabilities(),
				Sequence:        lease.GetSequence(),
				LeaseTTLSeconds: int(lease.GetLeaseTtlSeconds()),
			})
			if err != nil {
				return err
			}
			if err := stream.Send(&controlv1.ControlEnvelope{
				RequestId: envelope.GetRequestId(),
				Payload: &controlv1.ControlEnvelope_Ack{Ack: &controlv1.ControlAck{
					Code:    "LEASE_OK",
					Message: "lease renewed",
				}},
			}); err != nil {
				return err
			}
			continue
		}
		if result := envelope.GetRuntimeResult(); result != nil {
			s.runtime.HandleResult(runtimelab.ResultFromProto(result))
		}
	}
}

func NewGRPCServer(control *ControlServer) *grpc.Server {
	server := grpc.NewServer()
	controlv1.RegisterNodeControlServiceServer(server, control)
	return server
}

func RegisterLifecycle(lc fx.Lifecycle, cfg config.Config, server *grpc.Server, log *zap.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			listener, err := net.Listen("tcp", cfg.GRPC.Addr)
			if err != nil {
				return err
			}
			go func() {
				if err := server.Serve(listener); err != nil {
					log.Error("grpc server stopped", zap.Error(err))
				}
			}()
			log.Info("grpc server listening", zap.String("addr", cfg.GRPC.Addr))
			return nil
		},
		OnStop: func(ctx context.Context) error {
			stopped := make(chan struct{})
			go func() {
				server.GracefulStop()
				close(stopped)
			}()
			select {
			case <-stopped:
				return nil
			case <-ctx.Done():
				server.Stop()
				return ctx.Err()
			}
		},
	})
}
