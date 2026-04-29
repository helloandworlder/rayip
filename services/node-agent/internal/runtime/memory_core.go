package runtime

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"sync"
	"time"
)

type Core interface {
	UpsertAccount(ctx context.Context, account Account) error
	DisableAccount(ctx context.Context, proxyAccountID string, generation uint64) error
	DeleteAccount(ctx context.Context, proxyAccountID string) error
	Usage(ctx context.Context, proxyAccountID string) (Usage, error)
	Probe(ctx context.Context, proxyAccountID string) (Usage, error)
	Digest(ctx context.Context) (Digest, error)
}

type MemoryCore struct {
	mu          sync.RWMutex
	accounts    map[string]Account
	usage       map[string]Usage
	egress      map[string]*bucket
	ingress     map[string]*bucket
	windows     map[string]trafficWindow
	fairPoolBPS uint64
}

func NewMemoryCore() *MemoryCore {
	return &MemoryCore{
		accounts: map[string]Account{},
		usage:    map[string]Usage{},
		egress:   map[string]*bucket{},
		ingress:  map[string]*bucket{},
		windows:  map[string]trafficWindow{},
	}
}

func (c *MemoryCore) UpsertAccount(_ context.Context, account Account) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if account.RuntimeEmail == "" {
		account.RuntimeEmail = account.ProxyAccountID
	}
	if account.Status == "" {
		account.Status = AccountStatusEnabled
	}
	if account.Priority == 0 {
		account.Priority = 1
	}
	c.accounts[account.ProxyAccountID] = account
	c.ensureBucket(account.ProxyAccountID, DirectionEgress).setLimit(account.EgressLimitBPS)
	c.ensureBucket(account.ProxyAccountID, DirectionIngress).setLimit(account.IngressLimitBPS)
	return nil
}

func (c *MemoryCore) DisableAccount(_ context.Context, proxyAccountID string, generation uint64) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	account := c.accounts[proxyAccountID]
	account.Status = AccountStatusDisabled
	account.DesiredGeneration = generation
	c.accounts[proxyAccountID] = account
	return nil
}

func (c *MemoryCore) DeleteAccount(_ context.Context, proxyAccountID string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.accounts, proxyAccountID)
	delete(c.usage, proxyAccountID)
	delete(c.egress, proxyAccountID)
	delete(c.ingress, proxyAccountID)
	delete(c.windows, proxyAccountID)
	return nil
}

func (c *MemoryCore) Usage(_ context.Context, proxyAccountID string) (Usage, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	usage := c.usage[proxyAccountID]
	if usage.ProxyAccountID == "" {
		account := c.accounts[proxyAccountID]
		usage.ProxyAccountID = account.ProxyAccountID
		usage.RuntimeEmail = account.RuntimeEmail
	}
	return usage, nil
}

func (c *MemoryCore) Probe(ctx context.Context, proxyAccountID string) (Usage, error) {
	return c.Usage(ctx, proxyAccountID)
}

func (c *MemoryCore) Digest(_ context.Context) (Digest, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	items := make([]Account, 0, len(c.accounts))
	for _, account := range c.accounts {
		items = append(items, account)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ProxyAccountID < items[j].ProxyAccountID })

	digest := Digest{AccountCount: uint64(len(items))}
	for _, account := range items {
		switch account.Status {
		case AccountStatusDisabled:
			digest.DisabledCount++
		default:
			digest.EnabledCount++
		}
		if account.DesiredGeneration > digest.MaxGeneration {
			digest.MaxGeneration = account.DesiredGeneration
		}
	}
	payload, _ := json.Marshal(items)
	sum := sha256.Sum256(payload)
	digest.Hash = hex.EncodeToString(sum[:])
	return digest, nil
}

func (c *MemoryCore) Account(proxyAccountID string) (Account, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	account, ok := c.accounts[proxyAccountID]
	return account, ok
}

func (c *MemoryCore) SetFairPoolBPS(bytesPerSecond uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.fairPoolBPS = bytesPerSecond
}

func (c *MemoryCore) AcquireConnection(proxyAccountID string) (func(), error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	account, ok := c.accounts[proxyAccountID]
	if !ok {
		return nil, errors.New("account not found")
	}
	if account.Status == AccountStatusDisabled {
		return nil, errors.New("account disabled")
	}
	usage := c.usage[proxyAccountID]
	if usage.ProxyAccountID == "" {
		usage.ProxyAccountID = account.ProxyAccountID
		usage.RuntimeEmail = account.RuntimeEmail
	}
	if account.MaxConnections > 0 && usage.ActiveConnections >= uint64(account.MaxConnections) {
		return nil, errors.New("connection limit exceeded")
	}
	usage.ActiveConnections++
	c.usage[proxyAccountID] = usage
	var once sync.Once
	return func() {
		once.Do(func() {
			c.mu.Lock()
			defer c.mu.Unlock()
			usage := c.usage[proxyAccountID]
			if usage.ActiveConnections > 0 {
				usage.ActiveConnections--
				c.usage[proxyAccountID] = usage
			}
		})
	}, nil
}

