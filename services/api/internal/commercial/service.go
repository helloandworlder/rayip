package commercial

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrAlreadyExists        = errors.New("already exists")
	ErrUnauthorized         = errors.New("unauthorized")
	ErrNotFound             = errors.New("not found")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrIdempotencyRequired  = errors.New("idempotency key is required")
	ErrIdempotencyConflict  = errors.New("idempotency conflict")
	ErrInventoryUnavailable = errors.New("inventory is unavailable")
	ErrUnsupportedProtocol  = errors.New("unsupported protocol")
)

type RuntimeWriter interface {
	UpsertProxyAccount(ctx context.Context, input RuntimeProxyAccountInput) (RuntimeMutationResult, error)
	RemoveProxyAccount(ctx context.Context, proxyAccountID string) (RuntimeMutationResult, error)
}

type Service struct {
	repo    Repository
	runtime RuntimeWriter
	now     func() time.Time
}

func NewService(repo Repository, runtime RuntimeWriter, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repo: repo, runtime: runtime, now: now}
}

func (s *Service) BootstrapDefaults(ctx context.Context) error {
	now := s.now().UTC()
	if _, ok, err := s.repo.GetProduct(ctx, "static-residential"); err != nil {
		return err
	} else if !ok {
		if err := s.repo.SaveProduct(ctx, Product{
			ID:        "static-residential",
			Name:      "静态住宅代理",
			IPType:    "原生住宅",
			Enabled:   true,
			CreatedAt: now,
			UpdatedAt: now,
		}); err != nil {
			return err
		}
	}
	for _, price := range []ProductPrice{
		{ID: "static-socks5-30", ProductID: "static-residential", Protocol: ProtocolSOCKS5, DurationDays: 30, UnitCents: 3000},
		{ID: "static-http-30", ProductID: "static-residential", Protocol: ProtocolHTTP, DurationDays: 30, UnitCents: 3000},
		{ID: "static-socks5-60", ProductID: "static-residential", Protocol: ProtocolSOCKS5, DurationDays: 60, UnitCents: 5600},
		{ID: "static-http-60", ProductID: "static-residential", Protocol: ProtocolHTTP, DurationDays: 60, UnitCents: 5600},
		{ID: "static-socks5-90", ProductID: "static-residential", Protocol: ProtocolSOCKS5, DurationDays: 90, UnitCents: 8100},
		{ID: "static-http-90", ProductID: "static-residential", Protocol: ProtocolHTTP, DurationDays: 90, UnitCents: 8100},
		{ID: "static-socks5-180", ProductID: "static-residential", Protocol: ProtocolSOCKS5, DurationDays: 180, UnitCents: 15000},
		{ID: "static-http-180", ProductID: "static-residential", Protocol: ProtocolHTTP, DurationDays: 180, UnitCents: 15000},
	} {
		price.CreatedAt = now
		price.UpdatedAt = now
		if err := s.repo.SaveProductPrice(ctx, price); err != nil {
			return err
		}
	}
	if err := s.repo.SaveRegion(ctx, Region{ID: "us", Name: "美国", Country: "US", CreatedAt: now, UpdatedAt: now}); err != nil {
		return err
	}
	if err := s.repo.SaveCity(ctx, City{ID: "nyc", RegionID: "us", Name: "纽约", CreatedAt: now, UpdatedAt: now}); err != nil {
		return err
	}
	if err := s.repo.SaveRatePolicy(ctx, RatePolicy{
		ID:              "standard",
		Name:            "标准静态住宅",
		EgressLimitBPS:  0,
		IngressLimitBPS: 0,
		MaxConnections:  2,
		CreatedAt:       now,
		UpdatedAt:       now,
	}); err != nil {
		return err
	}
	return s.ensureDefaultAdmin(ctx)
}

func (s *Service) ensureDefaultAdmin(ctx context.Context) error {
	admins, err := s.repo.ListAdminUsers(ctx)
	if err != nil {
		return err
	}
	if len(admins) > 0 {
		return nil
	}
	hash, err := hashPassword("rayip-admin")
	if err != nil {
		return err
	}
	now := s.now().UTC()
	return s.repo.SaveAdminUser(ctx, AdminUser{
		ID:           uuid.NewString(),
		Username:     "admin",
		PasswordHash: hash,
		Role:         "SUPER_ADMIN",
		CreatedAt:    now,
		UpdatedAt:    now,
	})
}

func (s *Service) Register(ctx context.Context, input RegisterInput) (User, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || len(input.Password) < 6 {
		return User{}, errors.New("email and password are required")
	}
	if _, ok, err := s.repo.GetUserByEmail(ctx, email); err != nil {
		return User{}, err
	} else if ok {
		return User{}, ErrAlreadyExists
	}
	hash, err := hashPassword(input.Password)
	if err != nil {
		return User{}, err
	}
	now := s.now().UTC()
	user := User{
		ID:           uuid.NewString(),
		Email:        email,
		PasswordHash: hash,
		Status:       UserStatusActive,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.repo.SaveUser(ctx, user); err != nil {
		return User{}, err
	}
	if _, err := s.repo.EnsureWallet(ctx, user.ID, now); err != nil {
		return User{}, err
	}
	return user, nil
}

func (s *Service) Login(ctx context.Context, input LoginInput) (User, Session, error) {
	user, ok, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil {
		return User{}, Session{}, err
	}
	if !ok || bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(input.Password)) != nil {
		return User{}, Session{}, ErrUnauthorized
	}
	if user.Status != UserStatusActive {
		return User{}, Session{}, ErrUnauthorized
	}
	session := Session{
		ID:        randomToken(),
		SubjectID: user.ID,
		Scope:     SessionScopeUser,
		ExpiresAt: s.now().UTC().Add(7 * 24 * time.Hour),
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return User{}, Session{}, err
	}
	return user, session, nil
}

func (s *Service) AdminLogin(ctx context.Context, username string, password string) (AdminUser, Session, error) {
	admin, ok, err := s.repo.GetAdminByUsername(ctx, strings.TrimSpace(username))
	if err != nil {
		return AdminUser{}, Session{}, err
	}
	if !ok || bcrypt.CompareHashAndPassword([]byte(admin.PasswordHash), []byte(password)) != nil {
		return AdminUser{}, Session{}, ErrUnauthorized
	}
	session := Session{
		ID:        randomToken(),
		SubjectID: admin.ID,
		Scope:     SessionScopeAdmin,
		ExpiresAt: s.now().UTC().Add(12 * time.Hour),
		CreatedAt: s.now().UTC(),
	}
	if err := s.repo.SaveSession(ctx, session); err != nil {
		return AdminUser{}, Session{}, err
	}
	return admin, session, nil
}

