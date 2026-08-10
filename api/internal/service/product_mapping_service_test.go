package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/upstream"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

type failingSKUMappingRepo struct {
	err error
}

func (r *failingSKUMappingRepo) GetByID(id uint) (*models.SKUMapping, error) {
	return nil, nil
}

func (r *failingSKUMappingRepo) GetByLocalSKUID(skuID uint) (*models.SKUMapping, error) {
	return nil, nil
}

func (r *failingSKUMappingRepo) GetByMappingAndUpstreamSKUID(productMappingID, upstreamSKUID uint) (*models.SKUMapping, error) {
	return nil, nil
}

func (r *failingSKUMappingRepo) ListByProductMapping(productMappingID uint) ([]models.SKUMapping, error) {
	return nil, nil
}

func (r *failingSKUMappingRepo) ListByProductMappingIDs(productMappingIDs []uint) ([]models.SKUMapping, error) {
	return nil, nil
}

func (r *failingSKUMappingRepo) WithTx(tx *gorm.DB) repository.SKUMappingRepository {
	return r
}

func (r *failingSKUMappingRepo) Create(mapping *models.SKUMapping) error {
	return r.err
}

func (r *failingSKUMappingRepo) Update(mapping *models.SKUMapping) error {
	return nil
}

func (r *failingSKUMappingRepo) Delete(id uint) error {
	return nil
}

func (r *failingSKUMappingRepo) DeleteByProductMapping(productMappingID uint) error {
	return nil
}

func (r *failingSKUMappingRepo) BatchUpsert(mappings []models.SKUMapping) error {
	return r.err
}

func TestImportUpstreamProductRollbackWhenSKUMappingCreateFails(t *testing.T) {
	dsn := "file:product_mapping_import_rollback?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	if err := db.AutoMigrate(
		&models.Category{},
		&models.Product{},
		&models.ProductSKU{},
		&models.SiteConnection{},
		&models.ProductMapping{},
	); err != nil {
		t.Fatalf("auto migrate failed: %v", err)
	}

	categoryRepo := repository.NewCategoryRepository(db)
	if err := categoryRepo.Create(&models.Category{
		ParentID: 0,
		Slug:     "test-category",
		NameJSON: models.JSON{"zh-CN": "Test Category"},
	}); err != nil {
		t.Fatalf("create category failed: %v", err)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/upstream/products/101" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"ok": true,
			"product": upstream.UpstreamProduct{
				ID:              101,
				Title:           models.JSON{"zh-CN": "??????"},
				Description:     models.JSON{"zh-CN": "??"},
				Content:         models.JSON{"zh-CN": "??"},
				Images:          []string{},
				Tags:            []string{"tag-a"},
				PriceAmount:     "10.00",
				Currency:        "CNY",
				FulfillmentType: constants.FulfillmentTypeAuto,
				IsActive:        true,
				SKUs: []upstream.UpstreamSKU{
					{
						ID:          201,
						SKUCode:     "SKU-A",
						SpecValues:  models.JSON{"name": "A"},
						PriceAmount: "10.00",
						IsActive:    true,
					},
				},
			},
		})
	}))
	defer server.Close()

	connService := NewSiteConnectionService(repository.NewSiteConnectionRepository(db), "test-secret-key", t.TempDir())
	conn, err := connService.Create(CreateConnectionInput{
		Name:      "upstream-a",
		BaseURL:   server.URL,
		ApiKey:    "test-key",
		ApiSecret: "test-secret",
		Protocol:  constants.ConnectionProtocolDujiaoNext,
	})
	if err != nil {
		t.Fatalf("create connection failed: %v", err)
	}

	svc := NewProductMappingService(
		repository.NewProductMappingRepository(db),
		&failingSKUMappingRepo{err: errors.New("inject sku mapping failure")},
		repository.NewProductRepository(db),
		repository.NewProductSKURepository(db),
		categoryRepo,
		connService,
	)

	if _, err := svc.ImportUpstreamProduct(conn.ID, 101, 1, "rollback-slug"); err == nil {
		t.Fatalf("expected import upstream product to fail")
	}

	var productCount int64
	if err := db.Model(&models.Product{}).Count(&productCount).Error; err != nil {
		t.Fatalf("count products failed: %v", err)
	}
	if productCount != 0 {
		t.Fatalf("expected product rollback, got %d products", productCount)
	}

	var skuCount int64
	if err := db.Model(&models.ProductSKU{}).Count(&skuCount).Error; err != nil {
		t.Fatalf("count product skus failed: %v", err)
	}
	if skuCount != 0 {
		t.Fatalf("expected sku rollback, got %d skus", skuCount)
	}

	var mappingCount int64
	if err := db.Model(&models.ProductMapping{}).Count(&mappingCount).Error; err != nil {
		t.Fatalf("count product mappings failed: %v", err)
	}
	if mappingCount != 0 {
		t.Fatalf("expected mapping rollback, got %d mappings", mappingCount)
	}
}
