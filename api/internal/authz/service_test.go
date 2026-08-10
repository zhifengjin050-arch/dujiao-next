package authz

import (
	"fmt"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAuthzServiceTest(t *testing.T) *Service {
	t.Helper()
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	svc, err := NewService(db)
	if err != nil {
		t.Fatalf("new authz service failed: %v", err)
	}
	return svc
}

func TestEnforceAdminWithRolePolicy(t *testing.T) {
	svc := setupAuthzServiceTest(t)
	if err := svc.GrantRolePolicy("ops", "/admin/products/:id", "GET"); err != nil {
		t.Fatalf("grant role policy failed: %v", err)
	}
	if err := svc.SetAdminRoles(1, []string{"ops"}); err != nil {
		t.Fatalf("set admin roles failed: %v", err)
	}

	allow, err := svc.EnforceAdmin(1, "/api/v1/admin/products/42", "get")
	if err != nil {
		t.Fatalf("enforce allow failed: %v", err)
	}
	if !allow {
		t.Fatalf("expected allow=true")
	}

	allow, err = svc.EnforceAdmin(1, "/api/v1/admin/products/42", "POST")
	if err != nil {
		t.Fatalf("enforce deny failed: %v", err)
	}
	if allow {
		t.Fatalf("expected allow=false")
	}
}

func TestSetAdminRolesOverride(t *testing.T) {
	svc := setupAuthzServiceTest(t)
	if err := svc.GrantRolePolicy("ops", "/admin/orders", "GET"); err != nil {
		t.Fatalf("grant ops policy failed: %v", err)
	}
	if err := svc.GrantRolePolicy("finance", "/admin/payments", "GET"); err != nil {
		t.Fatalf("grant finance policy failed: %v", err)
	}

	if err := svc.SetAdminRoles(2, []string{"ops"}); err != nil {
		t.Fatalf("set first role failed: %v", err)
	}
	roles, err := svc.GetAdminRoles(2)
	if err != nil {
		t.Fatalf("get roles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "role:ops" {
		t.Fatalf("roles want [role:ops], got=%v", roles)
	}

	if err := svc.SetAdminRoles(2, []string{"finance"}); err != nil {
		t.Fatalf("set second role failed: %v", err)
	}
	roles, err = svc.GetAdminRoles(2)
	if err != nil {
		t.Fatalf("get roles failed: %v", err)
	}
	if len(roles) != 1 || roles[0] != "role:finance" {
		t.Fatalf("roles want [role:finance], got=%v", roles)
	}

	allow, err := svc.EnforceAdmin(2, "/admin/orders", "GET")
	if err != nil {
		t.Fatalf("enforce old role failed: %v", err)
	}
	if allow {
		t.Fatalf("expected old role permission removed")
	}

	allow, err = svc.EnforceAdmin(2, "/admin/payments", "GET")
	if err != nil {
		t.Fatalf("enforce new role failed: %v", err)
	}
	if !allow {
		t.Fatalf("expected new role permission granted")
	}
}

func TestNormalizeObject(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{in: "/api/v1/admin/orders/:id", want: "/admin/orders/:id"},
		{in: "/admin/orders/:id", want: "/admin/orders/:id"},
		{in: "admin/orders", want: "/admin/orders"},
		{in: "/api/v1", want: "/"},
		{in: "", want: "/"},
	}
	for _, item := range cases {
		got := NormalizeObject(item.in)
		if got != item.want {
			t.Fatalf("normalize object failed, in=%q want=%q got=%q", item.in, item.want, got)
		}
	}
}

func TestBootstrapBuiltinRoles(t *testing.T) {
	svc := setupAuthzServiceTest(t)
	if err := svc.BootstrapBuiltinRoles(); err != nil {
		t.Fatalf("bootstrap builtin roles failed: %v", err)
	}

	roles, err := svc.ListRoles()
	if err != nil {
		t.Fatalf("list roles failed: %v", err)
	}
	wantRoles := map[string]bool{
		"role:readonly_auditor": true,
		"role:operations":       true,
		"role:support":          true,
		"role:finance":          true,
	}
	for _, role := range roles {
		delete(wantRoles, role)
	}
	if len(wantRoles) != 0 {
		t.Fatalf("builtin roles missing: %v", wantRoles)
	}

	if err := svc.SetAdminRoles(3, []string{"operations"}); err != nil {
		t.Fatalf("set admin roles failed: %v", err)
	}

	allow, err := svc.EnforceAdmin(3, "/admin/settings", "GET")
	if err != nil {
		t.Fatalf("enforce inherited readonly failed: %v", err)
	}
	if !allow {
		t.Fatalf("expected inherited readonly permission")
	}

	allow, err = svc.EnforceAdmin(3, "/admin/settings", "PUT")
	if err != nil {
		t.Fatalf("enforce readonly write failed: %v", err)
	}
	if allow {
		t.Fatalf("expected readonly inherited role deny write")
	}
}
