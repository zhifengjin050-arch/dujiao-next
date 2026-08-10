package service

import (
	crand "crypto/rand"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

const (
	giftCardCodePrefix = "GC"
	giftCardBatchPrefx = "GCB"
)

// GiftCardService ?????
type GiftCardService struct {
	repo          repository.GiftCardRepository
	userRepo      repository.UserRepository
	walletService *WalletService
	settingSvc    *SettingService
}

// GenerateGiftCardsInput ???????
type GenerateGiftCardsInput struct {
	Name      string
	Quantity  int
	Amount    models.Money
	ExpiresAt *time.Time
	CreatedBy *uint
}

// GiftCardListInput ???????
type GiftCardListInput struct {
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

// UpdateGiftCardInput ???????
type UpdateGiftCardInput struct {
	Name           *string
	Status         *string
	ExpiresAt      *time.Time
	ClearExpiresAt bool
}

// GiftCardRedeemInput ???????
type GiftCardRedeemInput struct {
	UserID uint
	Code   string
}

// NewGiftCardService ???????
func NewGiftCardService(repo repository.GiftCardRepository, userRepo repository.UserRepository, walletService *WalletService, settingSvc *SettingService) *GiftCardService {
	return &GiftCardService{
		repo:          repo,
		userRepo:      userRepo,
		walletService: walletService,
		settingSvc:    settingSvc,
	}
}

// GenerateGiftCards ???????
func (s *GiftCardService) GenerateGiftCards(input GenerateGiftCardsInput) (*models.GiftCardBatch, int, error) {
	if s == nil || s.repo == nil {
		return nil, 0, ErrGiftCardCreateFailed
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, 0, ErrGiftCardInvalid
	}
	if input.Quantity <= 0 || input.Quantity > 10000 {
		return nil, 0, ErrGiftCardInvalid
	}
	amount := input.Amount.Decimal.Round(2)
	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, 0, ErrGiftCardInvalid
	}
	currency := resolveServiceSiteCurrency(s.settingSvc)

	now := time.Now()
	batch := &models.GiftCardBatch{
		BatchNo:   generateGiftCardBatchNo(now),
		Name:      name,
		Amount:    models.NewMoneyFromDecimal(amount),
		Currency:  currency,
		Quantity:  input.Quantity,
		ExpiresAt: normalizeGiftCardExpireAt(input.ExpiresAt),
		CreatedBy: input.CreatedBy,
		CreatedAt: now,
		UpdatedAt: now,
	}

	cards := make([]models.GiftCard, 0, input.Quantity)
	for i := 0; i < input.Quantity; i++ {
		code := generateGiftCardCode(now, i)
		cards = append(cards, models.GiftCard{
			Name:      name,
			Code:      code,
			Amount:    models.NewMoneyFromDecimal(amount),
			Currency:  currency,
			Status:    models.GiftCardStatusActive,
			ExpiresAt: normalizeGiftCardExpireAt(input.ExpiresAt),
			CreatedAt: now,
			UpdatedAt: now,
		})
	}

	if err := s.repo.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		if err := repo.CreateBatch(batch, cards); err != nil {
			return ErrGiftCardBatchCreateFailed
		}
		return nil
	}); err != nil {
		if errors.Is(err, ErrGiftCardBatchCreateFailed) {
			return nil, 0, ErrGiftCardBatchCreateFailed
		}
		return nil, 0, ErrGiftCardCreateFailed
	}

	return batch, input.Quantity, nil
}

// ListGiftCards ???????
func (s *GiftCardService) ListGiftCards(input GiftCardListInput) ([]models.GiftCard, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, ErrGiftCardFetchFailed
	}
	status := strings.TrimSpace(strings.ToLower(input.Status))
	filter := repository.GiftCardListFilter{
		Code:           strings.TrimSpace(strings.ToUpper(input.Code)),
		Status:         status,
		BatchNo:        strings.TrimSpace(strings.ToUpper(input.BatchNo)),
		RedeemedUserID: input.RedeemedUserID,
		CreatedFrom:    input.CreatedFrom,
		CreatedTo:      input.CreatedTo,
		RedeemedFrom:   input.RedeemedFrom,
		RedeemedTo:     input.RedeemedTo,
		ExpiresFrom:    input.ExpiresFrom,
		ExpiresTo:      input.ExpiresTo,
		Page:           input.Page,
		PageSize:       input.PageSize,
	}

	cards, total, err := s.repo.List(filter)
	if err != nil {
		return nil, 0, ErrGiftCardFetchFailed
	}
	return cards, total, nil
}

