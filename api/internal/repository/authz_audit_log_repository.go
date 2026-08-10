package repository

import (
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// AuthzAuditLogRepository ????????????
type AuthzAuditLogRepository interface {
	Create(log *models.AuthzAuditLog) error
	ListAdmin(filter AuthzAuditLogListFilter) ([]models.AuthzAuditLog, int64, error)
}

// GormAuthzAuditLogRepository GORM ??
type GormAuthzAuditLogRepository struct {
	db *gorm.DB
}

// NewAuthzAuditLogRepository ??????????
func NewAuthzAuditLogRepository(db *gorm.DB) *GormAuthzAuditLogRepository {
	return &GormAuthzAuditLogRepository{db: db}
}

// Create ????????
func (r *GormAuthzAuditLogRepository) Create(log *models.AuthzAuditLog) error {
	if log == nil {
		return nil
	}
	return r.db.Create(log).Error
}

// ListAdmin ???????????
func (r *GormAuthzAuditLogRepository) ListAdmin(filter AuthzAuditLogListFilter) ([]models.AuthzAuditLog, int64, error) {
	query := r.db.Model(&models.AuthzAuditLog{})
	if filter.OperatorAdminID != 0 {
		query = query.Where("operator_admin_id = ?", filter.OperatorAdminID)
	}
	if filter.TargetAdminID != 0 {
		query = query.Where("target_admin_id = ?", filter.TargetAdminID)
	}
	if filter.Action != "" {
		query = query.Where("action = ?", filter.Action)
	}
	if filter.Role != "" {
		query = query.Where("role = ?", filter.Role)
	}
	if filter.Object != "" {
		query = query.Where("object = ?", filter.Object)
	}
	if filter.Method != "" {
		query = query.Where("method = ?", filter.Method)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	query = applyPagination(query, filter.Page, filter.PageSize)

	logs := make([]models.AuthzAuditLog, 0)
	if err := query.Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
