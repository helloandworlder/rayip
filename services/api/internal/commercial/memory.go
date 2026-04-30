package commercial

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

type Repository interface {
	SaveUser(ctx context.Context, user User) error
	GetUser(ctx context.Context, userID string) (User, bool, error)
	GetUserByEmail(ctx context.Context, email string) (User, bool, error)
	ListUsers(ctx context.Context) ([]User, error)
	SaveAdminUser(ctx context.Context, admin AdminUser) error
	GetAdminByUsername(ctx context.Context, username string) (AdminUser, bool, error)
	ListAdminUsers(ctx context.Context) ([]AdminUser, error)
	SaveSession(ctx context.Context, session Session) error
	GetSession(ctx context.Context, sessionID string) (Session, bool, error)
	DeleteSession(ctx context.Context, sessionID string) error

	EnsureWallet(ctx context.Context, userID string, now time.Time) (Wallet, error)
	GetWallet(ctx context.Context, userID string) (Wallet, bool, error)
	SaveWallet(ctx context.Context, wallet Wallet) error
	AppendLedger(ctx context.Context, item WalletLedger) error
	GetLedgerByIdempotencyKey(ctx context.Context, key string) (WalletLedger, bool, error)
	ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error)
	SaveWalletHold(ctx context.Context, hold WalletHold) error
	GetWalletHold(ctx context.Context, holdID string) (WalletHold, bool, error)

	SavePaymentOrder(ctx context.Context, order PaymentOrder) error
	GetPaymentOrder(ctx context.Context, orderID string) (PaymentOrder, bool, error)
	GetPaymentOrderByProviderTrade(ctx context.Context, orderID string, providerTradeNo string) (PaymentOrder, bool, error)
	ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error)
	AppendAudit(ctx context.Context, item AuditLog) error
	ListAuditLogs(ctx context.Context) ([]AuditLog, error)

	SaveRegion(ctx context.Context, item Region) error
	SaveCity(ctx context.Context, item City) error
	SaveLine(ctx context.Context, item Line) error
	SaveProduct(ctx context.Context, item Product) error
	SaveProductPrice(ctx context.Context, item ProductPrice) error
	SaveRatePolicy(ctx context.Context, item RatePolicy) error
	GetLine(ctx context.Context, lineID string) (Line, bool, error)
	GetProduct(ctx context.Context, productID string) (Product, bool, error)
	FindPrice(ctx context.Context, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error)
	ListRegions(ctx context.Context) ([]Region, error)
	ListCities(ctx context.Context) ([]City, error)
	ListLines(ctx context.Context) ([]Line, error)
	ListProducts(ctx context.Context) ([]Product, error)
	ListPrices(ctx context.Context) ([]ProductPrice, error)
	ListRatePolicies(ctx context.Context) ([]RatePolicy, error)

	SaveNodeRuntimeStatus(ctx context.Context, item NodeRuntimeStatus) error
	GetNodeRuntimeStatus(ctx context.Context, nodeID string) (NodeRuntimeStatus, bool, error)
	ListNodeRuntimeStatuses(ctx context.Context) ([]NodeRuntimeStatus, error)
	SaveInventory(ctx context.Context, item NodeInventoryIP) error
	GetInventory(ctx context.Context, inventoryID string) (NodeInventoryIP, bool, error)
	ListInventory(ctx context.Context) ([]NodeInventoryIP, error)
	SaveReservation(ctx context.Context, item InventoryReservation) error
	GetReservation(ctx context.Context, reservationID string) (InventoryReservation, bool, error)

	SaveOrder(ctx context.Context, order ProxyOrder) error
	GetOrder(ctx context.Context, orderID string) (ProxyOrder, bool, error)
	GetOrderByIdempotencyKey(ctx context.Context, userID string, key string) (ProxyOrder, bool, error)
	ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error)
	SaveProxyAccount(ctx context.Context, account ProxyAccount) error
	GetProxyAccount(ctx context.Context, proxyID string) (ProxyAccount, bool, error)
	ListProxyAccounts(ctx context.Context, userID string) ([]ProxyAccount, error)
	SaveFulfillmentJob(ctx context.Context, job FulfillmentJob) error
	ListFulfillmentJobs(ctx context.Context, orderID string) ([]FulfillmentJob, error)
	SaveFulfillmentAttempt(ctx context.Context, attempt FulfillmentAttempt) error
	ListFulfillmentAttempts(ctx context.Context, jobID string) ([]FulfillmentAttempt, error)

	WithTx(ctx context.Context, fn func(Repository) error) error
}