func (s *Service) GetSession(ctx context.Context, sessionID string) (Session, bool, error) {
	session, ok, err := s.repo.GetSession(ctx, sessionID)
	if err != nil || !ok {
		return session, ok, err
	}
	if s.now().UTC().After(session.ExpiresAt) {
		_ = s.repo.DeleteSession(ctx, sessionID)
		return Session{}, false, nil
	}
	return session, true, nil
}

func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.repo.DeleteSession(ctx, sessionID)
}

func (s *Service) GetUser(ctx context.Context, userID string) (User, error) {
	user, ok, err := s.repo.GetUser(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if !ok {
		return User{}, ErrNotFound
	}
	return user, nil
}

func (s *Service) GetWallet(ctx context.Context, userID string) (Wallet, error) {
	wallet, ok, err := s.repo.GetWallet(ctx, userID)
	if err != nil {
		return Wallet{}, err
	}
	if ok {
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		return wallet, nil
	}
	return s.repo.EnsureWallet(ctx, userID, s.now().UTC())
}

func (s *Service) CreatePaymentOrder(ctx context.Context, userID string, input CreatePaymentOrderInput) (PaymentOrder, error) {
	if input.AmountCents <= 0 {
		return PaymentOrder{}, errors.New("amount_cents must be positive")
	}
	now := s.now().UTC()
	order := PaymentOrder{
		ID:          uuid.NewString(),
		UserID:      userID,
		AmountCents: input.AmountCents,
		Status:      PaymentOrderStatusPending,
		Provider:    "mock-yipay",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := s.repo.SavePaymentOrder(ctx, order); err != nil {
		return PaymentOrder{}, err
	}
	return order, nil
}

func (s *Service) HandleMockPaymentCallback(ctx context.Context, input PaymentCallbackInput) (PaymentOrder, error) {
	if input.PaymentOrderID == "" || input.ProviderTradeNo == "" || input.PaidAmountCents <= 0 {
		return PaymentOrder{}, errors.New("payment_order_id, provider_trade_no and paid_amount_cents are required")
	}
	var result PaymentOrder
	err := s.repo.WithTx(ctx, func(repo Repository) error {
		if existing, ok, err := repo.GetPaymentOrderByProviderTrade(ctx, input.PaymentOrderID, input.ProviderTradeNo); err != nil {
			return err
		} else if ok && existing.Status == PaymentOrderStatusPaid {
			result = existing
			return nil
		}
		order, ok, err := repo.GetPaymentOrder(ctx, input.PaymentOrderID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if order.Status == PaymentOrderStatusPaid {
			result = order
			return nil
		}
		if order.AmountCents != input.PaidAmountCents {
			return errors.New("paid amount does not match payment order")
		}
		now := s.now().UTC()
		wallet, err := repo.EnsureWallet(ctx, order.UserID, now)
		if err != nil {
			return err
		}
		wallet.BalanceCents += input.PaidAmountCents
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		wallet.UpdatedAt = now
		if err := repo.SaveWallet(ctx, wallet); err != nil {
			return err
		}
		order.Status = PaymentOrderStatusPaid
		order.ProviderTradeNo = input.ProviderTradeNo
		order.PaidAt = now
		order.UpdatedAt = now
		if err := repo.SavePaymentOrder(ctx, order); err != nil {
			return err
		}
		ledger := WalletLedger{
			ID:             uuid.NewString(),
			WalletID:       wallet.ID,
			UserID:         wallet.UserID,
			Type:           LedgerTypeCreditRecharge,
			AmountCents:    input.PaidAmountCents,
			BalanceAfter:   wallet.BalanceCents,
			HeldAfter:      wallet.HeldCents,
			ReferenceType:  "payment_order",
			ReferenceID:    order.ID,
			IdempotencyKey: "payment:" + order.ID + ":" + input.ProviderTradeNo,
			CreatedAt:      now,
		}
		if err := repo.AppendLedger(ctx, ledger); err != nil && !errors.Is(err, ErrIdempotencyConflict) {
			return err
		}
		result = order
		return nil
	})
	return result, err
}

func (s *Service) ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error) {
	return s.repo.ListWalletLedger(ctx, filter)
}

func (s *Service) ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error) {
	return s.repo.ListPaymentOrders(ctx)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.ListUsers(ctx)
}

func (s *Service) Quote(ctx context.Context, input QuoteInput) (Quote, error) {
	if input.ProductID == "" {
		input.ProductID = "static-residential"
	}
	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	if input.Protocol != ProtocolSOCKS5 && input.Protocol != ProtocolHTTP {
		return Quote{}, ErrUnsupportedProtocol
	}
	if input.DurationDays == 0 {
		input.DurationDays = 30
	}
	price, ok, err := s.repo.FindPrice(ctx, input.ProductID, input.Protocol, input.DurationDays)
	if err != nil {
		return Quote{}, err
	}
	if !ok {
		return Quote{}, ErrNotFound
	}
	return Quote{
		ProductID:    input.ProductID,
		Protocol:     input.Protocol,
		DurationDays: input.DurationDays,
		Quantity:     input.Quantity,
		UnitCents:    price.UnitCents,
		TotalCents:   price.UnitCents * int64(input.Quantity),
	}, nil
}

func (s *Service) GetStaticResidentialCatalog(ctx context.Context) (Catalog, error) {
	product, ok, err := s.repo.GetProduct(ctx, "static-residential")
	if err != nil {
		return Catalog{}, err
	}
	if !ok {
		return Catalog{}, ErrNotFound
	}
	prices, err := s.repo.ListPrices(ctx)
	if err != nil {
		return Catalog{}, err
	}
	regions, err := s.repo.ListRegions(ctx)
	if err != nil {
		return Catalog{}, err
	}
	cities, err := s.repo.ListCities(ctx)
	if err != nil {
		return Catalog{}, err
	}
	lines, err := s.repo.ListLines(ctx)
	if err != nil {
		return Catalog{}, err
	}
	inventory, err := s.repo.ListInventory(ctx)
	if err != nil {
		return Catalog{}, err
	}
	statuses, err := s.repo.ListNodeRuntimeStatuses(ctx)
	if err != nil {
		return Catalog{}, err
	}
	statusByNode := map[string]NodeRuntimeStatus{}
	for _, status := range statuses {
		statusByNode[status.NodeID] = status
	}
	availableByLine := map[string]int{}
	inventoryIDsByLine := map[string][]string{}
	reasonsByLine := map[string]map[string]struct{}{}
	for _, item := range inventory {
		line, lineOK := findLine(lines, item.LineID)
		status, statusOK := statusByNode[item.NodeID]
		reasons := sellableInventoryReasons(line, lineOK, item, status, statusOK, "")
		if len(reasons) == 0 {
			availableByLine[item.LineID]++
			inventoryIDsByLine[item.LineID] = append(inventoryIDsByLine[item.LineID], item.ID)
		} else {
			if reasonsByLine[item.LineID] == nil {
				reasonsByLine[item.LineID] = map[string]struct{}{}
			}
			for _, reason := range reasons {
				reasonsByLine[item.LineID][reason] = struct{}{}
			}
		}
	}

	cityByRegion := map[string][]CatalogCity{}
	for _, city := range cities {
		catalogCity := CatalogCity{City: city}
		for _, line := range lines {
			if line.CityID != city.ID {
				continue
			}
			reasons := reasonKeys(reasonsByLine[line.ID])
			available := availableByLine[line.ID]
			if !line.Enabled && len(reasons) == 0 {
				reasons = []string{"LINE_DISABLED"}
			}
			catalogCity.Lines = append(catalogCity.Lines, CatalogLine{
				Line:         line,
				Available:    available,
				InventoryIDs: append([]string(nil), inventoryIDsByLine[line.ID]...),
				Sellable:     available > 0,
				Reasons:      reasons,
			})
			catalogCity.Available += available
		}
		cityByRegion[city.RegionID] = append(cityByRegion[city.RegionID], catalogCity)
	}

	catalog := Catalog{Product: product}
	for _, price := range prices {
		if price.ProductID == product.ID {
			catalog.Prices = append(catalog.Prices, price)
		}
	}
	sort.Slice(catalog.Prices, func(i, j int) bool {
		if catalog.Prices[i].DurationDays == catalog.Prices[j].DurationDays {
			return catalog.Prices[i].Protocol < catalog.Prices[j].Protocol
		}
		return catalog.Prices[i].DurationDays < catalog.Prices[j].DurationDays
	})
	for _, region := range regions {
		entry := CatalogRegion{Region: region, Cities: cityByRegion[region.ID]}
		reasonSet := map[string]struct{}{}
		for _, city := range entry.Cities {
			entry.Available += city.Available
			for _, line := range city.Lines {
				for _, reason := range line.Reasons {
					reasonSet[reason] = struct{}{}
				}
			}
		}
		entry.Disabled = entry.Available == 0
		entry.DisabledWhy = reasonKeys(reasonSet)
		catalog.TotalAvailable += entry.Available
		catalog.Regions = append(catalog.Regions, entry)
	}
	return catalog, nil
}

func (s *Service) CreateReservation(ctx context.Context, userID string, input CreateReservationInput) (InventoryReservation, error) {
	var reservation InventoryReservation
	err := s.repo.WithTx(ctx, func(repo Repository) error {
		item, ok, err := repo.GetInventory(ctx, input.InventoryID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		line, lineOK, err := repo.GetLine(ctx, item.LineID)
		if err != nil {
			return err
		}
		status, statusOK, err := repo.GetNodeRuntimeStatus(ctx, item.NodeID)
		if err != nil {
			return err
		}
		reasons := sellableInventoryReasons(line, lineOK, item, status, statusOK, input.Protocol)
		if len(reasons) > 0 {
			return fmt.Errorf("%w: %s", ErrInventoryUnavailable, strings.Join(reasons, ","))
		}
		now := s.now().UTC()
		ttl := time.Duration(input.TTLSeconds) * time.Second
		if ttl <= 0 {
			ttl = 15 * time.Minute
		}
		reservation = InventoryReservation{
			ID:          uuid.NewString(),
			InventoryID: item.ID,
			UserID:      userID,
			OrderID:     input.OrderID,
			Status:      ReservationStatusActive,
			ExpiresAt:   now.Add(ttl),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		item.Status = InventoryStatusReserved
		item.ReservedOrderID = input.OrderID
		item.UpdatedAt = now
		if err := repo.SaveInventory(ctx, item); err != nil {
			return err
		}
		return repo.SaveReservation(ctx, reservation)
	})
	return reservation, err
}

func (s *Service) CreateOrder(ctx context.Context, userID string, input CreateOrderInput) (ProxyOrder, error) {
	if input.IdempotencyKey == "" {
		return ProxyOrder{}, ErrIdempotencyRequired
	}
	if input.Quantity <= 0 {
		input.Quantity = 1
	}
	if input.ProductID == "" {
		input.ProductID = "static-residential"
	}
	if input.DurationDays == 0 {
		input.DurationDays = 30
	}
	if existing, ok, err := s.repo.GetOrderByIdempotencyKey(ctx, userID, input.IdempotencyKey); err != nil {
		return ProxyOrder{}, err
	} else if ok {
		return existing, nil
	}
	quote, err := s.Quote(ctx, QuoteInput{
		ProductID:    input.ProductID,
		Protocol:     input.Protocol,
		DurationDays: input.DurationDays,
		Quantity:     input.Quantity,
	})
	if err != nil {
		return ProxyOrder{}, err
	}

	var order ProxyOrder
	var proxy ProxyAccount
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		if existing, ok, err := repo.GetOrderByIdempotencyKey(ctx, userID, input.IdempotencyKey); err != nil {
			return err
		} else if ok {
			order = existing
			return nil
		}
		now := s.now().UTC()
		wallet, err := repo.EnsureWallet(ctx, userID, now)
		if err != nil {
			return err
		}
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		if wallet.AvailableCents < quote.TotalCents {
			return ErrInsufficientBalance
		}
		inventory, ok, err := repo.GetInventory(ctx, input.InventoryID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		line, lineOK, err := repo.GetLine(ctx, inventory.LineID)
		if err != nil {
			return err
		}
		status, statusOK, err := repo.GetNodeRuntimeStatus(ctx, inventory.NodeID)
		if err != nil {
			return err
		}
		if reasons := sellableInventoryReasons(line, lineOK, inventory, status, statusOK, input.Protocol); len(reasons) > 0 {
			return fmt.Errorf("%w: %s", ErrInventoryUnavailable, strings.Join(reasons, ","))
		}
		orderID := uuid.NewString()
		proxyID := uuid.NewString()
		holdID := uuid.NewString()
		reservationID := uuid.NewString()
		wallet.HeldCents += quote.TotalCents
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		wallet.UpdatedAt = now
		if err := repo.SaveWallet(ctx, wallet); err != nil {
			return err
		}
		if err := repo.AppendLedger(ctx, WalletLedger{
			ID:             uuid.NewString(),
			WalletID:       wallet.ID,
			UserID:         userID,
			Type:           LedgerTypeHold,
			AmountCents:    quote.TotalCents,
			BalanceAfter:   wallet.BalanceCents,
			HeldAfter:      wallet.HeldCents,
			ReferenceType:  "proxy_order",
			ReferenceID:    orderID,
			IdempotencyKey: "order-hold:" + userID + ":" + input.IdempotencyKey,
			CreatedAt:      now,
		}); err != nil {
			return err
		}
		hold := WalletHold{
			ID:          holdID,
			WalletID:    wallet.ID,
			UserID:      userID,
			OrderID:     orderID,
			AmountCents: quote.TotalCents,
			Status:      "HELD",
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		if err := repo.SaveWalletHold(ctx, hold); err != nil {
			return err
		}
		reservation := InventoryReservation{
			ID:          reservationID,
			InventoryID: inventory.ID,
			UserID:      userID,
			OrderID:     orderID,
			Status:      ReservationStatusActive,
			ExpiresAt:   now.Add(15 * time.Minute),
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		inventory.Status = InventoryStatusReserved
		inventory.ReservedOrderID = orderID
		inventory.UpdatedAt = now
		if err := repo.SaveInventory(ctx, inventory); err != nil {
			return err
		}
		if err := repo.SaveReservation(ctx, reservation); err != nil {
			return err
		}
		username := "ray" + shortID(proxyID, 8)
		password := randomTokenN(12)
		expiresAt := now.Add(time.Duration(input.DurationDays) * 24 * time.Hour)
		proxy = ProxyAccount{
			ID:              proxyID,
			OrderID:         orderID,
			UserID:          userID,
			NodeID:          inventory.NodeID,
			InventoryID:     inventory.ID,
			Protocol:        input.Protocol,
			ListenIP:        inventory.IP,
			Port:            inventory.Port,
			Username:        username,
			Password:        password,
			RuntimeEmail:    proxyID,
			EgressLimitBPS:  input.EgressLimitBPS,
			IngressLimitBPS: input.IngressLimitBPS,
			MaxConnections:  input.MaxConnections,
			Status:          "PENDING_RUNTIME",
			LifecycleStatus: ProxyLifecycleRuntimePending,
			ExpiresAt:       expiresAt,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		order = ProxyOrder{
			ID:             orderID,
			UserID:         userID,
			ProductID:      input.ProductID,
			InventoryID:    inventory.ID,
			ReservationID:  reservation.ID,
			WalletHoldID:   hold.ID,
			ProxyAccountID: proxy.ID,
			IdempotencyKey: input.IdempotencyKey,
			Protocol:       input.Protocol,
			DurationDays:   input.DurationDays,
			Quantity:       input.Quantity,
			AmountCents:    quote.TotalCents,
			Status:         OrderStatusPendingRuntime,
			CreatedAt:      now,
			UpdatedAt:      now,
			ExpiresAt:      expiresAt,
		}
		if err := repo.SaveProxyAccount(ctx, proxy); err != nil {
			return err
		}
		if err := repo.SaveOrder(ctx, order); err != nil {
			return err
		}
		if err := repo.SaveFulfillmentJob(ctx, FulfillmentJob{
			ID:             uuid.NewString(),
			OrderID:        order.ID,
			ProxyAccountID: proxy.ID,
			Status:         FulfillmentJobPending,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil || proxy.ID == "" || s.runtime == nil {
		return order, err
	}
	_, runtimeErr := s.runtime.UpsertProxyAccount(ctx, RuntimeProxyAccountInput{
		ProxyAccountID:  proxy.ID,
		NodeID:          proxy.NodeID,
		RuntimeEmail:    proxy.RuntimeEmail,
		Protocol:        proxy.Protocol,
		ListenIP:        proxy.ListenIP,
		Port:            proxy.Port,
		Username:        proxy.Username,
		Password:        proxy.Password,
		EgressLimitBPS:  proxy.EgressLimitBPS,
		IngressLimitBPS: proxy.IngressLimitBPS,
		MaxConnections:  proxy.MaxConnections,
		ExpiresAt:       proxy.ExpiresAt,
	})
	if runtimeErr != nil {
		_ = s.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
			ProxyAccountID: proxy.ID,
			Status:         RuntimeApplyStatusFailed,
			ErrorDetail:    runtimeErr.Error(),
		})
		return order, runtimeErr
	}
	return order, nil
}

func (s *Service) HandleRuntimeApplyResult(ctx context.Context, input RuntimeApplySettlementInput) error {
	if input.ProxyAccountID == "" {
		return errors.New("proxy_account_id is required")
	}
	return s.repo.WithTx(ctx, func(repo Repository) error {
		proxy, ok, err := repo.GetProxyAccount(ctx, input.ProxyAccountID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		order, ok, err := repo.GetOrder(ctx, proxy.OrderID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if order.Status == OrderStatusDelivered && input.Status == RuntimeApplyStatusACK {
			return nil
		}
		now := s.now().UTC()
		if !input.AppliedAt.IsZero() {
			now = input.AppliedAt.UTC()
		}
		switch input.Status {
		case RuntimeApplyStatusACK:
			return s.settleRuntimeACK(ctx, repo, order, proxy, now)
		default:
			if input.ErrorDetail == "" {
				input.ErrorDetail = string(input.Status)
			}
			return s.settleRuntimeFailure(ctx, repo, order, proxy, input.ErrorDetail, now)
		}
	})
}

func (s *Service) settleRuntimeACK(ctx context.Context, repo Repository, order ProxyOrder, proxy ProxyAccount, now time.Time) error {
	if proxy.Status == "DISABLE_PENDING" || proxy.LifecycleStatus == ProxyLifecycleRuntimePending && order.Status == OrderStatusDisabled {
		proxy.Status = "DISABLED"
		proxy.LifecycleStatus = ProxyLifecycleDisabled
		proxy.Password = ""
		proxy.ConnectionURI = ""
		proxy.UpdatedAt = now
		if err := repo.SaveProxyAccount(ctx, proxy); err != nil {
			return err
		}
		order.Status = OrderStatusDisabled
		order.UpdatedAt = now
		return repo.SaveOrder(ctx, order)
	}

	hold, ok, err := repo.GetWalletHold(ctx, order.WalletHoldID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	wallet, ok, err := repo.GetWallet(ctx, order.UserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrNotFound
	}
	if hold.Status == "HELD" {
		wallet.HeldCents -= hold.AmountCents
		wallet.BalanceCents -= hold.AmountCents
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		wallet.UpdatedAt = now
		hold.Status = "CAPTURED"
		hold.UpdatedAt = now
		if err := repo.SaveWallet(ctx, wallet); err != nil {
			return err
		}
		if err := repo.SaveWalletHold(ctx, hold); err != nil {
			return err
		}
		if err := repo.AppendLedger(ctx, WalletLedger{
			ID:             uuid.NewString(),
			WalletID:       wallet.ID,
			UserID:         wallet.UserID,
			Type:           LedgerTypeDebitPurchase,
			AmountCents:    -hold.AmountCents,
			BalanceAfter:   wallet.BalanceCents,
			HeldAfter:      wallet.HeldCents,
			ReferenceType:  "proxy_order",
			ReferenceID:    order.ID,
			IdempotencyKey: "order-debit:" + order.ID,
			CreatedAt:      now,
		}); err != nil && !errors.Is(err, ErrIdempotencyConflict) {
			return err
		}
	}
	inventory, ok, err := repo.GetInventory(ctx, order.InventoryID)
	if err != nil {
		return err
	}
	if ok {
		inventory.Status = InventoryStatusSold
		inventory.SoldOrderID = order.ID
		inventory.ReservedOrderID = ""
		inventory.UpdatedAt = now
		if err := repo.SaveInventory(ctx, inventory); err != nil {
			return err
		}
	}
	if reservation, ok, err := repo.GetReservation(ctx, order.ReservationID); err != nil {
		return err
	} else if ok {
		reservation.Status = ReservationStatusConfirmed
		reservation.UpdatedAt = now
		if err := repo.SaveReservation(ctx, reservation); err != nil {
			return err
		}
	}
	order.Status = OrderStatusDelivered
	order.DeliveredAt = now
	order.UpdatedAt = now
	if err := repo.SaveOrder(ctx, order); err != nil {
		return err
	}
	proxy.Status = "ACTIVE"
	proxy.LifecycleStatus = ProxyLifecycleActive
	proxy.ConnectionURI = connectionURI(proxy)
	proxy.UpdatedAt = now
	return repo.SaveProxyAccount(ctx, proxy)
}

func (s *Service) settleRuntimeFailure(ctx context.Context, repo Repository, order ProxyOrder, proxy ProxyAccount, reason string, now time.Time) error {
	hold, ok, err := repo.GetWalletHold(ctx, order.WalletHoldID)
	if err != nil {
		return err
	}
	if ok && hold.Status == "HELD" {
		wallet, walletOK, err := repo.GetWallet(ctx, order.UserID)
		if err != nil {
			return err
		}
		if walletOK {
			wallet.HeldCents -= hold.AmountCents
			wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
			wallet.UpdatedAt = now
			if err := repo.SaveWallet(ctx, wallet); err != nil {
				return err
			}
			if err := repo.AppendLedger(ctx, WalletLedger{
				ID:             uuid.NewString(),
				WalletID:       wallet.ID,
				UserID:         wallet.UserID,
				Type:           LedgerTypeHoldRelease,
				AmountCents:    -hold.AmountCents,
				BalanceAfter:   wallet.BalanceCents,
				HeldAfter:      wallet.HeldCents,
				ReferenceType:  "proxy_order",
				ReferenceID:    order.ID,
				IdempotencyKey: "order-release:" + order.ID,
				CreatedAt:      now,
			}); err != nil && !errors.Is(err, ErrIdempotencyConflict) {
				return err
			}
		}
		hold.Status = "RELEASED"
		hold.UpdatedAt = now
		if err := repo.SaveWalletHold(ctx, hold); err != nil {
			return err
		}
	}
	if inventory, ok, err := repo.GetInventory(ctx, order.InventoryID); err != nil {
		return err
	} else if ok {
		inventory.Status = InventoryStatusAvailable
		inventory.ReservedOrderID = ""
		inventory.UpdatedAt = now
		if err := repo.SaveInventory(ctx, inventory); err != nil {
			return err
		}
	}
	if reservation, ok, err := repo.GetReservation(ctx, order.ReservationID); err != nil {
		return err
	} else if ok {
		reservation.Status = ReservationStatusReleased
		reservation.UpdatedAt = now
		if err := repo.SaveReservation(ctx, reservation); err != nil {
			return err
		}
	}
	order.Status = OrderStatusFulfillmentFailed
	order.FailureReason = reason
	order.UpdatedAt = now
	if err := repo.SaveOrder(ctx, order); err != nil {
		return err
	}
	proxy.Status = "FAILED"
	proxy.LifecycleStatus = ProxyLifecycleRuntimeFailed
	proxy.Password = ""
	proxy.ConnectionURI = ""
	proxy.UpdatedAt = now
	return repo.SaveProxyAccount(ctx, proxy)
}

func (s *Service) GetOrder(ctx context.Context, userID string, orderID string) (ProxyOrder, error) {
	order, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return ProxyOrder{}, err
	}
	if !ok || (userID != "" && order.UserID != userID) {
		return ProxyOrder{}, ErrNotFound
	}
	return order, nil
}

func (s *Service) ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error) {
	return s.repo.ListOrders(ctx, filter)
}

func (s *Service) ListUserProxies(ctx context.Context, userID string) ([]ProxyAccount, error) {
	items, err := s.repo.ListProxyAccounts(ctx, userID)
	if err != nil {
		return nil, err
	}
	visible := make([]ProxyAccount, 0, len(items))
	for _, item := range items {
		if item.Status != "ACTIVE" && item.LifecycleStatus != ProxyLifecycleDisabled {
			continue
		}
		if item.Status != "ACTIVE" {
			item.Password = ""
			item.ConnectionURI = ""
		} else if item.ConnectionURI == "" {
			item.ConnectionURI = connectionURI(item)
		}
		visible = append(visible, item)
	}
	return visible, nil
}

func (s *Service) GetProxy(ctx context.Context, userID string, proxyID string) (ProxyAccount, error) {
	proxy, ok, err := s.repo.GetProxyAccount(ctx, proxyID)
	if err != nil {
		return ProxyAccount{}, err
	}
	if !ok || (userID != "" && proxy.UserID != userID) {
		return ProxyAccount{}, ErrNotFound
	}
	if proxy.Status != "ACTIVE" {
		proxy.Password = ""
		proxy.ConnectionURI = ""
	} else if proxy.ConnectionURI == "" {
		proxy.ConnectionURI = connectionURI(proxy)
	}
	return proxy, nil
}

func (s *Service) RenewProxy(ctx context.Context, userID string, proxyID string, input RenewProxyInput) (ProxyAccount, error) {
	if input.IdempotencyKey == "" {
		return ProxyAccount{}, ErrIdempotencyRequired
	}
	if input.DurationDays <= 0 {
		input.DurationDays = 30
	}
	proxy, ok, err := s.repo.GetProxyAccount(ctx, proxyID)
	if err != nil {
		return ProxyAccount{}, err
	}
	if !ok || proxy.UserID != userID {
		return ProxyAccount{}, ErrNotFound
	}
	quote, err := s.Quote(ctx, QuoteInput{ProductID: "static-residential", Protocol: proxy.Protocol, DurationDays: input.DurationDays, Quantity: 1})
	if err != nil {
		return ProxyAccount{}, err
	}
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		now := s.now().UTC()
		wallet, ok, err := repo.GetWallet(ctx, userID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if wallet.BalanceCents-wallet.HeldCents < quote.TotalCents {
			return ErrInsufficientBalance
		}
		wallet.BalanceCents -= quote.TotalCents
		wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
		wallet.UpdatedAt = now
		if err := repo.SaveWallet(ctx, wallet); err != nil {
			return err
		}
		if err := repo.AppendLedger(ctx, WalletLedger{
			ID:             uuid.NewString(),
			WalletID:       wallet.ID,
			UserID:         userID,
			Type:           LedgerTypeDebitPurchase,
			AmountCents:    -quote.TotalCents,
			BalanceAfter:   wallet.BalanceCents,
			HeldAfter:      wallet.HeldCents,
			ReferenceType:  "proxy_renew",
			ReferenceID:    proxyID,
			IdempotencyKey: "renew:" + userID + ":" + input.IdempotencyKey,
			CreatedAt:      now,
		}); err != nil {
			return err
		}
		proxy.ExpiresAt = proxy.ExpiresAt.Add(time.Duration(input.DurationDays) * 24 * time.Hour)
		proxy.LifecycleStatus = ProxyLifecycleRuntimePending
		proxy.UpdatedAt = now
		return repo.SaveProxyAccount(ctx, proxy)
	})
	if err != nil {
		return ProxyAccount{}, err
	}
	if s.runtime != nil {
		_, err = s.runtime.UpsertProxyAccount(ctx, runtimeInputFromProxy(proxy))
	}
	return proxy, err
}

func (s *Service) DisableProxy(ctx context.Context, userID string, proxyID string) (ProxyAccount, error) {
	proxy, ok, err := s.repo.GetProxyAccount(ctx, proxyID)
	if err != nil {
		return ProxyAccount{}, err
	}
	if !ok || proxy.UserID != userID {
		return ProxyAccount{}, ErrNotFound
	}
	proxy.LifecycleStatus = ProxyLifecycleRuntimePending
	proxy.Status = "DISABLE_PENDING"
	proxy.UpdatedAt = s.now().UTC()
	if err := s.repo.WithTx(ctx, func(repo Repository) error {
		if err := repo.SaveProxyAccount(ctx, proxy); err != nil {
			return err
		}
		order, ok, err := repo.GetOrder(ctx, proxy.OrderID)
		if err != nil {
			return err
		}
		if ok {
			order.Status = OrderStatusDisabled
			order.UpdatedAt = proxy.UpdatedAt
			if err := repo.SaveOrder(ctx, order); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return ProxyAccount{}, err
	}
	if s.runtime != nil {
		_, err = s.runtime.RemoveProxyAccount(ctx, proxy.ID)
	}
	return proxy, err
}

func (s *Service) RetryFulfillment(ctx context.Context, orderID string) (ProxyOrder, error) {
	order, ok, err := s.repo.GetOrder(ctx, orderID)
	if err != nil {
		return ProxyOrder{}, err
	}
	if !ok {
		return ProxyOrder{}, ErrNotFound
	}
	if order.Status == OrderStatusDelivered {
		return order, nil
	}
	if order.Status != OrderStatusFulfillmentFailed {
		return ProxyOrder{}, errors.New("only failed fulfillment can be retried")
	}
	proxy, ok, err := s.repo.GetProxyAccount(ctx, order.ProxyAccountID)
	if err != nil {
		return ProxyOrder{}, err
	}
	if !ok {
		return ProxyOrder{}, ErrNotFound
	}
	err = s.repo.WithTx(ctx, func(repo Repository) error {
		now := s.now().UTC()
		wallet, ok, err := repo.GetWallet(ctx, order.UserID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		hold, ok, err := repo.GetWalletHold(ctx, order.WalletHoldID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		if hold.Status != "HELD" {
			if wallet.BalanceCents-wallet.HeldCents < hold.AmountCents {
				return ErrInsufficientBalance
			}
			wallet.HeldCents += hold.AmountCents
			wallet.AvailableCents = wallet.BalanceCents - wallet.HeldCents
			wallet.UpdatedAt = now
			hold.Status = "HELD"
			hold.UpdatedAt = now
			if err := repo.SaveWallet(ctx, wallet); err != nil {
				return err
			}
			if err := repo.SaveWalletHold(ctx, hold); err != nil {
				return err
			}
			if err := repo.AppendLedger(ctx, WalletLedger{
				ID:             uuid.NewString(),
				WalletID:       wallet.ID,
				UserID:         wallet.UserID,
				Type:           LedgerTypeHold,
				AmountCents:    hold.AmountCents,
				BalanceAfter:   wallet.BalanceCents,
				HeldAfter:      wallet.HeldCents,
				ReferenceType:  "proxy_order",
				ReferenceID:    order.ID,
				IdempotencyKey: "order-retry-hold:" + order.ID + ":" + fmt.Sprint(now.UnixNano()),
				CreatedAt:      now,
			}); err != nil {
				return err
			}
		}
		inventory, ok, err := repo.GetInventory(ctx, order.InventoryID)
		if err != nil {
			return err
		}
		if !ok {
			return ErrNotFound
		}
		line, lineOK, err := repo.GetLine(ctx, inventory.LineID)
		if err != nil {
			return err
		}
		status, statusOK, err := repo.GetNodeRuntimeStatus(ctx, inventory.NodeID)
		if err != nil {
			return err
		}
		if reasons := sellableInventoryReasons(line, lineOK, inventory, status, statusOK, order.Protocol); len(reasons) > 0 {
			return fmt.Errorf("%w: %s", ErrInventoryUnavailable, strings.Join(reasons, ","))
		}
		inventory.Status = InventoryStatusReserved
		inventory.ReservedOrderID = order.ID
		inventory.UpdatedAt = now
		if err := repo.SaveInventory(ctx, inventory); err != nil {
			return err
		}
		reservation, ok, err := repo.GetReservation(ctx, order.ReservationID)
		if err != nil {
			return err
		}
		if ok {
			reservation.Status = ReservationStatusActive
			reservation.ExpiresAt = now.Add(15 * time.Minute)
			reservation.UpdatedAt = now
			if err := repo.SaveReservation(ctx, reservation); err != nil {
				return err
			}
		}
		proxy.Status = "PENDING_RUNTIME"
		proxy.LifecycleStatus = ProxyLifecycleRuntimePending
		proxy.Password = randomTokenN(12)
		proxy.ConnectionURI = ""
		proxy.UpdatedAt = now
		if err := repo.SaveProxyAccount(ctx, proxy); err != nil {
			return err
		}
		order.Status = OrderStatusPendingRuntime
		order.FailureReason = ""
		order.UpdatedAt = now
		if err := repo.SaveOrder(ctx, order); err != nil {
			return err
		}
		if err := repo.SaveFulfillmentJob(ctx, FulfillmentJob{
			ID:             uuid.NewString(),
			OrderID:        order.ID,
			ProxyAccountID: proxy.ID,
			Status:         FulfillmentJobPending,
			CreatedAt:      now,
			UpdatedAt:      now,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ProxyOrder{}, err
	}
	if s.runtime != nil {
		if _, err := s.runtime.UpsertProxyAccount(ctx, runtimeInputFromProxy(proxy)); err != nil {
			_ = s.HandleRuntimeApplyResult(ctx, RuntimeApplySettlementInput{
				ProxyAccountID: proxy.ID,
				Status:         RuntimeApplyStatusFailed,
				ErrorDetail:    err.Error(),
			})
			return order, err
		}
	}
	return s.GetOrder(ctx, "", order.ID)
}

func (s *Service) ReconcileProxy(ctx context.Context, proxyID string) (ProxyAccount, error) {
	proxy, ok, err := s.repo.GetProxyAccount(ctx, proxyID)
	if err != nil {
		return ProxyAccount{}, err
	}
	if !ok {
		return ProxyAccount{}, ErrNotFound
	}
	proxy.LifecycleStatus = ProxyLifecycleRuntimePending
	proxy.UpdatedAt = s.now().UTC()
	if err := s.repo.SaveProxyAccount(ctx, proxy); err != nil {
		return ProxyAccount{}, err
	}
	if s.runtime == nil {
		return proxy, nil
	}
	if proxy.Status == "DISABLED" || proxy.Status == "DISABLE_PENDING" || proxy.LifecycleStatus == ProxyLifecycleDisabled {
		_, err = s.runtime.RemoveProxyAccount(ctx, proxy.ID)
	} else {
		_, err = s.runtime.UpsertProxyAccount(ctx, runtimeInputFromProxy(proxy))
	}
	return proxy, err
}

func (s *Service) ListAdminProxies(ctx context.Context) ([]ProxyAccount, error) {
	return s.repo.ListProxyAccounts(ctx, "")
}

func (s *Service) UpsertProduct(ctx context.Context, input UpsertProductInput) (Product, error) {
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Name == "" {
		return Product{}, errors.New("product name is required")
	}
	now := s.now().UTC()
	product := Product{
		ID:        input.ID,
		Name:      input.Name,
		IPType:    input.IPType,
		Enabled:   input.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err := s.repo.WithTx(ctx, func(repo Repository) error {
		if existing, ok, err := repo.GetProduct(ctx, product.ID); err != nil {
			return err
		} else if ok {
			product.CreatedAt = existing.CreatedAt
		}
		if err := repo.SaveProduct(ctx, product); err != nil {
			return err
		}
		for _, inputPrice := range input.Prices {
			if inputPrice.Protocol != ProtocolSOCKS5 && inputPrice.Protocol != ProtocolHTTP {
				return ErrUnsupportedProtocol
			}
			if inputPrice.DurationDays <= 0 || inputPrice.UnitCents <= 0 {
				return errors.New("duration_days and unit_cents must be positive")
			}
			if inputPrice.ProductID == "" {
				inputPrice.ProductID = product.ID
			}
			if inputPrice.ID == "" {
				inputPrice.ID = fmt.Sprintf("%s-%s-%d", inputPrice.ProductID, strings.ToLower(string(inputPrice.Protocol)), inputPrice.DurationDays)
			}
			price := ProductPrice{
				ID:           inputPrice.ID,
				ProductID:    inputPrice.ProductID,
				Protocol:     inputPrice.Protocol,
				DurationDays: inputPrice.DurationDays,
				UnitCents:    inputPrice.UnitCents,
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := repo.SaveProductPrice(ctx, price); err != nil {
				return err
			}
		}
		return nil
	})
	return product, err
}

func (s *Service) UpsertLine(ctx context.Context, input UpsertLineInput) (Line, error) {
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	now := s.now().UTC()
	line := Line{
		ID:        input.ID,
		RegionID:  input.RegionID,
		CityID:    input.CityID,
		NodeID:    input.NodeID,
		Name:      input.Name,
		Enabled:   input.Enabled,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.repo.SaveLine(ctx, line); err != nil {
		return Line{}, err
	}
	return line, nil
}

func (s *Service) UpsertNodeRuntimeStatus(ctx context.Context, input NodeRuntimeStatus) error {
	input.UpdatedAt = s.now().UTC()
	return s.repo.SaveNodeRuntimeStatus(ctx, input)
}

func (s *Service) UpsertInventory(ctx context.Context, input UpsertInventoryInput) (NodeInventoryIP, error) {
	if input.ID == "" {
		input.ID = uuid.NewString()
	}
	if input.Status == "" {
		input.Status = InventoryStatusAvailable
	}
	now := s.now().UTC()
	item := NodeInventoryIP{
		ID:             input.ID,
		LineID:         input.LineID,
		NodeID:         input.NodeID,
		IP:             input.IP,
		Port:           input.Port,
		Protocols:      append([]Protocol(nil), input.Protocols...),
		Status:         input.Status,
		ManualHold:     input.ManualHold,
		ComplianceHold: input.ComplianceHold,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if err := s.repo.SaveInventory(ctx, item); err != nil {
		return NodeInventoryIP{}, err
	}
	return item, nil
}

func (s *Service) ListInventory(ctx context.Context) ([]NodeInventoryIP, error) {
	return s.repo.ListInventory(ctx)
}

func (s *Service) ListProducts(ctx context.Context) ([]Product, error) {
	return s.repo.ListProducts(ctx)
}

func (s *Service) ListPrices(ctx context.Context) ([]ProductPrice, error) {
	return s.repo.ListPrices(ctx)
}

func (s *Service) ListLines(ctx context.Context) ([]Line, error) {
	return s.repo.ListLines(ctx)
}

func (s *Service) ListAuditLogs(ctx context.Context) ([]AuditLog, error) {
	return s.repo.ListAuditLogs(ctx)
}

func hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

func randomToken() string {
	return randomTokenN(32)
}

func randomTokenN(size int) string {
	if size <= 0 {
		size = 16
	}
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return strings.ReplaceAll(uuid.NewString(), "-", "")
	}
	return hex.EncodeToString(bytes)[:size]
}

func shortID(id string, n int) string {
	cleaned := strings.ReplaceAll(id, "-", "")
	if len(cleaned) <= n {
		return cleaned
	}
	return cleaned[:n]
}

func findLine(lines []Line, id string) (Line, bool) {
	for _, line := range lines {
		if line.ID == id {
			return line, true
		}
	}
	return Line{}, false
}

func sellableInventoryReasons(line Line, lineOK bool, item NodeInventoryIP, status NodeRuntimeStatus, statusOK bool, protocol Protocol) []string {
	reasons := []string{}
	if !lineOK {
		reasons = append(reasons, "LINE_MISSING")
	} else if !line.Enabled {
		reasons = append(reasons, "LINE_DISABLED")
	}
	if item.Status != InventoryStatusAvailable {
		reasons = append(reasons, "INVENTORY_"+string(item.Status))
	}
	if item.ManualHold {
		reasons = append(reasons, "MANUAL_HOLD")
	}
	if item.ComplianceHold {
		reasons = append(reasons, "COMPLIANCE_HOLD")
	}
	if protocol != "" && !protocolsContain(item.Protocols, protocol) {
		reasons = append(reasons, "UNSUPPORTED_PROTOCOL")
	}
	if !statusOK {
		reasons = append(reasons, "NODE_STATUS_MISSING")
	} else {
		if !status.Sellable {
			if len(status.UnsellableReasons) == 0 {
				reasons = append(reasons, "NODE_UNSELLABLE")
			}
			reasons = append(reasons, status.UnsellableReasons...)
		}
		if protocol != "" && !protocolCapabilitySupported(status.Capabilities, protocol) {
			reasons = append(reasons, "UNSUPPORTED_CAPABILITY")
		}
	}
	return uniqueStrings(reasons)
}

func protocolsContain(protocols []Protocol, protocol Protocol) bool {
	for _, item := range protocols {
		if item == protocol {
			return true
		}
	}
	return false
}

func stringSliceContains(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func protocolCapabilitySupported(capabilities []string, protocol Protocol) bool {
	needle := strings.ToLower(string(protocol))
	for _, item := range capabilities {
		if strings.ToLower(item) == needle {
			return true
		}
	}
	return false
}

func uniqueStrings(items []string) []string {
	set := map[string]struct{}{}
	for _, item := range items {
		if item != "" {
			set[item] = struct{}{}
		}
	}
	return reasonKeys(set)
}

func reasonKeys(set map[string]struct{}) []string {
	if len(set) == 0 {
		return nil
	}
	items := make([]string, 0, len(set))
	for item := range set {
		items = append(items, item)
	}
	sort.Strings(items)
	return items
}

func connectionURI(proxy ProxyAccount) string {
	user := url.QueryEscape(proxy.Username)
	pass := url.QueryEscape(proxy.Password)
	switch proxy.Protocol {
	case ProtocolHTTP:
		return fmt.Sprintf("http://%s:%s@%s:%d", user, pass, proxy.ListenIP, proxy.Port)
	default:
		return fmt.Sprintf("socks5://%s:%s@%s:%d", user, pass, proxy.ListenIP, proxy.Port)
	}
}

func runtimeInputFromProxy(proxy ProxyAccount) RuntimeProxyAccountInput {
	return RuntimeProxyAccountInput{
		ProxyAccountID:  proxy.ID,
		NodeID:          proxy.NodeID,
		RuntimeEmail:    proxy.RuntimeEmail,
		Protocol:        proxy.Protocol,
		ListenIP:        proxy.ListenIP,
		Port:            proxy.Port,
		Username:        proxy.Username,
		Password:        proxy.Password,
		EgressLimitBPS:  proxy.EgressLimitBPS,
		IngressLimitBPS: proxy.IngressLimitBPS,
		MaxConnections:  proxy.MaxConnections,
		ExpiresAt:       proxy.ExpiresAt,
	}
}
