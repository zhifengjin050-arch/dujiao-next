package repository

import (
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// NotificationLogRepository ??????????
type NotificationLogRepository interface {
	Create(log *models.NotificationLog) error
	ListAdmin(filter NotificationLogListFilter) ([]models.NotificationLog, int64, error)
}

// GormNotificationLogRepository GORM ??
type GormNotificationLogRepository struct {
	db *gorm.DB
}

// NewNotificationLogRepository ????????
func NewNotificationLogRepository(db *gorm.DB) *GormNotificationLogRepository {
	return &GormNotificationLogRepository{db: db}
}

// Create ??????
func (r *GormNotificationLogRepository) Create(log *models.NotificationLog) error {
	if log == nil {
		return nil
	}
	return r.db.Create(log).Error
}

// ListAdmin ?????????
func (r *GormNotificationLogRepository) ListAdmin(filter NotificationLogListFilter) ([]models.NotificationLog, int64, error) {
	query := r.db.Model(&models.NotificationLog{})
	if filter.Channel != "" {
		query = query.Where("channel = ?", filter.Channel)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.EventType != "" {
		query = query.Where("event_type = ?", filter.EventType)
	}
	if filter.IsTest != nil {
		query = query.Where("is_test = ?", *filter.IsTest)
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

	logs := make([]models.NotificationLog, 0)
	if err := query.Order("id DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
