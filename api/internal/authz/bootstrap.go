package authz

import "fmt"

// RoleSeed ??????
type RoleSeed struct {
	Role      string
	Inherits  []string
	Policies  []Policy
	Immutable bool
}

// BuiltinRoleSeeds ????????
func BuiltinRoleSeeds() []RoleSeed {
	return []RoleSeed{
		{
			Role: "readonly_auditor",
			Policies: []Policy{
				{Object: "/admin/*", Action: "GET"},
				{Object: "/admin/password", Action: "PUT"},        // ?????????????
				{Object: "/admin/ads/impression", Action: "POST"}, // ??????,????????
			},
			Immutable: true,
		},
		{
			Role:     "operations",
			Inherits: []string{"readonly_auditor"},
			Policies: []Policy{
				{Object: "/admin/products", Action: "*"},
				{Object: "/admin/products/:id", Action: "*"},
				{Object: "/admin/categories", Action: "*"},
				{Object: "/admin/categories/:id", Action: "*"},
				{Object: "/admin/posts", Action: "*"},
				{Object: "/admin/posts/:id", Action: "*"},
				{Object: "/admin/banners", Action: "*"},
				{Object: "/admin/banners/:id", Action: "*"},
				{Object: "/admin/coupons", Action: "*"},
				{Object: "/admin/coupons/:id", Action: "*"},
				{Object: "/admin/promotions", Action: "*"},
				{Object: "/admin/promotions/:id", Action: "*"},
				{Object: "/admin/card-secrets", Action: "*"},
				{Object: "/admin/card-secrets/:id", Action: "*"},
				{Object: "/admin/card-secrets/batch", Action: "POST"},
				{Object: "/admin/card-secrets/import", Action: "POST"},
				{Object: "/admin/card-secrets/batch-status", Action: "PATCH"},
				{Object: "/admin/card-secrets/batch-delete", Action: "POST"},
				{Object: "/admin/card-secrets/export", Action: "POST"},
				{Object: "/admin/card-secrets/stats", Action: "GET"},
				{Object: "/admin/card-secrets/batches", Action: "GET"},
				{Object: "/admin/card-secrets/template", Action: "GET"},
				{Object: "/admin/gift-cards", Action: "*"},
				{Object: "/admin/gift-cards/:id", Action: "*"},
				{Object: "/admin/gift-cards/generate", Action: "POST"},
				{Object: "/admin/gift-cards/batch-status", Action: "PATCH"},
				{Object: "/admin/gift-cards/export", Action: "POST"},
				{Object: "/admin/upload", Action: "POST"},
				{Object: "/admin/media/:id", Action: "PUT"},
				{Object: "/admin/media/:id", Action: "DELETE"},
				{Object: "/admin/affiliates/users", Action: "GET"},
				{Object: "/admin/affiliates/users/:id/status", Action: "PATCH"},
				{Object: "/admin/affiliates/users/batch-status", Action: "PATCH"},
				// ??????
				{Object: "/admin/member-levels", Action: "*"},
				{Object: "/admin/member-levels/:id", Action: "*"},
				{Object: "/admin/member-levels/backfill", Action: "POST"},
				{Object: "/admin/member-level-prices", Action: "*"},
				{Object: "/admin/member-level-prices/batch", Action: "POST"},
				{Object: "/admin/member-level-prices/:id", Action: "DELETE"},
			},
			Immutable: true,
		},
		{
			Role:     "support",
			Inherits: []string{"readonly_auditor"},
			Policies: []Policy{
				{Object: "/admin/orders", Action: "GET"},
				{Object: "/admin/orders/:id", Action: "GET"},
				{Object: "/admin/orders/:id", Action: "PATCH"},
				{Object: "/admin/orders/:id/fulfillment/download", Action: "GET"},
				{Object: "/admin/orders/:id/refund-to-wallet", Action: "POST"},
				{Object: "/admin/orders/:id/manual-refund", Action: "POST"},
				{Object: "/admin/order-refunds", Action: "GET"},
				{Object: "/admin/order-refunds/:id", Action: "GET"},
				{Object: "/admin/fulfillments", Action: "POST"},
				{Object: "/admin/users", Action: "GET"},
				{Object: "/admin/users/:id", Action: "GET"},
				{Object: "/admin/users/:id", Action: "PUT"},
				{Object: "/admin/users/batch-status", Action: "PUT"},
				{Object: "/admin/users/:id/coupon-usages", Action: "GET"},
				{Object: "/admin/users/:id/wallet", Action: "GET"},
				{Object: "/admin/users/:id/wallet/transactions", Action: "GET"},
				{Object: "/admin/users/:id/wallet/adjust", Action: "POST"},
				{Object: "/admin/users/:id/member-level", Action: "PUT"},
				{Object: "/admin/user-login-logs", Action: "GET"},
				{Object: "/admin/wallet/recharges", Action: "GET"},
				{Object: "/admin/payments", Action: "GET"},
				{Object: "/admin/payments/:id", Action: "GET"},
				{Object: "/admin/gift-cards", Action: "GET"},
			},
			Immutable: true,
		},
		{
			Role:     "integration",
			Inherits: []string{"readonly_auditor"},
			Policies: []Policy{
				{Object: "/admin/site-connections", Action: "*"},
				{Object: "/admin/site-connections/:id", Action: "*"},
				{Object: "/admin/site-connections/:id/ping", Action: "POST"},
				{Object: "/admin/site-connections/:id/status", Action: "PUT"},
				{Object: "/admin/site-connections/:id/reapply-markup", Action: "POST"},
				{Object: "/admin/product-mappings", Action: "*"},
				{Object: "/admin/product-mappings/:id", Action: "*"},
				{Object: "/admin/product-mappings/:id/sync", Action: "POST"},
				{Object: "/admin/product-mappings/:id/status", Action: "PUT"},
				{Object: "/admin/product-mappings/import", Action: "POST"},
				{Object: "/admin/product-mappings/batch-import", Action: "POST"},
				{Object: "/admin/product-mappings/batch-sync", Action: "POST"},
				{Object: "/admin/product-mappings/batch-status", Action: "POST"},
				{Object: "/admin/product-mappings/batch-delete", Action: "POST"},
				{Object: "/admin/procurement-orders", Action: "GET"},
				{Object: "/admin/procurement-orders/:id", Action: "GET"},
				{Object: "/admin/procurement-orders/:id/upstream-payload/download", Action: "GET"},
				{Object: "/admin/procurement-orders/:id/retry", Action: "POST"},
				{Object: "/admin/procurement-orders/:id/cancel", Action: "POST"},
				{Object: "/admin/reconciliation/run", Action: "POST"},
				{Object: "/admin/reconciliation/jobs", Action: "GET"},
				{Object: "/admin/reconciliation/jobs/:id", Action: "GET"},
				{Object: "/admin/reconciliation/items/:id/resolve", Action: "PUT"},
				{Object: "/admin/api-credentials", Action: "*"},
				{Object: "/admin/api-credentials/:id", Action: "*"},
				{Object: "/admin/api-credentials/:id/approve", Action: "POST"},
				{Object: "/admin/api-credentials/:id/reject", Action: "POST"},
				{Object: "/admin/api-credentials/:id/status", Action: "PUT"},
				{Object: "/admin/upstream-products", Action: "GET"},
			},
			Immutable: true,
		},
		{
			Role:     "finance",
			Inherits: []string{"readonly_auditor"},
			Policies: []Policy{
				{Object: "/admin/payments", Action: "GET"},
				{Object: "/admin/payments/:id", Action: "GET"},
				{Object: "/admin/payments/export", Action: "GET"},
				{Object: "/admin/payment-channels", Action: "*"},
				{Object: "/admin/payment-channels/:id", Action: "*"},
				{Object: "/admin/orders", Action: "GET"},
				{Object: "/admin/orders/:id", Action: "GET"},
				{Object: "/admin/orders/:id/refund-to-wallet", Action: "POST"},
				{Object: "/admin/orders/:id/manual-refund", Action: "POST"},
				{Object: "/admin/order-refunds", Action: "GET"},
				{Object: "/admin/order-refunds/:id", Action: "GET"},
				{Object: "/admin/affiliates/commissions", Action: "GET"},
				{Object: "/admin/affiliates/withdraws", Action: "GET"},
				{Object: "/admin/affiliates/withdraws/:id/reject", Action: "POST"},
				{Object: "/admin/affiliates/withdraws/:id/pay", Action: "POST"},
				{Object: "/admin/gift-cards", Action: "GET"},
				{Object: "/admin/gift-cards/export", Action: "POST"},
				{Object: "/admin/wallet/recharges", Action: "GET"},
			},
			Immutable: true,
		},
		{
			Role:     "system_admin",
			Inherits: []string{"readonly_auditor"},
			Policies: []Policy{
				// ????
				{Object: "/admin/settings", Action: "*"},
				{Object: "/admin/settings/smtp", Action: "*"},
				{Object: "/admin/settings/smtp/test", Action: "POST"},
				{Object: "/admin/settings/captcha", Action: "*"},
				{Object: "/admin/settings/telegram-auth", Action: "*"},
				{Object: "/admin/settings/notification-center", Action: "*"},
				{Object: "/admin/settings/notification-center/logs", Action: "GET"},
				{Object: "/admin/settings/notification-center/test", Action: "POST"},
				{Object: "/admin/settings/notifications", Action: "*"},
				{Object: "/admin/settings/notifications/logs", Action: "GET"},
				{Object: "/admin/settings/notifications/test", Action: "POST"},
				{Object: "/admin/settings/order-email-template", Action: "*"},
				{Object: "/admin/settings/order-email-template/reset", Action: "POST"},
				{Object: "/admin/settings/affiliate", Action: "*"},
				{Object: "/admin/settings/telegram-bot", Action: "*"},
				{Object: "/admin/settings/telegram-bot/runtime-status", Action: "GET"},
				// ????(? system_admin ???)
				{Object: "/admin/authz/me", Action: "GET"},
				{Object: "/admin/authz/roles", Action: "*"},
				{Object: "/admin/authz/roles/:role", Action: "*"},
				{Object: "/admin/authz/roles/:role/policies", Action: "GET"},
				{Object: "/admin/authz/admins", Action: "*"},
				{Object: "/admin/authz/admins/:id", Action: "*"},
				{Object: "/admin/authz/admins/:id/roles", Action: "*"},
				{Object: "/admin/authz/policies", Action: "*"},
				{Object: "/admin/authz/permissions/catalog", Action: "GET"},
				{Object: "/admin/authz/audit-logs", Action: "GET"},
				// ???????
				{Object: "/admin/channel-clients", Action: "*"},
				{Object: "/admin/channel-clients/:id", Action: "*"},
				{Object: "/admin/channel-clients/:id/status", Action: "PUT"},
				{Object: "/admin/channel-clients/:id/reset-secret", Action: "POST"},
				// Telegram Bot ??
				{Object: "/admin/telegram-bot/broadcasts", Action: "*"},
				{Object: "/admin/telegram-bot/users", Action: "GET"},
			},
			Immutable: true,
		},
	}
}

