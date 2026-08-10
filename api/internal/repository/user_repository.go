package repository

import (
	"errors"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// UserRepository ????????
type UserRepository interface {
	GetByEmail(email string) (*models.User, error)
	GetByID(id uint) (*models.User, error)
	ListByIDs(ids []uint) ([]models.User, error)
	Create(user *models.User) error
	Update(user *models.User) error
	List(filter UserListFilter) ([]models.User, int64, error)
	BatchUpdateStatus(userIDs []uint, status string) error
	AssignDefaultMemberLevel(defaultLevelID uint) (int64, error)
}

// GormUserRepository GORM ??
type GormUserRepository struct {
	db *gorm.DB
}

// NewUserRepository ??????
func NewUserRepository(db *gorm.DB) *GormUserRepository {
	return &GormUserRepository{db: db}
}

// GetByEmail ????????
func (r *GormUserRepository) GetByEmail(email string) (*models.User, error) {
	var user models.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByID ?? ID ????
func (r *GormUserRepository) GetByID(id uint) (*models.User, error) {
	var user models.User
	if err := r.db.First(&user, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// ListByIDs ??????
func (r *GormUserRepository) ListByIDs(ids []uint) ([]models.User, error) {
	if len(ids) == 0 {
		return []models.User{}, nil
	}
	var users []models.User
	if err := r.db.Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// Create ????
func (r *GormUserRepository) Create(user *models.User) error {
	return r.db.Create(user).Error
}

// Update ????
func (r *GormUserRepository) Update(user *models.User) error {
	return r.db.Save(user).Error
}

// List ????
func (r *GormUserRepository) List(filter UserListFilter) ([]models.User, int64, error) {
	query := r.db.Model(&models.User{})

	if filter.Keyword != "" {
		like := "%" + filter.Keyword + "%"
		query = query.Where(
			"email LIKE ? OR display_name LIKE ? OR EXISTS ("+
				"SELECT 1 FROM user_oauth_identities "+
				"WHERE user_oauth_identities.user_id = users.id "+
				"AND ("+
				"user_oauth_identities.provider LIKE ? OR "+
				"user_oauth_identities.provider_user_id LIKE ? OR "+
				"user_oauth_identities.username LIKE ?"+
				")"+
				")",
			like, like, like, like, like,
		)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.LastLoginFrom != nil {
		query = query.Where("last_login_at >= ?", *filter.LastLoginFrom)
	}
	if filter.LastLoginTo != nil {
		query = query.Where("last_login_at <= ?", *filter.LastLoginTo)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	var users []models.User
	if err := query.Order("id DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

// BatchUpdateStatus ????????
func (r *GormUserRepository) BatchUpdateStatus(userIDs []uint, status string) error {
	if len(userIDs) == 0 {
		return nil
	}
	now := time.Now()
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": now,
	}
	if strings.ToLower(strings.TrimSpace(status)) == constants.UserStatusDisabled {
		updates["token_invalid_before"] = now
		updates["token_version"] = gorm.Expr("token_version + 1")
	}
	return r.db.Model(&models.User{}).Where("id IN ?", userIDs).Updates(updates).Error
}

// AssignDefaultMemberLevel ????????(member_level_id=0)???????????
func (r *GormUserRepository) AssignDefaultMemberLevel(defaultLevelID uint) (int64, error) {
	if defaultLevelID == 0 {
		return 0, nil
	}
	result := r.db.Model(&models.User{}).
		Where("member_level_id = 0 OR member_level_id IS NULL").
		Updates(map[string]interface{}{
			"member_level_id": defaultLevelID,
			"updated_at":      time.Now(),
		})
	return result.RowsAffected, result.Error
}