type MemoryRepository struct {
	mu              sync.Mutex
	users           map[string]User
	usersByEmail    map[string]string
	adminUsers      map[string]AdminUser
	adminByUsername map[string]string
	sessions        map[string]Session
	wallets         map[string]Wallet
	ledger          map[string]WalletLedger
	ledgerByIDKey   map[string]string
	holds           map[string]WalletHold
	payments        map[string]PaymentOrder
	paymentsByTrade map[string]string
	audits          map[string]AuditLog
	regions         map[string]Region
	cities          map[string]City
	lines           map[string]Line
	products        map[string]Product
	prices          map[string]ProductPrice
	ratePolicies    map[string]RatePolicy
	nodeStatuses    map[string]NodeRuntimeStatus
	inventory       map[string]NodeInventoryIP
	reservations    map[string]InventoryReservation
	orders          map[string]ProxyOrder
	ordersByIDKey   map[string]string
	proxies         map[string]ProxyAccount
	jobs            map[string]FulfillmentJob
	attempts        map[string]FulfillmentAttempt
}

func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		users:           map[string]User{},
		usersByEmail:    map[string]string{},
		adminUsers:      map[string]AdminUser{},
		adminByUsername: map[string]string{},
		sessions:        map[string]Session{},
		wallets:         map[string]Wallet{},
		ledger:          map[string]WalletLedger{},
		ledgerByIDKey:   map[string]string{},
		holds:           map[string]WalletHold{},
		payments:        map[string]PaymentOrder{},
		paymentsByTrade: map[string]string{},
		audits:          map[string]AuditLog{},
		regions:         map[string]Region{},
		cities:          map[string]City{},
		lines:           map[string]Line{},
		products:        map[string]Product{},
		prices:          map[string]ProductPrice{},
		ratePolicies:    map[string]RatePolicy{},
		nodeStatuses:    map[string]NodeRuntimeStatus{},
		inventory:       map[string]NodeInventoryIP{},
		reservations:    map[string]InventoryReservation{},
		orders:          map[string]ProxyOrder{},
		ordersByIDKey:   map[string]string{},
		proxies:         map[string]ProxyAccount{},
		jobs:            map[string]FulfillmentJob{},
		attempts:        map[string]FulfillmentAttempt{},
	}
}

func (r *MemoryRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return fn((*lockedMemoryRepository)(r))
}

type lockedMemoryRepository MemoryRepository

func (r *MemoryRepository) locked() *lockedMemoryRepository {
	return (*lockedMemoryRepository)(r)
}

func (r *MemoryRepository) SaveUser(ctx context.Context, user User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveUser(ctx, user)
}

func (r *lockedMemoryRepository) SaveUser(ctx context.Context, user User) error {
	if existingID, ok := r.usersByEmail[user.Email]; ok && existingID != user.ID {
		return ErrAlreadyExists
	}
	r.users[user.ID] = user
	r.usersByEmail[user.Email] = user.ID
	return nil
}

func (r *MemoryRepository) GetUser(ctx context.Context, userID string) (User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetUser(ctx, userID)
}

func (r *lockedMemoryRepository) GetUser(ctx context.Context, userID string) (User, bool, error) {
	item, ok := r.users[userID]
	return item, ok, nil
}

func (r *MemoryRepository) GetUserByEmail(ctx context.Context, email string) (User, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetUserByEmail(ctx, email)
}