func (c *MemoryCore) AllowBytesAt(proxyAccountID string, direction Direction, requested uint64, now time.Time) uint64 {
	c.mu.Lock()
	defer c.mu.Unlock()
	account, ok := c.accounts[proxyAccountID]
	if !ok || requested == 0 || account.Status == AccountStatusDisabled {
		return 0
	}
	limit := account.EgressLimitBPS
	if direction == DirectionIngress {
		limit = account.IngressLimitBPS
	}
	b := c.ensureBucket(proxyAccountID, direction)
	if limit == 0 && c.fairPoolBPS > 0 {
		limit = c.fairShareBPSLocked(proxyAccountID, now)
		b.setLimit(limit)
	}
	if limit == 0 {
		return requested
	}
	return b.take(requested, now)
}

func (c *MemoryCore) FairShareBPS(proxyAccountID string, now time.Time) uint64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.fairShareBPSLocked(proxyAccountID, now)
}

func (c *MemoryCore) RecordTrafficAt(proxyAccountID string, direction Direction, bytes uint64, now time.Time) *AbuseEvent {
	c.mu.Lock()
	defer c.mu.Unlock()
	account, ok := c.accounts[proxyAccountID]
	if !ok || bytes == 0 {
		return nil
	}
	usage := c.usage[proxyAccountID]
	if usage.ProxyAccountID == "" {
		usage.ProxyAccountID = account.ProxyAccountID
		usage.RuntimeEmail = account.RuntimeEmail
	}
	if direction == DirectionIngress {
		usage.RxBytes += bytes
	} else {
		usage.TxBytes += bytes
	}
	c.usage[proxyAccountID] = usage

	window := c.windows[proxyAccountID]
	window.add(bytes, now)
	c.windows[proxyAccountID] = window
	if account.AbuseBytesPerMin == 0 || window.bytes <= account.AbuseBytesPerMin {
		return nil
	}
	action := account.AbuseAction
	if action == "" {
		action = AbuseActionReportOnly
	}
	if action == AbuseActionDisableAndReport {
		account.Status = AccountStatusDisabled
		c.accounts[proxyAccountID] = account
	}
	return &AbuseEvent{
		ProxyAccountID: proxyAccountID,
		RuntimeEmail:   account.RuntimeEmail,
		Action:         action,
		WindowBytes:    window.bytes,
		Threshold:      account.AbuseBytesPerMin,
	}
}

func (c *MemoryCore) ensureBucket(proxyAccountID string, direction Direction) *bucket {
	target := c.egress
	if direction == DirectionIngress {
		target = c.ingress
	}
	b := target[proxyAccountID]
	if b == nil {
		b = &bucket{}
		target[proxyAccountID] = b
	}
	return b
}

func (c *MemoryCore) fairShareBPSLocked(proxyAccountID string, now time.Time) uint64 {
	if c.fairPoolBPS == 0 {
		return 0
	}
	total := uint64(0)
	weights := map[string]uint64{}
	for id, account := range c.accounts {
		if account.Status == AccountStatusDisabled {
			continue
		}
		weight := uint64(account.Priority)
		if weight == 0 {
			weight = 1
		}
		window := c.windows[id]
		if !window.start.IsZero() && now.Sub(window.start) < time.Minute && window.bytes > c.fairPoolBPS {
			weight = maxUint64(1, weight/2)
		}
		weights[id] = weight
		total += weight
	}
	if total == 0 {
		return 0
	}
	return c.fairPoolBPS * weights[proxyAccountID] / total
}

type bucket struct {
	limit  uint64
	tokens uint64
	last   time.Time
}

func (b *bucket) setLimit(limit uint64) {
	if b.limit == limit {
		return
	}
	b.limit = limit
	b.tokens = limit
	b.last = time.Time{}
}

func (b *bucket) take(requested uint64, now time.Time) uint64 {
	if b.limit == 0 {
		return requested
	}
	if b.last.IsZero() {
		b.last = now
		if b.tokens == 0 {
			b.tokens = b.limit
		}
	} else if now.After(b.last) {
		refill := uint64(now.Sub(b.last).Seconds() * float64(b.limit))
		b.tokens = minUint64(b.limit, b.tokens+refill)
		b.last = now
	}
	allowed := minUint64(requested, b.tokens)
	b.tokens -= allowed
	return allowed
}

type trafficWindow struct {
	start time.Time
	bytes uint64
}

func (w *trafficWindow) add(bytes uint64, now time.Time) {
	if w.start.IsZero() || now.Sub(w.start) >= time.Minute {
		w.start = now
		w.bytes = 0
	}
	w.bytes += bytes
}

func minUint64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}

func maxUint64(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}
