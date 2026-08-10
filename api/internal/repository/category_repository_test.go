package repository

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupCategoryRepositoryTest(t *testing.T) *GormCategoryRepository {
	t.Helper()
	dsn := fmt.Sprintf("file:category_repository_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}); err != nil {
		t.Fatalf("migrate category failed: %v", err)
	}
	return NewCategoryRepository(db)
}

func TestCategoryRepositoryListSortOrderDescending(t *testing.T) {
	repo := setupCategoryRepositoryTest(t)

	high := &models.Category{
		Slug:      "high",
		NameJSON:  models.JSON{"zh-CN": "high"},
		SortOrder: 100,
	}
	low := &models.Category{
		Slug:      "low",
		NameJSON:  models.JSON{"zh-CN": "low"},
		SortOrder: 1,
	}
	if err := repo.Create(high); err != nil {
		t.Fatalf("create high sort category failed: %v", err)
	}
	if err := repo.Create(low); err != nil {
		t.Fatalf("create low sort category failed: %v", err)
	}

	rows, err := repo.List()
	if err != nil {
		t.Fatalf("list categories failed: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(rows))
	}
	if rows[0].Slug != "high" || rows[1].Slug != "low" {
		t.Fatalf("expected high sort_order first, got %s then %s", rows[0].Slug, rows[1].Slug)
	}
}
