package repository

import (
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// UserLoginLogRepository ????????????
type UserLoginLogRepository interface {
	Create(log *models.UserLoginLog) error
	ListAdmin(filter UserLoginLogListFilter) ([]models.UserLoginLog, int64, error)
	ListByUser(userID uint, page, pageSize int) ([]models.UserLoginLog, int64, error)
}

// GormUserLoginLogRepository GORM ??
type GormUserLoginLogRepository struct {
	db *gorm.DB
}

// NewUserLoginLogRepository ??????????
func NewUserLoginLogRepository(db *gorm.DB) *GormUserLoginLogRepository {
	return &GormUserLoginLogRepository{db: db}
}

// Create ??????
func (r *GormUserLoginLogRepository) Create(log *models.UserLoginLog) error {
	if log == nil {
		return nil
	}
	return r.db.Create(log).Error
}

// ListAdmin ?????????
func (r *GormUserLoginLogRepository) ListAdmin(filter UserLoginLogListFilter) ([]models.UserLoginLog, int64, error) {
	query := r.db.Model(&models.UserLoginLog{})
	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if filter.Email != "" {
		query = query.Where("email = ?", filter.Email)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.FailReason != "" {
		query = query.Where("fail_reason = ?", filter.FailReason)
	}
	if filter.ClientIP != "" {
		query = query.Where("client_ip = ?", filter.ClientIP)
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

	var logs []models.UserLoginLog
	if err := query.Order("id desc").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

// ListByUser ????????????
func (r *GormUserLoginLogRepository) ListByUser(userID uint, page, pageSize int) ([]models.UserLoginLog, int64, error) {
	query := r.db.Model(&models.UserLoginLog{}).Where("user_id = ?", userID)

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	var logs []models.UserLoginLog
	if err := query.Order("id desc").Limit(pageSize).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
