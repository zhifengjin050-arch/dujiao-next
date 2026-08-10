package service

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"gorm.io/gorm"
)

// CardSecretService ??????
type CardSecretService struct {
	secretRepo     repository.CardSecretRepository
	batchRepo      repository.CardSecretBatchRepository
	productRepo    repository.ProductRepository
	productSKURepo repository.ProductSKURepository
}

// NewCardSecretService ????????
func NewCardSecretService(secretRepo repository.CardSecretRepository, batchRepo repository.CardSecretBatchRepository, productRepo repository.ProductRepository, productSKURepo repository.ProductSKURepository) *CardSecretService {
	return &CardSecretService{
		secretRepo:     secretRepo,
		batchRepo:      batchRepo,
		productRepo:    productRepo,
		productSKURepo: productSKURepo,
	}
}

// CreateCardSecretBatchInput ????????
type CreateCardSecretBatchInput struct {
	ProductID uint
	SKUID     uint
	Secrets   []string
	BatchNo   string
	Note      string
	Source    string
	AdminID   uint
}

// CreateCardSecretBatch ??????
func (s *CardSecretService) CreateCardSecretBatch(input CreateCardSecretBatchInput) (*models.CardSecretBatch, int, error) {
	if input.ProductID == 0 {
		return nil, 0, ErrCardSecretInvalid
	}
	if len(input.Secrets) == 0 {
		return nil, 0, ErrCardSecretInvalid
	}

	product, err := s.productRepo.GetByID(strings.TrimSpace(strconv.FormatUint(uint64(input.ProductID), 10)))
	if err != nil {
		return nil, 0, ErrProductFetchFailed
	}
	if product == nil {
		return nil, 0, ErrProductNotFound
	}
	sku, err := s.resolveCardSecretSKU(product.ID, input.SKUID)
	if err != nil {
		return nil, 0, err
	}

	normalized := normalizeSecrets(input.Secrets)
	if len(normalized) == 0 {
		return nil, 0, ErrCardSecretInvalid
	}
	if s.batchRepo == nil {
		return nil, 0, ErrCardSecretBatchCreateFailed
	}

	batchNo := strings.TrimSpace(input.BatchNo)
	if batchNo == "" {
		batchNo = generateBatchNo()
	}
	source := strings.TrimSpace(input.Source)
	if source == "" {
		source = constants.CardSecretSourceManual
	}

	now := time.Now()
	batch := &models.CardSecretBatch{
		ProductID:  input.ProductID,
		SKUID:      sku.ID,
		BatchNo:    batchNo,
		Source:     source,
		TotalCount: len(normalized),
		Note:       strings.TrimSpace(input.Note),
		CreatedAt:  now,
		UpdatedAt:  now,
	}
	if input.AdminID > 0 {
		batch.CreatedBy = &input.AdminID
	}

	err = s.secretRepo.Transaction(func(tx *gorm.DB) error {
		batchRepo := s.batchRepo.WithTx(tx)
		secretRepo := s.secretRepo.WithTx(tx)
		if err := batchRepo.Create(batch); err != nil {
			return ErrCardSecretBatchCreateFailed
		}
		items := make([]models.CardSecret, 0, len(normalized))
		for _, secret := range normalized {
			items = append(items, models.CardSecret{
				ProductID: input.ProductID,
				SKUID:     sku.ID,
				BatchID:   &batch.ID,
				Secret:    secret,
				Status:    models.CardSecretStatusAvailable,
				CreatedAt: now,
				UpdatedAt: now,
			})
		}
		if err := secretRepo.CreateBatch(items); err != nil {
			return ErrCardSecretCreateFailed
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrCardSecretBatchCreateFailed) {
			return nil, 0, ErrCardSecretBatchCreateFailed
		}
		return nil, 0, ErrCardSecretCreateFailed
	}
	return batch, batch.TotalCount, nil
}

// ImportCardSecretCSVInput ?? CSV ??
type ImportCardSecretCSVInput struct {
	ProductID uint
	SKUID     uint
	File      *multipart.FileHeader
	BatchNo   string
	Note      string
	AdminID   uint
}

// ImportCardSecretCSV ? CSV ????
func (s *CardSecretService) ImportCardSecretCSV(input ImportCardSecretCSVInput) (*models.CardSecretBatch, int, error) {
	if input.ProductID == 0 || input.File == nil {
		return nil, 0, ErrCardSecretInvalid
	}

	file, err := input.File.Open()
	if err != nil {
		return nil, 0, ErrCardSecretImportFailed
	}
	defer file.Close()

	secrets, err := parseCSVSecrets(file)
	if err != nil {
		return nil, 0, ErrCardSecretImportFailed
	}
	return s.CreateCardSecretBatch(CreateCardSecretBatchInput{
		ProductID: input.ProductID,
		SKUID:     input.SKUID,
		Secrets:   secrets,
		BatchNo:   input.BatchNo,
		Note:      input.Note,
		Source:    constants.CardSecretSourceCSV,
		AdminID:   input.AdminID,
	})
}

// ListCardSecretInput ??????
type ListCardSecretInput struct {
	ProductID   uint
	SKUID       uint
	BatchID     uint
	Status      string
	Secret      string
	BatchNo     string
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Page        int
	PageSize    int
}

// ListCardSecrets ??????
func (s *CardSecretService) ListCardSecrets(input ListCardSecretInput) ([]models.CardSecret, int64, error) {
	if input.SKUID > 0 && input.ProductID == 0 {
		return nil, 0, ErrCardSecretInvalid
	}
	if input.ProductID > 0 && input.SKUID > 0 {
		if _, err := s.resolveCardSecretSKU(input.ProductID, input.SKUID); err != nil {
			return nil, 0, err
		}
	}

	items, total, err := s.secretRepo.List(repository.CardSecretListFilter{
		ProductID:   input.ProductID,
		SKUID:       input.SKUID,
		BatchID:     input.BatchID,
		Status:      strings.TrimSpace(input.Status),
		Secret:      strings.TrimSpace(input.Secret),
		BatchNo:     strings.TrimSpace(input.BatchNo),
		CreatedFrom: input.CreatedFrom,
		CreatedTo:   input.CreatedTo,
		Page:        input.Page,
		PageSize:    input.PageSize,
	})
	if err != nil {
		return nil, 0, ErrCardSecretFetchFailed
	}
	return items, total, nil
}

func (s *CardSecretService) buildRepositoryFilter(input ListCardSecretInput) repository.CardSecretListFilter {
	return repository.CardSecretListFilter{
		ProductID: input.ProductID,
		SKUID:     input.SKUID,
		BatchID:   input.BatchID,
		Status:    strings.TrimSpace(input.Status),
		Secret:    strings.TrimSpace(input.Secret),
		BatchNo:   strings.TrimSpace(input.BatchNo),
		Page:      input.Page,
		PageSize:  input.PageSize,
	}
}

func (s *CardSecretService) hasListFilter(input ListCardSecretInput) bool {
	filter := s.buildRepositoryFilter(input)
	return filter.ProductID > 0 ||
		filter.SKUID > 0 ||
		filter.BatchID > 0 ||
		filter.Status != "" ||
		filter.Secret != "" ||
		filter.BatchNo != ""
}

// BatchUpdateCardSecretStatus ????????
func (s *CardSecretService) BatchUpdateCardSecretStatus(ids []uint, batchID uint, filter ListCardSecretInput, status string) (int64, error) {
	normalizedStatus := strings.TrimSpace(status)
	switch normalizedStatus {
	case models.CardSecretStatusAvailable, models.CardSecretStatusReserved, models.CardSecretStatusUsed:
	default:
		return 0, ErrCardSecretInvalid
	}
	normalizedIDs, err := s.resolveBatchTargetCardSecretIDs(ids, batchID, filter)
	if err != nil {
		return 0, err
	}
	rows, err := s.secretRepo.BatchUpdateStatus(normalizedIDs, normalizedStatus, time.Now())
	if err != nil {
		return 0, ErrCardSecretUpdateFailed
	}
	return rows, nil
}

// BatchDeleteCardSecrets ??????
func (s *CardSecretService) BatchDeleteCardSecrets(ids []uint, batchID uint, filter ListCardSecretInput) (int64, error) {
	normalizedIDs, err := s.resolveBatchTargetCardSecretIDs(ids, batchID, filter)
	if err != nil {
		return 0, err
	}
	rows, err := s.secretRepo.BatchDeleteByIDs(normalizedIDs)
	if err != nil {
		return 0, ErrCardSecretDeleteFailed
	}
	return rows, nil
}

// ExportCardSecrets ??????(txt/csv)
func (s *CardSecretService) ExportCardSecrets(ids []uint, batchID uint, filter ListCardSecretInput, format string) ([]byte, string, error) {
	normalizedFormat := strings.ToLower(strings.TrimSpace(format))
	switch normalizedFormat {
	case constants.ExportFormatTXT, constants.ExportFormatCSV:
	default:
		return nil, "", ErrCardSecretInvalid
	}
	normalizedIDs, err := s.resolveBatchTargetCardSecretIDs(ids, batchID, filter)
	if err != nil {
		return nil, "", err
	}

	items, err := s.secretRepo.ListByIDs(normalizedIDs)
	if err != nil {
		return nil, "", ErrCardSecretFetchFailed
	}
	if len(items) == 0 {
		return nil, "", ErrNotFound
	}

	if normalizedFormat == constants.ExportFormatTXT {
		lines := make([]string, 0, len(items))
		for _, item := range items {
			secret := strings.TrimSpace(item.Secret)
			if secret == "" {
				continue
			}
			lines = append(lines, secret)
		}
		return []byte(strings.Join(lines, "\n")), "text/plain; charset=utf-8", nil
	}

	buffer := bytes.NewBuffer(nil)
	writer := csv.NewWriter(buffer)
	header := []string{"id", "secret", "status", "product_id", "sku_id", "order_id", "batch_id", "created_at"}
	if err := writer.Write(header); err != nil {
		return nil, "", ErrCardSecretFetchFailed
	}
	for _, item := range items {
		orderID := ""
		if item.OrderID != nil {
			orderID = strconv.FormatUint(uint64(*item.OrderID), 10)
		}
		batchID := ""
		if item.BatchID != nil {
			batchID = strconv.FormatUint(uint64(*item.BatchID), 10)
		}
		row := []string{
			strconv.FormatUint(uint64(item.ID), 10),
			item.Secret,
			item.Status,
			strconv.FormatUint(uint64(item.ProductID), 10),
			strconv.FormatUint(uint64(item.SKUID), 10),
			orderID,
			batchID,
			item.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(row); err != nil {
			return nil, "", ErrCardSecretFetchFailed
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", ErrCardSecretFetchFailed
	}
	return buffer.Bytes(), "text/csv; charset=utf-8", nil
}

func (s *CardSecretService) resolveBatchTargetCardSecretIDs(ids []uint, batchID uint, filter ListCardSecretInput) ([]uint, error) {
	normalizedIDs := normalizeCardSecretIDs(ids)
	if len(normalizedIDs) > 0 {
		return normalizedIDs, nil
	}
	if s.hasListFilter(filter) {
		targetIDs, err := s.secretRepo.ListIDs(s.buildRepositoryFilter(filter))
		if err != nil {
			return nil, ErrCardSecretFetchFailed
		}
		if len(targetIDs) == 0 {
			return nil, ErrNotFound
		}
		return targetIDs, nil
	}
	if batchID == 0 {
		return nil, ErrCardSecretInvalid
	}
	targetIDs, err := s.secretRepo.ListIDsByBatchID(batchID)
	if err != nil {
		return nil, ErrCardSecretFetchFailed
	}
	if len(targetIDs) == 0 {
		return nil, ErrNotFound
	}
	return targetIDs, nil
}

// UpdateCardSecret ????
func (s *CardSecretService) UpdateCardSecret(id uint, secret, status string) (*models.CardSecret, error) {
	if id == 0 {
		return nil, ErrCardSecretInvalid
	}
	item, err := s.secretRepo.GetByID(id)
	if err != nil {
		return nil, ErrCardSecretFetchFailed
	}
	if item == nil {
		return nil, ErrNotFound
	}
	trimmedSecret := strings.TrimSpace(secret)
	if trimmedSecret != "" {
		item.Secret = trimmedSecret
	}
	trimmedStatus := strings.TrimSpace(status)
	if trimmedStatus != "" {
		switch trimmedStatus {
		case models.CardSecretStatusAvailable, models.CardSecretStatusReserved, models.CardSecretStatusUsed:
			item.Status = trimmedStatus
		default:
			return nil, ErrCardSecretInvalid
		}
	}
	item.UpdatedAt = time.Now()
	if err := s.secretRepo.Update(item); err != nil {
		return nil, ErrCardSecretUpdateFailed
	}
	return item, nil
}

// CardSecretStats ????
type CardSecretStats struct {
	Total     int64 `json:"total"`
	Available int64 `json:"available"`
	Reserved  int64 `json:"reserved"`
	Used      int64 `json:"used"`
}

// CardSecretBatchSummary ????????
type CardSecretBatchSummary struct {
	ID             uint      `json:"id"`
	ProductID      uint      `json:"product_id"`
	SKUID          uint      `json:"sku_id"`
	Name           string    `json:"name"`
	BatchNo        string    `json:"batch_no"`
	Source         string    `json:"source"`
	Note           string    `json:"note"`
	TotalCount     int64     `json:"total_count"`
	AvailableCount int64     `json:"available_count"`
	ReservedCount  int64     `json:"reserved_count"`
	UsedCount      int64     `json:"used_count"`
	CreatedAt      time.Time `json:"created_at"`
}

// GetStats ??????
func (s *CardSecretService) GetStats(productID, skuID uint) (*CardSecretStats, error) {
	if productID == 0 {
		return nil, ErrCardSecretInvalid
	}
	if skuID > 0 {
		if _, err := s.resolveCardSecretSKU(productID, skuID); err != nil {
			return nil, err
		}
	}
	total, available, used, err := s.secretRepo.CountByProduct(productID, skuID)
	if err != nil {
		return nil, ErrCardSecretStatsFailed
	}
	reserved, err := s.secretRepo.CountReserved(productID, skuID)
	if err != nil {
		return nil, ErrCardSecretStatsFailed
	}
	return &CardSecretStats{
		Total:     total,
		Available: available,
		Reserved:  reserved,
		Used:      used,
	}, nil
}

// ListBatches ??????
func (s *CardSecretService) ListBatches(productID, skuID uint, page, pageSize int) ([]CardSecretBatchSummary, int64, error) {
	if productID == 0 {
		return nil, 0, ErrCardSecretInvalid
	}
	if skuID > 0 {
		if _, err := s.resolveCardSecretSKU(productID, skuID); err != nil {
			return nil, 0, err
		}
	}
	if s.batchRepo == nil {
		return nil, 0, ErrCardSecretBatchFetchFailed
	}
	items, total, err := s.batchRepo.ListByProduct(productID, skuID, page, pageSize)
	if err != nil {
		return nil, 0, ErrCardSecretBatchFetchFailed
	}
	if len(items) == 0 {
		return []CardSecretBatchSummary{}, total, nil
	}

	batchIDs := make([]uint, 0, len(items))
	for _, item := range items {
		batchIDs = append(batchIDs, item.ID)
	}
	countRows, err := s.secretRepo.CountByBatchIDs(batchIDs)
	if err != nil {
		return nil, 0, ErrCardSecretBatchFetchFailed
	}

	type batchCounter struct {
		available int64
		reserved  int64
		used      int64
	}
	counterMap := make(map[uint]batchCounter, len(batchIDs))
	for _, row := range countRows {
		counter := counterMap[row.BatchID]
		switch row.Status {
		case models.CardSecretStatusAvailable:
			counter.available = row.Total
		case models.CardSecretStatusReserved:
			counter.reserved = row.Total
		case models.CardSecretStatusUsed:
			counter.used = row.Total
		}
		counterMap[row.BatchID] = counter
	}

	result := make([]CardSecretBatchSummary, 0, len(items))
	for _, item := range items {
		counter := counterMap[item.ID]
		result = append(result, CardSecretBatchSummary{
			ID:             item.ID,
			ProductID:      item.ProductID,
			SKUID:          item.SKUID,
			Name:           "",
			BatchNo:        item.BatchNo,
			Source:         item.Source,
			Note:           item.Note,
			TotalCount:     counter.available + counter.reserved + counter.used,
			AvailableCount: counter.available,
			ReservedCount:  counter.reserved,
			UsedCount:      counter.used,
			CreatedAt:      item.CreatedAt,
		})
	}
	return result, total, nil
}

func (s *CardSecretService) resolveCardSecretSKU(productID, rawSKUID uint) (*models.ProductSKU, error) {
	if productID == 0 || s.productSKURepo == nil {
		return nil, ErrProductSKUInvalid
	}
	product, err := s.productRepo.GetByID(strings.TrimSpace(strconv.FormatUint(uint64(productID), 10)))
	if err != nil {
		return nil, err
	}
	if product == nil {
		return nil, ErrProductNotFound
	}
	skus, err := s.productSKURepo.ListByProduct(productID, false)
	if err != nil {
		return nil, err
	}
	activeSKUs := make([]models.ProductSKU, 0, len(skus))
	for _, sku := range skus {
		if !sku.IsActive {
			continue
		}
		activeSKUs = append(activeSKUs, sku)
	}
	if rawSKUID > 0 {
		sku, err := s.productSKURepo.GetByID(rawSKUID)
		if err != nil {
			return nil, err
		}
		if sku == nil || sku.ProductID != productID {
			return nil, ErrProductSKUInvalid
		}
		if strings.TrimSpace(product.FulfillmentType) == constants.FulfillmentTypeAuto && !sku.IsActive {
			return nil, ErrProductSKUInvalid
		}
		return sku, nil
	}

	if strings.TrimSpace(product.FulfillmentType) == constants.FulfillmentTypeAuto {
		switch len(activeSKUs) {
		case 0:
		case 1:
			return &activeSKUs[0], nil
		default:
			return nil, ErrProductSKURequired
		}
	}

	defaultSKU, err := s.productSKURepo.GetByProductAndCode(productID, models.DefaultSKUCode)
	if err != nil {
		return nil, err
	}
	if defaultSKU != nil {
		return defaultSKU, nil
	}
	if len(skus) == 1 {
		return &skus[0], nil
	}
	return nil, ErrProductSKURequired
}

func normalizeSecrets(values []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, val := range values {
		for _, line := range strings.Split(val, "\n") {
			trimmed := strings.TrimSpace(line)
			if trimmed == "" {
				continue
			}
			if _, ok := seen[trimmed]; ok {
				continue
			}
			seen[trimmed] = struct{}{}
			result = append(result, trimmed)
		}
	}
	return result
}

func parseCSVSecrets(reader io.Reader) ([]string, error) {
	csvReader := csv.NewReader(reader)
	csvReader.TrimLeadingSpace = true
	var (
		secrets    []string
		headerRead bool
		secretIdx  = 0
	)
	for {
		record, err := csvReader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(record) == 0 {
			continue
		}
		if !headerRead {
			headerRead = true
			skipRow := false
			for i, col := range record {
				if strings.EqualFold(strings.TrimSpace(strings.TrimPrefix(col, "\ufeff")), "secret") {
					secretIdx = i
					skipRow = true
					break
				}
			}
			if skipRow {
				continue
			}
		}
		if secretIdx >= len(record) {
			continue
		}
		secret := strings.TrimSpace(strings.TrimPrefix(record[secretIdx], "\ufeff"))
		if secret == "" {
			continue
		}
		secrets = append(secrets, secret)
	}
	return normalizeSecrets(secrets), nil
}

func generateBatchNo() string {
	now := time.Now().Format("20060102150405")
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	return fmt.Sprintf("BATCH-%s-%04d", now, rng.Intn(10000))
}

func normalizeCardSecretIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return []uint{}
	}
	seen := make(map[uint]struct{}, len(ids))
	result := make([]uint, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
