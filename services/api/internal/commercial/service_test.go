package commercial

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPaymentCallbackCreditsWalletOnce(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	user, err := svc.Register(ctx, RegisterInput{Email: "buyer@example.com", Password: "secret123"})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	order, err := svc.CreatePaymentOrder(ctx, user.ID, CreatePaymentOrderInput{AmountCents: 10000})
	if err != nil {
		t.Fatalf("CreatePaymentOrder() error = %v", err)
	}

	if _, err := svc.HandleMockPaymentCallback(ctx, PaymentCallbackInput{
		PaymentOrderID:  order.ID,
		ProviderTradeNo: "trade-001",
		PaidAmountCents: 10000,
	}); err != nil {
		t.Fatalf("first callback error = %v", err)
	}
	if _, err := svc.HandleMockPaymentCallback(ctx, PaymentCallbackInput{
		PaymentOrderID:  order.ID,
		ProviderTradeNo: "trade-001",
		PaidAmountCents: 10000,
	}); err != nil {
		t.Fatalf("duplicate callback error = %v", err)
	}

	wallet, err := svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 10000 || wallet.AvailableCents != 10000 {
		t.Fatalf("wallet = %#v, want exactly one credit", wallet)
	}
	ledger, err := svc.ListWalletLedger(ctx, LedgerFilter{UserID: user.ID})
	if err != nil {
		t.Fatalf("ListWalletLedger() error = %v", err)
	}
	if len(ledger) != 1 || ledger[0].Type != LedgerTypeCreditRecharge {
		t.Fatalf("ledger = %#v, want one recharge credit", ledger)
	}
}

func TestCreateOrderRequiresBalanceAndIsIdempotent(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-1", ProtocolSOCKS5)

	input := CreateOrderInput{
		ProductID:       "static-residential",
		InventoryID:     inventory.ID,
		Protocol:        ProtocolSOCKS5,
		DurationDays:    30,
		Quantity:        1,
		IdempotencyKey:  "idem-order-1",
		EgressLimitBPS:  1024,
		IngressLimitBPS: 1024,
		MaxConnections:  2,
	}
	first, err := svc.CreateOrder(ctx, user.ID, input)
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	second, err := svc.CreateOrder(ctx, user.ID, input)
	if err != nil {
		t.Fatalf("duplicate CreateOrder() error = %v", err)
	}
	if first.ID != second.ID {
		t.Fatalf("duplicate order id = %s, want %s", second.ID, first.ID)
	}

	wallet, err := svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 10000 || wallet.HeldCents != 3000 || wallet.AvailableCents != 7000 {
		t.Fatalf("wallet after idempotent order = %#v", wallet)
	}
	ledger, err := svc.ListWalletLedger(ctx, LedgerFilter{UserID: user.ID})
	if err != nil {
		t.Fatalf("ListWalletLedger() error = %v", err)
	}
	holdEntries := 0
	for _, item := range ledger {
		if item.Type == LedgerTypeHold {
			holdEntries++
		}
	}
	if holdEntries != 1 {
		t.Fatalf("hold entries = %d, want 1; ledger=%#v", holdEntries, ledger)
	}

	poorUser := testUserWithBalance(t, svc, 1000)
	otherInventory := seedSellableInventory(t, svc, "node-1", ProtocolSOCKS5)
	_, err = svc.CreateOrder(ctx, poorUser.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    otherInventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-poor",
	})
	if !errors.Is(err, ErrInsufficientBalance) {
		t.Fatalf("CreateOrder() error = %v, want ErrInsufficientBalance", err)
	}
}

func TestCreateOrderAcceptsNodeProtocolCapabilityCaseInsensitively(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)

	line, err := svc.UpsertLine(ctx, UpsertLineInput{
		ID:       "node-lower-cap-line",
		RegionID: "us",
		CityID:   "nyc",
		NodeID:   "node-lower-cap",
		Name:     "lower capability line",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("UpsertLine() error = %v", err)
	}
	if err := svc.UpsertNodeRuntimeStatus(ctx, NodeRuntimeStatus{
		NodeID:         "node-lower-cap",
		Sellable:       true,
		Capabilities:   []string{"socks5", "http"},
		RuntimeVerdict: "ACCEPTED",
		LeaseOnline:    true,
	}); err != nil {
		t.Fatalf("UpsertNodeRuntimeStatus() error = %v", err)
	}
	inventory, err := svc.UpsertInventory(ctx, UpsertInventoryInput{
		LineID:    line.ID,
		NodeID:    "node-lower-cap",
		IP:        "203.0.113.20",
		Port:      18080,
		Protocols: []Protocol{ProtocolSOCKS5, ProtocolHTTP},
		Status:    InventoryStatusAvailable,
	})
	if err != nil {
		t.Fatalf("UpsertInventory() error = %v", err)
	}

	_, err = svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-lower-cap",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
}

