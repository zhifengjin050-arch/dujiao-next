package models

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/glebarez/sqlite" // ? Go SQLite ??(?? modernc.org/sqlite)
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

const (
	manualStockRemainingMigrationSettingKey = "migration/manual_stock_remaining_v1"
	skuMigrationSettingKey                  = "migration/product_sku_v1"
	categoryParentMigrationSettingKey       = "migration/category_parent_v1"
	manualStockUnlimitedValue               = -1
)

// DBPoolConfig ????????
type DBPoolConfig struct {
	MaxOpenConns           int
	MaxIdleConns           int
	ConnMaxLifetimeSeconds int
	ConnMaxIdleTimeSeconds int
}

// InitDB ????????
func InitDB(driver, dsn string, pool DBPoolConfig) error {
	var err error
	normalized := strings.ToLower(strings.TrimSpace(driver))
	var dialector gorm.Dialector
	switch normalized {
	case "", "sqlite":
		// glebarez/sqlite ??? modernc.org/sqlite ?? Go ??
		// ?? PRAGMA ????????? busy-spin ?? CPU ??
		dialector = sqlite.Open(appendSQLitePragmas(dsn))
	case "postgres", "postgresql":
		dialector = postgres.Open(dsn)
	default:
		return fmt.Errorf("unsupported database driver: %s", driver)
	}
	DB, err = gorm.Open(dialector, &gorm.Config{
		Logger:  logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time { return time.Now().UTC() },
	})
	if err != nil {
		return err
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	applyDBPool(sqlDB, pool)

	// SQLite ???? PRAGMA ?? WAL ????
	if normalized == "" || normalized == "sqlite" {
		DB.Exec("PRAGMA journal_mode=WAL")
		DB.Exec("PRAGMA busy_timeout=5000")
		DB.Exec("PRAGMA synchronous=NORMAL")
	}
	return nil
}

// appendSQLitePragmas ? SQLite DSN ????? PRAGMA ??
func appendSQLitePragmas(dsn string) string {
	// glebarez/sqlite ?? ?_pragma=key=value ??
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep +
		"_pragma=busy_timeout(5000)" +
		"&_pragma=journal_mode(WAL)" +
		"&_pragma=synchronous(NORMAL)"
}

func applyDBPool(sqlDB *sql.DB, pool DBPoolConfig) {
	if sqlDB == nil {
		return
	}
	if pool.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(pool.MaxOpenConns)
	}
	if pool.MaxIdleConns >= 0 {
		sqlDB.SetMaxIdleConns(pool.MaxIdleConns)
	}
	if pool.ConnMaxLifetimeSeconds > 0 {
		sqlDB.SetConnMaxLifetime(time.Duration(pool.ConnMaxLifetimeSeconds) * time.Second)
	}
	if pool.ConnMaxIdleTimeSeconds > 0 {
		sqlDB.SetConnMaxIdleTime(time.Duration(pool.ConnMaxIdleTimeSeconds) * time.Second)
	}
}

// AutoMigrate ??????????
func AutoMigrate() error {
	if err := DB.AutoMigrate(
		&Admin{},
		&User{},
		&UserOAuthIdentity{},
		&AffiliateProfile{},
		&AffiliateClick{},
		&AffiliateCommission{},
		&AffiliateWithdrawRequest{},
		&WalletAccount{},
		&WalletTransaction{},
		&WalletRechargeOrder{},
		&UserLoginLog{},
		&AuthzAuditLog{},
		&NotificationLog{},
		&EmailVerifyCode{},
		&Order{},
		&OrderItem{},
		&OrderRefundRecord{},
		&CartItem{},
		&PaymentChannel{},
		&Payment{},
		&CardSecret{},
		&CardSecretBatch{},
		&GiftCard{},
		&GiftCardBatch{},
		&Fulfillment{},
		&Coupon{},
		&CouponUsage{},
		&Promotion{},
		&Category{},
		&Product{},
		&ProductSKU{},
		&Post{},
		&Banner{},
		&Setting{},
		&ApiCredential{},
		&SiteConnection{},
		&ProductMapping{},
		&SKUMapping{},
		&ProcurementOrder{},
		&DownstreamOrderRef{},
		&ReconciliationJob{},
		&ReconciliationItem{},
		&ChannelClient{},
		&TelegramBroadcast{},
		&MemberLevel{},
		&MemberLevelPrice{},
		&Media{},
	); err != nil {
		return err
	}

	if err := migrateCartSKUUniqueIndex(); err != nil {
		return err
	}

	if err := ensureProductSKUMigration(); err != nil {
		return err
	}
	if err := ensureManualStockRemainingMigration(); err != nil {
		return err
	}
	if err := ensureCategoryParentMigration(); err != nil {
		return err
	}
	if err := SeedAdmin("admin@php.net", "123456"); err != nil {
		return err
	}

	// ???????????,????????????
	if DB.Migrator().HasColumn(&Product{}, "price_currency") {
		if err := DB.Migrator().DropColumn(&Product{}, "price_currency"); err != nil {
			return err
		}
	}
	return nil
}

// SeedAdmin ???????????
func SeedAdmin(username, password string) error {
	username = strings.TrimSpace(username)
	if username == "" || password == "" {
		return nil
	}

	var existing Admin
	if err := DB.Unscoped().Where("username = ?", username).First(&existing).Error; err == nil {
		return nil
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := Admin{
		Username:     username,
		PasswordHash: string(hash),
		IsSuper:      true,
	}
	return DB.Create(&admin).Error
}
