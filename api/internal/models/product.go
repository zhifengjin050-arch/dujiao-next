package models

import (
	"time"

	"gorm.io/gorm"
)

// Product ???
type Product struct {
	ID                   uint           `gorm:"primarykey" json:"id"`                                               // ??
	CategoryID           uint           `gorm:"not null;index" json:"category_id"`                                  // ??ID
	Slug                 string         `gorm:"uniqueIndex;not null" json:"slug"`                                   // ????
	SeoMetaJSON          JSON           `gorm:"type:json" json:"seo_meta"`                                          // SEO ???
	TitleJSON            JSON           `gorm:"type:json;not null" json:"title"`                                    // ?????
	DescriptionJSON      JSON           `gorm:"type:json" json:"description"`                                       // ?????
	ContentJSON          JSON           `gorm:"type:json" json:"content"`                                           // ?????(Markdown)
	InstructionsJSON     JSON           `gorm:"type:json" json:"instructions"`                                      // ??????????(???????)
	PriceAmount          Money          `gorm:"type:decimal(20,2);not null;default:0" json:"price_amount"`          // ????
	CostPriceAmount      Money          `gorm:"type:decimal(20,2);not null;default:0" json:"cost_price_amount"`     // ???(?????SKU???)
	Images               StringArray    `gorm:"type:json" json:"images"`                                            // ????
	Tags                 StringArray    `gorm:"type:json" json:"tags"`                                              // ????
	PurchaseType         string         `gorm:"type:varchar(20);not null;default:'member'" json:"purchase_type"`    // ????(guest/member)
	MaxPurchaseQuantity  int            `gorm:"not null;default:0" json:"max_purchase_quantity"`                    // ????????(0 ?????)
	FulfillmentType      string         `gorm:"type:varchar(20);not null;default:'manual'" json:"fulfillment_type"` // ????(auto/manual)
	ManualFormSchemaJSON JSON           `gorm:"type:json" json:"manual_form_schema"`                                // ?????? schema
	ManualStockTotal     int            `gorm:"not null;default:0" json:"manual_stock_total"`                       // ??????(-1 ??????,>=0 ????????)
	ManualStockLocked    int            `gorm:"not null;default:0" json:"manual_stock_locked"`                      // ???????(???)
	ManualStockSold      int            `gorm:"not null;default:0" json:"manual_stock_sold"`                        // ???????(???????)
	PaymentChannelIDs    string         `gorm:"type:text" json:"payment_channel_ids"`                               // ???????ID(JSON?????,??????)
	IsAffiliateEnabled   bool           `gorm:"not null;default:false;index" json:"is_affiliate_enabled"`           // ????????
	AutoStockAvailable   int64          `gorm:"-" json:"auto_stock_available"`                                      // ?????????(???,??????)
	AutoStockTotal       int64          `gorm:"-" json:"auto_stock_total"`                                          // ????????(???,??????)
	AutoStockLocked      int64          `gorm:"-" json:"auto_stock_locked"`                                         // ?????????(???,??????)
	AutoStockSold        int64          `gorm:"-" json:"auto_stock_sold"`                                           // ?????????(???,??????)
	IsMapped             bool           `gorm:"not null;default:false;index" json:"is_mapped"`                      // ???????
	IsActive             bool           `gorm:"default:false;index" json:"is_active"`                               // ????
	SortOrder            int            `gorm:"default:0;index" json:"sort_order"`                                  // ????
	CreatedAt            time.Time      `gorm:"index" json:"created_at"`                                            // ????
	UpdatedAt            time.Time      `json:"updated_at"`                                                         // ????
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"-"`                                                     // ?????

	// ??
	Category Category     `gorm:"foreignKey:CategoryID" json:"category,omitempty"` // ????
	SKUs     []ProductSKU `gorm:"foreignKey:ProductID" json:"skus,omitempty"`      // SKU ??
}

// TableName ????
func (Product) TableName() string {
	return "products"
}
