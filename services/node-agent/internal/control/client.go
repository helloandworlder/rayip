package control

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	controlv1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/control/v1"
	"github.com/rayip/rayip/services/node-agent/internal/config"
	"github.com/rayip/rayip/services/node-agent/internal/runtime"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	cfg      config.Config
	endpoint *runtime.Endpoint
	runtime  *runtime.Manager
	discover func() (runtime.DiscoveryInfo, error)
	log      *zap.Logger
}

func NewClient(cfg config.Config, endpoint *runtime.Endpoint, manager *runtime.Manager, log *zap.Logger) *Client {
	return &Client{
		cfg:      cfg,
		endpoint: endpoint,
		runtime:  manager,
		discover: func() (runtime.DiscoveryInfo, error) {
			return runtime.Discover(runtime.DiscoveryConfig{
				AgentVersion: cfg.Runtime.AgentVersion,
				ManifestPath: cfg.Runtime.ManifestPath,
				CoreMode:     cfg.Runtime.CoreMode,
				XrayGRPCAddr: endpoint.GRPCAddr(),
			})
		},
		log: log,
	}
}

func (c *Client) Run(ctx context.Context) error {
	for {
		if err := c.connectOnce(ctx); err != nil {
			c.log.Warn("control stream ended", zap.Error(err))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
}

func (c *Client) connectOnce(ctx context.Context) error {
	conn, err := grpc.NewClient(c.cfg.API.GRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()

	client := controlv1.NewNodeControlServiceClient(conn)
	stream, err := client.Connect(ctx)
	if err != nil {
		return err
	}
	sendMu := &sync.Mutex{}

	seq := uint64(1)
	runtimeInfo, err := c.discover()
	if err != nil {
		return err
	}
	sendMu.Lock()
	if err := stream.Send(&controlv1.AgentEnvelope{
		RequestId: uuid.NewString(),
		Payload: &controlv1.AgentEnvelope_Hello{Hello: &controlv1.AgentHello{
			NodeCode:        c.cfg.Node.Code,
			EnrollmentToken: c.cfg.Node.EnrollmentToken,
			AgentVersion:    runtimeInfo.AgentVersion,
			XrayVersion:     runtimeInfo.XrayVersion,
			BundleVersion:   runtimeInfo.BundleVersion,
			Capabilities:    runtimeInfo.Capabilities,
			Sequence:        seq,
			LeaseTtlSeconds: int64(c.cfg.Lease.TTL / time.Second),
			Runtime:         runtime.ObservationToProto(runtimeInfo),
		}},
	}); err != nil {
		sendMu.Unlock()
		return err
	}
	sendMu.Unlock()

	welcomeEnvelope, err := stream.Recv()
	if err != nil {
		return err
	}
	welcome := welcomeEnvelope.GetWelcome()
	if welcome == nil {
		return nil
	}
	c.log.Info("registered with api",
		zap.String("node_id", welcome.GetNodeId()),
		zap.String("session_id", welcome.GetSessionId()),
		zap.String("api_instance_id", welcome.GetApiInstanceId()),
	)

	ticker := time.NewTicker(c.cfg.Lease.Interval)
	defer ticker.Stop()
	recvErr := make(chan error, 1)
	go c.receiveControl(ctx, stream, sendMu, recvErr)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-recvErr:
			return err
		case <-ticker.C:
			seq++
			runtimeInfo, discoverErr := c.discover()
			if discoverErr != nil {
				return discoverErr
			}
			sendMu.Lock()
			sendErr := stream.Send(&controlv1.AgentEnvelope{
				RequestId: uuid.NewString(),
				Payload: &controlv1.AgentEnvelope_Lease{Lease: &controlv1.LeaseRenewal{
					NodeId:          welcome.GetNodeId(),
					NodeCode:        c.cfg.Node.Code,
					SessionId:       welcome.GetSessionId(),
					AgentVersion:    runtimeInfo.AgentVersion,
					XrayVersion:     runtimeInfo.XrayVersion,
					BundleVersion:   runtimeInfo.BundleVersion,
					Capabilities:    runtimeInfo.Capabilities,
					Sequence:        seq,
					LeaseTtlSeconds: int64(c.cfg.Lease.TTL / time.Second),
					Runtime:         runtime.ObservationToProto(runtimeInfo),
				}},
			})
			sendMu.Unlock()
			if sendErr != nil {
				return sendErr
			}
		}
	}
}

func (c *Client) receiveControl(ctx context.Context, stream grpc.BidiStreamingClient[controlv1.AgentEnvelope, controlv1.ControlEnvelope], sendMu *sync.Mutex, recvErr chan<- error) {
	for {
		envelope, err := stream.Recv()
		if err != nil {
			recvErr <- err
			return
		}
		if cmd := envelope.GetRuntimeCommand(); cmd != nil {
			result, applyErr := c.runtime.Apply(ctx, runtime.CommandFromProto(cmd))
			if applyErr != nil {
				c.log.Warn("runtime command failed", zap.String("command_id", cmd.GetCommandId()), zap.Error(applyErr))
			}
			sendMu.Lock()
			err := stream.Send(&controlv1.AgentEnvelope{
				RequestId: envelope.GetRequestId(),
				Payload: &controlv1.AgentEnvelope_RuntimeResult{
					RuntimeResult: runtime.ResultToProto(result),
				},
			})
			sendMu.Unlock()
			if err != nil {
				recvErr <- err
				return
			}
		}
	}
}
