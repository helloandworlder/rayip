package httpapi

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/rayip/rayip/services/api/internal/commercial"
)

const (
	userSessionCookie  = "rayip_session"
	adminSessionCookie = "rayip_admin_session"
)

func RegisterCommercialRoutes(app *fiber.App, svc *commercial.Service) {
	app.Post("/api/auth/register", func(c fiber.Ctx) error {
		var input commercial.RegisterInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		user, err := svc.Register(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"user": user})
	})

	app.Post("/api/auth/login", func(c fiber.Ctx) error {
		var input commercial.LoginInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		user, session, err := svc.Login(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		setSessionCookie(c, userSessionCookie, session)
		wallet, _ := svc.GetWallet(c.Context(), user.ID)
		return c.JSON(fiber.Map{"user": user, "wallet": wallet})
	})

	app.Post("/api/auth/logout", func(c fiber.Ctx) error {
		_ = svc.Logout(c.Context(), c.Req().Cookies(userSessionCookie))
		clearSessionCookie(c, userSessionCookie)
		return c.JSON(fiber.Map{"ok": true})
	})

	app.Get("/api/me", func(c fiber.Ctx) error {
		user, ok, err := currentUser(c, svc)
		if err != nil {
			return httpCommercialError(err)
		}
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "login required")
		}
		wallet, _ := svc.GetWallet(c.Context(), user.ID)
		return c.JSON(fiber.Map{"user": user, "wallet": wallet})
	})

	app.Get("/api/wallet", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		wallet, err := svc.GetWallet(c.Context(), user.ID)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"wallet": wallet})
	}))

	app.Post("/api/payments/orders", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		var input commercial.CreatePaymentOrderInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		order, err := svc.CreatePaymentOrder(c.Context(), user.ID, input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"order": order})
	}))

	app.Post("/api/payments/mock-callback", func(c fiber.Ctx) error {
		var input commercial.PaymentCallbackInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		order, err := svc.HandleMockPaymentCallback(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"order": order})
	})

	app.Get("/api/catalog/static-residential", func(c fiber.Ctx) error {
		catalog, err := svc.GetStaticResidentialCatalog(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"catalog": catalog})
	})

	app.Post("/api/catalog/quote", func(c fiber.Ctx) error {
		var input commercial.QuoteInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		quote, err := svc.Quote(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"quote": quote})
	})

	app.Post("/api/inventory/reservations", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		var input commercial.CreateReservationInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		reservation, err := svc.CreateReservation(c.Context(), user.ID, input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"reservation": reservation})
	}))

	app.Post("/api/orders", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		var input commercial.CreateOrderInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if input.IdempotencyKey == "" {
			input.IdempotencyKey = c.Get("Idempotency-Key")
		}
		order, err := svc.CreateOrder(c.Context(), user.ID, input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"order": order})
	}))

	app.Get("/api/orders", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		items, err := svc.ListOrders(c.Context(), commercial.OrderFilter{UserID: user.ID})
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/orders/:id", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		order, err := svc.GetOrder(c.Context(), user.ID, c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"order": order})
	}))

	app.Get("/api/proxies", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		items, err := svc.ListUserProxies(c.Context(), user.ID)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/proxies/:id", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		proxy, err := svc.GetProxy(c.Context(), user.ID, c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"proxy": proxy})
	}))

	app.Post("/api/proxies/:id/renew", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		var input commercial.RenewProxyInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if input.IdempotencyKey == "" {
			input.IdempotencyKey = c.Get("Idempotency-Key")
		}
		proxy, err := svc.RenewProxy(c.Context(), user.ID, c.Params("id"), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"proxy": proxy})
	}))

	app.Post("/api/proxies/:id/disable", userRequired(svc, func(c fiber.Ctx, user commercial.User) error {
		proxy, err := svc.DisableProxy(c.Context(), user.ID, c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"proxy": proxy})
	}))

	registerCommercialAdminRoutes(app, svc)
}

