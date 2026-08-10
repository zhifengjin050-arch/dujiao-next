package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AffiliateRepository ??????????
type AffiliateRepository interface {
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) AffiliateRepository

	GetProfileByID(id uint) (*models.AffiliateProfile, error)
	UpdateProfileStatus(id uint, status string, updatedAt time.Time) error
	BatchUpdateProfileStatus(ids []uint, status string, updatedAt time.Time) (int64, error)
	GetProfileByUserID(userID uint) (*models.AffiliateProfile, error)
	GetProfileByCode(code string) (*models.AffiliateProfile, error)
	CreateProfile(profile *models.AffiliateProfile) error
	ListProfiles(filter AffiliateProfileListFilter) ([]models.AffiliateProfile, int64, error)

	CreateClick(click *models.AffiliateClick) error
	HasRecentClick(profileID uint, visitorKey, landingPath string, since time.Time) (bool, error)
	GetLatestActiveProfileByVisitorKey(visitorKey string, since time.Time) (*models.AffiliateProfile, error)
	CountClicksByProfile(profileID uint) (int64, error)

	GetCommissionByOrderAndProfile(orderID, profileID uint, commissionType string) (*models.AffiliateCommission, error)
	CreateCommission(commission *models.AffiliateCommission) error
	UpdateCommission(commission *models.AffiliateCommission) error
	ListCommissions(filter AffiliateCommissionListFilter) ([]models.AffiliateCommission, int64, error)
	ListCommissionsByOrder(orderID uint, statuses []string) ([]models.AffiliateCommission, error)
	ListCommissionsByOrderForUpdate(orderID uint, statuses []string) ([]models.AffiliateCommission, error)
	ListCommissionsByWithdrawIDForUpdate(withdrawID uint) ([]models.AffiliateCommission, error)
	MarkPendingCommissionsAvailable(before, now time.Time) (int64, error)
	CountValidOrdersByProfile(profileID uint) (int64, error)
	SumCommissionByProfile(profileID uint, statuses []string, unboundOnly bool) (decimal.Decimal, error)
	ListAvailableCommissionsForUpdate(profileID uint) ([]models.AffiliateCommission, error)
	BatchUpdateCommissions(ids []uint, updates map[string]interface{}) error

	CreateWithdraw(req *models.AffiliateWithdrawRequest) error
	UpdateWithdraw(req *models.AffiliateWithdrawRequest) error
	GetWithdrawByID(id uint) (*models.AffiliateWithdrawRequest, error)
	GetWithdrawByIDForUpdate(id uint) (*models.AffiliateWithdrawRequest, error)
	ListWithdraws(filter AffiliateWithdrawListFilter) ([]models.AffiliateWithdrawRequest, int64, error)
	GetProfileStatsBatch(profileIDs []uint) (map[uint]AffiliateProfileStatsAggregate, error)
}

// GormAffiliateRepository GORM ??????
type GormAffiliateRepository struct {
	BaseRepository
}

// NewAffiliateRepository ????????
func NewAffiliateRepository(db *gorm.DB) *GormAffiliateRepository {
	return &GormAffiliateRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormAffiliateRepository) WithTx(tx *gorm.DB) AffiliateRepository {
	if tx == nil {
		return r
	}
	return &GormAffiliateRepository{BaseRepository: BaseRepository{db: tx}}
}