func (r *lockedMemoryRepository) GetUserByEmail(ctx context.Context, email string) (User, bool, error) {
	id, ok := r.usersByEmail[email]
	if !ok {
		return User{}, false, nil
	}
	return r.users[id], true, nil
}

func (r *MemoryRepository) ListUsers(ctx context.Context) ([]User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListUsers(ctx)
}

func (r *lockedMemoryRepository) ListUsers(ctx context.Context) ([]User, error) {
	items := make([]User, 0, len(r.users))
	for _, item := range r.users {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveAdminUser(ctx context.Context, admin AdminUser) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveAdminUser(ctx, admin)
}

func (r *lockedMemoryRepository) SaveAdminUser(ctx context.Context, admin AdminUser) error {
	if existingID, ok := r.adminByUsername[admin.Username]; ok && existingID != admin.ID {
		return ErrAlreadyExists
	}
	r.adminUsers[admin.ID] = admin
	r.adminByUsername[admin.Username] = admin.ID
	return nil
}

func (r *MemoryRepository) GetAdminByUsername(ctx context.Context, username string) (AdminUser, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetAdminByUsername(ctx, username)
}

func (r *lockedMemoryRepository) GetAdminByUsername(ctx context.Context, username string) (AdminUser, bool, error) {
	id, ok := r.adminByUsername[username]
	if !ok {
		return AdminUser{}, false, nil
	}
	return r.adminUsers[id], true, nil
}

func (r *MemoryRepository) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListAdminUsers(ctx)
}

func (r *lockedMemoryRepository) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	items := make([]AdminUser, 0, len(r.adminUsers))
	for _, item := range r.adminUsers {
		items = append(items, item)
	}
	return items, nil
}

func (r *MemoryRepository) SaveSession(ctx context.Context, session Session) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveSession(ctx, session)
}

func (r *lockedMemoryRepository) SaveSession(ctx context.Context, session Session) error {
	r.sessions[session.ID] = session
	return nil
}

func (r *MemoryRepository) GetSession(ctx context.Context, sessionID string) (Session, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetSession(ctx, sessionID)
}

func (r *lockedMemoryRepository) GetSession(ctx context.Context, sessionID string) (Session, bool, error) {
	item, ok := r.sessions[sessionID]
	return item, ok, nil
}

func (r *MemoryRepository) DeleteSession(ctx context.Context, sessionID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().DeleteSession(ctx, sessionID)
}

func (r *lockedMemoryRepository) DeleteSession(ctx context.Context, sessionID string) error {
	delete(r.sessions, sessionID)
	return nil
}

func (r *MemoryRepository) EnsureWallet(ctx context.Context, userID string, now time.Time) (Wallet, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().EnsureWallet(ctx, userID, now)
}

func (r *lockedMemoryRepository) EnsureWallet(ctx context.Context, userID string, now time.Time) (Wallet, error) {
	if wallet, ok := r.wallets[userID]; ok {
		return wallet, nil
	}
	wallet := Wallet{ID: "wallet-" + userID, UserID: userID, UpdatedAt: now}
	r.wallets[userID] = wallet
	return wallet, nil
}

func (r *MemoryRepository) GetWallet(ctx context.Context, userID string) (Wallet, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetWallet(ctx, userID)
}

func (r *lockedMemoryRepository) GetWallet(ctx context.Context, userID string) (Wallet, bool, error) {
	item, ok := r.wallets[userID]
	return item, ok, nil
}

func (r *MemoryRepository) SaveWallet(ctx context.Context, wallet Wallet) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveWallet(ctx, wallet)
}

func (r *lockedMemoryRepository) SaveWallet(ctx context.Context, wallet Wallet) error {
	r.wallets[wallet.UserID] = wallet
	return nil
}

func (r *MemoryRepository) AppendLedger(ctx context.Context, item WalletLedger) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().AppendLedger(ctx, item)
}

func (r *lockedMemoryRepository) AppendLedger(ctx context.Context, item WalletLedger) error {
	if item.IdempotencyKey != "" {
		if _, ok := r.ledgerByIDKey[item.IdempotencyKey]; ok {
			return ErrIdempotencyConflict
		}
		r.ledgerByIDKey[item.IdempotencyKey] = item.ID
	}
	r.ledger[item.ID] = item
	return nil
}

