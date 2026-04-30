package commercial

import (
	"context"
	"time"

	apiEnt "github.com/rayip/rayip/services/api/ent"
	entAdmin "github.com/rayip/rayip/services/api/ent/adminuser"
	entAudit "github.com/rayip/rayip/services/api/ent/auditlog"
	entCity "github.com/rayip/rayip/services/api/ent/city"
	entAttempt "github.com/rayip/rayip/services/api/ent/fulfillmentattempt"
	entJob "github.com/rayip/rayip/services/api/ent/fulfillmentjob"
	entLine "github.com/rayip/rayip/services/api/ent/line"
	entInventory "github.com/rayip/rayip/services/api/ent/nodeinventoryip"
	entPayment "github.com/rayip/rayip/services/api/ent/paymentorder"
	entProduct "github.com/rayip/rayip/services/api/ent/product"
	entPrice "github.com/rayip/rayip/services/api/ent/productprice"
	entProxy "github.com/rayip/rayip/services/api/ent/proxyaccount"
	entOrder "github.com/rayip/rayip/services/api/ent/proxyorder"
	entRate "github.com/rayip/rayip/services/api/ent/ratepolicy"
	entRegion "github.com/rayip/rayip/services/api/ent/region"
	entUser "github.com/rayip/rayip/services/api/ent/user"
	entWallet "github.com/rayip/rayip/services/api/ent/wallet"
	entLedger "github.com/rayip/rayip/services/api/ent/walletledger"
)

type EntRepository struct {
	client *apiEnt.Client
}

func NewEntRepository(client *apiEnt.Client) *EntRepository {
	return &EntRepository{client: client}
}

