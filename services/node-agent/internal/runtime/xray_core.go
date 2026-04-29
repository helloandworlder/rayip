package runtime

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	runtimev1 "github.com/rayip/rayip/packages/proto/gen/go/rayip/runtime/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type XrayCore struct {
	client   runtimev1.RuntimeServiceClient
	conn     *grpc.ClientConn
	mu       sync.RWMutex
	emails   map[string]string
	policies map[string]*runtimev1.AccountPolicy
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
		conn:     conn,
		emails:   map[string]string{},
		policies: map[string]*runtimev1.AccountPolicy{},
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
	return &XrayCore{client: client, emails: map[string]string{}, policies: map[string]*runtimev1.AccountPolicy{}}
}

func (c *XrayCore) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}

func (c *XrayCore) UpsertAccount(ctx context.Context, account Account) error {
	email := account.RuntimeEmail
	if email == "" {
		email = account.ProxyAccountID
	}
	policy := toRuntimePolicy(account)
	_, err := c.client.UpsertAccountPolicy(ctx, &runtimev1.UpsertAccountPolicyRequest{
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
	_, err := c.client.UpsertAccountPolicy(ctx, &runtimev1.UpsertAccountPolicyRequest{
		Policy: policy,
	})
	if err == nil {
		c.rememberPolicy(proxyAccountID, policy.GetEmail(), policy)
	}
	return err
}

func (c *XrayCore) DeleteAccount(ctx context.Context, proxyAccountID string) error {
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

func (c *XrayCore) forgetPolicy(proxyAccountID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.emails, proxyAccountID)
	delete(c.policies, proxyAccountID)
}

func (c *XrayCore) runtimeEmail(proxyAccountID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if email := c.emails[proxyAccountID]; email != "" {
		return email
	}
	return proxyAccountID
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
