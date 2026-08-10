package dto

import (
	"github.com/dujiao-next/internal/models"
)

// ProductResp ??????
type ProductResp struct {
	ID                   uint               `json:"id"`
	CategoryID           uint               `json:"category_id"`
	Slug                 string             `json:"slug"`
	Title                models.JSON        `json:"title"`
	Description          models.JSON        `json:"description"`
	Content              models.JSON        `json:"content"`
	PriceAmount          models.Money       `json:"price_amount"`
	Images               models.StringArray `json:"images"`
	Tags                 models.StringArray `json:"tags"`
	PurchaseType         string             `json:"purchase_type"`
	MaxPurchaseQuantity  int                `json:"max_purchase_quantity"`
	FulfillmentType      string             `json:"fulfillment_type"`
	ManualFormSchema     models.JSON        `json:"manual_form_schema"`
	ManualStockAvailable int                `json:"manual_stock_available"`
	AutoStockAvailable   int64              `json:"auto_stock_available"`
	StockStatus          string             `json:"stock_status"`
	IsSoldOut            bool               `json:"is_sold_out"`

	// ??????
	PaymentChannelIDs []uint `json:"payment_channel_ids,omitempty"`

	// ??
	Category CategoryResp `json:"category,omitempty"`
	SKUs     []SKUResp    `json:"skus,omitempty"`

	// ??/???
	PromotionID          *uint               `json:"promotion_id,omitempty"`
	PromotionName        string              `json:"promotion_name,omitempty"`
	PromotionType        string              `json:"promotion_type,omitempty"`
	PromotionPriceAmount *models.Money       `json:"promotion_price_amount,omitempty"`
	PromotionRules       []PromotionRuleResp `json:"promotion_rules,omitempty"`
	MemberPrices         []MemberLevelPrice  `json:"member_prices,omitempty"`
}

// SKUResp ?? SKU ????
type SKUResp struct {
	ID                 uint         `json:"id"`
	SKUCode            string       `json:"sku_code"`
	SpecValues         models.JSON  `json:"spec_values"`
	PriceAmount        models.Money `json:"price_amount"`
	ManualStockTotal   int          `json:"manual_stock_total"`
	ManualStockSold    int          `json:"manual_stock_sold"`
	AutoStockAvailable int64        `json:"auto_stock_available"`
	UpstreamStock      int          `json:"upstream_stock"`
	IsActive           bool         `json:"is_active"`

	// ??/?????
	PromotionPriceAmount *models.Money `json:"promotion_price_amount,omitempty"`
	MemberPriceAmount    *models.Money `json:"member_price_amount,omitempty"`
}

// CategoryResp ??????
type CategoryResp struct {
	ID        uint        `json:"id"`
	ParentID  uint        `json:"parent_id"`
	Slug      string      `json:"slug"`
	Name      models.JSON `json:"name"`
	Icon      string      `json:"icon,omitempty"`
	SortOrder int         `json:"sort_order"`
}

// NewCategoryResp ? models.Category ????
func NewCategoryResp(c *models.Category) CategoryResp {
	return CategoryResp{
		ID:        c.ID,
		ParentID:  c.ParentID,
		Slug:      c.Slug,
		Name:      c.NameJSON,
		Icon:      c.Icon,
		SortOrder: c.SortOrder,
	}
}

// NewCategoryRespList ????????
func NewCategoryRespList(categories []models.Category) []CategoryResp {
	result := make([]CategoryResp, 0, len(categories))
	for i := range categories {
		result = append(result, NewCategoryResp(&categories[i]))
	}
	return result
}

// PromotionRuleResp ??????
type PromotionRuleResp struct {
	ID        uint         `json:"id"`
	Name      string       `json:"name"`
	Type      string       `json:"type"`
	Value     models.Money `json:"value"`
	MinAmount models.Money `json:"min_amount"`
}

// MemberLevelPrice ??????
type MemberLevelPrice struct {
	MemberLevelID uint         `json:"member_level_id"`
	SKUID         uint         `json:"sku_id"`
	PriceAmount   models.Money `json:"price_amount"`
}

// MemberLevelResp ????????
type MemberLevelResp struct {
	ID                uint        `json:"id"`
	Name              models.JSON `json:"name"`
	Slug              string      `json:"slug"`
	Icon              string      `json:"icon"`
	DiscountRate      float64     `json:"discount_rate"`
	RechargeThreshold float64     `json:"recharge_threshold"`
	SpendThreshold    float64     `json:"spend_threshold"`
	IsDefault         bool        `json:"is_default"`
	SortOrder         int         `json:"sort_order"`
}
