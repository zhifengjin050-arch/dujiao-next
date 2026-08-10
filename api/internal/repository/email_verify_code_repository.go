package repository

import (
	"errors"
	"time"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// EmailVerifyCodeRepository ???????????
type EmailVerifyCodeRepository interface {
	Create(code *models.EmailVerifyCode) error
	GetLatest(email, purpose string) (*models.EmailVerifyCode, error)
	MarkVerified(id uint, verifiedAt time.Time) error
	IncrementAttempt(id uint) error
}

// GormEmailVerifyCodeRepository GORM ??
type GormEmailVerifyCodeRepository struct {
	db *gorm.DB
}

// NewEmailVerifyCodeRepository ?????????
func NewEmailVerifyCodeRepository(db *gorm.DB) *GormEmailVerifyCodeRepository {
	return &GormEmailVerifyCodeRepository{db: db}
}

// Create ???????
func (r *GormEmailVerifyCodeRepository) Create(code *models.EmailVerifyCode) error {
	return r.db.Create(code).Error
}

// GetLatest ?????????
func (r *GormEmailVerifyCodeRepository) GetLatest(email, purpose string) (*models.EmailVerifyCode, error) {
	var record models.EmailVerifyCode
	if err := r.db.Where("email = ? AND purpose = ?", email, purpose).
		Order("sent_at desc, id desc").
		First(&record).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}

// MarkVerified ????????
func (r *GormEmailVerifyCodeRepository) MarkVerified(id uint, verifiedAt time.Time) error {
	return r.db.Model(&models.EmailVerifyCode{}).
		Where("id = ?", id).
		Update("verified_at", verifiedAt).Error
}

// IncrementAttempt ??????
func (r *GormEmailVerifyCodeRepository) IncrementAttempt(id uint) error {
	return r.db.Model(&models.EmailVerifyCode{}).
		Where("id = ?", id).
		UpdateColumn("attempt_count", gorm.Expr("attempt_count + 1")).Error
}
