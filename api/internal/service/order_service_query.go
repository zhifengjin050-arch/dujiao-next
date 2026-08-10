package service

import (
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
)

// BuildLocalRefundRecordsForOrder ????????????????
// ????? order_refund_records ??,???????????
func (s *OrderService) BuildLocalRefundRecordsForOrder(order *models.Order) ([]models.JSON, error) {
	recordsJSON := make([]models.JSON, 0)
	if order == nil || s.orderRefundRecordRepo == nil {
		return recordsJSON, nil
	}

	idSet := make(map[uint]struct{}, 4)
	idSet[order.ID] = struct{}{}
	if order.ParentID != nil && *order.ParentID > 0 {
		idSet[*order.ParentID] = struct{}{}
	}
	for i := range order.Children {
		if order.Children[i].ID > 0 {
			idSet[order.Children[i].ID] = struct{}{}
		}
	}

	orderIDs := make([]uint, 0, len(idSet))
	for id := range idSet {
		orderIDs = append(orderIDs, id)
	}

	records, err := s.orderRefundRecordRepo.ListByOrderIDs(orderIDs)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return recordsJSON, nil
	}

	sort.SliceStable(records, func(i, j int) bool {
		if records[i].CreatedAt.Equal(records[j].CreatedAt) {
			return records[i].ID < records[j].ID
		}
		return records[i].CreatedAt.Before(records[j].CreatedAt)
	})

	recordsJSON = make([]models.JSON, 0, len(records))
	for idx, record := range records {
		recordsJSON = append(recordsJSON, models.JSON{
			// ?????????,?????????
			"id":          idx + 1,
			"user_id":     record.UserID,
			"guest_email": record.GuestEmail,
			"order_id":    record.OrderID,
			"type":        record.Type,
			"amount":      record.Amount,
			"currency":    record.Currency,
			"remark":      record.Remark,
			"created_at":  record.CreatedAt,
			"updated_at":  record.UpdatedAt,
		})
	}
	return recordsJSON, nil
}

// ensureOrderCanceledIfExpired ????????????
func (s *OrderService) ensureOrderCanceledIfExpired(order *models.Order) error {
	if order == nil {
		return nil
	}
	if order.Status != constants.OrderStatusPendingPayment {
		return nil
	}
	if order.ExpiresAt == nil {
		return nil
	}
	if order.ExpiresAt.After(time.Now()) {
		return nil
	}
	if err := s.cancelOrderWithChildren(order, true); err != nil {
		return err
	}
	if s.queueClient != nil {
		if _, err := enqueueOrderStatusEmailTaskIfEligible(s.orderRepo, s.queueClient, s.settingService, s.defaultEmailConfig, order.ID, constants.OrderStatusCanceled); err != nil {
			logger.Warnw("order_enqueue_status_email_failed",
				"order_id", order.ID,
				"target_order_id", order.ID,
				"status", constants.OrderStatusCanceled,
				"error", err,
			)
		}
	}
	return nil
}

// ensureOrdersCanceledIfExpired ???????????
func (s *OrderService) ensureOrdersCanceledIfExpired(orders []models.Order) error {
	if len(orders) == 0 {
		return nil
	}
	for i := range orders {
		if err := s.ensureOrderCanceledIfExpired(&orders[i]); err != nil {
			return err
		}
	}
	return nil
}

// expectedRefundStatus ???????????????????????
func expectedRefundStatus(order *models.Order) string {
	if order == nil {
		return ""
	}
	if strings.ToLower(strings.TrimSpace(order.Status)) == constants.OrderStatusCanceled {
		return ""
	}
	if order.PaidAt == nil {
		return ""
	}
	total := order.TotalAmount.Decimal.Round(2)
	if total.LessThanOrEqual(decimal.Zero) {
		return ""
	}
	refunded := order.RefundedAmount.Decimal.Round(2)
	if refunded.LessThanOrEqual(decimal.Zero) {
		return ""
	}
	if refunded.GreaterThanOrEqual(total) {
		return constants.OrderStatusRefunded
	}
	return constants.OrderStatusPartiallyRefunded
}