// BootstrapBuiltinRoles ????????????
func (s *Service) BootstrapBuiltinRoles() error {
	if s == nil || s.enforcer == nil {
		return fmt.Errorf("authz service unavailable")
	}

	changed := false
	for _, seed := range BuiltinRoleSeeds() {
		role, err := NormalizeRole(seed.Role)
		if err != nil {
			return err
		}

		exists, err := s.enforcer.HasNamedGroupingPolicy("g", role, roleAnchor)
		if err != nil {
			return fmt.Errorf("check builtin role failed: %w", err)
		}
		if !exists {
			added, err := s.enforcer.AddNamedGroupingPolicy("g", role, roleAnchor)
			if err != nil {
				return fmt.Errorf("create builtin role failed: %w", err)
			}
			if added {
				changed = true
			}
		}

		for _, parent := range seed.Inherits {
			parentRole, err := NormalizeRole(parent)
			if err != nil {
				return err
			}
			added, err := s.enforcer.AddNamedGroupingPolicy("g", role, parentRole)
			if err != nil {
				return fmt.Errorf("link role inheritance failed: %w", err)
			}
			if added {
				changed = true
			}
		}

		for _, policy := range seed.Policies {
			action := NormalizeAction(policy.Action)
			if action == "" {
				return fmt.Errorf("builtin policy action is required")
			}
			added, err := s.enforcer.AddPolicy(role, NormalizeObject(policy.Object), action)
			if err != nil {
				return fmt.Errorf("add builtin policy failed: %w", err)
			}
			if added {
				changed = true
			}
		}
	}

	if changed {
		return s.saveAndReload()
	}
	return nil
}
