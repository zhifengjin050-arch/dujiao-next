package repository

import (
	"errors"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// OrderRepository ????????
type OrderRepository interface {
	Create(order *models.Order, items []models.OrderItem) error
	GetByID(id uint) (*models.Order, error)
	GetByIDs(ids []uint) ([]models.Order, error)
	ResolveReceiverEmailByOrderID(orderID uint) (string, error)
	GetByIDAndUser(id uint, userID uint) (*models.Order, error)
	GetByOrderNoAndUser(orderNo string, userID uint) (*models.Order, error)
	GetAnyByOrderNoAndUser(orderNo string, userID uint) (*models.Order, error)
	GetByIDAndGuest(id uint, email, password string) (*models.Order, error)
	GetByOrderNoAndGuest(orderNo, email, password string) (*models.Order, error)
	GetAnyByOrderNoAndGuest(orderNo, email, password string) (*models.Order, error)
	GetByOrderNoPublic(orderNo string) (*models.Order, error)
	ListChildren(parentID uint) ([]models.Order, error)
	ListByUser(filter OrderListFilter) ([]models.Order, int64, error)
	ListByGuest(email, password string, page, pageSize int) ([]models.Order, int64, error)
	ListAdmin(filter OrderListFilter) ([]models.Order, int64, error)
	UpdateStatus(id uint, status string, updates map[string]interface{}) error
	CountOrderItemsByProduct(productID uint) (int64, error)
	CountPendingByUserID(userID uint) (int64, error)
	CountPendingByClientIP(clientIP string) (int64, error)
	CountPendingByGuestEmail(email string) (int64, error)
	Transaction(fn func(tx *gorm.DB) error) error
	WithTx(tx *gorm.DB) *GormOrderRepository
}

// GormOrderRepository GORM ??
type GormOrderRepository struct {
	BaseRepository
}

// NewOrderRepository ??????
func NewOrderRepository(db *gorm.DB) *GormOrderRepository {
	return &GormOrderRepository{BaseRepository: BaseRepository{db: db}}
}

// WithTx ????
func (r *GormOrderRepository) WithTx(tx *gorm.DB) *GormOrderRepository {
	if tx == nil {
		return r
	}
	return &GormOrderRepository{BaseRepository: BaseRepository{db: tx}}
}

func (r *GormOrderRepository) withChildren(query *gorm.DB) *gorm.DB {
	return query.Preload("Children").Preload("Children.Items").Preload("Children.Fulfillment")
}

// Create ????????
func (r *GormOrderRepository) Create(order *models.Order, items []models.OrderItem) error {
	if err := r.db.Create(order).Error; err != nil {
		return err
	}
	for i := range items {
		items[i].OrderID = order.ID
	}
	if len(items) > 0 {
		if err := r.db.Create(&items).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetByID ?? ID ????
func (r *GormOrderRepository) GetByID(id uint) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.First(&order, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByIDs ?? ID ????????
func (r *GormOrderRepository) GetByIDs(ids []uint) ([]models.Order, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	var orders []models.Order
	if err := r.db.Where("id IN ?", ids).Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// ResolveReceiverEmailByOrderID ???? ID ????????????
func (r *GormOrderRepository) ResolveReceiverEmailByOrderID(orderID uint) (string, error) {
	if orderID == 0 {
		return "", nil
	}

	var orderRow struct {
		UserID     uint
		GuestEmail string
	}
	if err := r.db.Model(&models.Order{}).
		Select("user_id", "guest_email").
		Where("id = ?", orderID).
		Take(&orderRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	if orderRow.UserID == 0 {
		return strings.TrimSpace(orderRow.GuestEmail), nil
	}

	var userRow struct {
		Email string
	}
	if err := r.db.Model(&models.User{}).
		Select("email").
		Where("id = ?", orderRow.UserID).
		Take(&userRow).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(userRow.Email), nil
}

// GetByIDAndUser ????????
func (r *GormOrderRepository) GetByIDAndUser(id uint, userID uint) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.Where("id = ? AND user_id = ? AND parent_id IS NULL", id, userID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}
func (r *GormOrderRepository) GetByOrderNoAndUser(orderNo string, userID uint) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.Where("order_no = ? AND user_id = ? AND parent_id IS NULL", orderNo, userID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetAnyByOrderNoAndUser ????????(???/?),?????????
func (r *GormOrderRepository) GetAnyByOrderNoAndUser(orderNo string, userID uint) (*models.Order, error) {
	var order models.Order
	query := r.db.Preload("Items").Preload("Fulfillment").Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Items").Preload("Fulfillment")
	})
	if err := query.Where("order_no = ? AND user_id = ?", orderNo, userID).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetAnyByOrderNoAndGuest ??????????(???/?),?????????
func (r *GormOrderRepository) GetAnyByOrderNoAndGuest(orderNo, email, password string) (*models.Order, error) {
	var order models.Order
	query := r.db.Preload("Items").Preload("Fulfillment").Preload("Children", func(db *gorm.DB) *gorm.DB {
		return db.Preload("Items").Preload("Fulfillment")
	})
	if err := query.Where("order_no = ? AND user_id = 0 AND guest_email = ? AND guest_password = ?", orderNo, email, password).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByIDAndGuest ????????
func (r *GormOrderRepository) GetByIDAndGuest(id uint, email, password string) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.
		Where("id = ? AND user_id = 0 AND guest_email = ? AND guest_password = ? AND parent_id IS NULL", id, email, password).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderNoAndGuest ????????(????)
func (r *GormOrderRepository) GetByOrderNoAndGuest(orderNo, email, password string) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.
		Where("order_no = ? AND user_id = 0 AND guest_email = ? AND guest_password = ? AND parent_id IS NULL", orderNo, email, password).
		First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderNoPublic ????????(??????,??????)
// ????????,????????
func (r *GormOrderRepository) GetByOrderNoPublic(orderNo string) (*models.Order, error) {
	var order models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.
		Where("order_no = ? AND parent_id IS NULL", orderNo).
		First(&order).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		var child models.Order
		childQuery := r.db.Preload("Items").Preload("Fulfillment")
		if err := childQuery.
			Where("order_no = ?", orderNo).
			First(&child).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, nil
			}
			return nil, err
		}
		if child.ParentID != nil && *child.ParentID > 0 {
			parentQuery := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
			if err := parentQuery.
				Where("id = ? AND parent_id IS NULL", *child.ParentID).
				First(&order).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return nil, nil
				}
				return nil, err
			}
			return &order, nil
		}
		return nil, nil
	}
	return &order, nil
}

// ListChildren ???????
func (r *GormOrderRepository) ListChildren(parentID uint) ([]models.Order, error) {
	var orders []models.Order
	if parentID == 0 {
		return orders, nil
	}
	if err := r.db.Preload("Items").Preload("Fulfillment").
		Where("parent_id = ?", parentID).
		Order("id asc").
		Find(&orders).Error; err != nil {
		return nil, err
	}
	return orders, nil
}

// ListAdmin ???????
func (r *GormOrderRepository) ListAdmin(filter OrderListFilter) ([]models.Order, int64, error) {
	var orders []models.Order
	query := r.db.Model(&models.Order{}).Where("parent_id IS NULL")

	if filter.UserID != 0 {
		query = query.Where("user_id = ?", filter.UserID)
	}
	if keyword := strings.TrimSpace(filter.UserKeyword); keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where(
			"user_id IN ("+
				"SELECT users.id FROM users "+
				"WHERE users.deleted_at IS NULL AND ("+
				"users.email LIKE ? OR "+
				"users.display_name LIKE ? OR "+
				"EXISTS ("+
				"SELECT 1 FROM user_oauth_identities "+
				"WHERE user_oauth_identities.user_id = users.id AND ("+
				"user_oauth_identities.provider LIKE ? OR "+
				"user_oauth_identities.provider_user_id LIKE ? OR "+
				"user_oauth_identities.username LIKE ?"+
				")"+
				")"+
				")"+
				")",
			like, like, like, like, like,
		)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OrderNo != "" {
		query = query.Where("order_no = ?", filter.OrderNo)
	}
	if filter.GuestEmail != "" {
		query = query.Where("guest_email = ?", filter.GuestEmail)
	}
	if keyword := strings.TrimSpace(filter.ProductKeyword); keyword != "" {
		like := "%" + keyword + "%"
		cond1, argCount1 := buildLocalizedLikeCondition(r.db, nil, []string{"oi.title_json"})
		cond2, argCount2 := buildLocalizedLikeCondition(r.db, nil, []string{"oi2.title_json"})
		if cond1 != "" {
			args1 := repeatLikeArgs(like, argCount1)
			args2 := repeatLikeArgs(like, argCount2)
			query = query.Where(
				"id IN (SELECT DISTINCT oi.order_id FROM order_items oi WHERE oi.order_id IN (SELECT o2.id FROM orders o2 WHERE o2.parent_id IS NULL) AND ("+cond1+")) "+
					"OR id IN (SELECT DISTINCT o3.parent_id FROM orders o3 WHERE o3.parent_id IS NOT NULL AND o3.id IN (SELECT DISTINCT oi2.order_id FROM order_items oi2 WHERE "+cond2+"))",
				append(args1, args2...)...,
			)
		}
	}
	if filter.CreatedFrom != nil {
		query = query.Where("created_at >= ?", *filter.CreatedFrom)
	}
	if filter.CreatedTo != nil {
		query = query.Where("created_at <= ?", *filter.CreatedTo)
	}
	if filter.ProductID > 0 {
		query = query.Where(
			"(id IN (SELECT oi.order_id FROM order_items oi WHERE oi.product_id = ?) "+
				"OR id IN (SELECT o3.parent_id FROM orders o3 WHERE o3.parent_id IS NOT NULL AND o3.id IN (SELECT oi2.order_id FROM order_items oi2 WHERE oi2.product_id = ?)))",
			filter.ProductID, filter.ProductID,
		)
	}
	if filter.CategoryID > 0 {
		query = query.Where(
			"(id IN (SELECT oi.order_id FROM order_items oi JOIN products p ON oi.product_id = p.id WHERE p.category_id = ?) "+
				"OR id IN (SELECT o3.parent_id FROM orders o3 WHERE o3.parent_id IS NOT NULL AND o3.id IN (SELECT oi2.order_id FROM order_items oi2 JOIN products p2 ON oi2.product_id = p2.id WHERE p2.category_id = ?)))",
			filter.CategoryID, filter.CategoryID,
		)
	}
	if filter.SKUID > 0 {
		query = query.Where(
			"(id IN (SELECT oi.order_id FROM order_items oi WHERE oi.sku_id = ?) "+
				"OR id IN (SELECT o3.parent_id FROM orders o3 WHERE o3.parent_id IS NOT NULL AND o3.id IN (SELECT oi2.order_id FROM order_items oi2 WHERE oi2.sku_id = ?)))",
			filter.SKUID, filter.SKUID,
		)
	}

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	dataQuery := applyPagination(query.Session(&gorm.Session{}), filter.Page, filter.PageSize)
	dataQuery = r.withChildren(dataQuery.Preload("Items").Preload("Fulfillment"))

	orderClause := resolveAdminOrderSort(filter.SortBy, filter.SortOrder)
	if err := dataQuery.Order(orderClause).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// resolveAdminOrderSort ??????,????? ORDER BY ???
func resolveAdminOrderSort(sortBy, sortOrder string) string {
	allowedColumns := map[string]bool{
		"created_at":   true,
		"updated_at":   true,
		"total_amount": true,
	}
	direction := "desc"
	if strings.ToLower(strings.TrimSpace(sortOrder)) == "asc" {
		direction = "asc"
	}
	col := strings.ToLower(strings.TrimSpace(sortBy))
	if !allowedColumns[col] {
		return "id " + direction
	}
	return col + " " + direction
}

// UpdateStatus ??????
func (r *GormOrderRepository) UpdateStatus(id uint, status string, updates map[string]interface{}) error {
	if updates == nil {
		updates = map[string]interface{}{}
	}
	updates["status"] = status
	return r.db.Model(&models.Order{}).Where("id = ?", id).Updates(updates).Error
}

// ListByUser ????????
func (r *GormOrderRepository) ListByUser(filter OrderListFilter) ([]models.Order, int64, error) {
	var orders []models.Order
	query := r.db.Model(&models.Order{}).Where("user_id = ? AND parent_id IS NULL", filter.UserID)

	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.OrderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+filter.OrderNo+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	query = applyPagination(query, filter.Page, filter.PageSize)

	query = r.withChildren(query.Preload("Items").Preload("Fulfillment"))
	if err := query.Order("id desc").Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// ListByGuest ????????
func (r *GormOrderRepository) ListByGuest(email, password string, page, pageSize int) ([]models.Order, int64, error) {
	var total int64
	if err := r.db.Model(&models.Order{}).
		Where("user_id = 0 AND guest_email = ? AND guest_password = ? AND parent_id IS NULL", email, password).
		Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var orders []models.Order
	query := r.withChildren(r.db.Preload("Items").Preload("Fulfillment"))
	if err := query.
		Where("user_id = 0 AND guest_email = ? AND guest_password = ? AND parent_id IS NULL", email, password).
		Order("id desc").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

// CountPendingByUserID ?????????????
func (r *GormOrderRepository) CountPendingByUserID(userID uint) (int64, error) {
	if userID == 0 {
		return 0, nil
	}
	var count int64
	if err := r.db.Model(&models.Order{}).
		Where("user_id = ? AND status = ? AND parent_id IS NULL", userID, constants.OrderStatusPendingPayment).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountPendingByClientIP ??? IP ?????????
func (r *GormOrderRepository) CountPendingByClientIP(clientIP string) (int64, error) {
	if clientIP == "" {
		return 0, nil
	}
	var count int64
	if err := r.db.Model(&models.Order{}).
		Where("client_ip = ? AND status = ? AND parent_id IS NULL", clientIP, constants.OrderStatusPendingPayment).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountPendingByGuestEmail ???????????????
func (r *GormOrderRepository) CountPendingByGuestEmail(email string) (int64, error) {
	if email == "" {
		return 0, nil
	}
	var count int64
	if err := r.db.Model(&models.Order{}).
		Where("guest_email = ? AND status = ? AND parent_id IS NULL", email, constants.OrderStatusPendingPayment).
		Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}

// CountOrderItemsByProduct ????????????
func (r *GormOrderRepository) CountOrderItemsByProduct(productID uint) (int64, error) {
	if productID == 0 {
		return 0, errors.New("invalid product id")
	}
	var count int64
	if err := r.db.Model(&models.OrderItem{}).Where("product_id = ?", productID).Count(&count).Error; err != nil {
		return 0, err
	}
	return count, nil
}