// resolvedParentStatus ?????????????(??????)?
func resolvedParentStatus(order *models.Order) string {
	if order == nil {
		return ""
	}
	if refundStatus := expectedRefundStatus(order); refundStatus != "" {
		return refundStatus
	}
	return calcParentStatus(order.Children, order.Status)
}

// ensureSingleOrderRefundStatusSynced ????????????????
func (s *OrderService) ensureSingleOrderRefundStatusSynced(order *models.Order, now time.Time) (bool, error) {
	target := expectedRefundStatus(order)
	if target == "" || strings.EqualFold(strings.TrimSpace(order.Status), target) {
		return false, nil
	}
	if err := s.orderRepo.UpdateStatus(order.ID, target, map[string]interface{}{
		"updated_at": now,
	}); err != nil {
		return false, err
	}
	order.Status = target
	order.UpdatedAt = now
	return true, nil
}

// ensureOrderRefundStatusSynced ????????????
func (s *OrderService) ensureOrderRefundStatusSynced(order *models.Order) error {
	if order == nil {
		return nil
	}

	now := time.Now()
	orderChanged, err := s.ensureSingleOrderRefundStatusSynced(order, now)
	if err != nil {
		return err
	}

	for i := range order.Children {
		changed, childErr := s.ensureSingleOrderRefundStatusSynced(&order.Children[i], now)
		if childErr != nil {
			return childErr
		}
		if changed {
			orderChanged = true
		}
	}

	if len(order.Children) > 0 {
		parentStatus := resolvedParentStatus(order)
		if !strings.EqualFold(strings.TrimSpace(order.Status), parentStatus) {
			if err := s.orderRepo.UpdateStatus(order.ID, parentStatus, map[string]interface{}{
				"updated_at": now,
			}); err != nil {
				return err
			}
			order.Status = parentStatus
			order.UpdatedAt = now
		}
		return nil
	}

	if order.ParentID != nil && orderChanged {
		if _, err := syncParentStatus(s.orderRepo, *order.ParentID, now); err != nil {
			return err
		}
	}
	return nil
}

// ensureOrdersRefundStatusSynced ???????????
func (s *OrderService) ensureOrdersRefundStatusSynced(orders []models.Order) error {
	if len(orders) == 0 {
		return nil
	}
	for i := range orders {
		if err := s.ensureOrderRefundStatusSynced(&orders[i]); err != nil {
			return err
		}
	}
	return nil
}