func registerCommercialAdminRoutes(app *fiber.App, svc *commercial.Service) {
	app.Post("/api/admin/auth/login", func(c fiber.Ctx) error {
		var input struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		admin, session, err := svc.AdminLogin(c.Context(), input.Username, input.Password)
		if err != nil {
			return httpCommercialError(err)
		}
		setSessionCookie(c, adminSessionCookie, session)
		return c.JSON(fiber.Map{"admin": admin})
	})

	app.Get("/api/admin/users", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListUsers(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/admin/wallet-ledger", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListWalletLedger(c.Context(), commercial.LedgerFilter{})
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/admin/payment-orders", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListPaymentOrders(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/admin/audit-logs", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListAuditLogs(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/admin/products", adminRequired(svc, func(c fiber.Ctx) error {
		products, err := svc.ListProducts(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		prices, _ := svc.ListPrices(c.Context())
		lines, _ := svc.ListLines(c.Context())
		return c.JSON(fiber.Map{"items": products, "prices": prices, "lines": lines, "total": len(products)})
	}))

	app.Post("/api/admin/products", adminRequired(svc, func(c fiber.Ctx) error {
		var input commercial.UpsertProductInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		product, err := svc.UpsertProduct(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"product": product})
	}))

	app.Post("/api/admin/lines", adminRequired(svc, func(c fiber.Ctx) error {
		var input commercial.UpsertLineInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		line, err := svc.UpsertLine(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"line": line})
	}))

	app.Post("/api/admin/node-runtime-status", adminRequired(svc, func(c fiber.Ctx) error {
		var input commercial.NodeRuntimeStatus
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		if err := svc.UpsertNodeRuntimeStatus(c.Context(), input); err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"ok": true})
	}))

	app.Get("/api/admin/inventory", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListInventory(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Post("/api/admin/inventory", adminRequired(svc, func(c fiber.Ctx) error {
		var input commercial.UpsertInventoryInput
		if err := c.Bind().Body(&input); err != nil {
			return fiber.NewError(fiber.StatusBadRequest, err.Error())
		}
		item, err := svc.UpsertInventory(c.Context(), input)
		if err != nil {
			return httpCommercialError(err)
		}
		return c.Status(fiber.StatusCreated).JSON(fiber.Map{"inventory": item})
	}))

	app.Get("/api/admin/orders", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListOrders(c.Context(), commercial.OrderFilter{})
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Get("/api/admin/orders/:id", adminRequired(svc, func(c fiber.Ctx) error {
		order, err := svc.GetOrder(c.Context(), "", c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"order": order})
	}))

	app.Post("/api/admin/orders/:id/retry-fulfillment", adminRequired(svc, func(c fiber.Ctx) error {
		order, err := svc.RetryFulfillment(c.Context(), c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"order": order, "accepted": true})
	}))

	app.Get("/api/admin/proxies", adminRequired(svc, func(c fiber.Ctx) error {
		items, err := svc.ListAdminProxies(c.Context())
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"items": items, "total": len(items)})
	}))

	app.Post("/api/admin/proxies/:id/reconcile", adminRequired(svc, func(c fiber.Ctx) error {
		proxy, err := svc.ReconcileProxy(c.Context(), c.Params("id"))
		if err != nil {
			return httpCommercialError(err)
		}
		return c.JSON(fiber.Map{"accepted": true, "proxy": proxy})
	}))
}

func userRequired(svc *commercial.Service, next func(fiber.Ctx, commercial.User) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		user, ok, err := currentUser(c, svc)
		if err != nil {
			return httpCommercialError(err)
		}
		if !ok {
			return fiber.NewError(fiber.StatusUnauthorized, "login required")
		}
		return next(c, user)
	}
}

func adminRequired(svc *commercial.Service, next func(fiber.Ctx) error) fiber.Handler {
	return func(c fiber.Ctx) error {
		session, ok, err := svc.GetSession(c.Context(), c.Req().Cookies(adminSessionCookie))
		if err != nil {
			return httpCommercialError(err)
		}
		if !ok || session.Scope != commercial.SessionScopeAdmin {
			return fiber.NewError(fiber.StatusUnauthorized, "admin login required")
		}
		return next(c)
	}
}

func currentUser(c fiber.Ctx, svc *commercial.Service) (commercial.User, bool, error) {
	sessionID := c.Req().Cookies(userSessionCookie)
	if sessionID == "" {
		return commercial.User{}, false, nil
	}
	session, ok, err := svc.GetSession(c.Context(), sessionID)
	if err != nil || !ok || session.Scope != commercial.SessionScopeUser {
		return commercial.User{}, false, err
	}
	user, err := svc.GetUser(c.Context(), session.SubjectID)
	if err != nil {
		return commercial.User{}, false, err
	}
	return user, true, nil
}

func setSessionCookie(c fiber.Ctx, name string, session commercial.Session) {
	c.Res().Cookie(&fiber.Cookie{
		Name:     name,
		Value:    session.ID,
		Path:     "/",
		Expires:  session.ExpiresAt,
		SameSite: "Lax",
		HTTPOnly: true,
	})
}

func clearSessionCookie(c fiber.Ctx, name string) {
	c.Res().Cookie(&fiber.Cookie{
		Name:     name,
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		MaxAge:   -1,
		HTTPOnly: true,
	})
}

func httpCommercialError(err error) error {
	switch {
	case errors.Is(err, commercial.ErrUnauthorized):
		return fiber.NewError(fiber.StatusUnauthorized, err.Error())
	case errors.Is(err, commercial.ErrNotFound):
		return fiber.NewError(fiber.StatusNotFound, err.Error())
	case errors.Is(err, commercial.ErrAlreadyExists), errors.Is(err, commercial.ErrInsufficientBalance), errors.Is(err, commercial.ErrIdempotencyRequired), errors.Is(err, commercial.ErrInventoryUnavailable), errors.Is(err, commercial.ErrUnsupportedProtocol):
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	default:
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
}