func TestRuntimeAckControlsSettlementAndCredentialVisibility(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-ack", ProtocolSOCKS5)

	order, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-ack",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	proxies, err := svc.ListUserProxies(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserProxies() before ACK error = %v", err)
	}
	if len(proxies) != 0 {
		t.Fatalf("proxies before ACK = %#v, want no credentials", proxies)
	}

	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusACK,
		AppliedAt:      time.Date(2026, 4, 30, 12, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatalf("HandleRuntimeApplyResult(ACK) error = %v", err)
	}

	got, err := svc.GetOrder(ctx, user.ID, order.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if got.Status != OrderStatusDelivered {
		t.Fatalf("order status = %s, want DELIVERED", got.Status)
	}
	wallet, err := svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 7000 || wallet.HeldCents != 0 || wallet.AvailableCents != 7000 {
		t.Fatalf("wallet after ACK = %#v", wallet)
	}
	proxies, err = svc.ListUserProxies(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserProxies() after ACK error = %v", err)
	}
	if len(proxies) != 1 || proxies[0].Password == "" || proxies[0].ConnectionURI == "" {
		t.Fatalf("proxies after ACK = %#v, want visible credentials", proxies)
	}
}

func TestRuntimeFailureReleasesHoldAndInventory(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-nack", ProtocolHTTP)

	order, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolHTTP,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-nack",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}

	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusNACK,
		ErrorDetail:    "digest mismatch",
	}); err != nil {
		t.Fatalf("HandleRuntimeApplyResult(NACK) error = %v", err)
	}

	got, err := svc.GetOrder(ctx, user.ID, order.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if got.Status != OrderStatusFulfillmentFailed || got.FailureReason != "digest mismatch" {
		t.Fatalf("order after NACK = %#v", got)
	}
	wallet, err := svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 10000 || wallet.HeldCents != 0 || wallet.AvailableCents != 10000 {
		t.Fatalf("wallet after NACK = %#v", wallet)
	}
	catalog, err := svc.GetStaticResidentialCatalog(ctx)
	if err != nil {
		t.Fatalf("GetStaticResidentialCatalog() error = %v", err)
	}
	if catalog.TotalAvailable != 1 {
		t.Fatalf("catalog after release = %#v, want inventory available again", catalog)
	}
}

func TestRetryFulfillmentReholdsInventoryAndCanDeliver(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-retry", ProtocolHTTP)

	order, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolHTTP,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-retry",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusNACK,
		ErrorDetail:    "node rejected",
	}); err != nil {
		t.Fatalf("HandleRuntimeApplyResult(NACK) error = %v", err)
	}

	retried, err := svc.RetryFulfillment(ctx, order.ID)
	if err != nil {
		t.Fatalf("RetryFulfillment() error = %v", err)
	}
	if retried.Status != OrderStatusPendingRuntime || retried.FailureReason != "" {
		t.Fatalf("retried order = %#v, want pending runtime without failure reason", retried)
	}
	wallet, err := svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 10000 || wallet.HeldCents != 3000 || wallet.AvailableCents != 7000 {
		t.Fatalf("wallet after retry = %#v, want funds held again", wallet)
	}
	runtime := svc.runtime.(*fakeRuntimeWriter)
	if len(runtime.upserts) != 2 {
		t.Fatalf("runtime upserts = %#v, want initial attempt and retry", runtime.upserts)
	}

	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: retried.ProxyAccountID,
		Status:         RuntimeApplyStatusACK,
	}); err != nil {
		t.Fatalf("HandleRuntimeApplyResult(ACK) after retry error = %v", err)
	}
	delivered, err := svc.GetOrder(ctx, user.ID, order.ID)
	if err != nil {
		t.Fatalf("GetOrder() error = %v", err)
	}
	if delivered.Status != OrderStatusDelivered {
		t.Fatalf("order after retry ACK = %#v, want delivered", delivered)
	}
	wallet, err = svc.GetWallet(ctx, user.ID)
	if err != nil {
		t.Fatalf("GetWallet() error = %v", err)
	}
	if wallet.BalanceCents != 7000 || wallet.HeldCents != 0 || wallet.AvailableCents != 7000 {
		t.Fatalf("wallet after retry ACK = %#v", wallet)
	}
}

