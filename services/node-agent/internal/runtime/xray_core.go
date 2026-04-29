package runtime

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	runtimev1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/runtime/v1"
	"github.com/xtls/xray-core/app/proxyman"
	handlercmd "github.com/xtls/xray-core/app/proxyman/command"
	xnet "github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/common/protocol"
	"github.com/xtls/xray-core/common/serial"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/proxy/http"
	"github.com/xtls/xray-core/proxy/socks"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type XrayCore struct {
	client   runtimev1.RuntimeServiceClient
	handler  handlercmd.HandlerServiceClient
	conn     *grpc.ClientConn
	mu       sync.RWMutex
	emails   map[string]string
	policies map[string]*runtimev1.AccountPolicy
	inbounds map[string]string
}

func NewXrayCore(ctx context.Context, addr string) (*XrayCore, error) {
	if strings.TrimSpace(addr) == "" {
		return nil, errors.New("xray runtime gRPC addr is required")
	}
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	core := &XrayCore{
		client:   runtimev1.NewRuntimeServiceClient(conn),
		handler:  handlercmd.NewHandlerServiceClient(conn),
		conn:     conn,
		emails:   map[string]string{},
		policies: map[string]*runtimev1.AccountPolicy{},
		inbounds: map[string]string{},
	}
	for {
		if _, err := core.client.GetCapabilities(ctx, &runtimev1.GetCapabilitiesRequest{}); err == nil {
			return core, nil
		} else if ctx.Err() != nil {
			_ = conn.Close()
			return nil, err
		}
		timer := time.NewTimer(100 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			_ = conn.Close()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}

func NewXrayCoreWithClient(client runtimev1.RuntimeServiceClient) *XrayCore {
	return NewXrayCoreWithClients(client, nil)
}

func NewXrayCoreWithClients(client runtimev1.RuntimeServiceClient, handler handlercmd.HandlerServiceClient) *XrayCore {
	return &XrayCore{
		client:   client,
		handler:  handler,
		emails:   map[string]string{},
		policies: map[string]*runtimev1.AccountPolicy{},
		inbounds: map[string]string{},
	}
}

func (c *XrayCore) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *XrayCore) UpsertAccount(ctx context.Context, account Account) error {
	tag, err := c.ensureInbound(ctx, account)
	if err != nil {
		return err
	}
	email := account.RuntimeEmail
	if email == "" {
		email = account.ProxyAccountID
	}
	if err := c.replaceInboundUser(ctx, tag, account, email); err != nil {
		return err
	}
	policy := toRuntimePolicy(account)
	_, err = c.client.UpsertAccountPolicy(ctx, &runtimev1.UpsertAccountPolicyRequest{
		Policy: policy,
	})
	if err == nil {
		c.rememberPolicy(account.ProxyAccountID, email, policy)
	}
	return err
}

func (c *XrayCore) DisableAccount(ctx context.Context, proxyAccountID string, generation uint64) error {
	policy := c.runtimePolicy(proxyAccountID)
	policy.Disabled = true
	policy.Generation = generation
	if tag := c.runtimeInboundTag(proxyAccountID); tag != "" {
		if err := c.removeInboundUser(ctx, tag, policy.GetEmail()); err != nil {
			return err
		}
	}
	_, err := c.client.UpsertAccountPolicy(ctx, &runtimev1.UpsertAccountPolicyRequest{
		Policy: policy,
	})
	if err == nil {
		c.rememberPolicy(proxyAccountID, policy.GetEmail(), policy)
	}
	return err
}

func (c *XrayCore) DeleteAccount(ctx context.Context, proxyAccountID string) error {
	email := c.runtimeEmail(proxyAccountID)
	if tag := c.runtimeInboundTag(proxyAccountID); tag != "" {
		if err := c.removeInboundUser(ctx, tag, email); err != nil {
			return err
		}
	}
	_, err := c.client.RemoveAccountPolicy(ctx, &runtimev1.RemoveAccountPolicyRequest{Email: c.runtimeEmail(proxyAccountID)})
	if err == nil {
		c.forgetPolicy(proxyAccountID)
	}
	return err
}

func (c *XrayCore) Usage(ctx context.Context, proxyAccountID string) (Usage, error) {
	response, err := c.client.GetUserSpeed(ctx, &runtimev1.GetUserSpeedRequest{Email: c.runtimeEmail(proxyAccountID)})
	if err != nil {
		return Usage{}, err
	}
	return fromRuntimeSpeed(proxyAccountID, response.GetSpeed()), nil
}

func (c *XrayCore) rememberPolicy(proxyAccountID string, email string, policy *runtimev1.AccountPolicy) {
	if proxyAccountID == "" || email == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.emails[proxyAccountID] = email
	if policy != nil {
		copied := *policy
		c.policies[proxyAccountID] = &copied
	}
}

func (c *XrayCore) rememberInbound(proxyAccountID string, tag string) {
	if proxyAccountID == "" || tag == "" {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.inbounds[proxyAccountID] = tag
}

func (c *XrayCore) forgetPolicy(proxyAccountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.emails, proxyAccountID)
	delete(c.policies, proxyAccountID)
	delete(c.inbounds, proxyAccountID)
}

func (c *XrayCore) runtimeEmail(proxyAccountID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if email := c.emails[proxyAccountID]; email != "" {
		return email
	}
	return proxyAccountID
}

func (c *XrayCore) runtimeInboundTag(proxyAccountID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.inbounds[proxyAccountID]
}

func (c *XrayCore) ensureInbound(ctx context.Context, account Account) (string, error) {
	if c.handler == nil {
		return "", errors.New("xray handler service client is required")
	}
	tag := inboundTag(account)
	response, err := c.handler.ListInbounds(ctx, &handlercmd.ListInboundsRequest{IsOnlyTags: true})
	if err != nil {
		return "", err
	}
	for _, inbound := range response.GetInbounds() {
		if inbound.GetTag() == tag {
			c.rememberInbound(account.ProxyAccountID, tag)
			return tag, nil
		}
	}
	if _, err := c.handler.AddInbound(ctx, &handlercmd.AddInboundRequest{Inbound: inboundConfig(account, tag)}); err != nil {
		if status.Code(err) != codes.AlreadyExists {
			return "", err
		}
	}
	c.rememberInbound(account.ProxyAccountID, tag)
	return tag, nil
}

func (c *XrayCore) replaceInboundUser(ctx context.Context, tag string, account Account, email string) error {
	_ = c.removeInboundUser(ctx, tag, email)
	_, err := c.handler.AlterInbound(ctx, &handlercmd.AlterInboundRequest{
		Tag: tag,
		Operation: serial.ToTypedMessage(&handlercmd.AddUserOperation{
			User: xrayUser(account, email),
		}),
	})
	return err
}

func (c *XrayCore) removeInboundUser(ctx context.Context, tag string, email string) error {
	if c.handler == nil || tag == "" || email == "" {
		return nil
	}
	_, err := c.handler.AlterInbound(ctx, &handlercmd.AlterInboundRequest{
		Tag:       tag,
		Operation: serial.ToTypedMessage(&handlercmd.RemoveUserOperation{Email: email}),
	})
	if status.Code(err) == codes.NotFound {
		return nil
	}
	return err
}

func inboundConfig(account Account, tag string) *core.InboundHandlerConfig {
	return &core.InboundHandlerConfig{
		Tag: tag,
		ReceiverSettings: serial.ToTypedMessage(&proxyman.ReceiverConfig{
			PortList: &xnet.PortList{Range: []*xnet.PortRange{{From: account.Port, To: account.Port}}},
			Listen:   ipOrDomain(account.ListenIP),
		}),
		ProxySettings: serial.ToTypedMessage(proxySettings(account)),
	}
}

func proxySettings(account Account) proto.Message {
	switch account.Protocol {
	case ProtocolHTTP:
		return &http.ServerConfig{
			Accounts:      map[string]string{},
			AccountEmails: map[string]string{},
			UserLevel:     0,
		}
	default:
		return &socks.ServerConfig{
			AuthType:      socks.AuthType_PASSWORD,
			Accounts:      map[string]string{},
			AccountEmails: map[string]string{},
			UdpEnabled:    false,
			UserLevel:     0,
		}
	}
}

func xrayUser(account Account, email string) *protocol.User {
	switch account.Protocol {
	case ProtocolHTTP:
		return &protocol.User{
			Email:   email,
			Account: serial.ToTypedMessage(&http.Account{Username: account.Username, Password: account.Password}),
		}
	default:
		return &protocol.User{
			Email:   email,
			Account: serial.ToTypedMessage(&socks.Account{Username: account.Username, Password: account.Password}),
		}
	}
}

func inboundTag(account Account) string {
	return "rayip-" + strings.ToLower(string(account.Protocol)) + "-" + sanitizeTagPart(account.ListenIP) + "-" + fmt.Sprint(account.Port)
}

func sanitizeTagPart(value string) string {
	replacer := strings.NewReplacer(":", "_", ".", "_", "/", "_", "[", "", "]", "")
	return replacer.Replace(value)
}

func ipOrDomain(value string) *xnet.IPOrDomain {
	if value == "" {
		value = "127.0.0.1"
	}
	if ip := net.ParseIP(value); ip != nil {
		return &xnet.IPOrDomain{Address: &xnet.IPOrDomain_Ip{Ip: ip}}
	}
	return &xnet.IPOrDomain{Address: &xnet.IPOrDomain_Domain{Domain: value}}
}

func (c *XrayCore) runtimePolicy(proxyAccountID string) *runtimev1.AccountPolicy {
	c.mu.RLock()
	if policy := c.policies[proxyAccountID]; policy != nil {
		copied := *policy
		c.mu.RUnlock()
		return &copied
	}
	email := c.emails[proxyAccountID]
	c.mu.RUnlock()
	if email == "" {
		email = proxyAccountID
	}
	return &runtimev1.AccountPolicy{Email: email}
}

func (c *XrayCore) Probe(ctx context.Context, proxyAccountID string) (Usage, error) {
	return c.Usage(ctx, proxyAccountID)
}

func (c *XrayCore) Digest(ctx context.Context) (Digest, error) {
	response, err := c.client.GetDigest(ctx, &runtimev1.GetDigestRequest{})
	if err != nil {
		return Digest{}, err
	}
	return fromRuntimeDigest(response.GetDigest()), nil
}

func toRuntimePolicy(account Account) *runtimev1.AccountPolicy {
	email := account.RuntimeEmail
	if email == "" {
		email = account.ProxyAccountID
	}
	return &runtimev1.AccountPolicy{
		Email:            email,
		EgressLimitBps:   account.EgressLimitBPS,
		IngressLimitBps:  account.IngressLimitBPS,
		MaxConnections:   account.MaxConnections,
		Priority:         account.Priority,
		Generation:       account.DesiredGeneration,
		Disabled:         account.Status == AccountStatusDisabled,
		AbuseBytesPerMin: account.AbuseBytesPerMin,
		AbuseAction:      toRuntimeAbuseAction(account.AbuseAction),
	}
}

func fromRuntimeSpeed(proxyAccountID string, speed *runtimev1.UserSpeed) Usage {
	if speed == nil {
		return Usage{ProxyAccountID: proxyAccountID, RuntimeEmail: proxyAccountID}
	}
	return Usage{
		ProxyAccountID:    proxyAccountID,
		RuntimeEmail:      speed.GetEmail(),
		RxBytes:           speed.GetRxBytes(),
		TxBytes:           speed.GetTxBytes(),
		ActiveConnections: speed.GetActiveConnections(),
		RxBytesPerSecond:  speed.GetRxBytesPerSecond(),
		TxBytesPerSecond:  speed.GetTxBytesPerSecond(),
	}
}

func fromRuntimeDigest(digest *runtimev1.Digest) Digest {
	if digest == nil {
		return Digest{}
	}
	return Digest{
		AccountCount:  digest.GetAccountCount(),
		EnabledCount:  digest.GetEnabledCount(),
		DisabledCount: digest.GetDisabledCount(),
		MaxGeneration: digest.GetMaxGeneration(),
		Hash:          digest.GetHash(),
	}
}

func toRuntimeAbuseAction(action AbuseAction) runtimev1.AbuseAction {
	switch action {
	case AbuseActionDisableAndReport:
		return runtimev1.AbuseAction_ABUSE_ACTION_DISABLE_AND_REPORT
	case AbuseActionReportOnly:
		return runtimev1.AbuseAction_ABUSE_ACTION_REPORT_ONLY
	default:
		return runtimev1.AbuseAction_ABUSE_ACTION_UNSPECIFIED
	}
}
