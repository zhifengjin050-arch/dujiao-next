package service

import "github.com/dujiao-next/internal/models"

// normalizeMaxPurchaseQuantity ??????????????
func normalizeMaxPurchaseQuantity(value int) int {
	if value <= 0 {
		return 0
	}
	return value
}

// productMaxPurchaseQuantity ????????????????
func productMaxPurchaseQuantity(product *models.Product) int {
	if product == nil {
		return 0
	}
	return normalizeMaxPurchaseQuantity(product.MaxPurchaseQuantity)
}

// validateProductPurchaseQuantity ?????????????????
func validateProductPurchaseQuantity(product *models.Product, quantity int) error {
	if quantity <= 0 {
		return ErrInvalidOrderItem
	}
	limit := productMaxPurchaseQuantity(product)
	if limit > 0 && quantity > limit {
		return ErrProductMaxPurchaseExceeded
	}
	return nil
}
