package cache

import (
	"context"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
)

func TestBuildUserAuthStateNil(t *testing.T) {
	state := BuildUserAuthState(nil)
	if state != nil {
		t.Fatal("BuildUserAuthState(nil) should return nil")
	}
}

func TestBuildUserAuthState(t *testing.T) {
	now := time.Now()
	user := &models.User{
		ID:           123,
		Status:       "active",
		TokenVersion: 5,
	}
	user.TokenInvalidBefore = &now

	state := BuildUserAuthState(user)
	if state == nil {
		t.Fatal("BuildUserAuthState should not return nil")
	}
	if state.UserID != 123 {
		t.Fatalf("expected UserID=123, got %d", state.UserID)
	}
	if state.Status != "active" {
		t.Fatalf("expected Status=active, got %s", state.Status)
	}
	if state.TokenVersion != 5 {
		t.Fatalf("expected TokenVersion=5, got %d", state.TokenVersion)
	}
	if state.TokenInvalidBefore != now.Unix() {
		t.Fatalf("expected TokenInvalidBefore=%d, got %d", now.Unix(), state.TokenInvalidBefore)
	}
	if state.UpdatedAt == 0 {
		t.Fatal("expected UpdatedAt to be set")
	}
}

func TestBuildUserAuthStateWithoutTokenInvalidBefore(t *testing.T) {
	user := &models.User{
		ID:           456,
		Status:       "banned",
		TokenVersion: 0,
	}

	state := BuildUserAuthState(user)
	if state == nil {
		t.Fatal("BuildUserAuthState should not return nil")
	}
	if state.UserID != 456 {
		t.Fatalf("expected UserID=456, got %d", state.UserID)
	}
	if state.TokenInvalidBefore != 0 {
		t.Fatalf("expected TokenInvalidBefore=0 when nil, got %d", state.TokenInvalidBefore)
	}
}

func TestBuildAdminAuthStateNil(t *testing.T) {
	state := BuildAdminAuthState(nil)
	if state != nil {
		t.Fatal("BuildAdminAuthState(nil) should return nil")
	}
}

func TestBuildAdminAuthState(t *testing.T) {
	now := time.Now()
	admin := &models.Admin{
		ID:            1,
		Username:      "superadmin",
		TokenVersion:  10,
		IsSuper:       true,
	}
	admin.TokenInvalidBefore = &now

	state := BuildAdminAuthState(admin)
	if state == nil {
		t.Fatal("BuildAdminAuthState should not return nil")
	}
	if state.AdminID != 1 {
		t.Fatalf("expected AdminID=1, got %d", state.AdminID)
	}
	if state.Username != "superadmin" {
		t.Fatalf("expected Username=superadmin, got %s", state.Username)
	}
	if state.TokenVersion != 10 {
		t.Fatalf("expected TokenVersion=10, got %d", state.TokenVersion)
	}
	if !state.IsSuper {
		t.Fatal("expected IsSuper=true")
	}
	if state.TokenInvalidBefore != now.Unix() {
		t.Fatalf("expected TokenInvalidBefore=%d, got %d", now.Unix(), state.TokenInvalidBefore)
	}
}

func TestBuildAdminAuthStateWithoutTokenInvalidBefore(t *testing.T) {
	admin := &models.Admin{
		ID:       2,
		Username: "editor",
		IsSuper:  false,
	}

	state := BuildAdminAuthState(admin)
	if state == nil {
		t.Fatal("BuildAdminAuthState should not return nil")
	}
	if state.IsSuper {
		t.Fatal("expected IsSuper=false")
	}
	if state.TokenInvalidBefore != 0 {
		t.Fatalf("expected TokenInvalidBefore=0 when nil, got %d", state.TokenInvalidBefore)
	}
}

func TestUserAuthStateKey(t *testing.T) {
	key := userAuthStateKey(123)
	if key != "auth:user:123" {
		t.Fatalf("expected auth:user:123, got %s", key)
	}
}

func TestAdminAuthStateKey(t *testing.T) {
	key := adminAuthStateKey(1)
	if key != "auth:admin:1" {
		t.Fatalf("expected auth:admin:1, got %s", key)
	}
}

func TestGetUserAuthStateZeroID(t *testing.T) {
	state, hit, err := GetUserAuthState(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Fatal("expected hit=false for zero user ID")
	}
	if state != nil {
		t.Fatal("expected nil state for zero user ID")
	}
}

func TestSetUserAuthStateNil(t *testing.T) {
	err := SetUserAuthState(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetUserAuthStateZeroID(t *testing.T) {
	state := &UserAuthState{UserID: 0, Status: "active"}
	err := SetUserAuthState(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelUserAuthStateZeroID(t *testing.T) {
	err := DelUserAuthState(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestGetAdminAuthStateZeroID(t *testing.T) {
	state, hit, err := GetAdminAuthState(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if hit {
		t.Fatal("expected hit=false for zero admin ID")
	}
	if state != nil {
		t.Fatal("expected nil state for zero admin ID")
	}
}

func TestSetAdminAuthStateNil(t *testing.T) {
	err := SetAdminAuthState(context.Background(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSetAdminAuthStateZeroID(t *testing.T) {
	state := &AdminAuthState{AdminID: 0, Username: "test"}
	err := SetAdminAuthState(context.Background(), state)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDelAdminAuthStateZeroID(t *testing.T) {
	err := DelAdminAuthState(context.Background(), 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
