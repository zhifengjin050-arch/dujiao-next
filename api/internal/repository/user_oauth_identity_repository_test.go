package repository

import (
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestListTelegramUsers(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserOAuthIdentity{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	user1 := &models.User{
		Email:        "alice@example.com",
		PasswordHash: "hash",
		DisplayName:  "Alice",
		Status:       "active",
	}
	user2 := &models.User{
		Email:        "bob@example.com",
		PasswordHash: "hash",
		DisplayName:  "Bob",
		Status:       "active",
	}
	if err := db.Create(user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}

	boundAt1 := time.Date(2026, 3, 1, 10, 0, 0, 0, time.UTC)
	boundAt2 := time.Date(2026, 3, 5, 10, 0, 0, 0, time.UTC)
	identity1 := &models.UserOAuthIdentity{
		UserID:         user1.ID,
		Provider:       "telegram",
		ProviderUserID: "10001",
		Username:       "alice_tg",
		CreatedAt:      boundAt1,
		UpdatedAt:      boundAt1,
	}
	identity2 := &models.UserOAuthIdentity{
		UserID:         user2.ID,
		Provider:       "telegram",
		ProviderUserID: "20002",
		Username:       "bob_tg",
		CreatedAt:      boundAt2,
		UpdatedAt:      boundAt2,
	}
	if err := db.Create(identity1).Error; err != nil {
		t.Fatalf("create identity1 failed: %v", err)
	}
	if err := db.Create(identity2).Error; err != nil {
		t.Fatalf("create identity2 failed: %v", err)
	}

	repo := NewUserOAuthIdentityRepository(db)

	items, total, err := repo.ListTelegramUsers(TelegramUserListFilter{
		Keyword:  "alice",
		Page:     1,
		PageSize: 10,
	})
	if err != nil {
		t.Fatalf("list telegram users failed: %v", err)
	}
	if total != 1 {
		t.Fatalf("expected total 1, got %d", total)
	}
	if len(items) != 1 || items[0].TelegramUserID != "10001" {
		t.Fatalf("unexpected list result: %+v", items)
	}

	items, total, err = repo.ListTelegramUsers(TelegramUserListFilter{
		TelegramUserID: "200",
		Page:           1,
		PageSize:       10,
	})
	if err != nil {
		t.Fatalf("list by telegram id failed: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].UserID != user2.ID {
		t.Fatalf("unexpected list by telegram id: total=%d items=%+v", total, items)
	}

	items, total, err = repo.ListTelegramUsers(TelegramUserListFilter{
		CreatedFrom: ptrTime(boundAt2.Add(-time.Hour)),
		CreatedTo:   ptrTime(boundAt2.Add(time.Hour)),
		Page:        1,
		PageSize:    10,
	})
	if err != nil {
		t.Fatalf("list by created range failed: %v", err)
	}
	if total != 1 || len(items) != 1 || items[0].TelegramUsername != "bob_tg" {
		t.Fatalf("unexpected list by created range: total=%d items=%+v", total, items)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
