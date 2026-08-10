package models

import (
	"time"

	"gorm.io/gorm"
)

const (
	// DefaultSKUCode ??????????????? SKU ??
	DefaultSKUCode = "DEFAULT"
)

// ProductSKU ?? SKU ?(v1:??+????)
type ProductSKU struct {
	ID                 uint           `gorm:"primarykey" json:"id"`                                                                       // ??
	ProductID          uint           `gorm:"not null;index;uniqueIndex:idx_product_sku_code" json:"product_id"`                          // ??ID
	SKUCode            string         `gorm:"column:sku_code;type:varchar(64);not null;uniqueIndex:idx_product_sku_code" json:"sku_code"` // SKU??(??????)
	SpecValuesJSON     JSON           `gorm:"type:json" json:"spec_values"`                                                               // ???(???/??)
	PriceAmount        Money          `gorm:"type:decimal(20,2);not null;default:0" json:"price_amount"`                                  // SKU??
	CostPriceAmount    Money          `gorm:"type:decimal(20,2);not null;default:0" json:"cost_price_amount"`                             // ???
	ManualStockTotal   int            `gorm:"not null;default:0" json:"manual_stock_total"`                                               // ??????(-1 ??????,>=0 ????????)
	ManualStockLocked  int            `gorm:"not null;default:0" json:"manual_stock_locked"`                                              // ???????(???)
	ManualStockSold    int            `gorm:"not null;default:0" json:"manual_stock_sold"`                                                // ???????(???????)
	AutoStockAvailable int64          `gorm:"-" json:"auto_stock_available"`                                                              // ?????????(???,??????)
	AutoStockTotal     int64          `gorm:"-" json:"auto_stock_total"`                                                                  // ????????(???,??????)
	AutoStockLocked    int64          `gorm:"-" json:"auto_stock_locked"`                                                                 // ?????????(???,??????)
	AutoStockSold      int64          `gorm:"-" json:"auto_stock_sold"`                                                                   // ?????????(???,??????)
	UpstreamStock      int            `gorm:"-" json:"upstream_stock"`                                                                    // ????(-1=??, 0=??, >0=??;???,??????)
	IsActive           bool           `gorm:"default:true;index" json:"is_active"`                                                        // ????
	SortOrder          int            `gorm:"default:0;index" json:"sort_order"`                                                          // ????
	CreatedAt          time.Time      `gorm:"index" json:"created_at"`                                                                    // ????
	UpdatedAt          time.Time      `gorm:"index" json:"updated_at"`                                                                    // ????
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`                                                                             // ?????

	Product *Product `gorm:"foreignKey:ProductID" json:"product,omitempty"` // ????
}

// TableName ????
func (ProductSKU) TableName() string {
	return "product_skus"
}
