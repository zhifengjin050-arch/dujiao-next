package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupTelegramBroadcastRepositoryTest(t *testing.T) *GormTelegramBroadcastRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:telegram_broadcast_repo_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.TelegramBroadcast{}); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}
	return NewTelegramBroadcastRepository(db)
}

func TestTelegramBroadcastRepositoryListFilters(t *testing.T) {
	repo := setupTelegramBroadcastRepositoryTest(t)
	now := time.Now().UTC().Truncate(time.Second)
	items := []models.TelegramBroadcast{
		{
			Title:         "Spring Promo",
			RecipientType: constants.TelegramBroadcastRecipientTypeAll,
			Status:        constants.TelegramBroadcastStatusCompleted,
			MessageHTML:   "<b>1</b>",
			CreatedAt:     now.Add(-2 * time.Hour),
			UpdatedAt:     now.Add(-2 * time.Hour),
		},
		{
			Title:         "VIP Users",
			RecipientType: constants.TelegramBroadcastRecipientTypeSpecific,
			Status:        constants.TelegramBroadcastStatusFailed,
			MessageHTML:   "<b>2</b>",
			CreatedAt:     now,
			UpdatedAt:     now,
		},
	}
	for i := range items {
		if err := repo.Create(&items[i]); err != nil {
			t.Fatalf("create broadcast failed: %v", err)
		}
	}

	rows, total, err := repo.List(TelegramBroadcastListFilter{
		Page:     1,
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("list broadcasts failed: %v", err)
	}
	if total != 2 || len(rows) != 1 {
		t.Fatalf("unexpected filter result: total=%d rows=%d", total, len(rows))
	}
	if rows[0].Title != "VIP Users" {
		t.Fatalf("unexpected broadcast title: %s", rows[0].Title)
	}
}