// GetOrderByUser ??????
func (s *OrderService) GetOrderByUser(orderID uint, userID uint) (*models.Order, error) {
	order, err := s.orderRepo.GetByIDAndUser(orderID, userID)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

// GetOrderByUserOrderNo ????????????
func (s *OrderService) GetOrderByUserOrderNo(orderNo string, userID uint) (*models.Order, error) {
	orderNo = strings.TrimSpace(orderNo)
	if orderNo == "" {
		return nil, ErrOrderNotFound
	}
	order, err := s.orderRepo.GetByOrderNoAndUser(orderNo, userID)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

// GetOrderByGuest ????????
func (s *OrderService) GetOrderByGuest(orderID uint, email, password string) (*models.Order, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	order, err := s.orderRepo.GetByIDAndGuest(orderID, email, password)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrGuestOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

// GetOrderByGuestOrderNo ????????(????)
func (s *OrderService) GetOrderByGuestOrderNo(orderNo, email, password string) (*models.Order, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	order, err := s.orderRepo.GetByOrderNoAndGuest(orderNo, email, password)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrGuestOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

// ListOrdersByUser ??????
func (s *OrderService) ListOrdersByUser(filter repository.OrderListFilter) ([]models.Order, int64, error) {
	if filter.UserID == 0 {
		return nil, 0, ErrOrderFetchFailed
	}
	orders, total, err := s.orderRepo.ListByUser(filter)
	if err != nil {
		return nil, 0, ErrOrderFetchFailed
	}
	if err := s.ensureOrdersCanceledIfExpired(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	if err := s.ensureOrdersRefundStatusSynced(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	fillOrdersItemsFromChildren(orders)
	return orders, total, nil
}

func (s *OrderService) LookupGuestOrder(keyword string) (*models.Order, error) {
	keyword = strings.TrimSpace(keyword)
	if keyword == "" {
		return nil, ErrGuestOrderNotFound
	}
	order, err := s.orderRepo.GetByOrderNoPublic(keyword)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order != nil {
		if err := s.ensureOrderCanceledIfExpired(order); err != nil {
			return nil, ErrOrderUpdateFailed
		}
		if err := s.ensureOrderRefundStatusSynced(order); err != nil {
			return nil, ErrOrderUpdateFailed
		}
		fillOrderItemsFromChildren(order)
		return order, nil
	}
	cs, err := s.cardSecretRepo.GetBySecret(keyword)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if cs != nil && cs.OrderID != nil && *cs.OrderID > 0 {
		order, err = s.orderRepo.GetByID(*cs.OrderID)
		if err != nil {
			return nil, ErrOrderFetchFailed
		}
		if order != nil {
			if order.ParentID != nil && *order.ParentID > 0 {
				parent, parentErr := s.orderRepo.GetByID(*order.ParentID)
				if parentErr != nil {
					return nil, ErrOrderFetchFailed
				}
				if parent != nil {
					order = parent
				}
			}
			if err := s.ensureOrderCanceledIfExpired(order); err != nil {
				return nil, ErrOrderUpdateFailed
			}
			if err := s.ensureOrderRefundStatusSynced(order); err != nil {
				return nil, ErrOrderUpdateFailed
			}
			fillOrderItemsFromChildren(order)
			return order, nil
		}
	}
	payment, err := s.paymentRepo.GetByGatewayOrderNo(keyword)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if payment != nil && payment.OrderID > 0 {
		order, err = s.orderRepo.GetByID(payment.OrderID)
		if err != nil {
			return nil, ErrOrderFetchFailed
		}
		if order != nil {
			if order.ParentID != nil && *order.ParentID > 0 {
				parent, parentErr := s.orderRepo.GetByID(*order.ParentID)
				if parentErr != nil {
					return nil, ErrOrderFetchFailed
				}
				if parent != nil {
					order = parent
				}
			}
			if err := s.ensureOrderCanceledIfExpired(order); err != nil {
				return nil, ErrOrderUpdateFailed
			}
			if err := s.ensureOrderRefundStatusSynced(order); err != nil {
				return nil, ErrOrderUpdateFailed
			}
			fillOrderItemsFromChildren(order)
			return order, nil
		}
	}
	return nil, ErrGuestOrderNotFound
}

func (s *OrderService) GetOrderByOrderNoPublic(orderNo string) (*models.Order, error) {
	orderNo = strings.TrimSpace(orderNo)
	order, err := s.orderRepo.GetByOrderNoPublic(orderNo)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrGuestOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}

// ListOrdersByGuest ????????
func (s *OrderService) ListOrdersByGuest(email, password string, page, pageSize int) ([]models.Order, int64, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	orders, total, err := s.orderRepo.ListByGuest(email, password, page, pageSize)
	if err != nil {
		return nil, 0, ErrOrderFetchFailed
	}
	if err := s.ensureOrdersCanceledIfExpired(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	if err := s.ensureOrdersRefundStatusSynced(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	fillOrdersItemsFromChildren(orders)
	return orders, total, nil
}

// ListOrdersForAdmin ???????
func (s *OrderService) ListOrdersForAdmin(filter repository.OrderListFilter) ([]models.Order, int64, error) {
	orders, total, err := s.orderRepo.ListAdmin(filter)
	if err != nil {
		return nil, 0, ErrOrderFetchFailed
	}
	if err := s.ensureOrdersCanceledIfExpired(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	if err := s.ensureOrdersRefundStatusSynced(orders); err != nil {
		return nil, 0, ErrOrderUpdateFailed
	}
	fillOrdersItemsFromChildren(orders)
	return orders, total, nil
}

// GetOrderForAdmin ???????
func (s *OrderService) GetOrderForAdmin(orderID uint) (*models.Order, error) {
	if orderID == 0 {
		return nil, ErrOrderNotFound
	}
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, ErrOrderFetchFailed
	}
	if order == nil {
		return nil, ErrOrderNotFound
	}
	if err := s.ensureOrderCanceledIfExpired(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	if err := s.ensureOrderRefundStatusSynced(order); err != nil {
		return nil, ErrOrderUpdateFailed
	}
	fillOrderItemsFromChildren(order)
	return order, nil
}