// UpdateGiftCard ?????
func (s *GiftCardService) UpdateGiftCard(id uint, input UpdateGiftCardInput) (*models.GiftCard, error) {
	if s == nil || s.repo == nil || id == 0 {
		return nil, ErrGiftCardInvalid
	}
	card, err := s.repo.GetByID(id)
	if err != nil {
		return nil, ErrGiftCardFetchFailed
	}
	if card == nil {
		return nil, ErrGiftCardNotFound
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, ErrGiftCardInvalid
		}
		card.Name = name
	}
	if input.Status != nil {
		status := strings.TrimSpace(strings.ToLower(*input.Status))
		switch status {
		case models.GiftCardStatusActive, models.GiftCardStatusDisabled:
			if card.Status == models.GiftCardStatusRedeemed {
				return nil, ErrGiftCardInvalid
			}
			card.Status = status
		default:
			return nil, ErrGiftCardInvalid
		}
	}
	if input.ClearExpiresAt {
		card.ExpiresAt = nil
	} else if input.ExpiresAt != nil {
		normalized := normalizeGiftCardExpireAt(input.ExpiresAt)
		if normalized != nil && normalized.Before(time.Now()) {
			return nil, ErrGiftCardInvalid
		}
		card.ExpiresAt = normalized
	}
	card.UpdatedAt = time.Now()
	if err := s.repo.Update(card); err != nil {
		return nil, ErrGiftCardUpdateFailed
	}
	return card, nil
}

// DeleteGiftCard ?????
func (s *GiftCardService) DeleteGiftCard(id uint) error {
	if s == nil || s.repo == nil || id == 0 {
		return ErrGiftCardInvalid
	}
	card, err := s.repo.GetByID(id)
	if err != nil {
		return ErrGiftCardFetchFailed
	}
	if card == nil {
		return ErrGiftCardNotFound
	}
	if card.Status == models.GiftCardStatusRedeemed {
		return ErrGiftCardInvalid
	}
	if err := s.repo.Delete(id); err != nil {
		return ErrGiftCardDeleteFailed
	}
	return nil
}

// BatchUpdateStatus ?????????
func (s *GiftCardService) BatchUpdateStatus(ids []uint, status string) (int64, error) {
	if s == nil || s.repo == nil {
		return 0, ErrGiftCardInvalid
	}
	normalizedIDs := normalizeGiftCardIDs(ids)
	if len(normalizedIDs) == 0 {
		return 0, ErrGiftCardInvalid
	}
	normalizedStatus := strings.TrimSpace(strings.ToLower(status))
	switch normalizedStatus {
	case models.GiftCardStatusActive, models.GiftCardStatusDisabled:
	default:
		return 0, ErrGiftCardInvalid
	}
	rows, err := s.repo.BatchUpdateStatus(normalizedIDs, normalizedStatus, time.Now())
	if err != nil {
		return 0, ErrGiftCardUpdateFailed
	}
	return rows, nil
}

