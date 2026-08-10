package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	giftCardListStatusExpired = "expired"
)

// GiftCardListFilter ???????
type GiftCardListFilter struct {
	Code           string
	Status         string
	BatchNo        string
	RedeemedUserID uint
	CreatedFrom    *time.Time
	CreatedTo      *time.Time
	RedeemedFrom   *time.Time
	RedeemedTo     *time.Time
	ExpiresFrom    *time.Time
	ExpiresTo      *time.Time
	Page           int
	PageSize       int
}

// GiftCardRepository ???????
type GiftCardRepository interface {
	CreateBatch(batch *models.GiftCardBatch, cards []models.GiftCard) error
	GetByID(id uint) (*models.GiftCard, error)
	GetByCodeForUpdate(code string) (*models.GiftCard, error)
	List(filter GiftCardListFilter) ([]models.GiftCard, int64, error)
	ListByIDs(ids []uint) ([]models.GiftCard, error)
	Update(card *models.GiftCard) error
	Delete(id uint) error
	BatchUpdateStatus(ids []uint, status string, updatedAt time.Time) (int64, error)
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) *GormGiftCardRepository
}

// GormGiftCardRepository GORM ???????
type GormGiftCardRepository struct {
	BaseRepository
}

// NewGiftCardRepository ???????
func NewGiftCardRepository(db *gorm.DB) *GormGiftCardRepository {
	return &GormGiftCardRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormGiftCardRepository) WithTx(tx *gorm.DB) *GormGiftCardRepository {
	if tx == nil {
		return r
	}
	return &GormGiftCardRepository{BaseRepository: BaseRepository{db: tx}}
}

// CreateBatch ??????????
func (r *GormGiftCardRepository) CreateBatch(batch *models.GiftCardBatch, cards []models.GiftCard) error {
	if batch == nil {
		return errors.New("invalid gift card batch")
	}
	if err := r.db.Create(batch).Error; err != nil {
		return err
	}
	if len(cards) == 0 {
		return nil
	}
	for idx := range cards {
		cards[idx].BatchID = &batch.ID
	}
	return r.db.Create(&cards).Error
}

// GetByID ?? ID ?????
func (r *GormGiftCardRepository) GetByID(id uint) (*models.GiftCard, error) {
	if id == 0 {
		return nil, nil
	}
	var card models.GiftCard
	if err := r.db.Preload("Batch").First(&card, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &card, nil
}

// GetByCodeForUpdate ???????????
func (r *GormGiftCardRepository) GetByCodeForUpdate(code string) (*models.GiftCard, error) {
	code = strings.TrimSpace(strings.ToUpper(code))
	if code == "" {
		return nil, nil
	}
	var card models.GiftCard
	if err := r.db.Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("code = ?", code).
		First(&card).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &card, nil
}

// List ???????
func (r *GormGiftCardRepository) List(filter GiftCardListFilter) ([]models.GiftCard, int64, error) {
	query := r.db.Model(&models.GiftCard{}).Preload("Batch")
	if code := strings.TrimSpace(strings.ToUpper(filter.Code)); code != "" {
		query = query.Where("code LIKE ?", "%"+code+"%")
	}
	if status := strings.TrimSpace(filter.Status); status != "" {
		now := time.Now()
		switch status {
		case giftCardListStatusExpired:
			query = query.Where("status = ? AND expires_at IS NOT NULL AND expires_at < ?", models.GiftCardStatusActive, now)
		case models.GiftCardStatusActive:
			query = query.Where("status = ? AND (expires_at IS NULL OR expires_at >= ?)", models.GiftCardStatusActive, now)
		default:
			query = query.Where("status = ?", status)
		}
	}
	if batchNo := strings.TrimSpace(strings.ToUpper(filter.BatchNo)); batchNo != "" {
		query = query.Joins("LEFT JOIN gift_card_batches ON gift_card_batches.id = gift_cards.batch_id").
			Where("gift_card_batches.batch_no LIKE ?", "%"+batchNo+"%")
	}
	if filter.RedeemedUserID > 0 {
		query = query.Where("redeemed_user_id = ?", filter.RedeemedUserID)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.RedeemedFrom != nil {
		query = query.Where("redeemed_at >= ?", *filter.RedeemedFrom)
	}
	if filter.RedeemedTo != nil {
		query = query.Where("redeemed_at <= ?", *filter.RedeemedTo)
	}
	if filter.ExpiresFrom != nil {
		query = query.Where("expires_at >= ?", *filter.ExpiresFrom)
	}
	if filter.ExpiresTo != nil {
		query = query.Where("expires_at <= ?", *filter.ExpiresTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var cards []models.GiftCard
	if err := query.Order("id desc").Find(&cards).Error; err != nil {
		return nil, 0, err
	}
	return cards, total, nil
}

// ListByIDs ? ID ???????
func (r *GormGiftCardRepository) ListByIDs(ids []uint) ([]models.GiftCard, error) {
	if len(ids) == 0 {
		return []models.GiftCard{}, nil
	}
	var cards []models.GiftCard
	if err := r.db.Preload("Batch").Where("id IN ?", ids).Order("id asc").Find(&cards).Error; err != nil {
		return nil, err
	}
	return cards, nil
}

// Update ?????
func (r *GormGiftCardRepository) Update(card *models.GiftCard) error {
	if card == nil {
		return errors.New("invalid gift card")
	}
	return r.db.Save(card).Error
}

// Delete ?????
func (r *GormGiftCardRepository) Delete(id uint) error {
	if id == 0 {
		return nil
	}
	return r.db.Delete(&models.GiftCard{}, id).Error
}

// BatchUpdateStatus ?????????
func (r *GormGiftCardRepository) BatchUpdateStatus(ids []uint, status string, updatedAt time.Time) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now()
	}
	result := r.db.Model(&models.GiftCard{}).
		Where("id IN ? AND status <> ?", ids, models.GiftCardStatusRedeemed).
		Updates(map[string]interface{}{
			"status":     strings.TrimSpace(status),
			"updated_at": updatedAt,
		})
	return result.RowsAffected, result.Error
}