func TestDisableAckMarksProxyDisabledInsteadOfReactivating(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-disable", ProtocolSOCKS5)
	order, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-disable",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusACK,
	}); err != nil {
		t.Fatalf("initial ACK error = %v", err)
	}

	if _, err := svc.DisableProxy(ctx, user.ID, order.ProxyAccountID); err != nil {
		t.Fatalf("DisableProxy() error = %v", err)
	}
	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusACK,
	}); err != nil {
		t.Fatalf("disable ACK error = %v", err)
	}

	proxy, err := svc.GetProxy(ctx, user.ID, order.ProxyAccountID)
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if proxy.Status != "DISABLED" || proxy.LifecycleStatus != ProxyLifecycleDisabled {
		t.Fatalf("proxy after disable ACK = %#v, want disabled", proxy)
	}
	if proxy.Password != "" || proxy.ConnectionURI != "" {
		t.Fatalf("disabled proxy leaked credentials: %#v", proxy)
	}
}

func TestAdminProxyListingIncludesPendingAndFailedProxies(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	pendingInventory := seedSellableInventory(t, svc, "node-pending", ProtocolSOCKS5)
	failedInventory := seedSellableInventory(t, svc, "node-failed", ProtocolHTTP)

	pending, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    pendingInventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-pending-admin",
	})
	if err != nil {
		t.Fatalf("CreateOrder(pending) error = %v", err)
	}
	failed, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    failedInventory.ID,
		Protocol:       ProtocolHTTP,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-failed-admin",
	})
	if err != nil {
		t.Fatalf("CreateOrder(failed) error = %v", err)
	}
	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: failed.ProxyAccountID,
		Status:         RuntimeApplyStatusNACK,
		ErrorDetail:    "runtime failed",
	}); err != nil {
		t.Fatalf("NACK failed order error = %v", err)
	}

	userProxies, err := svc.ListUserProxies(ctx, user.ID)
	if err != nil {
		t.Fatalf("ListUserProxies() error = %v", err)
	}
	if len(userProxies) != 0 {
		t.Fatalf("user proxies before ACK = %#v, want no pending or failed proxies", userProxies)
	}
	adminProxies, err := svc.ListAdminProxies(ctx)
	if err != nil {
		t.Fatalf("ListAdminProxies() error = %v", err)
	}
	ids := map[string]bool{}
	for _, item := range adminProxies {
		ids[item.ID] = true
	}
	if !ids[pending.ProxyAccountID] || !ids[failed.ProxyAccountID] {
		t.Fatalf("admin proxies = %#v, want pending %s and failed %s", adminProxies, pending.ProxyAccountID, failed.ProxyAccountID)
	}
}

func TestUpsertProductPersistsProductAndPrices(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)

	product, err := svc.UpsertProduct(ctx, UpsertProductInput{
		ID:      "static-residential",
		Name:    "RayIP 静态住宅",
		IPType:  "ISP住宅",
		Enabled: true,
		Prices: []UpsertProductPriceInput{
			{ID: "static-socks5-30", Protocol: ProtocolSOCKS5, DurationDays: 30, UnitCents: 3200},
			{Protocol: ProtocolHTTP, DurationDays: 180, UnitCents: 16000},
		},
	})
	if err != nil {
		t.Fatalf("UpsertProduct() error = %v", err)
	}
	if product.Name != "RayIP 静态住宅" || product.IPType != "ISP住宅" {
		t.Fatalf("product = %#v", product)
	}
	quote, err := svc.Quote(ctx, QuoteInput{ProductID: "static-residential", Protocol: ProtocolSOCKS5, DurationDays: 30, Quantity: 1})
	if err != nil {
		t.Fatalf("Quote() error = %v", err)
	}
	if quote.UnitCents != 3200 {
		t.Fatalf("quote = %#v, want updated unit price", quote)
	}
	quote, err = svc.Quote(ctx, QuoteInput{ProductID: "static-residential", Protocol: ProtocolHTTP, DurationDays: 180, Quantity: 1})
	if err != nil {
		t.Fatalf("Quote(new price) error = %v", err)
	}
	if quote.UnitCents != 16000 {
		t.Fatalf("quote new price = %#v", quote)
	}
}