func (r *MemoryRepository) GetLedgerByIdempotencyKey(ctx context.Context, key string) (WalletLedger, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetLedgerByIdempotencyKey(ctx, key)
}

func (r *lockedMemoryRepository) GetLedgerByIdempotencyKey(ctx context.Context, key string) (WalletLedger, bool, error) {
	id, ok := r.ledgerByIDKey[key]
	if !ok {
		return WalletLedger{}, false, nil
	}
	return r.ledger[id], true, nil
}

func (r *MemoryRepository) ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListWalletLedger(ctx, filter)
}

func (r *lockedMemoryRepository) ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error) {
	items := make([]WalletLedger, 0, len(r.ledger))
	for _, item := range r.ledger {
		if filter.UserID != "" && item.UserID != filter.UserID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveWalletHold(ctx context.Context, hold WalletHold) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveWalletHold(ctx, hold)
}

func (r *lockedMemoryRepository) SaveWalletHold(ctx context.Context, hold WalletHold) error {
	r.holds[hold.ID] = hold
	return nil
}

func (r *MemoryRepository) GetWalletHold(ctx context.Context, holdID string) (WalletHold, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetWalletHold(ctx, holdID)
}

func (r *lockedMemoryRepository) GetWalletHold(ctx context.Context, holdID string) (WalletHold, bool, error) {
	item, ok := r.holds[holdID]
	return item, ok, nil
}

func (r *MemoryRepository) SavePaymentOrder(ctx context.Context, order PaymentOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SavePaymentOrder(ctx, order)
}

func (r *lockedMemoryRepository) SavePaymentOrder(ctx context.Context, order PaymentOrder) error {
	r.payments[order.ID] = order
	if order.ProviderTradeNo != "" {
		r.paymentsByTrade[order.ID+"|"+order.ProviderTradeNo] = order.ID
	}
	return nil
}

func (r *MemoryRepository) GetPaymentOrder(ctx context.Context, orderID string) (PaymentOrder, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetPaymentOrder(ctx, orderID)
}

func (r *lockedMemoryRepository) GetPaymentOrder(ctx context.Context, orderID string) (PaymentOrder, bool, error) {
	item, ok := r.payments[orderID]
	return item, ok, nil
}

func (r *MemoryRepository) GetPaymentOrderByProviderTrade(ctx context.Context, orderID string, providerTradeNo string) (PaymentOrder, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetPaymentOrderByProviderTrade(ctx, orderID, providerTradeNo)
}

func (r *lockedMemoryRepository) GetPaymentOrderByProviderTrade(ctx context.Context, orderID string, providerTradeNo string) (PaymentOrder, bool, error) {
	id, ok := r.paymentsByTrade[orderID+"|"+providerTradeNo]
	if !ok {
		return PaymentOrder{}, false, nil
	}
	return r.payments[id], true, nil
}

func (r *MemoryRepository) ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListPaymentOrders(ctx)
}

func (r *lockedMemoryRepository) ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error) {
	items := make([]PaymentOrder, 0, len(r.payments))
	for _, item := range r.payments {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) AppendAudit(ctx context.Context, item AuditLog) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().AppendAudit(ctx, item)
}

func (r *lockedMemoryRepository) AppendAudit(ctx context.Context, item AuditLog) error {
	r.audits[item.ID] = item
	return nil
}

func (r *MemoryRepository) ListAuditLogs(ctx context.Context) ([]AuditLog, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListAuditLogs(ctx)
}

func (r *lockedMemoryRepository) ListAuditLogs(ctx context.Context) ([]AuditLog, error) {
	items := make([]AuditLog, 0, len(r.audits))
	for _, item := range r.audits {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveRegion(ctx context.Context, item Region) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveRegion(ctx, item)
}

func (r *lockedMemoryRepository) SaveRegion(ctx context.Context, item Region) error {
	r.regions[item.ID] = item
	return nil
}

func (r *MemoryRepository) SaveCity(ctx context.Context, item City) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveCity(ctx, item)
}

