package models

import (
	"fmt"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func setupInitDefaultAdminTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := fmt.Sprintf("file:init_default_admin_test_%d?mode=memory&cache=shared", time.Now().UnixNano())
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})
	if err != nil {
		t.Fatalf("open sqlite failed: %v", err)
	}
	previousDB := DB
	DB = db
	t.Cleanup(func() {
		DB = previousDB
	})
	if err := db.AutoMigrate(&Admin{}); err != nil {
		t.Fatalf("auto migrate admin failed: %v", err)
	}
	return db
}

func TestInitDefaultAdminMarksBootstrapAdminAsSuper(t *testing.T) {
	db := setupInitDefaultAdminTestDB(t)

	if err := InitDefaultAdmin("root-admin", "secret123"); err != nil {
		t.Fatalf("init default admin failed: %v", err)
	}

	var admin Admin
	if err := db.Where("username = ?", "root-admin").First(&admin).Error; err != nil {
		t.Fatalf("query bootstrap admin failed: %v", err)
	}
	if !admin.IsSuper {
		t.Fatalf("bootstrap admin should be super")
	}
}

func TestInitDefaultAdminRepairsExistingBootstrapAdminSuperFlag(t *testing.T) {
	db := setupInitDefaultAdminTestDB(t)

	admin := &Admin{
		Username:     "root-admin",
		PasswordHash: "hashed-password",
		IsSuper:      false,
	}
	if err := db.Create(admin).Error; err != nil {
		t.Fatalf("create bootstrap admin failed: %v", err)
	}

	if err := InitDefaultAdmin("root-admin", "ignored"); err != nil {
		t.Fatalf("repair bootstrap admin failed: %v", err)
	}

	var refreshed Admin
	if err := db.First(&refreshed, admin.ID).Error; err != nil {
		t.Fatalf("reload bootstrap admin failed: %v", err)
	}
	if !refreshed.IsSuper {
		t.Fatalf("existing bootstrap admin should be repaired to super")
	}
}
