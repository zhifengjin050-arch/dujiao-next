package repository

import (
	"time"

	"github.com/shopspring/decimal"
)

// Pagination ??????
type Pagination struct {
	Page     int
	PageSize int
}

// ProductListFilter ???????????
type ProductListFilter struct {
	Page              int
	PageSize          int
	CategoryID        string
	CategoryIDs       []uint
	Search            string
	FulfillmentType   string
	StockStatus       string
	LowStockThreshold int // ?????
	OnlyActive        bool
	WithCategory      bool
	UpdatedAfter      *time.Time // ?????????????
}

// PostListFilter ???????????
type PostListFilter struct {
	Page          int
	PageSize      int
	Type          string
	Search        string
	OnlyPublished bool
	OrderBy       string
}

// BannerListFilter ?? Banner ???????
type BannerListFilter struct {
	Page      int
	PageSize  int
	Position  string
	Search    string
	IsActive  *bool
	OrderBy   string
	OnlyValid bool
}

// OrderListFilter ???????????
type OrderListFilter struct {
	Page           int
	PageSize       int
	UserID         uint
	UserKeyword    string
	Status         string
	OrderNo        string
	GuestEmail     string
	ProductKeyword string
	ProductID      uint
	CategoryID     uint
	SKUID          uint
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	SortBy         string
	SortOrder      string
}

// PaymentListFilter ???????????
type PaymentListFilter struct {
	Page         int
	PageSize     int
	UserID       uint
	OrderID      uint
	ChannelID    uint
	ProviderType string
	ChannelType  string
	Status       string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	SkipCount    bool
	Lightweight  bool
}

// OrderRefundRecordListFilter ???????????????
type OrderRefundRecordListFilter struct {
	Page           int
	PageSize       int
	UserID         uint
	UserKeyword    string
	OrderNo        string
	GuestEmail     string
	ProductKeyword string
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
}

// PaymentChannelListFilter ?????????????
type PaymentChannelListFilter struct {
	Page         int
	PageSize     int
	ProviderType string
	ChannelType  string
	ActiveOnly   bool
}

// CouponUsageListFilter ????????????????
type CouponUsageListFilter struct {
	Page     int
	PageSize int
	UserID   uint
}

// UserListFilter ???????????
type UserListFilter struct {
	Page          int
	PageSize      int
	Keyword       string
	Status        string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
	LastLoginFrom *time.Time
	LastLoginTo   *time.Time
}

// WalletAccountListFilter ?????????????
type WalletAccountListFilter struct {
	Page     int
	PageSize int
	UserID   uint
}

// WalletTransactionListFilter ?????????????
type WalletTransactionListFilter struct {
	Page        int
	PageSize    int
	UserID      uint
	OrderID     uint
	Type        string
	Direction   string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// WalletRechargeListFilter ??????????????
type WalletRechargeListFilter struct {
	Page         int
	PageSize     int
	RechargeNo   string
	UserID       uint
	UserKeyword  string
	PaymentID    uint
	ChannelID    uint
	ProviderType string
	ChannelType  string
	Status       string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	PaidFrom     *time.Time
	PaidTo       *time.Time
}

// UserLoginLogListFilter ???????????????
type UserLoginLogListFilter struct {
	Page        int
	PageSize    int
	UserID      uint
	Email       string
	Status      string
	FailReason  string
	ClientIP    string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// AuthzAuditLogListFilter ???????????????
type AuthzAuditLogListFilter struct {
	Page            int
	PageSize        int
	OperatorAdminID uint
	TargetAdminID   uint
	Action          string
	Role            string
	Object          string
	Method          string
	CreatedFrom     *time.Time
	CreatedTo       *time.Time
}

// NotificationLogListFilter ???????????????
type NotificationLogListFilter struct {
	Page        int
	PageSize    int
	Channel     string
	Status      string
	EventType   string
	IsTest      *bool
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// AffiliateProfileListFilter ??????????
type AffiliateProfileListFilter struct {
	Page     int
	PageSize int
	UserID   uint
	Status   string
	Code     string
	Keyword  string
}

// AffiliateCommissionListFilter ??????????
type AffiliateCommissionListFilter struct {
	Page               int
	PageSize           int
	AffiliateProfileID uint
	OrderID            uint
	OrderNo            string
	Status             string
	Keyword            string
	CreatedFrom        *time.Time
	CreatedTo          *time.Time
}

// AffiliateWithdrawListFilter ??????????
type AffiliateWithdrawListFilter struct {
	Page               int
	PageSize           int
	AffiliateProfileID uint
	Status             string
	Keyword            string
	CreatedFrom        *time.Time
	CreatedTo          *time.Time
}

// MediaListFilter ???????????
type MediaListFilter struct {
	Page     int
	PageSize int
	Scene    string
	Search   string // ?????/?????????
}

// AffiliateProfileStatsAggregate ??????????
type AffiliateProfileStatsAggregate struct {
	ClickCount          int64
	ValidOrderCount     int64
	PendingCommission   decimal.Decimal
	AvailableCommission decimal.Decimal
	WithdrawnCommission decimal.Decimal
}