func (r *lockedMemoryRepository) SaveCity(ctx context.Context, item City) error {
	r.cities[item.ID] = item
	return nil
}

func (r *MemoryRepository) SaveLine(ctx context.Context, item Line) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveLine(ctx, item)
}

func (r *lockedMemoryRepository) SaveLine(ctx context.Context, item Line) error {
	r.lines[item.ID] = item
	return nil
}

func (r *MemoryRepository) SaveProduct(ctx context.Context, item Product) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveProduct(ctx, item)
}

func (r *lockedMemoryRepository) SaveProduct(ctx context.Context, item Product) error {
	r.products[item.ID] = item
	return nil
}

func (r *MemoryRepository) SaveProductPrice(ctx context.Context, item ProductPrice) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveProductPrice(ctx, item)
}

func (r *lockedMemoryRepository) SaveProductPrice(ctx context.Context, item ProductPrice) error {
	for id, price := range r.prices {
		if price.ProductID == item.ProductID && price.Protocol == item.Protocol && price.DurationDays == item.DurationDays {
			item.ID = id
			item.CreatedAt = price.CreatedAt
			break
		}
	}
	r.prices[item.ID] = item
	return nil
}

func (r *MemoryRepository) SaveRatePolicy(ctx context.Context, item RatePolicy) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveRatePolicy(ctx, item)
}

func (r *lockedMemoryRepository) SaveRatePolicy(ctx context.Context, item RatePolicy) error {
	r.ratePolicies[item.ID] = item
	return nil
}

func (r *MemoryRepository) GetLine(ctx context.Context, lineID string) (Line, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetLine(ctx, lineID)
}

func (r *lockedMemoryRepository) GetLine(ctx context.Context, lineID string) (Line, bool, error) {
	item, ok := r.lines[lineID]
	return item, ok, nil
}

func (r *MemoryRepository) GetProduct(ctx context.Context, productID string) (Product, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetProduct(ctx, productID)
}

func (r *lockedMemoryRepository) GetProduct(ctx context.Context, productID string) (Product, bool, error) {
	item, ok := r.products[productID]
	return item, ok, nil
}

func (r *MemoryRepository) FindPrice(ctx context.Context, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().FindPrice(ctx, productID, protocol, durationDays)
}

func (r *lockedMemoryRepository) FindPrice(ctx context.Context, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error) {
	for _, price := range r.prices {
		if price.ProductID == productID && price.Protocol == protocol && price.DurationDays == durationDays {
			return price, true, nil
		}
	}
	return ProductPrice{}, false, nil
}

func (r *MemoryRepository) ListRegions(ctx context.Context) ([]Region, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListRegions(ctx)
}

func (r *lockedMemoryRepository) ListRegions(ctx context.Context) ([]Region, error) {
	items := mapValues(r.regions)
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *MemoryRepository) ListCities(ctx context.Context) ([]City, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListCities(ctx)
}

func (r *lockedMemoryRepository) ListCities(ctx context.Context) ([]City, error) {
	items := mapValues(r.cities)
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *MemoryRepository) ListLines(ctx context.Context) ([]Line, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListLines(ctx)
}

