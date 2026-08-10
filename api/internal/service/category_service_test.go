package service

import (
	"fmt"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/glebarez/sqlite"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

func newCategoryServiceForTest(t *testing.T) (*CategoryService, *gorm.DB) {
	t.Helper()

	dsn := fmt.Sprintf("file:category_service_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(&models.Category{}, &models.Product{}); err != nil {
		t.Fatalf("auto migrate category/product failed: %v", err)
	}

	return NewCategoryService(repository.NewCategoryRepository(db)), db
}

func createCategoryFixture(t *testing.T, db *gorm.DB, slug string, parentID uint) models.Category {
	t.Helper()

	category := models.Category{
		ParentID: parentID,
		Slug:     slug,
		NameJSON: models.JSON{
			"zh-CN": slug,
		},
	}
	if err := db.Create(&category).Error; err != nil {
		t.Fatalf("create category fixture failed: %v", err)
	}
	return category
}

func createProductFixture(t *testing.T, db *gorm.DB, categoryID uint, slug string) {
	t.Helper()

	product := models.Product{
		CategoryID:  categoryID,
		Slug:        slug,
		TitleJSON:   models.JSON{"zh-CN": slug},
		PriceAmount: models.NewMoneyFromDecimal(decimal.NewFromInt(10)),
		IsActive:    true,
	}
	if err := db.Create(&product).Error; err != nil {
		t.Fatalf("create product fixture failed: %v", err)
	}
}

func TestCategoryServiceCreateSupportsSecondLevelCategory(t *testing.T) {
	svc, db := newCategoryServiceForTest(t)
	parent := createCategoryFixture(t, db, "games", 0)

	category, err := svc.Create(CreateCategoryInput{
		ParentID: parent.ID,
		Slug:     "steam",
		NameJSON: map[string]interface{}{
			"zh-CN": "Steam",
		},
	})
	if err != nil {
		t.Fatalf("create second-level category failed: %v", err)
	}
	if category.ParentID != parent.ID {
		t.Fatalf("expected parent_id=%d, got %d", parent.ID, category.ParentID)
	}
}

func TestCategoryServiceCreateRejectsMissingOrSecondLevelParent(t *testing.T) {
	svc, db := newCategoryServiceForTest(t)
	parent := createCategoryFixture(t, db, "games", 0)
	child := createCategoryFixture(t, db, "steam", parent.ID)

	_, err := svc.Create(CreateCategoryInput{
		ParentID: 9999,
		Slug:     "missing-parent",
		NameJSON: map[string]interface{}{"zh-CN": "missing-parent"},
	})
	if err != ErrCategoryParentInvalid {
		t.Fatalf("expected ErrCategoryParentInvalid for missing parent, got %v", err)
	}

	_, err = svc.Create(CreateCategoryInput{
		ParentID: child.ID,
		Slug:     "steam-gift-card",
		NameJSON: map[string]interface{}{"zh-CN": "steam-gift-card"},
	})
	if err != ErrCategoryParentInvalid {
		t.Fatalf("expected ErrCategoryParentInvalid for second-level parent, got %v", err)
	}
}

func TestCategoryServiceUpdateRejectsInvalidParentAssignment(t *testing.T) {
	svc, db := newCategoryServiceForTest(t)
	rootA := createCategoryFixture(t, db, "games", 0)
	rootB := createCategoryFixture(t, db, "cards", 0)
	_ = createCategoryFixture(t, db, "steam", rootA.ID)

	_, err := svc.Update(fmt.Sprintf("%d", rootA.ID), CreateCategoryInput{
		ParentID: rootA.ID,
		Slug:     rootA.Slug,
		NameJSON: map[string]interface{}{"zh-CN": rootA.Slug},
	})
	if err != ErrCategoryParentInvalid {
		t.Fatalf("expected ErrCategoryParentInvalid for self parent, got %v", err)
	}

	_, err = svc.Update(fmt.Sprintf("%d", rootA.ID), CreateCategoryInput{
		ParentID: rootB.ID,
		Slug:     rootA.Slug,
		NameJSON: map[string]interface{}{"zh-CN": rootA.Slug},
	})
	if err != ErrCategoryParentInvalid {
		t.Fatalf("expected ErrCategoryParentInvalid when moving parent with children, got %v", err)
	}
}

func TestCategoryServiceDeleteRejectsCategoriesWithChildrenOrProducts(t *testing.T) {
	svc, db := newCategoryServiceForTest(t)
	parent := createCategoryFixture(t, db, "games", 0)
	child := createCategoryFixture(t, db, "steam", parent.ID)

	if err := svc.Delete(fmt.Sprintf("%d", parent.ID)); err != ErrCategoryInUse {
		t.Fatalf("expected ErrCategoryInUse for category with children, got %v", err)
	}

	createProductFixture(t, db, child.ID, "steam-product")
	if err := svc.Delete(fmt.Sprintf("%d", child.ID)); err != ErrCategoryInUse {
		t.Fatalf("expected ErrCategoryInUse for category with products, got %v", err)
	}
}

func TestCategoryServiceListSortOrderDescending(t *testing.T) {
	svc, db := newCategoryServiceForTest(t)

	high := models.Category{
		Slug:      "high",
		NameJSON:  models.JSON{"zh-CN": "high"},
		SortOrder: 100,
	}
	low := models.Category{
		Slug:      "low",
		NameJSON:  models.JSON{"zh-CN": "low"},
		SortOrder: 1,
	}
	if err := db.Create(&high).Error; err != nil {
		t.Fatalf("create high sort category failed: %v", err)
	}
	if err := db.Create(&low).Error; err != nil {
		t.Fatalf("create low sort category failed: %v", err)
	}

	rows, err := svc.List()
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