// GetProfileByID ?ID??????
func (r *GormAffiliateRepository) GetProfileByID(id uint) (*models.AffiliateProfile, error) {
	if id == 0 {
		return nil, nil
	}
	var profile models.AffiliateProfile
	if err := r.db.Preload("User").First(&profile, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// UpdateProfileStatus ????????
func (r *GormAffiliateRepository) UpdateProfileStatus(id uint, status string, updatedAt time.Time) error {
	if id == 0 {
		return nil
	}
	return r.db.Model(&models.AffiliateProfile{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     strings.TrimSpace(status),
			"updated_at": updatedAt,
		}).Error
}

// BatchUpdateProfileStatus ??????????
func (r *GormAffiliateRepository) BatchUpdateProfileStatus(ids []uint, status string, updatedAt time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	result := r.db.Model(&models.AffiliateProfile{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     strings.TrimSpace(status),
			"updated_at": updatedAt,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// GetProfileByUserID ???ID??????
func (r *GormAffiliateRepository) GetProfileByUserID(userID uint) (*models.AffiliateProfile, error) {
	if userID == 0 {
		return nil, nil
	}
	var profile models.AffiliateProfile
	if err := r.db.Preload("User").Where("user_id = ?", userID).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// GetProfileByCode ???ID??????
func (r *GormAffiliateRepository) GetProfileByCode(code string) (*models.AffiliateProfile, error) {
	normalized := strings.ToUpper(strings.TrimSpace(code))
	if normalized == "" {
		return nil, nil
	}
	var profile models.AffiliateProfile
	if err := r.db.Preload("User").Where("affiliate_code = ?", normalized).First(&profile).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// CreateProfile ??????
func (r *GormAffiliateRepository) CreateProfile(profile *models.AffiliateProfile) error {
	return r.db.Create(profile).Error
}

// ListProfiles ????????
func (r *GormAffiliateRepository) ListProfiles(filter AffiliateProfileListFilter) ([]models.AffiliateProfile, int64, error) {
	query := r.db.Model(&models.AffiliateProfile{}).Preload("User")
	if filter.UserID != 0 {
		query = query.Where("affiliate_profiles.user_id = ?", filter.UserID)
	}
	if code := strings.TrimSpace(filter.Code); code != "" {
		query = query.Where("affiliate_profiles.affiliate_code = ?", strings.ToUpper(code))
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("affiliate_profiles.status = ?", status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.
			Joins("LEFT JOIN users ON users.id = affiliate_profiles.user_id").
			Where("(users.email LIKE ? OR users.display_name LIKE ? OR affiliate_profiles.affiliate_code LIKE ?)", like, like, like)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyPagination(query, filter.Page, filter.PageSize)

	var rows []models.AffiliateProfile
	if err := query.Order("affiliate_profiles.id desc").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// CreateClick ????????
func (r *GormAffiliateRepository) CreateClick(click *models.AffiliateClick) error {
	return r.db.Create(click).Error
}

// HasRecentClick ??????????????
func (r *GormAffiliateRepository) HasRecentClick(profileID uint, visitorKey, landingPath string, since time.Time) (bool, error) {
	if profileID == 0 || strings.TrimSpace(visitorKey) == "" {
		return false, nil
	}
	query := r.db.Model(&models.AffiliateClick{}).
		Where("affiliate_profile_id = ? AND visitor_key = ? AND created_at >= ?",
			profileID,
			strings.TrimSpace(visitorKey),
			since,
		)
	if path := strings.TrimSpace(landingPath); path != "" {
		query = query.Where("landing_path = ?", path)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return false, err
	}
	return total > 0, nil
}

// GetLatestActiveProfileByVisitorKey ???????????????????
func (r *GormAffiliateRepository) GetLatestActiveProfileByVisitorKey(visitorKey string, since time.Time) (*models.AffiliateProfile, error) {
	key := strings.TrimSpace(visitorKey)
	if key == "" {
		return nil, nil
	}

	var profile models.AffiliateProfile
	err := r.db.Model(&models.AffiliateProfile{}).
		Joins("JOIN affiliate_clicks ac ON ac.affiliate_profile_id = affiliate_profiles.id").
		Where("ac.visitor_key = ? AND ac.created_at >= ? AND affiliate_profiles.status = ?",
			key,
			since,
			constants.AffiliateProfileStatusActive,
		).
		Order("ac.created_at DESC, ac.id DESC").
		Limit(1).
		Preload("User").
		First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &profile, nil
}

// CountClicksByProfile ???????
func (r *GormAffiliateRepository) CountClicksByProfile(profileID uint) (int64, error) {
	if profileID == 0 {
		return 0, nil
	}
	var total int64
	if err := r.db.Model(&models.AffiliateClick{}).Where("affiliate_profile_id = ?", profileID).Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// GetCommissionByOrderAndProfile ???????????
func (r *GormAffiliateRepository) GetCommissionByOrderAndProfile(orderID, profileID uint, commissionType string) (*models.AffiliateCommission, error) {
	if orderID == 0 || profileID == 0 {
		return nil, nil
	}
	ctype := strings.TrimSpace(commissionType)
	if ctype == "" {
		ctype = constants.AffiliateCommissionTypeOrder
	}
	var commission models.AffiliateCommission
	if err := r.db.Where("order_id = ? AND affiliate_profile_id = ? AND commission_type = ?", orderID, profileID, ctype).
		First(&commission).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &commission, nil
}

// CreateCommission ??????
func (r *GormAffiliateRepository) CreateCommission(commission *models.AffiliateCommission) error {
	return r.db.Create(commission).Error
}

// UpdateCommission ??????
func (r *GormAffiliateRepository) UpdateCommission(commission *models.AffiliateCommission) error {
	return r.db.Save(commission).Error
}

// ListCommissions ??????
func (r *GormAffiliateRepository) ListCommissions(filter AffiliateCommissionListFilter) ([]models.AffiliateCommission, int64, error) {
	query := r.db.Model(&models.AffiliateCommission{}).
		Preload("AffiliateProfile").
		Preload("AffiliateProfile.User").
		Preload("Order")
	if filter.AffiliateProfileID != 0 {
		query = query.Where("affiliate_commissions.affiliate_profile_id = ?", filter.AffiliateProfileID)
	}
	if filter.OrderID != 0 {
		query = query.Where("affiliate_commissions.order_id = ?", filter.OrderID)
	}
	if orderNo := strings.TrimSpace(filter.OrderNo); orderNo != "" {
		query = query.Joins("LEFT JOIN orders ON orders.id = affiliate_commissions.order_id").
			Where("orders.order_no LIKE ?", "%"+orderNo+"%")
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("affiliate_commissions.status = ?", status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.
			Joins("LEFT JOIN affiliate_profiles ap ON ap.id = affiliate_commissions.affiliate_profile_id").
			Joins("LEFT JOIN users u ON u.id = ap.user_id").
			Where("(u.email LIKE ? OR u.display_name LIKE ? OR ap.affiliate_code LIKE ?)", like, like, like)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("affiliate_commissions.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("affiliate_commissions.created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyPagination(query, filter.Page, filter.PageSize)

	var rows []models.AffiliateCommission
	if err := query.Order("affiliate_commissions.id desc").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// ListCommissionsByOrder ?????????
func (r *GormAffiliateRepository) ListCommissionsByOrder(orderID uint, statuses []string) ([]models.AffiliateCommission, error) {
	if orderID == 0 {
		return []models.AffiliateCommission{}, nil
	}
	query := r.db.Model(&models.AffiliateCommission{}).Where("order_id = ?", orderID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	var rows []models.AffiliateCommission
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListCommissionsByOrderForUpdate ??????????
func (r *GormAffiliateRepository) ListCommissionsByOrderForUpdate(orderID uint, statuses []string) ([]models.AffiliateCommission, error) {
	if orderID == 0 {
		return []models.AffiliateCommission{}, nil
	}
	query := r.db.Model(&models.AffiliateCommission{}).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("order_id = ?", orderID)
	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}
	var rows []models.AffiliateCommission
	if err := query.Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// ListCommissionsByWithdrawIDForUpdate ?????????????
func (r *GormAffiliateRepository) ListCommissionsByWithdrawIDForUpdate(withdrawID uint) ([]models.AffiliateCommission, error) {
	if withdrawID == 0 {
		return []models.AffiliateCommission{}, nil
	}
	var rows []models.AffiliateCommission
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("withdraw_request_id = ?", withdrawID).
		Order("id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// MarkPendingCommissionsAvailable ????????????
func (r *GormAffiliateRepository) MarkPendingCommissionsAvailable(before, now time.Time) (int64, error) {
	result := r.db.Model(&models.AffiliateCommission{}).
		Where("status = ? AND confirm_at IS NOT NULL AND confirm_at <= ? AND withdraw_request_id IS NULL",
			constants.AffiliateCommissionStatusPendingConfirm, before).
		Updates(map[string]interface{}{
			"status":       constants.AffiliateCommissionStatusAvailable,
			"available_at": now,
			"updated_at":   now,
		})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

// CountValidOrdersByProfile ???????
func (r *GormAffiliateRepository) CountValidOrdersByProfile(profileID uint) (int64, error) {
	if profileID == 0 {
		return 0, nil
	}
	var total int64
	if err := r.db.Model(&models.AffiliateCommission{}).
		Where("affiliate_profile_id = ? AND status <> ?", profileID, constants.AffiliateCommissionStatusRejected).
		Distinct("order_id").
		Count(&total).Error; err != nil {
		return 0, err
	}
	return total, nil
}

// SumCommissionByProfile ??????????
func (r *GormAffiliateRepository) SumCommissionByProfile(profileID uint, statuses []string, unboundOnly bool) (decimal.Decimal, error) {
	if profileID == 0 || len(statuses) == 0 {
		return decimal.Zero, nil
	}
	query := r.db.Model(&models.AffiliateCommission{}).
		Where("affiliate_profile_id = ? AND status IN ?", profileID, statuses)
	if unboundOnly {
		query = query.Where("withdraw_request_id IS NULL")
	}

	var row struct {
		Total decimal.Decimal `gorm:"column:total"`
	}
	if err := query.Select("COALESCE(SUM(commission_amount), 0) AS total").Scan(&row).Error; err != nil {
		return decimal.Zero, err
	}
	return row.Total.Round(2), nil
}

// ListAvailableCommissionsForUpdate ??????????
func (r *GormAffiliateRepository) ListAvailableCommissionsForUpdate(profileID uint) ([]models.AffiliateCommission, error) {
	if profileID == 0 {
		return []models.AffiliateCommission{}, nil
	}
	var rows []models.AffiliateCommission
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("affiliate_profile_id = ? AND status = ? AND withdraw_request_id IS NULL",
			profileID, constants.AffiliateCommissionStatusAvailable).
		Order("id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// BatchUpdateCommissions ????????
func (r *GormAffiliateRepository) BatchUpdateCommissions(ids []uint, updates map[string]interface{}) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Model(&models.AffiliateCommission{}).Where("id IN ?", ids).Updates(updates).Error
}

// CreateWithdraw ??????
func (r *GormAffiliateRepository) CreateWithdraw(req *models.AffiliateWithdrawRequest) error {
	return r.db.Create(req).Error
}

// UpdateWithdraw ??????
func (r *GormAffiliateRepository) UpdateWithdraw(req *models.AffiliateWithdrawRequest) error {
	return r.db.Save(req).Error
}

// GetWithdrawByID ?ID??????
func (r *GormAffiliateRepository) GetWithdrawByID(id uint) (*models.AffiliateWithdrawRequest, error) {
	if id == 0 {
		return nil, nil
	}
	var row models.AffiliateWithdrawRequest
	if err := r.db.Preload("AffiliateProfile").Preload("AffiliateProfile.User").Preload("Processor").First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// GetWithdrawByIDForUpdate ?ID????????
func (r *GormAffiliateRepository) GetWithdrawByIDForUpdate(id uint) (*models.AffiliateWithdrawRequest, error) {
	if id == 0 {
		return nil, nil
	}
	var row models.AffiliateWithdrawRequest
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

// ListWithdraws ????????
func (r *GormAffiliateRepository) ListWithdraws(filter AffiliateWithdrawListFilter) ([]models.AffiliateWithdrawRequest, int64, error) {
	query := r.db.Model(&models.AffiliateWithdrawRequest{}).
		Preload("AffiliateProfile").
		Preload("AffiliateProfile.User").
		Preload("Processor")

	if filter.AffiliateProfileID != 0 {
		query = query.Where("affiliate_withdraw_requests.affiliate_profile_id = ?", filter.AffiliateProfileID)
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		query = query.Where("affiliate_withdraw_requests.status = ?", status)
	}
	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.
			Joins("LEFT JOIN affiliate_profiles ap ON ap.id = affiliate_withdraw_requests.affiliate_profile_id").
			Joins("LEFT JOIN users u ON u.id = ap.user_id").
			Where("(u.email LIKE ? OR u.display_name LIKE ? OR ap.affiliate_code LIKE ? OR affiliate_withdraw_requests.account LIKE ?)",
				like, like, like, like)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("affiliate_withdraw_requests.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("affiliate_withdraw_requests.created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyPagination(query, filter.Page, filter.PageSize)

	var rows []models.AffiliateWithdrawRequest
	if err := query.Order("affiliate_withdraw_requests.id desc").Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	return rows, total, nil
}

// GetProfileStatsBatch ????????????
func (r *GormAffiliateRepository) GetProfileStatsBatch(profileIDs []uint) (map[uint]AffiliateProfileStatsAggregate, error) {
	result := make(map[uint]AffiliateProfileStatsAggregate, len(profileIDs))
	if len(profileIDs) == 0 {
		return result, nil
	}

	for _, id := range profileIDs {
		if id == 0 {
			continue
		}
		result[id] = AffiliateProfileStatsAggregate{
			PendingCommission:   decimal.Zero,
			AvailableCommission: decimal.Zero,
			WithdrawnCommission: decimal.Zero,
		}
	}

	var clickRows []struct {
		AffiliateProfileID uint  `gorm:"column:affiliate_profile_id"`
		Total              int64 `gorm:"column:total"`
	}
	if err := r.db.Model(&models.AffiliateClick{}).
		Select("affiliate_profile_id, COUNT(*) AS total").
		Where("affiliate_profile_id IN ?", profileIDs).
		Group("affiliate_profile_id").
		Scan(&clickRows).Error; err != nil {
		return nil, err
	}
	for _, row := range clickRows {
		item := result[row.AffiliateProfileID]
		item.ClickCount = row.Total
		result[row.AffiliateProfileID] = item
	}

	var validRows []struct {
		AffiliateProfileID uint  `gorm:"column:affiliate_profile_id"`
		Total              int64 `gorm:"column:total"`
	}
	if err := r.db.Model(&models.AffiliateCommission{}).
		Select("affiliate_profile_id, COUNT(DISTINCT order_id) AS total").
		Where("affiliate_profile_id IN ? AND status <> ?", profileIDs, constants.AffiliateCommissionStatusRejected).
		Group("affiliate_profile_id").
		Scan(&validRows).Error; err != nil {
		return nil, err
	}
	for _, row := range validRows {
		item := result[row.AffiliateProfileID]
		item.ValidOrderCount = row.Total
		result[row.AffiliateProfileID] = item
	}

	var pendingRows []struct {
		AffiliateProfileID uint            `gorm:"column:affiliate_profile_id"`
		Total              decimal.Decimal `gorm:"column:total"`
	}
	if err := r.db.Model(&models.AffiliateCommission{}).
		Select("affiliate_profile_id, COALESCE(SUM(commission_amount), 0) AS total").
		Where("affiliate_profile_id IN ? AND status = ?", profileIDs, constants.AffiliateCommissionStatusPendingConfirm).
		Group("affiliate_profile_id").
		Scan(&pendingRows).Error; err != nil {
		return nil, err
	}
	for _, row := range pendingRows {
		item := result[row.AffiliateProfileID]
		item.PendingCommission = row.Total.Round(2)
		result[row.AffiliateProfileID] = item
	}

	var availableRows []struct {
		AffiliateProfileID uint            `gorm:"column:affiliate_profile_id"`
		Total              decimal.Decimal `gorm:"column:total"`
	}
	if err := r.db.Model(&models.AffiliateCommission{}).
		Select("affiliate_profile_id, COALESCE(SUM(commission_amount), 0) AS total").
		Where("affiliate_profile_id IN ? AND status = ? AND withdraw_request_id IS NULL",
			profileIDs,
			constants.AffiliateCommissionStatusAvailable,
		).
		Group("affiliate_profile_id").
		Scan(&availableRows).Error; err != nil {
		return nil, err
	}
	for _, row := range availableRows {
		item := result[row.AffiliateProfileID]
		item.AvailableCommission = row.Total.Round(2)
		result[row.AffiliateProfileID] = item
	}

	var withdrawnRows []struct {
		AffiliateProfileID uint            `gorm:"column:affiliate_profile_id"`
		Total              decimal.Decimal `gorm:"column:total"`
	}
	if err := r.db.Model(&models.AffiliateCommission{}).
		Select("affiliate_profile_id, COALESCE(SUM(commission_amount), 0) AS total").
		Where("affiliate_profile_id IN ? AND status = ?", profileIDs, constants.AffiliateCommissionStatusWithdrawn).
		Group("affiliate_profile_id").
		Scan(&withdrawnRows).Error; err != nil {
		return nil, err
	}
	for _, row := range withdrawnRows {
		item := result[row.AffiliateProfileID]
		item.WithdrawnCommission = row.Total.Round(2)
		result[row.AffiliateProfileID] = item
	}

	return result, nil
}