func TestRenewAndDisableUseRuntimeDesiredState(t *testing.T) {
	ctx := context.Background()
	svc := newTestService(t)
	user := testUserWithBalance(t, svc, 10000)
	inventory := seedSellableInventory(t, svc, "node-life", ProtocolSOCKS5)
	order, err := svc.CreateOrder(ctx, user.ID, CreateOrderInput{
		ProductID:      "static-residential",
		InventoryID:    inventory.ID,
		Protocol:       ProtocolSOCKS5,
		DurationDays:   30,
		Quantity:       1,
		IdempotencyKey: "idem-life",
	})
	if err != nil {
		t.Fatalf("CreateOrder() error = %v", err)
	}
	if err := svc.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
		ProxyAccountID: order.ProxyAccountID,
		Status:         RuntimeApplyStatusACK,
	}); err != nil {
		t.Fatalf("ACK error = %v", err)
	}

	proxies, err := svc.ListUserProxies(ctx, user.ID)
	if err != nil || len(proxies) != 1 {
		t.Fatalf("ListUserProxies() = %#v, %v", proxies, err)
	}
	proxy := proxies[0]
	if _, err := svc.RenewProxy(ctx, user.ID, proxy.ID, RenewProxyInput{DurationDays: 30, IdempotencyKey: "renew-1"}); err != nil {
		t.Fatalf("RenewProxy() error = %v", err)
	}
	if _, err := svc.DisableProxy(ctx, user.ID, proxy.ID); err != nil {
		t.Fatalf("DisableProxy() error = %v", err)
	}

	runtime := svc.runtime.(*fakeRuntimeWriter)
	if len(runtime.upserts) != 2 {
		t.Fatalf("runtime upserts = %#v, want create and renew", runtime.upserts)
	}
	if len(runtime.removes) != 1 || runtime.removes[0] != proxy.ID {
		t.Fatalf("runtime removes = %#v, want proxy removal", runtime.removes)
	}
	got, err := svc.GetProxy(ctx, user.ID, proxy.ID)
	if err != nil {
		t.Fatalf("GetProxy() error = %v", err)
	}
	if got.LifecycleStatus != ProxyLifecycleRuntimePending {
		t.Fatalf("lifecycle status = %s, want RUNTIME_PENDING after disable", got.LifecycleStatus)
	}
}

func newTestService(t *testing.T) *Service {
	t.Helper()
	now := time.Date(2026, 4, 30, 11, 0, 0, 0, time.UTC)
	repo := NewMemoryRepository()
	runtime := &fakeRuntimeWriter{}
	svc := NewService(repo, runtime, func() time.Time { return now })
	if err := svc.BootstrapDefaults(context.Background()); err != nil {
		t.Fatalf("BootstrapDefaults() error = %v", err)
	}
	return svc
}

func testUserWithBalance(t *testing.T, svc *Service, amountCents int64) User {
	t.Helper()
	user, err := svc.Register(context.Background(), RegisterInput{
		Email:    randomTestEmail(),
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	order, err := svc.CreatePaymentOrder(context.Background(), user.ID, CreatePaymentOrderInput{AmountCents: amountCents})
	if err != nil {
		t.Fatalf("CreatePaymentOrder() error = %v", err)
	}
	if _, err := svc.HandleMockPaymentCallback(context.Background(), PaymentCallbackInput{
		PaymentOrderID:  order.ID,
		ProviderTradeNo: order.ID + "-trade",
		PaidAmountCents: amountCents,
	}); err != nil {
		t.Fatalf("HandleMockPaymentCallback() error = %v", err)
	}
	return user
}

func seedSellableInventory(t *testing.T, svc *Service, nodeID string, protocol Protocol) NodeInventoryIP {
	t.Helper()
	line, err := svc.UpsertLine(context.Background(), UpsertLineInput{
		ID:       nodeID + "-line",
		RegionID: "us",
		CityID:   "nyc",
		NodeID:   nodeID,
		Name:     "测试线路",
		Enabled:  true,
	})
	if err != nil {
		t.Fatalf("UpsertLine() error = %v", err)
	}
	if err := svc.UpsertNodeRuntimeStatus(context.Background(), NodeRuntimeStatus{
		NodeID:         nodeID,
		Sellable:       true,
		Capabilities:   []string{string(protocol)},
		RuntimeVerdict: "ACCEPTED",
		LeaseOnline:    true,
	}); err != nil {
		t.Fatalf("UpsertNodeRuntimeStatus() error = %v", err)
	}
	item, err := svc.UpsertInventory(context.Background(), UpsertInventoryInput{
		LineID:    line.ID,
		NodeID:    nodeID,
		IP:        "203.0.113.10",
		Port:      18080,
		Protocols: []Protocol{protocol},
		Status:    InventoryStatusAvailable,
	})
	if err != nil {
		t.Fatalf("UpsertInventory() error = %v", err)
	}
	return item
}

type fakeRuntimeWriter struct {
	upserts []RuntimeProxyAccountInput
	removes []string
}

func (f *fakeRuntimeWriter) UpsertProxyAccount(ctx context.Context, input RuntimeProxyAccountInput) (RuntimeMutationResult, error) {
	f.upserts = append(f.upserts, input)
	return RuntimeMutationResult{ProxyAccountID: input.ProxyAccountID, NodeID: input.NodeID}, nil
}

func (f *fakeRuntimeWriter) RemoveProxyAccount(ctx context.Context, proxyAccountID string) (RuntimeMutationResult, error) {
	f.removes = append(f.removes, proxyAccountID)
	return RuntimeMutationResult{ProxyAccountID: proxyAccountID}, nil
}

var randomEmailCounter int

func randomTestEmail() string {
	randomEmailCounter++
	return "buyer" + string(rune('a'+randomEmailCounter)) + "@example.com"
}
