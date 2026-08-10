package upstream

import (
	"context"
	"fmt"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

// PingResult ??????
type PingResult struct {
	SiteName        string                 `json:"site_name"`
	ProtocolVersion string                 `json:"protocol_version"`
	UserID          uint                   `json:"user_id"`
	Balance         string                 `json:"balance"`
	Currency        string                 `json:"currency"`
	MemberLevel     map[string]interface{} `json:"member_level,omitempty"`
}

// ListProductsOpts ??????
type ListProductsOpts struct {
	Page         int
	PageSize     int
	UpdatedAfter *time.Time
}

// ProductListResult ??????
type ProductListResult struct {
	Total int               `json:"total"`
	Items []UpstreamProduct `json:"items"`
}

// UpstreamProduct ??????
type UpstreamProduct struct {
	ID               uint          `json:"id"`
	SeoMeta          models.JSON   `json:"seo_meta"`
	Title            models.JSON   `json:"title"`
	Description      models.JSON   `json:"description"`
	Content          models.JSON   `json:"content"`
	Images           []string      `json:"images"`
	Tags             []string      `json:"tags"`
	PriceAmount      string        `json:"price_amount"`
	OriginalPrice    string        `json:"original_price,omitempty"`
	MemberPrice      string        `json:"member_price,omitempty"`
	Currency         string        `json:"currency"`
	FulfillmentType  string        `json:"fulfillment_type"`
	ManualFormSchema models.JSON   `json:"manual_form_schema"`
	IsActive         bool          `json:"is_active"`
	CategoryID       uint          `json:"category_id"`
	SKUs             []UpstreamSKU `json:"skus"`
	UpdatedAt        time.Time     `json:"updated_at"`
}

// UpstreamCategory ??????
type UpstreamCategory struct {
	ID        uint        `json:"id"`
	ParentID  uint        `json:"parent_id"`
	Slug      string      `json:"slug"`
	Name      models.JSON `json:"name"`
	Icon      string      `json:"icon"`
	SortOrder int         `json:"sort_order"`
}

// CategoryListResult ??????
type CategoryListResult struct {
	Supported  bool               `json:"supported"`
	Categories []UpstreamCategory `json:"categories"`
}

// UpstreamSKU ?? SKU ??
type UpstreamSKU struct {
	ID            uint        `json:"id"`
	SKUCode       string      `json:"sku_code"`
	SpecValues    models.JSON `json:"spec_values"`
	PriceAmount   string      `json:"price_amount"`
	OriginalPrice string      `json:"original_price,omitempty"`
	MemberPrice   string      `json:"member_price,omitempty"`
	StockStatus   string      `json:"stock_status"`
	StockQuantity int         `json:"stock_quantity"` // ??????(-1=??)
	IsActive      bool        `json:"is_active"`
}

// CreateUpstreamOrderReq ?????????
type CreateUpstreamOrderReq struct {
	SKUID             uint        `json:"sku_id"`
	Quantity          int         `json:"quantity"`
	ManualFormData    models.JSON `json:"manual_form_data,omitempty"`
	DownstreamOrderNo string      `json:"downstream_order_no"`
	TraceID           string      `json:"trace_id"`
	CallbackURL       string      `json:"callback_url"`
}

// CreateUpstreamOrderResp ?????????
type CreateUpstreamOrderResp struct {
	OK           bool   `json:"ok"`
	OrderID      uint   `json:"order_id,omitempty"`
	OrderNo      string `json:"order_no,omitempty"`
	Status       string `json:"status,omitempty"`
	Amount       string `json:"amount,omitempty"`
	Currency     string `json:"currency,omitempty"`
	ErrorCode    string `json:"error_code,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// UpstreamFulfillment ??????
type UpstreamFulfillment struct {
	Type         string      `json:"type"`
	Status       string      `json:"status"`
	Payload      string      `json:"payload"`
	DeliveryData models.JSON `json:"delivery_data"`
	DeliveredAt  *time.Time  `json:"delivered_at,omitempty"`
}

// UpstreamOrderDetail ??????
type UpstreamOrderDetail struct {
	OrderID        uint                 `json:"order_id"`
	OrderNo        string               `json:"order_no"`
	Status         string               `json:"status"`
	Amount         string               `json:"amount"`
	RefundedAmount string               `json:"refunded_amount,omitempty"`
	Currency       string               `json:"currency"`
	Fulfillment    *UpstreamFulfillment `json:"fulfillment,omitempty"`
	// RefundRecords ??????(??????)
	RefundRecords []models.JSON `json:"refund_records,omitempty"`
}

// Adapter ?????????
type Adapter interface {
	// Ping ????
	Ping(ctx context.Context) (*PingResult, error)

	// ListCategories ????????
	ListCategories(ctx context.Context) (*CategoryListResult, error)

	// ListProducts ????????
	ListProducts(ctx context.Context, opts ListProductsOpts) (*ProductListResult, error)

	// GetProduct ????????
	GetProduct(ctx context.Context, productID uint) (*UpstreamProduct, error)

	// CreateOrder ?????
	CreateOrder(ctx context.Context, req CreateUpstreamOrderReq) (*CreateUpstreamOrderResp, error)

	// GetOrder ????????
	GetOrder(ctx context.Context, orderID uint) (*UpstreamOrderDetail, error)

	// CancelOrder ?????
	CancelOrder(ctx context.Context, orderID uint) error

	// DownloadImage ???????
	DownloadImage(ctx context.Context, imageURL string) (localPath string, err error)
}

// NewAdapter ???????????
func NewAdapter(conn *models.SiteConnection, uploadsDir string) (Adapter, error) {
	switch conn.Protocol {
	case constants.ConnectionProtocolDujiaoNext:
		return NewDujiaoNextAdapter(conn, uploadsDir), nil
	default:
		return nil, fmt.Errorf("unsupported protocol: %s", conn.Protocol)
	}
}
