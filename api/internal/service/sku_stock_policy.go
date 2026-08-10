package service

import (
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
)

func hasMultipleActiveSKUs(product *models.Product) bool {
	if product == nil || len(product.SKUs) == 0 {
		return false
	}
	activeCount := 0
	for i := range product.SKUs {
		if !product.SKUs[i].IsActive {
			continue
		}
		activeCount++
		if activeCount > 1 {
			return true
		}
	}
	return false
}

func manualSKUAvailable(sku *models.ProductSKU) int {
	if sku == nil {
		return 0
	}
	if sku.ManualStockTotal == constants.ManualStockUnlimited {
		return int(^uint(0) >> 1)
	}
	if sku.ManualStockTotal < 0 {
		return 0
	}
	return sku.ManualStockTotal
}

func shouldEnforceManualSKUStock(product *models.Product, sku *models.ProductSKU) bool {
	if product == nil || sku == nil {
		return false
	}
	if sku.ManualStockTotal == constants.ManualStockUnlimited {
		return false
	}
	if sku.ManualStockTotal >= 0 {
		return true
	}
	if strings.ToUpper(strings.TrimSpace(sku.SKUCode)) != models.DefaultSKUCode {
		return true
	}
	return hasMultipleActiveSKUs(product)
}