func (r *lockedMemoryRepository) ListLines(ctx context.Context) ([]Line, error) {
	items := mapValues(r.lines)
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func (r *MemoryRepository) ListProducts(ctx context.Context) ([]Product, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListProducts(ctx)
}

func (r *lockedMemoryRepository) ListProducts(ctx context.Context) ([]Product, error) {
	return mapValues(r.products), nil
}

func (r *MemoryRepository) ListPrices(ctx context.Context) ([]ProductPrice, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListPrices(ctx)
}

func (r *lockedMemoryRepository) ListPrices(ctx context.Context) ([]ProductPrice, error) {
	return mapValues(r.prices), nil
}

func (r *MemoryRepository) ListRatePolicies(ctx context.Context) ([]RatePolicy, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListRatePolicies(ctx)
}

func (r *lockedMemoryRepository) ListRatePolicies(ctx context.Context) ([]RatePolicy, error) {
	return mapValues(r.ratePolicies), nil
}

func (r *MemoryRepository) SaveNodeRuntimeStatus(ctx context.Context, item NodeRuntimeStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveNodeRuntimeStatus(ctx, item)
}

func (r *lockedMemoryRepository) SaveNodeRuntimeStatus(ctx context.Context, item NodeRuntimeStatus) error {
	r.nodeStatuses[item.NodeID] = item
	return nil
}

func (r *MemoryRepository) GetNodeRuntimeStatus(ctx context.Context, nodeID string) (NodeRuntimeStatus, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetNodeRuntimeStatus(ctx, nodeID)
}

func (r *lockedMemoryRepository) GetNodeRuntimeStatus(ctx context.Context, nodeID string) (NodeRuntimeStatus, bool, error) {
	item, ok := r.nodeStatuses[nodeID]
	return item, ok, nil
}

func (r *MemoryRepository) ListNodeRuntimeStatuses(ctx context.Context) ([]NodeRuntimeStatus, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListNodeRuntimeStatuses(ctx)
}

func (r *lockedMemoryRepository) ListNodeRuntimeStatuses(ctx context.Context) ([]NodeRuntimeStatus, error) {
	return mapValues(r.nodeStatuses), nil
}

func (r *MemoryRepository) SaveInventory(ctx context.Context, item NodeInventoryIP) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveInventory(ctx, item)
}

func (r *lockedMemoryRepository) SaveInventory(ctx context.Context, item NodeInventoryIP) error {
	r.inventory[item.ID] = item
	return nil
}

func (r *MemoryRepository) GetInventory(ctx context.Context, inventoryID string) (NodeInventoryIP, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetInventory(ctx, inventoryID)
}

func (r *lockedMemoryRepository) GetInventory(ctx context.Context, inventoryID string) (NodeInventoryIP, bool, error) {
	item, ok := r.inventory[inventoryID]
	return item, ok, nil
}

func (r *MemoryRepository) ListInventory(ctx context.Context) ([]NodeInventoryIP, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListInventory(ctx)
}

func (r *lockedMemoryRepository) ListInventory(ctx context.Context) ([]NodeInventoryIP, error) {
	items := mapValues(r.inventory)
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveReservation(ctx context.Context, item InventoryReservation) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveReservation(ctx, item)
}

func (r *lockedMemoryRepository) SaveReservation(ctx context.Context, item InventoryReservation) error {
	r.reservations[item.ID] = item
	return nil
}

func (r *MemoryRepository) GetReservation(ctx context.Context, reservationID string) (InventoryReservation, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetReservation(ctx, reservationID)
}

func (r *lockedMemoryRepository) GetReservation(ctx context.Context, reservationID string) (InventoryReservation, bool, error) {
	item, ok := r.reservations[reservationID]
	return item, ok, nil
}

func (r *MemoryRepository) SaveOrder(ctx context.Context, order ProxyOrder) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveOrder(ctx, order)
}

func (r *lockedMemoryRepository) SaveOrder(ctx context.Context, order ProxyOrder) error {
	r.orders[order.ID] = order
	if order.IdempotencyKey != "" {
		r.ordersByIDKey[order.UserID+"|"+order.IdempotencyKey] = order.ID
	}
	return nil
}

func (r *MemoryRepository) GetOrder(ctx context.Context, orderID string) (ProxyOrder, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetOrder(ctx, orderID)
}

func (r *lockedMemoryRepository) GetOrder(ctx context.Context, orderID string) (ProxyOrder, bool, error) {
	item, ok := r.orders[orderID]
	return item, ok, nil
}

func (r *MemoryRepository) GetOrderByIdempotencyKey(ctx context.Context, userID string, key string) (ProxyOrder, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetOrderByIdempotencyKey(ctx, userID, key)
}