func (r *EntRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	tx, err := r.client.Tx(ctx)
	if err != nil {
		return err
	}
	repo := &entTxRepository{client: tx.Client()}
	if err := fn(repo); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

type entTxRepository struct {
	client *apiEnt.Client
}

func (r *EntRepository) ent() *apiEnt.Client   { return r.client }
func (r *entTxRepository) ent() *apiEnt.Client { return r.client }

func (r *entTxRepository) WithTx(ctx context.Context, fn func(Repository) error) error {
	return fn(r)
}

func (r *EntRepository) SaveUser(ctx context.Context, user User) error {
	return saveUser(ctx, r.ent(), user)
}
func (r *entTxRepository) SaveUser(ctx context.Context, user User) error {
	return saveUser(ctx, r.ent(), user)
}
func saveUser(ctx context.Context, c *apiEnt.Client, user User) error {
	if _, err := c.User.Get(ctx, user.ID); err == nil {
		return c.User.UpdateOneID(user.ID).
			SetEmail(user.Email).
			SetPasswordHash(user.PasswordHash).
			SetStatus(string(user.Status)).
			SetUpdatedAt(user.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.User.Create().
		SetID(user.ID).
		SetEmail(user.Email).
		SetPasswordHash(user.PasswordHash).
		SetStatus(string(user.Status)).
		SetCreatedAt(user.CreatedAt).
		SetUpdatedAt(user.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetUser(ctx context.Context, userID string) (User, bool, error) {
	return getUser(ctx, r.ent(), userID)
}
func (r *entTxRepository) GetUser(ctx context.Context, userID string) (User, bool, error) {
	return getUser(ctx, r.ent(), userID)
}
func getUser(ctx context.Context, c *apiEnt.Client, userID string) (User, bool, error) {
	item, err := c.User.Get(ctx, userID)
	if apiEnt.IsNotFound(err) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return userFromEnt(item), true, nil
}

func (r *EntRepository) GetUserByEmail(ctx context.Context, email string) (User, bool, error) {
	return getUserByEmail(ctx, r.ent(), email)
}
func (r *entTxRepository) GetUserByEmail(ctx context.Context, email string) (User, bool, error) {
	return getUserByEmail(ctx, r.ent(), email)
}
func getUserByEmail(ctx context.Context, c *apiEnt.Client, email string) (User, bool, error) {
	item, err := c.User.Query().Where(entUser.Email(email)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return User{}, false, nil
	}
	if err != nil {
		return User{}, false, err
	}
	return userFromEnt(item), true, nil
}

func (r *EntRepository) ListUsers(ctx context.Context) ([]User, error) {
	return listUsers(ctx, r.ent())
}
func (r *entTxRepository) ListUsers(ctx context.Context) ([]User, error) {
	return listUsers(ctx, r.ent())
}
func listUsers(ctx context.Context, c *apiEnt.Client) ([]User, error) {
	items, err := c.User.Query().Order(apiEnt.Desc(entUser.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]User, 0, len(items))
	for _, item := range items {
		out = append(out, userFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveAdminUser(ctx context.Context, admin AdminUser) error {
	return saveAdminUser(ctx, r.ent(), admin)
}
func (r *entTxRepository) SaveAdminUser(ctx context.Context, admin AdminUser) error {
	return saveAdminUser(ctx, r.ent(), admin)
}
func saveAdminUser(ctx context.Context, c *apiEnt.Client, admin AdminUser) error {
	if _, err := c.AdminUser.Get(ctx, admin.ID); err == nil {
		return c.AdminUser.UpdateOneID(admin.ID).
			SetUsername(admin.Username).
			SetPasswordHash(admin.PasswordHash).
			SetRole(admin.Role).
			SetUpdatedAt(admin.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.AdminUser.Create().
		SetID(admin.ID).
		SetUsername(admin.Username).
		SetPasswordHash(admin.PasswordHash).
		SetRole(admin.Role).
		SetCreatedAt(admin.CreatedAt).
		SetUpdatedAt(admin.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetAdminByUsername(ctx context.Context, username string) (AdminUser, bool, error) {
	return getAdminByUsername(ctx, r.ent(), username)
}
func (r *entTxRepository) GetAdminByUsername(ctx context.Context, username string) (AdminUser, bool, error) {
	return getAdminByUsername(ctx, r.ent(), username)
}
func getAdminByUsername(ctx context.Context, c *apiEnt.Client, username string) (AdminUser, bool, error) {
	item, err := c.AdminUser.Query().Where(entAdmin.Username(username)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return AdminUser{}, false, nil
	}
	if err != nil {
		return AdminUser{}, false, err
	}
	return AdminUser{ID: item.ID, Username: item.Username, PasswordHash: item.PasswordHash, Role: item.Role, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, true, nil
}

func (r *EntRepository) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	return listAdminUsers(ctx, r.ent())
}
func (r *entTxRepository) ListAdminUsers(ctx context.Context) ([]AdminUser, error) {
	return listAdminUsers(ctx, r.ent())
}
func listAdminUsers(ctx context.Context, c *apiEnt.Client) ([]AdminUser, error) {
	items, err := c.AdminUser.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AdminUser, 0, len(items))
	for _, item := range items {
		out = append(out, AdminUser{ID: item.ID, Username: item.Username, PasswordHash: item.PasswordHash, Role: item.Role, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) SaveSession(ctx context.Context, session Session) error {
	return saveSession(ctx, r.ent(), session)
}
func (r *entTxRepository) SaveSession(ctx context.Context, session Session) error {
	return saveSession(ctx, r.ent(), session)
}
func saveSession(ctx context.Context, c *apiEnt.Client, session Session) error {
	if _, err := c.Session.Get(ctx, session.ID); err == nil {
		return nil
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.Session.Create().
		SetID(session.ID).
		SetSubjectID(session.SubjectID).
		SetScope(string(session.Scope)).
		SetExpiresAt(session.ExpiresAt).
		SetCreatedAt(session.CreatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetSession(ctx context.Context, sessionID string) (Session, bool, error) {
	return getSession(ctx, r.ent(), sessionID)
}
func (r *entTxRepository) GetSession(ctx context.Context, sessionID string) (Session, bool, error) {
	return getSession(ctx, r.ent(), sessionID)
}
func getSession(ctx context.Context, c *apiEnt.Client, sessionID string) (Session, bool, error) {
	item, err := c.Session.Get(ctx, sessionID)
	if apiEnt.IsNotFound(err) {
		return Session{}, false, nil
	}
	if err != nil {
		return Session{}, false, err
	}
	return Session{ID: item.ID, SubjectID: item.SubjectID, Scope: SessionScope(item.Scope), ExpiresAt: item.ExpiresAt, CreatedAt: item.CreatedAt}, true, nil
}

func (r *EntRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return deleteSession(ctx, r.ent(), sessionID)
}
func (r *entTxRepository) DeleteSession(ctx context.Context, sessionID string) error {
	return deleteSession(ctx, r.ent(), sessionID)
}
func deleteSession(ctx context.Context, c *apiEnt.Client, sessionID string) error {
	err := c.Session.DeleteOneID(sessionID).Exec(ctx)
	if apiEnt.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *EntRepository) EnsureWallet(ctx context.Context, userID string, now time.Time) (Wallet, error) {
	return ensureWallet(ctx, r.ent(), userID, now)
}
func (r *entTxRepository) EnsureWallet(ctx context.Context, userID string, now time.Time) (Wallet, error) {
	return ensureWallet(ctx, r.ent(), userID, now)
}
func ensureWallet(ctx context.Context, c *apiEnt.Client, userID string, now time.Time) (Wallet, error) {
	item, err := c.Wallet.Query().Where(entWallet.UserID(userID)).Only(ctx)
	if err == nil {
		return walletFromEnt(item), nil
	}
	if !apiEnt.IsNotFound(err) {
		return Wallet{}, err
	}
	wallet := Wallet{ID: "wallet-" + userID, UserID: userID, UpdatedAt: now}
	err = c.Wallet.Create().
		SetID(wallet.ID).
		SetUserID(userID).
		SetBalanceCents(0).
		SetHeldCents(0).
		SetUpdatedAt(now).
		Exec(ctx)
	if err != nil {
		return Wallet{}, err
	}
	return wallet, nil
}

func (r *EntRepository) GetWallet(ctx context.Context, userID string) (Wallet, bool, error) {
	return getWallet(ctx, r.ent(), userID)
}
func (r *entTxRepository) GetWallet(ctx context.Context, userID string) (Wallet, bool, error) {
	return getWallet(ctx, r.ent(), userID)
}
func getWallet(ctx context.Context, c *apiEnt.Client, userID string) (Wallet, bool, error) {
	item, err := c.Wallet.Query().Where(entWallet.UserID(userID)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return Wallet{}, false, nil
	}
	if err != nil {
		return Wallet{}, false, err
	}
	return walletFromEnt(item), true, nil
}

func (r *EntRepository) SaveWallet(ctx context.Context, wallet Wallet) error {
	return saveWallet(ctx, r.ent(), wallet)
}
func (r *entTxRepository) SaveWallet(ctx context.Context, wallet Wallet) error {
	return saveWallet(ctx, r.ent(), wallet)
}
func saveWallet(ctx context.Context, c *apiEnt.Client, wallet Wallet) error {
	if _, err := c.Wallet.Get(ctx, wallet.ID); err == nil {
		return c.Wallet.UpdateOneID(wallet.ID).
			SetUserID(wallet.UserID).
			SetBalanceCents(wallet.BalanceCents).
			SetHeldCents(wallet.HeldCents).
			SetUpdatedAt(wallet.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.Wallet.Create().
		SetID(wallet.ID).
		SetUserID(wallet.UserID).
		SetBalanceCents(wallet.BalanceCents).
		SetHeldCents(wallet.HeldCents).
		SetUpdatedAt(wallet.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) AppendLedger(ctx context.Context, item WalletLedger) error {
	return appendLedger(ctx, r.ent(), item)
}
func (r *entTxRepository) AppendLedger(ctx context.Context, item WalletLedger) error {
	return appendLedger(ctx, r.ent(), item)
}
func appendLedger(ctx context.Context, c *apiEnt.Client, item WalletLedger) error {
	if item.IdempotencyKey != "" {
		if _, ok, err := getLedgerByIdempotencyKey(ctx, c, item.IdempotencyKey); err != nil {
			return err
		} else if ok {
			return ErrIdempotencyConflict
		}
	}
	return c.WalletLedger.Create().
		SetID(item.ID).
		SetWalletID(item.WalletID).
		SetUserID(item.UserID).
		SetType(string(item.Type)).
		SetAmountCents(item.AmountCents).
		SetBalanceAfterCents(item.BalanceAfter).
		SetHeldAfterCents(item.HeldAfter).
		SetReferenceType(item.ReferenceType).
		SetReferenceID(item.ReferenceID).
		SetIdempotencyKey(item.IdempotencyKey).
		SetCreatedAt(item.CreatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetLedgerByIdempotencyKey(ctx context.Context, key string) (WalletLedger, bool, error) {
	return getLedgerByIdempotencyKey(ctx, r.ent(), key)
}
func (r *entTxRepository) GetLedgerByIdempotencyKey(ctx context.Context, key string) (WalletLedger, bool, error) {
	return getLedgerByIdempotencyKey(ctx, r.ent(), key)
}
func getLedgerByIdempotencyKey(ctx context.Context, c *apiEnt.Client, key string) (WalletLedger, bool, error) {
	item, err := c.WalletLedger.Query().Where(entLedger.IdempotencyKey(key)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return WalletLedger{}, false, nil
	}
	if err != nil {
		return WalletLedger{}, false, err
	}
	return ledgerFromEnt(item), true, nil
}

func (r *EntRepository) ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error) {
	return listWalletLedger(ctx, r.ent(), filter)
}
func (r *entTxRepository) ListWalletLedger(ctx context.Context, filter LedgerFilter) ([]WalletLedger, error) {
	return listWalletLedger(ctx, r.ent(), filter)
}
func listWalletLedger(ctx context.Context, c *apiEnt.Client, filter LedgerFilter) ([]WalletLedger, error) {
	query := c.WalletLedger.Query().Order(apiEnt.Desc(entLedger.FieldCreatedAt))
	if filter.UserID != "" {
		query = query.Where(entLedger.UserID(filter.UserID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]WalletLedger, 0, len(items))
	for _, item := range items {
		out = append(out, ledgerFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveWalletHold(ctx context.Context, hold WalletHold) error {
	return saveWalletHold(ctx, r.ent(), hold)
}
func (r *entTxRepository) SaveWalletHold(ctx context.Context, hold WalletHold) error {
	return saveWalletHold(ctx, r.ent(), hold)
}
func saveWalletHold(ctx context.Context, c *apiEnt.Client, hold WalletHold) error {
	if _, err := c.WalletHold.Get(ctx, hold.ID); err == nil {
		return c.WalletHold.UpdateOneID(hold.ID).
			SetWalletID(hold.WalletID).
			SetUserID(hold.UserID).
			SetOrderID(hold.OrderID).
			SetAmountCents(hold.AmountCents).
			SetStatus(hold.Status).
			SetUpdatedAt(hold.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.WalletHold.Create().
		SetID(hold.ID).
		SetWalletID(hold.WalletID).
		SetUserID(hold.UserID).
		SetOrderID(hold.OrderID).
		SetAmountCents(hold.AmountCents).
		SetStatus(hold.Status).
		SetCreatedAt(hold.CreatedAt).
		SetUpdatedAt(hold.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetWalletHold(ctx context.Context, holdID string) (WalletHold, bool, error) {
	return getWalletHold(ctx, r.ent(), holdID)
}
func (r *entTxRepository) GetWalletHold(ctx context.Context, holdID string) (WalletHold, bool, error) {
	return getWalletHold(ctx, r.ent(), holdID)
}
func getWalletHold(ctx context.Context, c *apiEnt.Client, holdID string) (WalletHold, bool, error) {
	item, err := c.WalletHold.Get(ctx, holdID)
	if apiEnt.IsNotFound(err) {
		return WalletHold{}, false, nil
	}
	if err != nil {
		return WalletHold{}, false, err
	}
	return WalletHold{ID: item.ID, WalletID: item.WalletID, UserID: item.UserID, OrderID: item.OrderID, AmountCents: item.AmountCents, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, true, nil
}

func (r *EntRepository) SavePaymentOrder(ctx context.Context, order PaymentOrder) error {
	return savePaymentOrder(ctx, r.ent(), order)
}
func (r *entTxRepository) SavePaymentOrder(ctx context.Context, order PaymentOrder) error {
	return savePaymentOrder(ctx, r.ent(), order)
}
func savePaymentOrder(ctx context.Context, c *apiEnt.Client, order PaymentOrder) error {
	if _, err := c.PaymentOrder.Get(ctx, order.ID); err == nil {
		update := c.PaymentOrder.UpdateOneID(order.ID).
			SetUserID(order.UserID).
			SetAmountCents(order.AmountCents).
			SetStatus(string(order.Status)).
			SetProvider(order.Provider).
			SetProviderTradeNo(order.ProviderTradeNo).
			SetUpdatedAt(order.UpdatedAt)
		if !order.PaidAt.IsZero() {
			update.SetPaidAt(order.PaidAt)
		}
		return update.Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	create := c.PaymentOrder.Create().
		SetID(order.ID).
		SetUserID(order.UserID).
		SetAmountCents(order.AmountCents).
		SetStatus(string(order.Status)).
		SetProvider(order.Provider).
		SetProviderTradeNo(order.ProviderTradeNo).
		SetCreatedAt(order.CreatedAt).
		SetUpdatedAt(order.UpdatedAt)
	if !order.PaidAt.IsZero() {
		create.SetPaidAt(order.PaidAt)
	}
	return create.Exec(ctx)
}

func (r *EntRepository) GetPaymentOrder(ctx context.Context, orderID string) (PaymentOrder, bool, error) {
	return getPaymentOrder(ctx, r.ent(), orderID)
}
func (r *entTxRepository) GetPaymentOrder(ctx context.Context, orderID string) (PaymentOrder, bool, error) {
	return getPaymentOrder(ctx, r.ent(), orderID)
}
func getPaymentOrder(ctx context.Context, c *apiEnt.Client, orderID string) (PaymentOrder, bool, error) {
	item, err := c.PaymentOrder.Get(ctx, orderID)
	if apiEnt.IsNotFound(err) {
		return PaymentOrder{}, false, nil
	}
	if err != nil {
		return PaymentOrder{}, false, err
	}
	return paymentFromEnt(item), true, nil
}

func (r *EntRepository) GetPaymentOrderByProviderTrade(ctx context.Context, orderID string, providerTradeNo string) (PaymentOrder, bool, error) {
	return getPaymentOrderByProviderTrade(ctx, r.ent(), orderID, providerTradeNo)
}
func (r *entTxRepository) GetPaymentOrderByProviderTrade(ctx context.Context, orderID string, providerTradeNo string) (PaymentOrder, bool, error) {
	return getPaymentOrderByProviderTrade(ctx, r.ent(), orderID, providerTradeNo)
}
func getPaymentOrderByProviderTrade(ctx context.Context, c *apiEnt.Client, orderID string, providerTradeNo string) (PaymentOrder, bool, error) {
	item, err := c.PaymentOrder.Query().Where(entPayment.ID(orderID), entPayment.ProviderTradeNo(providerTradeNo)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return PaymentOrder{}, false, nil
	}
	if err != nil {
		return PaymentOrder{}, false, err
	}
	return paymentFromEnt(item), true, nil
}

func (r *EntRepository) ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error) {
	return listPaymentOrders(ctx, r.ent())
}
func (r *entTxRepository) ListPaymentOrders(ctx context.Context) ([]PaymentOrder, error) {
	return listPaymentOrders(ctx, r.ent())
}
func listPaymentOrders(ctx context.Context, c *apiEnt.Client) ([]PaymentOrder, error) {
	items, err := c.PaymentOrder.Query().Order(apiEnt.Desc(entPayment.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PaymentOrder, 0, len(items))
	for _, item := range items {
		out = append(out, paymentFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) AppendAudit(ctx context.Context, item AuditLog) error {
	return appendAudit(ctx, r.ent(), item)
}
func (r *entTxRepository) AppendAudit(ctx context.Context, item AuditLog) error {
	return appendAudit(ctx, r.ent(), item)
}
func appendAudit(ctx context.Context, c *apiEnt.Client, item AuditLog) error {
	return c.AuditLog.Create().
		SetID(item.ID).
		SetActorID(item.ActorID).
		SetActorType(item.ActorType).
		SetAction(item.Action).
		SetTargetID(item.TargetID).
		SetMetadata(item.Metadata).
		SetCreatedAt(item.CreatedAt).
		Exec(ctx)
}

func (r *EntRepository) ListAuditLogs(ctx context.Context) ([]AuditLog, error) {
	return listAuditLogs(ctx, r.ent())
}
func (r *entTxRepository) ListAuditLogs(ctx context.Context) ([]AuditLog, error) {
	return listAuditLogs(ctx, r.ent())
}
func listAuditLogs(ctx context.Context, c *apiEnt.Client) ([]AuditLog, error) {
	items, err := c.AuditLog.Query().Order(apiEnt.Desc(entAudit.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]AuditLog, 0, len(items))
	for _, item := range items {
		out = append(out, AuditLog{ID: item.ID, ActorID: item.ActorID, ActorType: item.ActorType, Action: item.Action, TargetID: item.TargetID, Metadata: item.Metadata, CreatedAt: item.CreatedAt})
	}
	return out, nil
}

func (r *EntRepository) SaveRegion(ctx context.Context, item Region) error {
	return saveRegion(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveRegion(ctx context.Context, item Region) error {
	return saveRegion(ctx, r.ent(), item)
}
func saveRegion(ctx context.Context, c *apiEnt.Client, item Region) error {
	if _, err := c.Region.Get(ctx, item.ID); err == nil {
		return c.Region.UpdateOneID(item.ID).SetName(item.Name).SetCountry(item.Country).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.Region.Create().SetID(item.ID).SetName(item.Name).SetCountry(item.Country).SetCreatedAt(item.CreatedAt).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
}

func (r *EntRepository) SaveCity(ctx context.Context, item City) error {
	return saveCity(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveCity(ctx context.Context, item City) error {
	return saveCity(ctx, r.ent(), item)
}
func saveCity(ctx context.Context, c *apiEnt.Client, item City) error {
	if _, err := c.City.Get(ctx, item.ID); err == nil {
		return c.City.UpdateOneID(item.ID).SetRegionID(item.RegionID).SetName(item.Name).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.City.Create().SetID(item.ID).SetRegionID(item.RegionID).SetName(item.Name).SetCreatedAt(item.CreatedAt).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
}

func (r *EntRepository) SaveLine(ctx context.Context, item Line) error {
	return saveLine(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveLine(ctx context.Context, item Line) error {
	return saveLine(ctx, r.ent(), item)
}
func saveLine(ctx context.Context, c *apiEnt.Client, item Line) error {
	if _, err := c.Line.Get(ctx, item.ID); err == nil {
		return c.Line.UpdateOneID(item.ID).SetRegionID(item.RegionID).SetCityID(item.CityID).SetNodeID(item.NodeID).SetName(item.Name).SetEnabled(item.Enabled).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.Line.Create().SetID(item.ID).SetRegionID(item.RegionID).SetCityID(item.CityID).SetNodeID(item.NodeID).SetName(item.Name).SetEnabled(item.Enabled).SetCreatedAt(item.CreatedAt).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
}

func (r *EntRepository) SaveProduct(ctx context.Context, item Product) error {
	return saveProduct(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveProduct(ctx context.Context, item Product) error {
	return saveProduct(ctx, r.ent(), item)
}
func saveProduct(ctx context.Context, c *apiEnt.Client, item Product) error {
	if _, err := c.Product.Get(ctx, item.ID); err == nil {
		return c.Product.UpdateOneID(item.ID).SetName(item.Name).SetIPType(item.IPType).SetEnabled(item.Enabled).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.Product.Create().SetID(item.ID).SetName(item.Name).SetIPType(item.IPType).SetEnabled(item.Enabled).SetCreatedAt(item.CreatedAt).SetUpdatedAt(item.UpdatedAt).Exec(ctx)
}

func (r *EntRepository) SaveProductPrice(ctx context.Context, item ProductPrice) error {
	return saveProductPrice(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveProductPrice(ctx context.Context, item ProductPrice) error {
	return saveProductPrice(ctx, r.ent(), item)
}
func saveProductPrice(ctx context.Context, c *apiEnt.Client, item ProductPrice) error {
	if existing, err := c.ProductPrice.Query().Where(entPrice.ProductID(item.ProductID), entPrice.Protocol(string(item.Protocol)), entPrice.DurationDays(item.DurationDays)).Only(ctx); err == nil {
		return c.ProductPrice.UpdateOneID(existing.ID).
			SetProductID(item.ProductID).
			SetProtocol(string(item.Protocol)).
			SetDurationDays(item.DurationDays).
			SetUnitCents(item.UnitCents).
			SetUpdatedAt(item.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.ProductPrice.Create().
		SetID(item.ID).
		SetProductID(item.ProductID).
		SetProtocol(string(item.Protocol)).
		SetDurationDays(item.DurationDays).
		SetUnitCents(item.UnitCents).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) SaveRatePolicy(ctx context.Context, item RatePolicy) error {
	return saveRatePolicy(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveRatePolicy(ctx context.Context, item RatePolicy) error {
	return saveRatePolicy(ctx, r.ent(), item)
}
func saveRatePolicy(ctx context.Context, c *apiEnt.Client, item RatePolicy) error {
	if _, err := c.RatePolicy.Get(ctx, item.ID); err == nil {
		return c.RatePolicy.UpdateOneID(item.ID).
			SetName(item.Name).
			SetEgressLimitBps(item.EgressLimitBPS).
			SetIngressLimitBps(item.IngressLimitBPS).
			SetMaxConnections(item.MaxConnections).
			SetUpdatedAt(item.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.RatePolicy.Create().
		SetID(item.ID).
		SetName(item.Name).
		SetEgressLimitBps(item.EgressLimitBPS).
		SetIngressLimitBps(item.IngressLimitBPS).
		SetMaxConnections(item.MaxConnections).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetLine(ctx context.Context, lineID string) (Line, bool, error) {
	return getLine(ctx, r.ent(), lineID)
}
func (r *entTxRepository) GetLine(ctx context.Context, lineID string) (Line, bool, error) {
	return getLine(ctx, r.ent(), lineID)
}
func getLine(ctx context.Context, c *apiEnt.Client, lineID string) (Line, bool, error) {
	item, err := c.Line.Get(ctx, lineID)
	if apiEnt.IsNotFound(err) {
		return Line{}, false, nil
	}
	if err != nil {
		return Line{}, false, err
	}
	return lineFromEnt(item), true, nil
}

func (r *EntRepository) GetProduct(ctx context.Context, productID string) (Product, bool, error) {
	return getProduct(ctx, r.ent(), productID)
}
func (r *entTxRepository) GetProduct(ctx context.Context, productID string) (Product, bool, error) {
	return getProduct(ctx, r.ent(), productID)
}
func getProduct(ctx context.Context, c *apiEnt.Client, productID string) (Product, bool, error) {
	item, err := c.Product.Get(ctx, productID)
	if apiEnt.IsNotFound(err) {
		return Product{}, false, nil
	}
	if err != nil {
		return Product{}, false, err
	}
	return Product{ID: item.ID, Name: item.Name, IPType: item.IPType, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, true, nil
}

func (r *EntRepository) FindPrice(ctx context.Context, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error) {
	return findPrice(ctx, r.ent(), productID, protocol, durationDays)
}
func (r *entTxRepository) FindPrice(ctx context.Context, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error) {
	return findPrice(ctx, r.ent(), productID, protocol, durationDays)
}
func findPrice(ctx context.Context, c *apiEnt.Client, productID string, protocol Protocol, durationDays int) (ProductPrice, bool, error) {
	item, err := c.ProductPrice.Query().Where(entPrice.ProductID(productID), entPrice.Protocol(string(protocol)), entPrice.DurationDays(durationDays)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return ProductPrice{}, false, nil
	}
	if err != nil {
		return ProductPrice{}, false, err
	}
	return priceFromEnt(item), true, nil
}

func (r *EntRepository) ListRegions(ctx context.Context) ([]Region, error) {
	return listRegions(ctx, r.ent())
}
func (r *entTxRepository) ListRegions(ctx context.Context) ([]Region, error) {
	return listRegions(ctx, r.ent())
}
func listRegions(ctx context.Context, c *apiEnt.Client) ([]Region, error) {
	items, err := c.Region.Query().Order(apiEnt.Asc(entRegion.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Region, 0, len(items))
	for _, item := range items {
		out = append(out, Region{ID: item.ID, Name: item.Name, Country: item.Country, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) ListCities(ctx context.Context) ([]City, error) {
	return listCities(ctx, r.ent())
}
func (r *entTxRepository) ListCities(ctx context.Context) ([]City, error) {
	return listCities(ctx, r.ent())
}
func listCities(ctx context.Context, c *apiEnt.Client) ([]City, error) {
	items, err := c.City.Query().Order(apiEnt.Asc(entCity.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]City, 0, len(items))
	for _, item := range items {
		out = append(out, City{ID: item.ID, RegionID: item.RegionID, Name: item.Name, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) ListLines(ctx context.Context) ([]Line, error) {
	return listLines(ctx, r.ent())
}
func (r *entTxRepository) ListLines(ctx context.Context) ([]Line, error) {
	return listLines(ctx, r.ent())
}
func listLines(ctx context.Context, c *apiEnt.Client) ([]Line, error) {
	items, err := c.Line.Query().Order(apiEnt.Asc(entLine.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Line, 0, len(items))
	for _, item := range items {
		out = append(out, lineFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) ListProducts(ctx context.Context) ([]Product, error) {
	return listProducts(ctx, r.ent())
}
func (r *entTxRepository) ListProducts(ctx context.Context) ([]Product, error) {
	return listProducts(ctx, r.ent())
}
func listProducts(ctx context.Context, c *apiEnt.Client) ([]Product, error) {
	items, err := c.Product.Query().Order(apiEnt.Asc(entProduct.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]Product, 0, len(items))
	for _, item := range items {
		out = append(out, Product{ID: item.ID, Name: item.Name, IPType: item.IPType, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) ListPrices(ctx context.Context) ([]ProductPrice, error) {
	return listPrices(ctx, r.ent())
}
func (r *entTxRepository) ListPrices(ctx context.Context) ([]ProductPrice, error) {
	return listPrices(ctx, r.ent())
}
func listPrices(ctx context.Context, c *apiEnt.Client) ([]ProductPrice, error) {
	items, err := c.ProductPrice.Query().Order(apiEnt.Asc(entPrice.FieldDurationDays)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProductPrice, 0, len(items))
	for _, item := range items {
		out = append(out, priceFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) ListRatePolicies(ctx context.Context) ([]RatePolicy, error) {
	return listRatePolicies(ctx, r.ent())
}
func (r *entTxRepository) ListRatePolicies(ctx context.Context) ([]RatePolicy, error) {
	return listRatePolicies(ctx, r.ent())
}
func listRatePolicies(ctx context.Context, c *apiEnt.Client) ([]RatePolicy, error) {
	items, err := c.RatePolicy.Query().Order(apiEnt.Asc(entRate.FieldName)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]RatePolicy, 0, len(items))
	for _, item := range items {
		out = append(out, RatePolicy{ID: item.ID, Name: item.Name, EgressLimitBPS: item.EgressLimitBps, IngressLimitBPS: item.IngressLimitBps, MaxConnections: item.MaxConnections, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) SaveNodeRuntimeStatus(ctx context.Context, item NodeRuntimeStatus) error {
	return saveNodeRuntimeStatus(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveNodeRuntimeStatus(ctx context.Context, item NodeRuntimeStatus) error {
	return saveNodeRuntimeStatus(ctx, r.ent(), item)
}
func saveNodeRuntimeStatus(ctx context.Context, c *apiEnt.Client, item NodeRuntimeStatus) error {
	if _, err := c.NodeRuntimeStatus.Get(ctx, item.NodeID); err == nil {
		return c.NodeRuntimeStatus.UpdateOneID(item.NodeID).
			SetLeaseOnline(item.LeaseOnline).
			SetRuntimeVerdict(item.RuntimeVerdict).
			SetCapabilities(item.Capabilities).
			SetSellable(item.Sellable).
			SetUnsellableReasons(item.UnsellableReasons).
			SetUpdatedAt(item.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.NodeRuntimeStatus.Create().
		SetID(item.NodeID).
		SetLeaseOnline(item.LeaseOnline).
		SetRuntimeVerdict(item.RuntimeVerdict).
		SetCapabilities(item.Capabilities).
		SetSellable(item.Sellable).
		SetUnsellableReasons(item.UnsellableReasons).
		SetUpdatedAt(item.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetNodeRuntimeStatus(ctx context.Context, nodeID string) (NodeRuntimeStatus, bool, error) {
	return getNodeRuntimeStatus(ctx, r.ent(), nodeID)
}
func (r *entTxRepository) GetNodeRuntimeStatus(ctx context.Context, nodeID string) (NodeRuntimeStatus, bool, error) {
	return getNodeRuntimeStatus(ctx, r.ent(), nodeID)
}
func getNodeRuntimeStatus(ctx context.Context, c *apiEnt.Client, nodeID string) (NodeRuntimeStatus, bool, error) {
	item, err := c.NodeRuntimeStatus.Get(ctx, nodeID)
	if apiEnt.IsNotFound(err) {
		return NodeRuntimeStatus{}, false, nil
	}
	if err != nil {
		return NodeRuntimeStatus{}, false, err
	}
	return nodeRuntimeStatusFromEnt(item), true, nil
}

func (r *EntRepository) ListNodeRuntimeStatuses(ctx context.Context) ([]NodeRuntimeStatus, error) {
	return listNodeRuntimeStatuses(ctx, r.ent())
}
func (r *entTxRepository) ListNodeRuntimeStatuses(ctx context.Context) ([]NodeRuntimeStatus, error) {
	return listNodeRuntimeStatuses(ctx, r.ent())
}
func listNodeRuntimeStatuses(ctx context.Context, c *apiEnt.Client) ([]NodeRuntimeStatus, error) {
	items, err := c.NodeRuntimeStatus.Query().All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]NodeRuntimeStatus, 0, len(items))
	for _, item := range items {
		out = append(out, nodeRuntimeStatusFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveInventory(ctx context.Context, item NodeInventoryIP) error {
	return saveInventory(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveInventory(ctx context.Context, item NodeInventoryIP) error {
	return saveInventory(ctx, r.ent(), item)
}
func saveInventory(ctx context.Context, c *apiEnt.Client, item NodeInventoryIP) error {
	protocols := protocolsToStrings(item.Protocols)
	if _, err := c.NodeInventoryIP.Get(ctx, item.ID); err == nil {
		return c.NodeInventoryIP.UpdateOneID(item.ID).
			SetLineID(item.LineID).
			SetNodeID(item.NodeID).
			SetIP(item.IP).
			SetPort(item.Port).
			SetProtocols(protocols).
			SetStatus(string(item.Status)).
			SetManualHold(item.ManualHold).
			SetComplianceHold(item.ComplianceHold).
			SetSoldOrderID(item.SoldOrderID).
			SetReservedOrderID(item.ReservedOrderID).
			SetUpdatedAt(item.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.NodeInventoryIP.Create().
		SetID(item.ID).
		SetLineID(item.LineID).
		SetNodeID(item.NodeID).
		SetIP(item.IP).
		SetPort(item.Port).
		SetProtocols(protocols).
		SetStatus(string(item.Status)).
		SetManualHold(item.ManualHold).
		SetComplianceHold(item.ComplianceHold).
		SetSoldOrderID(item.SoldOrderID).
		SetReservedOrderID(item.ReservedOrderID).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetInventory(ctx context.Context, inventoryID string) (NodeInventoryIP, bool, error) {
	return getInventory(ctx, r.ent(), inventoryID)
}
func (r *entTxRepository) GetInventory(ctx context.Context, inventoryID string) (NodeInventoryIP, bool, error) {
	return getInventory(ctx, r.ent(), inventoryID)
}
func getInventory(ctx context.Context, c *apiEnt.Client, inventoryID string) (NodeInventoryIP, bool, error) {
	item, err := c.NodeInventoryIP.Get(ctx, inventoryID)
	if apiEnt.IsNotFound(err) {
		return NodeInventoryIP{}, false, nil
	}
	if err != nil {
		return NodeInventoryIP{}, false, err
	}
	return inventoryFromEnt(item), true, nil
}

func (r *EntRepository) ListInventory(ctx context.Context) ([]NodeInventoryIP, error) {
	return listInventory(ctx, r.ent())
}
func (r *entTxRepository) ListInventory(ctx context.Context) ([]NodeInventoryIP, error) {
	return listInventory(ctx, r.ent())
}
func listInventory(ctx context.Context, c *apiEnt.Client) ([]NodeInventoryIP, error) {
	items, err := c.NodeInventoryIP.Query().Order(apiEnt.Desc(entInventory.FieldCreatedAt)).All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]NodeInventoryIP, 0, len(items))
	for _, item := range items {
		out = append(out, inventoryFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveReservation(ctx context.Context, item InventoryReservation) error {
	return saveReservation(ctx, r.ent(), item)
}
func (r *entTxRepository) SaveReservation(ctx context.Context, item InventoryReservation) error {
	return saveReservation(ctx, r.ent(), item)
}
func saveReservation(ctx context.Context, c *apiEnt.Client, item InventoryReservation) error {
	if _, err := c.InventoryReservation.Get(ctx, item.ID); err == nil {
		return c.InventoryReservation.UpdateOneID(item.ID).
			SetInventoryID(item.InventoryID).
			SetUserID(item.UserID).
			SetOrderID(item.OrderID).
			SetStatus(string(item.Status)).
			SetExpiresAt(item.ExpiresAt).
			SetUpdatedAt(item.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.InventoryReservation.Create().
		SetID(item.ID).
		SetInventoryID(item.InventoryID).
		SetUserID(item.UserID).
		SetOrderID(item.OrderID).
		SetStatus(string(item.Status)).
		SetExpiresAt(item.ExpiresAt).
		SetCreatedAt(item.CreatedAt).
		SetUpdatedAt(item.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetReservation(ctx context.Context, reservationID string) (InventoryReservation, bool, error) {
	return getReservation(ctx, r.ent(), reservationID)
}
func (r *entTxRepository) GetReservation(ctx context.Context, reservationID string) (InventoryReservation, bool, error) {
	return getReservation(ctx, r.ent(), reservationID)
}
func getReservation(ctx context.Context, c *apiEnt.Client, reservationID string) (InventoryReservation, bool, error) {
	item, err := c.InventoryReservation.Get(ctx, reservationID)
	if apiEnt.IsNotFound(err) {
		return InventoryReservation{}, false, nil
	}
	if err != nil {
		return InventoryReservation{}, false, err
	}
	return InventoryReservation{ID: item.ID, InventoryID: item.InventoryID, UserID: item.UserID, OrderID: item.OrderID, Status: ReservationStatus(item.Status), ExpiresAt: item.ExpiresAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}, true, nil
}

func (r *EntRepository) SaveOrder(ctx context.Context, order ProxyOrder) error {
	return saveOrder(ctx, r.ent(), order)
}
func (r *entTxRepository) SaveOrder(ctx context.Context, order ProxyOrder) error {
	return saveOrder(ctx, r.ent(), order)
}
func saveOrder(ctx context.Context, c *apiEnt.Client, order ProxyOrder) error {
	if _, err := c.ProxyOrder.Get(ctx, order.ID); err == nil {
		update := c.ProxyOrder.UpdateOneID(order.ID).
			SetUserID(order.UserID).
			SetProductID(order.ProductID).
			SetInventoryID(order.InventoryID).
			SetReservationID(order.ReservationID).
			SetWalletHoldID(order.WalletHoldID).
			SetProxyAccountID(order.ProxyAccountID).
			SetIdempotencyKey(order.IdempotencyKey).
			SetProtocol(string(order.Protocol)).
			SetDurationDays(order.DurationDays).
			SetQuantity(order.Quantity).
			SetAmountCents(order.AmountCents).
			SetStatus(string(order.Status)).
			SetFailureReason(order.FailureReason).
			SetUpdatedAt(order.UpdatedAt)
		if !order.DeliveredAt.IsZero() {
			update.SetDeliveredAt(order.DeliveredAt)
		}
		if !order.ExpiresAt.IsZero() {
			update.SetExpiresAt(order.ExpiresAt)
		}
		return update.Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	create := c.ProxyOrder.Create().
		SetID(order.ID).
		SetUserID(order.UserID).
		SetProductID(order.ProductID).
		SetInventoryID(order.InventoryID).
		SetReservationID(order.ReservationID).
		SetWalletHoldID(order.WalletHoldID).
		SetProxyAccountID(order.ProxyAccountID).
		SetIdempotencyKey(order.IdempotencyKey).
		SetProtocol(string(order.Protocol)).
		SetDurationDays(order.DurationDays).
		SetQuantity(order.Quantity).
		SetAmountCents(order.AmountCents).
		SetStatus(string(order.Status)).
		SetFailureReason(order.FailureReason).
		SetCreatedAt(order.CreatedAt).
		SetUpdatedAt(order.UpdatedAt)
	if !order.DeliveredAt.IsZero() {
		create.SetDeliveredAt(order.DeliveredAt)
	}
	if !order.ExpiresAt.IsZero() {
		create.SetExpiresAt(order.ExpiresAt)
	}
	return create.Exec(ctx)
}

func (r *EntRepository) GetOrder(ctx context.Context, orderID string) (ProxyOrder, bool, error) {
	return getOrder(ctx, r.ent(), orderID)
}
func (r *entTxRepository) GetOrder(ctx context.Context, orderID string) (ProxyOrder, bool, error) {
	return getOrder(ctx, r.ent(), orderID)
}
func getOrder(ctx context.Context, c *apiEnt.Client, orderID string) (ProxyOrder, bool, error) {
	item, err := c.ProxyOrder.Get(ctx, orderID)
	if apiEnt.IsNotFound(err) {
		return ProxyOrder{}, false, nil
	}
	if err != nil {
		return ProxyOrder{}, false, err
	}
	return orderFromEnt(item), true, nil
}

func (r *EntRepository) GetOrderByIdempotencyKey(ctx context.Context, userID string, key string) (ProxyOrder, bool, error) {
	return getOrderByIdempotencyKey(ctx, r.ent(), userID, key)
}
func (r *entTxRepository) GetOrderByIdempotencyKey(ctx context.Context, userID string, key string) (ProxyOrder, bool, error) {
	return getOrderByIdempotencyKey(ctx, r.ent(), userID, key)
}
func getOrderByIdempotencyKey(ctx context.Context, c *apiEnt.Client, userID string, key string) (ProxyOrder, bool, error) {
	item, err := c.ProxyOrder.Query().Where(entOrder.UserID(userID), entOrder.IdempotencyKey(key)).Only(ctx)
	if apiEnt.IsNotFound(err) {
		return ProxyOrder{}, false, nil
	}
	if err != nil {
		return ProxyOrder{}, false, err
	}
	return orderFromEnt(item), true, nil
}

func (r *EntRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error) {
	return listOrders(ctx, r.ent(), filter)
}
func (r *entTxRepository) ListOrders(ctx context.Context, filter OrderFilter) ([]ProxyOrder, error) {
	return listOrders(ctx, r.ent(), filter)
}
func listOrders(ctx context.Context, c *apiEnt.Client, filter OrderFilter) ([]ProxyOrder, error) {
	query := c.ProxyOrder.Query().Order(apiEnt.Desc(entOrder.FieldCreatedAt))
	if filter.UserID != "" {
		query = query.Where(entOrder.UserID(filter.UserID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProxyOrder, 0, len(items))
	for _, item := range items {
		out = append(out, orderFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveProxyAccount(ctx context.Context, account ProxyAccount) error {
	return saveProxyAccount(ctx, r.ent(), account)
}
func (r *entTxRepository) SaveProxyAccount(ctx context.Context, account ProxyAccount) error {
	return saveProxyAccount(ctx, r.ent(), account)
}
func saveProxyAccount(ctx context.Context, c *apiEnt.Client, account ProxyAccount) error {
	if _, err := c.ProxyAccount.Get(ctx, account.ID); err == nil {
		return c.ProxyAccount.UpdateOneID(account.ID).
			SetOrderID(account.OrderID).
			SetUserID(account.UserID).
			SetNodeID(account.NodeID).
			SetInventoryID(account.InventoryID).
			SetProtocol(string(account.Protocol)).
			SetListenIP(account.ListenIP).
			SetPort(account.Port).
			SetUsername(account.Username).
			SetPassword(account.Password).
			SetConnectionURI(account.ConnectionURI).
			SetRuntimeEmail(account.RuntimeEmail).
			SetEgressLimitBps(account.EgressLimitBPS).
			SetIngressLimitBps(account.IngressLimitBPS).
			SetMaxConnections(account.MaxConnections).
			SetStatus(account.Status).
			SetLifecycleStatus(string(account.LifecycleStatus)).
			SetExpiresAt(account.ExpiresAt).
			SetUpdatedAt(account.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.ProxyAccount.Create().
		SetID(account.ID).
		SetOrderID(account.OrderID).
		SetUserID(account.UserID).
		SetNodeID(account.NodeID).
		SetInventoryID(account.InventoryID).
		SetProtocol(string(account.Protocol)).
		SetListenIP(account.ListenIP).
		SetPort(account.Port).
		SetUsername(account.Username).
		SetPassword(account.Password).
		SetConnectionURI(account.ConnectionURI).
		SetRuntimeEmail(account.RuntimeEmail).
		SetEgressLimitBps(account.EgressLimitBPS).
		SetIngressLimitBps(account.IngressLimitBPS).
		SetMaxConnections(account.MaxConnections).
		SetStatus(account.Status).
		SetLifecycleStatus(string(account.LifecycleStatus)).
		SetExpiresAt(account.ExpiresAt).
		SetCreatedAt(account.CreatedAt).
		SetUpdatedAt(account.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) GetProxyAccount(ctx context.Context, proxyID string) (ProxyAccount, bool, error) {
	return getProxyAccount(ctx, r.ent(), proxyID)
}
func (r *entTxRepository) GetProxyAccount(ctx context.Context, proxyID string) (ProxyAccount, bool, error) {
	return getProxyAccount(ctx, r.ent(), proxyID)
}
func getProxyAccount(ctx context.Context, c *apiEnt.Client, proxyID string) (ProxyAccount, bool, error) {
	item, err := c.ProxyAccount.Get(ctx, proxyID)
	if apiEnt.IsNotFound(err) {
		return ProxyAccount{}, false, nil
	}
	if err != nil {
		return ProxyAccount{}, false, err
	}
	return proxyFromEnt(item), true, nil
}

func (r *EntRepository) ListProxyAccounts(ctx context.Context, userID string) ([]ProxyAccount, error) {
	return listProxyAccounts(ctx, r.ent(), userID)
}
func (r *entTxRepository) ListProxyAccounts(ctx context.Context, userID string) ([]ProxyAccount, error) {
	return listProxyAccounts(ctx, r.ent(), userID)
}
func listProxyAccounts(ctx context.Context, c *apiEnt.Client, userID string) ([]ProxyAccount, error) {
	query := c.ProxyAccount.Query().Order(apiEnt.Desc(entProxy.FieldCreatedAt))
	if userID != "" {
		query = query.Where(entProxy.UserID(userID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProxyAccount, 0, len(items))
	for _, item := range items {
		out = append(out, proxyFromEnt(item))
	}
	return out, nil
}

func (r *EntRepository) SaveFulfillmentJob(ctx context.Context, job FulfillmentJob) error {
	return saveFulfillmentJob(ctx, r.ent(), job)
}
func (r *entTxRepository) SaveFulfillmentJob(ctx context.Context, job FulfillmentJob) error {
	return saveFulfillmentJob(ctx, r.ent(), job)
}
func saveFulfillmentJob(ctx context.Context, c *apiEnt.Client, job FulfillmentJob) error {
	if _, err := c.FulfillmentJob.Get(ctx, job.ID); err == nil {
		return c.FulfillmentJob.UpdateOneID(job.ID).
			SetOrderID(job.OrderID).
			SetProxyAccountID(job.ProxyAccountID).
			SetStatus(string(job.Status)).
			SetErrorDetail(job.ErrorDetail).
			SetUpdatedAt(job.UpdatedAt).
			Exec(ctx)
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.FulfillmentJob.Create().
		SetID(job.ID).
		SetOrderID(job.OrderID).
		SetProxyAccountID(job.ProxyAccountID).
		SetStatus(string(job.Status)).
		SetErrorDetail(job.ErrorDetail).
		SetCreatedAt(job.CreatedAt).
		SetUpdatedAt(job.UpdatedAt).
		Exec(ctx)
}

func (r *EntRepository) ListFulfillmentJobs(ctx context.Context, orderID string) ([]FulfillmentJob, error) {
	return listFulfillmentJobs(ctx, r.ent(), orderID)
}
func (r *entTxRepository) ListFulfillmentJobs(ctx context.Context, orderID string) ([]FulfillmentJob, error) {
	return listFulfillmentJobs(ctx, r.ent(), orderID)
}
func listFulfillmentJobs(ctx context.Context, c *apiEnt.Client, orderID string) ([]FulfillmentJob, error) {
	query := c.FulfillmentJob.Query().Order(apiEnt.Desc(entJob.FieldCreatedAt))
	if orderID != "" {
		query = query.Where(entJob.OrderID(orderID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FulfillmentJob, 0, len(items))
	for _, item := range items {
		out = append(out, FulfillmentJob{ID: item.ID, OrderID: item.OrderID, ProxyAccountID: item.ProxyAccountID, Status: FulfillmentJobStatus(item.Status), ErrorDetail: item.ErrorDetail, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt})
	}
	return out, nil
}

func (r *EntRepository) SaveFulfillmentAttempt(ctx context.Context, attempt FulfillmentAttempt) error {
	return saveFulfillmentAttempt(ctx, r.ent(), attempt)
}
func (r *entTxRepository) SaveFulfillmentAttempt(ctx context.Context, attempt FulfillmentAttempt) error {
	return saveFulfillmentAttempt(ctx, r.ent(), attempt)
}
func saveFulfillmentAttempt(ctx context.Context, c *apiEnt.Client, attempt FulfillmentAttempt) error {
	if _, err := c.FulfillmentAttempt.Get(ctx, attempt.ID); err == nil {
		return nil
	} else if !apiEnt.IsNotFound(err) {
		return err
	}
	return c.FulfillmentAttempt.Create().
		SetID(attempt.ID).
		SetJobID(attempt.JobID).
		SetStatus(attempt.Status).
		SetError(attempt.Error).
		SetCreatedAt(attempt.CreatedAt).
		Exec(ctx)
}

func (r *EntRepository) ListFulfillmentAttempts(ctx context.Context, jobID string) ([]FulfillmentAttempt, error) {
	return listFulfillmentAttempts(ctx, r.ent(), jobID)
}
func (r *entTxRepository) ListFulfillmentAttempts(ctx context.Context, jobID string) ([]FulfillmentAttempt, error) {
	return listFulfillmentAttempts(ctx, r.ent(), jobID)
}
func listFulfillmentAttempts(ctx context.Context, c *apiEnt.Client, jobID string) ([]FulfillmentAttempt, error) {
	query := c.FulfillmentAttempt.Query().Order(apiEnt.Desc(entAttempt.FieldCreatedAt))
	if jobID != "" {
		query = query.Where(entAttempt.JobID(jobID))
	}
	items, err := query.All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]FulfillmentAttempt, 0, len(items))
	for _, item := range items {
		out = append(out, FulfillmentAttempt{ID: item.ID, JobID: item.JobID, Status: item.Status, Error: item.Error, CreatedAt: item.CreatedAt})
	}
	return out, nil
}

func userFromEnt(item *apiEnt.User) User {
	return User{ID: item.ID, Email: item.Email, PasswordHash: item.PasswordHash, Status: UserStatus(item.Status), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func walletFromEnt(item *apiEnt.Wallet) Wallet {
	return Wallet{ID: item.ID, UserID: item.UserID, BalanceCents: item.BalanceCents, HeldCents: item.HeldCents, AvailableCents: item.BalanceCents - item.HeldCents, UpdatedAt: item.UpdatedAt}
}

func ledgerFromEnt(item *apiEnt.WalletLedger) WalletLedger {
	return WalletLedger{ID: item.ID, WalletID: item.WalletID, UserID: item.UserID, Type: LedgerType(item.Type), AmountCents: item.AmountCents, BalanceAfter: item.BalanceAfterCents, HeldAfter: item.HeldAfterCents, ReferenceType: item.ReferenceType, ReferenceID: item.ReferenceID, IdempotencyKey: item.IdempotencyKey, CreatedAt: item.CreatedAt}
}

func paymentFromEnt(item *apiEnt.PaymentOrder) PaymentOrder {
	paidAt := time.Time{}
	if item.PaidAt != nil {
		paidAt = *item.PaidAt
	}
	return PaymentOrder{ID: item.ID, UserID: item.UserID, AmountCents: item.AmountCents, Status: PaymentOrderStatus(item.Status), Provider: item.Provider, ProviderTradeNo: item.ProviderTradeNo, PaidAt: paidAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func lineFromEnt(item *apiEnt.Line) Line {
	return Line{ID: item.ID, RegionID: item.RegionID, CityID: item.CityID, NodeID: item.NodeID, Name: item.Name, Enabled: item.Enabled, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func priceFromEnt(item *apiEnt.ProductPrice) ProductPrice {
	return ProductPrice{ID: item.ID, ProductID: item.ProductID, Protocol: Protocol(item.Protocol), DurationDays: item.DurationDays, UnitCents: item.UnitCents, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func nodeRuntimeStatusFromEnt(item *apiEnt.NodeRuntimeStatus) NodeRuntimeStatus {
	return NodeRuntimeStatus{NodeID: item.ID, LeaseOnline: item.LeaseOnline, RuntimeVerdict: item.RuntimeVerdict, Sellable: item.Sellable, Capabilities: item.Capabilities, UnsellableReasons: item.UnsellableReasons, UpdatedAt: item.UpdatedAt}
}

func inventoryFromEnt(item *apiEnt.NodeInventoryIP) NodeInventoryIP {
	return NodeInventoryIP{ID: item.ID, LineID: item.LineID, NodeID: item.NodeID, IP: item.IP, Port: item.Port, Protocols: stringsToProtocols(item.Protocols), Status: InventoryStatus(item.Status), ManualHold: item.ManualHold, ComplianceHold: item.ComplianceHold, SoldOrderID: item.SoldOrderID, ReservedOrderID: item.ReservedOrderID, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func orderFromEnt(item *apiEnt.ProxyOrder) ProxyOrder {
	deliveredAt := time.Time{}
	if item.DeliveredAt != nil {
		deliveredAt = *item.DeliveredAt
	}
	expiresAt := time.Time{}
	if item.ExpiresAt != nil {
		expiresAt = *item.ExpiresAt
	}
	return ProxyOrder{ID: item.ID, UserID: item.UserID, ProductID: item.ProductID, InventoryID: item.InventoryID, ReservationID: item.ReservationID, WalletHoldID: item.WalletHoldID, ProxyAccountID: item.ProxyAccountID, IdempotencyKey: item.IdempotencyKey, Protocol: Protocol(item.Protocol), DurationDays: item.DurationDays, Quantity: item.Quantity, AmountCents: item.AmountCents, Status: OrderStatus(item.Status), FailureReason: item.FailureReason, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt, DeliveredAt: deliveredAt, ExpiresAt: expiresAt}
}

func proxyFromEnt(item *apiEnt.ProxyAccount) ProxyAccount {
	return ProxyAccount{ID: item.ID, OrderID: item.OrderID, UserID: item.UserID, NodeID: item.NodeID, InventoryID: item.InventoryID, Protocol: Protocol(item.Protocol), ListenIP: item.ListenIP, Port: item.Port, Username: item.Username, Password: item.Password, ConnectionURI: item.ConnectionURI, RuntimeEmail: item.RuntimeEmail, EgressLimitBPS: item.EgressLimitBps, IngressLimitBPS: item.IngressLimitBps, MaxConnections: item.MaxConnections, Status: item.Status, LifecycleStatus: ProxyLifecycleStatus(item.LifecycleStatus), ExpiresAt: item.ExpiresAt, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func protocolsToStrings(protocols []Protocol) []string {
	out := make([]string, 0, len(protocols))
	for _, protocol := range protocols {
		out = append(out, string(protocol))
	}
	return out
}

func stringsToProtocols(protocols []string) []Protocol {
	out := make([]Protocol, 0, len(protocols))
	for _, protocol := range protocols {
		out = append(out, Protocol(protocol))
	}
	return out
}

var _ Repository = (*EntRepository)(nil)
var _ Repository = (*entTxRepository)(nil)
