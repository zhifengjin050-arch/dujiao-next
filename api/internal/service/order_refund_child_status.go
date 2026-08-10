package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// applyParentRefundChildStatusUpdatesTx ???????????????????
// ??:??????????????,?????????????
func applyParentRefundChildStatusUpdatesTx(tx *gorm.DB, parentOrderID uint, parentTargetStatus string, now time.Time) error {
	if tx == nil || parentOrderID == 0 {
		return nil
	}

	target := strings.ToLower(strings.TrimSpace(parentTargetStatus))
	if target != constants.OrderStatusPartiallyRefunded && target != constants.OrderStatusRefunded {
		return nil
	}

	return tx.Model(&models.Order{}).
		Where("parent_id = ? AND status <> ?", parentOrderID, target).
		Updates(map[string]interface{}{
			"status":     target,
			"updated_at": now,
		}).Error
}
