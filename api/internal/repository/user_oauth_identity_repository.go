package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// TelegramUserListFilter Telegram ???????
type TelegramUserListFilter struct {
	Page             int
	PageSize         int
	Keyword          string
	DisplayName      string
	TelegramUsername string
	TelegramUserID   string
	CreatedFrom      *time.Time
	CreatedTo        *time.Time
	UserIDs          []uint
}

// TelegramUserListItem Telegram ??????
type TelegramUserListItem struct {
	UserID           uint      `json:"user_id"`
	DisplayName      string    `json:"display_name"`
	UserEmail        string    `json:"user_email"`
	TelegramUsername string    `json:"telegram_username"`
	TelegramUserID   string    `json:"telegram_user_id"`
	BoundAt          time.Time `json:"bound_at"`
	UserCreatedAt    time.Time `json:"user_created_at"`
}

// UserOAuthIdentityRepository ?????????????
type UserOAuthIdentityRepository interface {
	GetByProviderUserID(provider, providerUserID string) (*models.UserOAuthIdentity, error)
	GetByUserProvider(userID uint, provider string) (*models.UserOAuthIdentity, error)
	ListByUserID(userID uint) ([]models.UserOAuthIdentity, error)
	ListTelegramUsers(filter TelegramUserListFilter) ([]TelegramUserListItem, int64, error)
	Create(identity *models.UserOAuthIdentity) error
	Update(identity *models.UserOAuthIdentity) error
	DeleteByID(id uint) error
	WithTx(tx *gorm.DB) *GormUserOAuthIdentityRepository
}

// GormUserOAuthIdentityRepository GORM ??
type GormUserOAuthIdentityRepository struct {
	db *gorm.DB
}

// NewUserOAuthIdentityRepository ????
func NewUserOAuthIdentityRepository(db *gorm.DB) *GormUserOAuthIdentityRepository {
	return &GormUserOAuthIdentityRepository{db: db}
}

// WithTx ????
func (r *GormUserOAuthIdentityRepository) WithTx(tx *gorm.DB) *GormUserOAuthIdentityRepository {
	if tx == nil {
		return r
	}
	return &GormUserOAuthIdentityRepository{db: tx}
}

// GetByProviderUserID ??????ID????
func (r *GormUserOAuthIdentityRepository) GetByProviderUserID(provider, providerUserID string) (*models.UserOAuthIdentity, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	providerUserID = strings.TrimSpace(providerUserID)
	if provider == "" || providerUserID == "" {
		return nil, nil
	}
	var identity models.UserOAuthIdentity
	if err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&identity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

// GetByUserProvider ????????????
func (r *GormUserOAuthIdentityRepository) GetByUserProvider(userID uint, provider string) (*models.UserOAuthIdentity, error) {
	provider = strings.ToLower(strings.TrimSpace(provider))
	if userID == 0 || provider == "" {
		return nil, nil
	}
	var identity models.UserOAuthIdentity
	if err := r.db.Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &identity, nil
}

// ListByUserID ????????????
func (r *GormUserOAuthIdentityRepository) ListByUserID(userID uint) ([]models.UserOAuthIdentity, error) {
	if userID == 0 {
		return []models.UserOAuthIdentity{}, nil
	}
	var identities []models.UserOAuthIdentity
	if err := r.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&identities).Error; err != nil {
		return nil, err
	}
	return identities, nil
}

// ListTelegramUsers ?? Telegram ???????
func (r *GormUserOAuthIdentityRepository) ListTelegramUsers(filter TelegramUserListFilter) ([]TelegramUserListItem, int64, error) {
	query := r.db.Table("user_oauth_identities").
		Select(""+
			"users.id AS user_id, "+
			"users.display_name AS display_name, "+
			"users.email AS user_email, "+
			"user_oauth_identities.username AS telegram_username, "+
			"user_oauth_identities.provider_user_id AS telegram_user_id, "+
			"user_oauth_identities.created_at AS bound_at, "+
			"users.created_at AS user_created_at").
		Joins("JOIN users ON users.id = user_oauth_identities.user_id").
		Where("user_oauth_identities.provider = ?", "telegram").
		Where("users.deleted_at IS NULL")

	if len(filter.UserIDs) > 0 {
		query = query.Where("users.id IN ?", filter.UserIDs)
	}

	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"users.display_name LIKE ? OR user_oauth_identities.username LIKE ? OR user_oauth_identities.provider_user_id LIKE ?",
			like, like, like,
		)
	}
	if value := strings.TrimSpace(filter.DisplayName); value != "" {
		query = query.Where("users.display_name LIKE ?", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.TelegramUsername); value != "" {
		query = query.Where("user_oauth_identities.username LIKE ?", "%"+value+"%")
	}
	if value := strings.TrimSpace(filter.TelegramUserID); value != "" {
		query = query.Where("user_oauth_identities.provider_user_id LIKE ?", "%"+value+"%")
	}
	if filter.CreatedFrom != nil {
		query = query.Where("user_oauth_identities.created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("user_oauth_identities.created_at <= ?", *filter.CreatedTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if filter.PageSize > 0 {
		query = applyPagination(query, filter.Page, filter.PageSize)
	}

	var items []TelegramUserListItem
	if err := query.Order("user_oauth_identities.created_at DESC").Scan(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

// Create ????
func (r *GormUserOAuthIdentityRepository) Create(identity *models.UserOAuthIdentity) error {
	if identity == nil {
		return nil
	}
	return r.db.Create(identity).Error
}

// Update ????
func (r *GormUserOAuthIdentityRepository) Update(identity *models.UserOAuthIdentity) error {
	if identity == nil {
		return nil
	}
	return r.db.Save(identity).Error
}

// DeleteByID ????
func (r *GormUserOAuthIdentityRepository) DeleteByID(id uint) error {
	if id == 0 {
		return nil
	}
	return r.db.Delete(&models.UserOAuthIdentity{}, id).Error
}
