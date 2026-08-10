package repository

import (
	"github.com/dujiao-next/internal/models"
	"gorm.io/gorm"
)

// ReconciliationJobRepository ????????
type ReconciliationJobRepository interface {
	Create(job *models.ReconciliationJob) error
	GetByID(id uint) (*models.ReconciliationJob, error)
	Update(job *models.ReconciliationJob) error
	List(filter ReconciliationJobListFilter) ([]models.ReconciliationJob, int64, error)
}

// ReconciliationItemRepository ????????
type ReconciliationItemRepository interface {
	BatchCreate(items []models.ReconciliationItem) error
	GetByID(id uint) (*models.ReconciliationItem, error)
	Update(item *models.ReconciliationItem) error
	ListByJobID(jobID uint, page, pageSize int) ([]models.ReconciliationItem, int64, error)
}

// ReconciliationJobListFilter ????????
type ReconciliationJobListFilter struct {
	Pagination
	ConnectionID uint   `form:"connection_id"`
	Status       string `form:"status"`
	Type         string `form:"type"`
}

// --- Job Repository Implementation ---

type GormReconciliationJobRepository struct {
	db *gorm.DB
}

func NewReconciliationJobRepository(db *gorm.DB) ReconciliationJobRepository {
	return &GormReconciliationJobRepository{db: db}
}

func (r *GormReconciliationJobRepository) Create(job *models.ReconciliationJob) error {
	return r.db.Create(job).Error
}

func (r *GormReconciliationJobRepository) GetByID(id uint) (*models.ReconciliationJob, error) {
	var job models.ReconciliationJob
	if err := r.db.Preload("Connection").First(&job, id).Error; err != nil {
		return nil, err
	}
	return &job, nil
}

func (r *GormReconciliationJobRepository) Update(job *models.ReconciliationJob) error {
	return r.db.Save(job).Error
}

func (r *GormReconciliationJobRepository) List(filter ReconciliationJobListFilter) ([]models.ReconciliationJob, int64, error) {
	var jobs []models.ReconciliationJob
	var total int64

	query := r.db.Model(&models.ReconciliationJob{})
	if filter.ConnectionID > 0 {
		query = query.Where("connection_id = ?", filter.ConnectionID)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page, pageSize := filter.Page, filter.PageSize
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if err := query.Preload("Connection").Order("id DESC").Offset(offset).Limit(pageSize).Find(&jobs).Error; err != nil {
		return nil, 0, err
	}

	return jobs, total, nil
}

// --- Item Repository Implementation ---

type GormReconciliationItemRepository struct {
	db *gorm.DB
}

func NewReconciliationItemRepository(db *gorm.DB) ReconciliationItemRepository {
	return &GormReconciliationItemRepository{db: db}
}

func (r *GormReconciliationItemRepository) BatchCreate(items []models.ReconciliationItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.Create(&items).Error
}

func (r *GormReconciliationItemRepository) GetByID(id uint) (*models.ReconciliationItem, error) {
	var item models.ReconciliationItem
	if err := r.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	return &item, nil
}

func (r *GormReconciliationItemRepository) Update(item *models.ReconciliationItem) error {
	return r.db.Save(item).Error
}

func (r *GormReconciliationItemRepository) ListByJobID(jobID uint, page, pageSize int) ([]models.ReconciliationItem, int64, error) {
	var items []models.ReconciliationItem
	var total int64

	query := r.db.Model(&models.ReconciliationItem{}).Where("job_id = ?", jobID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("id ASC").Offset(offset).Limit(pageSize).Find(&items).Error; err != nil {
		return nil, 0, err
	}

	return items, total, nil
}
