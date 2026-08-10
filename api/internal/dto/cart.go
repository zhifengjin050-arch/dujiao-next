package dto

import (
	"github.com/dujiao-next/internal/models"
)

// CartProductResp ???????
type CartProductResp struct {
	Slug                string             `json:"slug"`
	Title               models.JSON        `json:"title"`
	PriceAmount         models.Money       `json:"price_amount"`
	Images              models.StringArray `json:"images"`
	Tags                models.StringArray `json:"tags"`
	PurchaseType        string             `json:"purchase_type"`
	MaxPurchaseQuantity int                `json:"max_purchase_quantity"`
	FulfillmentType     string             `json:"fulfillment_type"`
	IsActive            bool               `json:"is_active"`
}

// CartItemResp ??????
type CartItemResp struct {
	ProductID       uint            `json:"product_id"`
	SKUID           uint            `json:"sku_id"`
	Quantity        int             `json:"quantity"`
	FulfillmentType string          `json:"fulfillment_type"`
	UnitPrice       models.Money    `json:"unit_price"`
	OriginalPrice   models.Money    `json:"original_price"`
	Currency        string          `json:"currency"`
	Product         CartProductResp `json:"product"`
}
