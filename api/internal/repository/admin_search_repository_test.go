package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupAdminSearchRepositoryTest(t *testing.T) (*GormUserRepository, *GormOrderRepository, *gorm.DB) {
	t.Helper()
	dsn := fmt.Sprintf("file:admin_search_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.UserOAuthIdentity{}, &models.Order{}, &models.OrderItem{}, &models.Fulfillment{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return NewUserRepository(db), NewOrderRepository(db), db
}

func TestUserRepositoryListSupportsOAuthKeyword(t *testing.T) {
	userRepo, _, db := setupAdminSearchRepositoryTest(t)

	user1 := models.User{Email: "telegram_6059928735@login.local", PasswordHash: "hash", DisplayName: "TG Buyer", Status: "active"}
	user2 := models.User{Email: "normal@example.com", PasswordHash: "hash", DisplayName: "Normal User", Status: "active"}
	if err := db.Create(&user1).Error; err != nil {
		t.Fatalf("create user1 failed: %v", err)
	}
	if err := db.Create(&user2).Error; err != nil {
		t.Fatalf("create user2 failed: %v", err)
	}
	if err := db.Create(&models.UserOAuthIdentity{
		UserID:         user1.ID,
		Provider:       "telegram",
		ProviderUserID: "6059928735",
		Username:       "tg_demo_user",
	}).Error; err != nil {
		t.Fatalf("create oauth identity failed: %v", err)
	}

	users, total, err := userRepo.List(UserListFilter{Page: 1, PageSize: 20, Keyword: "tg_demo_user"})
	if err != nil {
		t.Fatalf("list by username failed: %v", err)
	}
	if total != 1 || len(users) != 1 || users[0].ID != user1.ID {
		t.Fatalf("unexpected result by username: total=%d users=%+v", total, users)
	}

	users, total, err = userRepo.List(UserListFilter{Page: 1, PageSize: 20, Keyword: "6059928735"})
	if err != nil {
		t.Fatalf("list by provider user id failed: %v", err)
	}
	if total != 1 || len(users) != 1 || users[0].ID != user1.ID {
		t.Fatalf("unexpected result by provider user id: total=%d users=%+v", total, users)
	}
}

func TestOrderRepositoryListAdminSupportsUserKeyword(t *testing.T) {
	_, orderRepo, db := setupAdminSearchRepositoryTest(t)

	user := models.User{Email: "telegram_6059928735@login.local", PasswordHash: "hash", DisplayName: "TG Buyer", Status: "active"}
	if err := db.Create(&user).Error; err != nil {
		t.Fatalf("create user failed: %v", err)
	}
	if err := db.Create(&models.UserOAuthIdentity{
		UserID:         user.ID,
		Provider:       "telegram",
		ProviderUserID: "6059928735",
		Username:       "tg_demo_user",
	}).Error; err != nil {
		t.Fatalf("create oauth identity failed: %v", err)
	}

	order := models.Order{
		OrderNo:   "DJ-ORDER-TG-001",
		UserID:    user.ID,
		Status:    "paid",
		Currency:  "USD",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	if err := db.Create(&order).Error; err != nil {
		t.Fatalf("create order failed: %v", err)
	}

	rows, total, err := orderRepo.ListAdmin(OrderListFilter{Page: 1, PageSize: 20, UserKeyword: "tg_demo_user"})
	if err != nil {
		t.Fatalf("list orders by oauth username failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != order.ID {
		t.Fatalf("unexpected result by oauth username: total=%d rows=%+v", total, rows)
	}

	rows, total, err = orderRepo.ListAdmin(OrderListFilter{Page: 1, PageSize: 20, UserKeyword: "6059928735"})
	if err != nil {
		t.Fatalf("list orders by provider user id failed: %v", err)
	}
	if total != 1 || len(rows) != 1 || rows[0].ID != order.ID {
		t.Fatalf("unexpected result by provider user id: total=%d rows=%+v", total, rows)
	}
}