func (r *lockedMemoryRepository) GetOrderByIdempotencyKey(ctx context.Context, userID string, key string) (ProxyOrder, bool, error) {
	id, ok := r.ordersByIDKey[userID+"|"+key]
	if !ok {
		return ProxyOrder{}, false, nil
	}
	return r.orders[id], true, nil
}

func (r *MemoryRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListOrders(ctx, filter)
}

func (r *lockedMemoryRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error) {
	items := make([]ProxyOrder, 0, len(r.orders))
	for _, item := range r.orders {
		if filter.UserID != "" && item.UserID != filter.UserID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveProxyAccount(ctx context.Context, account ProxyAccount) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveProxyAccount(ctx, account)
}

func (r *lockedMemoryRepository) SaveProxyAccount(ctx context.Context, account ProxyAccount) error {
	r.proxies[account.ID] = account
	return nil
}

func (r *MemoryRepository) GetProxyAccount(ctx context.Context, proxyID string) (ProxyAccount, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().GetProxyAccount(ctx, proxyID)
}

func (r *lockedMemoryRepository) GetProxyAccount(ctx context.Context, proxyID string) (ProxyAccount, bool, error) {
	item, ok := r.proxies[proxyID]
	return item, ok, nil
}

func (r *MemoryRepository) ListProxyAccounts(ctx context.Context, userID string) ([]ProxyAccount, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListProxyAccounts(ctx, userID)
}

func (r *lockedMemoryRepository) ListProxyAccounts(ctx context.Context, userID string) ([]ProxyAccount, error) {
	items := make([]ProxyAccount, 0, len(r.proxies))
	for _, item := range r.proxies {
		if userID != "" && item.UserID != userID {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool { return items[i].CreatedAt.After(items[j].CreatedAt) })
	return items, nil
}

func (r *MemoryRepository) SaveFulfillmentJob(ctx context.Context, job FulfillmentJob) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveFulfillmentJob(ctx, job)
}

func (r *lockedMemoryRepository) SaveFulfillmentJob(ctx context.Context, job FulfillmentJob) error {
	r.jobs[job.ID] = job
	return nil
}

func (r *MemoryRepository) ListFulfillmentJobs(ctx context.Context, orderID string) ([]FulfillmentJob, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListFulfillmentJobs(ctx, orderID)
}

func (r *lockedMemoryRepository) ListFulfillmentJobs(ctx context.Context, orderID string) ([]FulfillmentJob, error) {
	items := []FulfillmentJob{}
	for _, item := range r.jobs {
		if orderID != "" && item.OrderID != orderID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func (r *MemoryRepository) SaveFulfillmentAttempt(ctx context.Context, attempt FulfillmentAttempt) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().SaveFulfillmentAttempt(ctx, attempt)
}

func (r *lockedMemoryRepository) SaveFulfillmentAttempt(ctx context.Context, attempt FulfillmentAttempt) error {
	r.attempts[attempt.ID] = attempt
	return nil
}

func (r *MemoryRepository) ListFulfillmentAttempts(ctx context.Context, jobID string) ([]FulfillmentAttempt, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.locked().ListFulfillmentAttempts(ctx, jobID)
}

func (r *lockedMemoryRepository) ListFulfillmentAttempts(ctx context.Context, jobID string) ([]FulfillmentAttempt, error) {
	items := []FulfillmentAttempt{}
	for _, item := range r.attempts {
		if jobID != "" && item.JobID != jobID {
			continue
		}
		items = append(items, item)
	}
	return items, nil
}

func mapValues[T any](m map[string]T) []T {
	items := make([]T, 0, len(m))
	for _, item := range m {
		items = append(items, item)
	}
	return items
}

var _ Repository = (*MemoryRepository)(nil)
var _ Repository = (*lockedMemoryRepository)(nil)

var errUnsupportedTx = errors.New("nested transaction is not supported")

func (r *lockedMemoryRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	return errUnsupportedTx
}