// ExportGiftCards ?????
func (s *GiftCardService) ExportGiftCards(ids []uint, format string) ([]byte, string, error) {
	if s == nil || s.repo == nil {
		return nil, "", ErrGiftCardFetchFailed
	}
	normalizedIDs := normalizeGiftCardIDs(ids)
	if len(normalizedIDs) == 0 {
		return nil, "", ErrGiftCardInvalid
	}
	normalizedFormat := strings.TrimSpace(strings.ToLower(format))
	if normalizedFormat != constants.ExportFormatCSV && normalizedFormat != constants.ExportFormatTXT {
		return nil, "", ErrGiftCardInvalid
	}

	cards, err := s.repo.ListByIDs(normalizedIDs)
	if err != nil {
		return nil, "", ErrGiftCardFetchFailed
	}
	if len(cards) == 0 {
		return nil, "", ErrGiftCardNotFound
	}

	if normalizedFormat == constants.ExportFormatTXT {
		lines := make([]string, 0, len(cards))
		for _, card := range cards {
			lines = append(lines, strings.TrimSpace(card.Code))
		}
		return []byte(strings.Join(lines, "\n")), "text/plain; charset=utf-8", nil
	}

	builder := &strings.Builder{}
	writer := csv.NewWriter(builder)
	if err := writer.Write([]string{
		"id",
		"batch_no",
		"name",
		"code",
		"amount",
		"currency",
		"status",
		"redeemed_user_id",
		"redeemed_at",
		"expires_at",
		"created_at",
	}); err != nil {
		return nil, "", ErrGiftCardFetchFailed
	}
	for _, card := range cards {
		batchNo := ""
		if card.Batch != nil {
			batchNo = card.Batch.BatchNo
		}
		redeemedUserID := ""
		if card.RedeemedUserID != nil {
			redeemedUserID = strconv.FormatUint(uint64(*card.RedeemedUserID), 10)
		}
		record := []string{
			strconv.FormatUint(uint64(card.ID), 10),
			batchNo,
			card.Name,
			card.Code,
			card.Amount.String(),
			card.Currency,
			card.Status,
			redeemedUserID,
			formatNullableTime(card.RedeemedAt),
			formatNullableTime(card.ExpiresAt),
			card.CreatedAt.Format(time.RFC3339),
		}
		if err := writer.Write(record); err != nil {
			return nil, "", ErrGiftCardFetchFailed
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, "", ErrGiftCardFetchFailed
	}
	return []byte(builder.String()), "text/csv; charset=utf-8", nil
}

// RedeemGiftCard ?????
func (s *GiftCardService) RedeemGiftCard(input GiftCardRedeemInput) (*models.GiftCard, *models.WalletAccount, *models.WalletTransaction, error) {
	if s == nil || s.repo == nil || s.walletService == nil {
		return nil, nil, nil, ErrGiftCardFetchFailed
	}
	code := strings.TrimSpace(strings.ToUpper(input.Code))
	if input.UserID == 0 || code == "" {
		return nil, nil, nil, ErrGiftCardInvalid
	}

	var (
		resultCard  *models.GiftCard
		resultAcc   *models.WalletAccount
		resultTxn   *models.WalletTransaction
		resultError error
	)
	err := s.repo.Transaction(func(tx *gorm.DB) error {
		repo := s.repo.WithTx(tx)
		card, err := repo.GetByCodeForUpdate(code)
		if err != nil {
			return ErrGiftCardFetchFailed
		}
		if card == nil {
			return ErrGiftCardNotFound
		}
		switch card.Status {
		case models.GiftCardStatusRedeemed:
			return ErrGiftCardRedeemed
		case models.GiftCardStatusDisabled:
			return ErrGiftCardDisabled
		case models.GiftCardStatusActive:
		default:
			return ErrGiftCardInvalid
		}
		if isGiftCardExpired(card.ExpiresAt, time.Now()) {
			return ErrGiftCardExpired
		}
		if card.Amount.Decimal.Round(2).LessThanOrEqual(decimal.Zero) {
			return ErrGiftCardInvalid
		}

		now := time.Now()
		account, txn, creditErr := s.walletService.CreditInTx(tx, WalletCreditInput{
			UserID:    input.UserID,
			Amount:    card.Amount,
			Currency:  card.Currency,
			TxnType:   constants.WalletTxnTypeGiftCard,
			Reference: fmt.Sprintf("gift_card:%d", card.ID),
			Remark:    fmt.Sprintf("?????:%s", card.Code),
			OrderID:   nil,
		})
		if creditErr != nil {
			return creditErr
		}

		card.Status = models.GiftCardStatusRedeemed
		card.RedeemedUserID = &input.UserID
		card.RedeemedAt = &now
		if txn != nil && txn.ID > 0 {
			card.WalletTxnID = &txn.ID
		}
		card.UpdatedAt = now
		if err := repo.Update(card); err != nil {
			return ErrGiftCardUpdateFailed
		}
		resultCard = card
		resultAcc = account
		resultTxn = txn
		return nil
	})
	if err != nil {
		resultError = err
	}
	return resultCard, resultAcc, resultTxn, resultError
}

// ResolveRedeemedUsers ???????????
func (s *GiftCardService) ResolveRedeemedUsers(cards []models.GiftCard) (map[uint]models.User, error) {
	result := make(map[uint]models.User)
	if s == nil || s.userRepo == nil || len(cards) == 0 {
		return result, nil
	}
	userIDs := make([]uint, 0, len(cards))
	seen := make(map[uint]struct{})
	for _, card := range cards {
		if card.RedeemedUserID == nil || *card.RedeemedUserID == 0 {
			continue
		}
		if _, ok := seen[*card.RedeemedUserID]; ok {
			continue
		}
		seen[*card.RedeemedUserID] = struct{}{}
		userIDs = append(userIDs, *card.RedeemedUserID)
	}
	if len(userIDs) == 0 {
		return result, nil
	}
	users, err := s.userRepo.ListByIDs(userIDs)
	if err != nil {
		return nil, err
	}
	for _, user := range users {
		result[user.ID] = user
	}
	return result, nil
}

func normalizeGiftCardIDs(ids []uint) []uint {
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

func normalizeGiftCardExpireAt(raw *time.Time) *time.Time {
	if raw == nil || raw.IsZero() {
		return nil
	}
	value := raw.UTC()
	return &value
}

func formatNullableTime(raw *time.Time) string {
	if raw == nil || raw.IsZero() {
		return ""
	}
	return raw.Format(time.RFC3339)
}

func isGiftCardExpired(expiresAt *time.Time, now time.Time) bool {
	if expiresAt == nil || expiresAt.IsZero() {
		return false
	}
	return expiresAt.Before(now)
}

func generateGiftCardBatchNo(now time.Time) string {
	return strings.ToUpper(fmt.Sprintf("%s%s%s", giftCardBatchPrefx, now.Format("20060102150405"), randomHex(4)))
}

func generateGiftCardCode(now time.Time, index int) string {
	return strings.ToUpper(fmt.Sprintf("%s%s%04d%s", giftCardCodePrefix, now.Format("060102150405"), index%10000, randomHex(5)))
}

func randomHex(n int) string {
	if n <= 0 {
		return ""
	}
	buf := make([]byte, n)
	if _, err := crand.Read(buf); err != nil {
		fallback := make([]byte, n)
		for i := range fallback {
			fallback[i] = byte('A' + (i % 26))
		}
		return hex.EncodeToString(fallback)
	}
	return hex.EncodeToString(buf)
}
