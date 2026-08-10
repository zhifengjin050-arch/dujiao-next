package service

import (
	"strconv"
	"strings"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

type manualStockSummary struct {
	BySKU           map[uint]int
	ByProductAll    map[uint]int
	ByLegacyProduct map[uint]int
}

func summarizeManualStockItems(items []models.OrderItem) manualStockSummary {
	result := manualStockSummary{
		BySKU:           make(map[uint]int),
		ByProductAll:    make(map[uint]int),
		ByLegacyProduct: make(map[uint]int),
	}
	for _, item := range items {
		if strings.TrimSpace(item.FulfillmentType) != constants.FulfillmentTypeManual {
			continue
		}
		if item.ProductID == 0 || item.Quantity <= 0 {
			continue
		}
		result.ByProductAll[item.ProductID] += item.Quantity
		if item.SKUID > 0 {
			result.BySKU[item.SKUID] += item.Quantity
			continue
		}
		result.ByLegacyProduct[item.ProductID] += item.Quantity
	}
	return result
}

func releaseManualStockByItems(productRepo repository.ProductRepository, productSKURepo repository.ProductSKURepository, items []models.OrderItem) error {
	summary := summarizeManualStockItems(items)
	if productSKURepo != nil {
		for skuID, quantity := range summary.BySKU {
			sku, err := productSKURepo.GetByID(skuID)
			if err != nil {
				return err
			}
			if sku == nil || sku.ManualStockTotal == constants.ManualStockUnlimited {
				continue
			}
			if _, err := productSKURepo.ReleaseManualStock(skuID, quantity); err != nil {
				return err
			}
		}
	}

	productSummary := summary.ByLegacyProduct
	if productSKURepo == nil {
		productSummary = summary.ByProductAll
	}
	if productRepo == nil {
		return nil
	}
	for productID, quantity := range productSummary {
		product, err := productRepo.GetByID(strconv.FormatUint(uint64(productID), 10))
		if err != nil {
			return err
		}
		if product == nil || product.ManualStockTotal == constants.ManualStockUnlimited {
			continue
		}
		if _, err := productRepo.ReleaseManualStock(productID, quantity); err != nil {
			return err
		}
	}
	return nil
}

func consumeManualStockByItems(productRepo repository.ProductRepository, productSKURepo repository.ProductSKURepository, items []models.OrderItem) error {
	summary := summarizeManualStockItems(items)
	if productSKURepo != nil {
		for skuID, quantity := range summary.BySKU {
			sku, err := productSKURepo.GetByID(skuID)
			if err != nil {
				return err
			}
			if sku == nil || sku.ManualStockTotal == constants.ManualStockUnlimited {
				continue
			}
			if _, err := productSKURepo.ConsumeManualStock(skuID, quantity); err != nil {
				return err
			}
		}
	}

	productSummary := summary.ByLegacyProduct
	if productSKURepo == nil {
		productSummary = summary.ByProductAll
	}
	if productRepo == nil {
		return nil
	}
	for productID, quantity := range productSummary {
		product, err := productRepo.GetByID(strconv.FormatUint(uint64(productID), 10))
		if err != nil {
			return err
		}
		if product == nil || product.ManualStockTotal == constants.ManualStockUnlimited {
			continue
		}
		if _, err := productRepo.ConsumeManualStock(productID, quantity); err != nil {
			return err
		}
	}
	return nil
}
