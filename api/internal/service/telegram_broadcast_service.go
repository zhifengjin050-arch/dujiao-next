package service

import (
	"context"
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/queue"
	"github.com/dujiao-next/internal/repository"
)

// TelegramBroadcastListInput Telegram ???????
type TelegramBroadcastListInput struct {
	Page     int
	PageSize int
}

// TelegramBroadcastUserQuery Telegram ???????
type TelegramBroadcastUserQuery struct {
	Page             int
	PageSize         int
	Keyword          string
	DisplayName      string
	TelegramUsername string
	TelegramUserID   string
	CreatedFrom      *time.Time
	CreatedTo        *time.Time
}

// TelegramBroadcastCreateInput Telegram ???????
type TelegramBroadcastCreateInput struct {
	Title          string      `json:"title"`
	RecipientType  string      `json:"recipient_type"`
	UserIDs        []uint      `json:"user_ids"`
	Filters        models.JSON `json:"filters"`
	AttachmentURL  string      `json:"attachment_url"`
	AttachmentName string      `json:"attachment_name"`
	MessageHTML    string      `json:"message_html"`
}

// TelegramBroadcastService Telegram ?????
type TelegramBroadcastService struct {
	repo                  repository.TelegramBroadcastRepository
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository
	channelClientRepo     repository.ChannelClientRepository
	channelClientService  *ChannelClientService
	queueClient           *queue.Client
	telegramSender        telegramBroadcastSender
}

type telegramBroadcastSender interface {
	SendWithBotToken(ctx context.Context, botToken string, options TelegramSendOptions) error
}

// NewTelegramBroadcastService ?? Telegram ?????
func NewTelegramBroadcastService(
	repo repository.TelegramBroadcastRepository,
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository,
	channelClientRepo repository.ChannelClientRepository,
	channelClientService *ChannelClientService,
	queueClient *queue.Client,
	telegramSender telegramBroadcastSender,
) *TelegramBroadcastService {
	return &TelegramBroadcastService{
		repo:                  repo,
		userOAuthIdentityRepo: userOAuthIdentityRepo,
		channelClientRepo:     channelClientRepo,
		channelClientService:  channelClientService,
		queueClient:           queueClient,
		telegramSender:        telegramSender,
	}
}

// GetBroadcast ?????????
func (s *TelegramBroadcastService) GetBroadcast(id uint) (*models.TelegramBroadcast, error) {
	return s.repo.GetByID(id)
}

// ListBroadcasts ???????
func (s *TelegramBroadcastService) ListBroadcasts(input TelegramBroadcastListInput) ([]models.TelegramBroadcast, int64, error) {
	return s.repo.List(repository.TelegramBroadcastListFilter{
		Page:     input.Page,
		PageSize: input.PageSize,
	})
}

// ListTelegramUsers ?? Telegram ???????
func (s *TelegramBroadcastService) ListTelegramUsers(input TelegramBroadcastUserQuery) ([]repository.TelegramUserListItem, int64, error) {
	return s.userOAuthIdentityRepo.ListTelegramUsers(repository.TelegramUserListFilter{
		Page:             input.Page,
		PageSize:         input.PageSize,
		Keyword:          input.Keyword,
		DisplayName:      input.DisplayName,
		TelegramUsername: input.TelegramUsername,
		TelegramUserID:   input.TelegramUserID,
		CreatedFrom:      input.CreatedFrom,
		CreatedTo:        input.CreatedTo,
	})
}

// CreateBroadcast ??????????
func (s *TelegramBroadcastService) CreateBroadcast(ctx context.Context, input TelegramBroadcastCreateInput) (*models.TelegramBroadcast, error) {
	if s == nil || s.repo == nil || s.userOAuthIdentityRepo == nil || s.channelClientRepo == nil || s.channelClientService == nil {
		return nil, ErrTelegramBroadcastInvalid
	}

	title := strings.TrimSpace(input.Title)
	messageHTML := strings.TrimSpace(input.MessageHTML)
	recipientType := strings.ToLower(strings.TrimSpace(input.RecipientType))
	if title == "" || messageHTML == "" {
		return nil, ErrTelegramBroadcastInvalid
	}
	if recipientType != constants.TelegramBroadcastRecipientTypeAll && recipientType != constants.TelegramBroadcastRecipientTypeSpecific {
		return nil, ErrTelegramBroadcastInvalid
	}

	if _, err := s.resolveActiveBotToken(); err != nil {
		return nil, err
	}

	recipientChatIDs, filtersSnapshot, err := s.resolveRecipients(input)
	if err != nil {
		return nil, err
	}
	if len(recipientChatIDs) == 0 {
		return nil, ErrTelegramBroadcastNoRecipients
	}

	broadcast := &models.TelegramBroadcast{
		Title:            title,
		RecipientType:    recipientType,
		FiltersJSON:      filtersSnapshot,
		RecipientChatIDs: models.StringArray(recipientChatIDs),
		RecipientCount:   len(recipientChatIDs),
		Status:           constants.TelegramBroadcastStatusPending,
		MessageHTML:      messageHTML,
		AttachmentURL:    strings.TrimSpace(input.AttachmentURL),
		AttachmentName:   strings.TrimSpace(input.AttachmentName),
	}
	if err := s.repo.Create(broadcast); err != nil {
		return nil, err
	}
	if err := s.dispatchBroadcast(ctx, broadcast.ID); err != nil {
		return nil, err
	}
	return broadcast, nil
}

// ProcessBroadcast ???????
func (s *TelegramBroadcastService) ProcessBroadcast(ctx context.Context, broadcastID uint) error {
	if s == nil || s.repo == nil {
		return ErrTelegramBroadcastInvalid
	}
	broadcast, err := s.repo.GetByID(broadcastID)
	if err != nil {
		return err
	}
	if broadcast == nil {
		return ErrTelegramBroadcastNotFound
	}
	if broadcast.Status == constants.TelegramBroadcastStatusCompleted {
		return nil
	}

	now := time.Now()
	broadcast.Status = constants.TelegramBroadcastStatusRunning
	if broadcast.StartedAt == nil {
		broadcast.StartedAt = &now
	}
	broadcast.CompletedAt = nil
	broadcast.LastError = ""
	if err := s.repo.Update(broadcast); err != nil {
		return err
	}

	token, err := s.resolveActiveBotToken()
	if err != nil {
		return s.markBroadcastFailed(broadcast, err.Error())
	}

	chatIDs := dedupeStrings([]string(broadcast.RecipientChatIDs))
	if len(chatIDs) == 0 {
		return s.markBroadcastFailed(broadcast, ErrTelegramBroadcastNoRecipients.Error())
	}

	successCount := 0
	failedCount := 0
	lastError := ""
	for _, chatID := range chatIDs {
		err := s.telegramSender.SendWithBotToken(ctx, token, TelegramSendOptions{
			ChatID:                chatID,
			Message:               broadcast.MessageHTML,
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
			AttachmentURL:         broadcast.AttachmentURL,
			AttachmentDisplayName: broadcast.AttachmentName,
		})
		if err != nil {
			failedCount++
			lastError = err.Error()
			continue
		}
		successCount++
	}

	completedAt := time.Now()
	broadcast.SuccessCount = successCount
	broadcast.FailedCount = failedCount
	broadcast.CompletedAt = &completedAt
	broadcast.LastError = lastError
	if successCount == 0 && failedCount > 0 {
		broadcast.Status = constants.TelegramBroadcastStatusFailed
	} else {
		broadcast.Status = constants.TelegramBroadcastStatusCompleted
	}
	return s.repo.Update(broadcast)
}

func (s *TelegramBroadcastService) resolveRecipients(input TelegramBroadcastCreateInput) ([]string, models.JSON, error) {
	filtersSnapshot := cloneJSONMap(input.Filters)
	if filtersSnapshot == nil {
		filtersSnapshot = models.JSON{}
	}

	var (
		items []repository.TelegramUserListItem
		err   error
	)
	switch strings.ToLower(strings.TrimSpace(input.RecipientType)) {
	case constants.TelegramBroadcastRecipientTypeAll:
		items, _, err = s.userOAuthIdentityRepo.ListTelegramUsers(repository.TelegramUserListFilter{
			Page:     1,
			PageSize: 0,
		})
	case constants.TelegramBroadcastRecipientTypeSpecific:
		uniqueUserIDs := uniqueUintIDs(input.UserIDs)
		if len(uniqueUserIDs) == 0 {
			return nil, nil, ErrTelegramBroadcastNoRecipients
		}
		filtersSnapshot["selected_user_ids"] = uniqueUserIDs
		items, _, err = s.userOAuthIdentityRepo.ListTelegramUsers(repository.TelegramUserListFilter{
			Page:     1,
			PageSize: 0,
			UserIDs:  uniqueUserIDs,
		})
	default:
		return nil, nil, ErrTelegramBroadcastInvalid
	}
	if err != nil {
		return nil, nil, err
	}

	chatIDs := make([]string, 0, len(items))
	for _, item := range items {
		chatID := strings.TrimSpace(item.TelegramUserID)
		if chatID == "" {
			continue
		}
		chatIDs = append(chatIDs, chatID)
	}
	chatIDs = dedupeStrings(chatIDs)
	if len(chatIDs) == 0 {
		return nil, nil, ErrTelegramBroadcastNoRecipients
	}
	return chatIDs, filtersSnapshot, nil
}

func (s *TelegramBroadcastService) resolveActiveBotToken() (string, error) {
	client, err := s.channelClientRepo.FindActiveByChannelType("telegram_bot")
	if err != nil {
		return "", err
	}
	if client == nil {
		return "", ErrTelegramBotTokenUnavailable
	}
	token, err := s.channelClientService.DecryptBotToken(client)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(token) == "" {
		return "", ErrTelegramBotTokenUnavailable
	}
	return strings.TrimSpace(token), nil
}

func (s *TelegramBroadcastService) dispatchBroadcast(_ context.Context, broadcastID uint) error {
	if s.queueClient != nil && s.queueClient.Enabled() {
		return s.queueClient.EnqueueTelegramBroadcast(queue.TelegramBroadcastPayload{BroadcastID: broadcastID})
	}

	go func() {
		_ = s.ProcessBroadcast(context.Background(), broadcastID)
	}()
	return nil
}

func (s *TelegramBroadcastService) markBroadcastFailed(broadcast *models.TelegramBroadcast, reason string) error {
	if broadcast == nil {
		return ErrTelegramBroadcastNotFound
	}
	completedAt := time.Now()
	broadcast.Status = constants.TelegramBroadcastStatusFailed
	broadcast.CompletedAt = &completedAt
	broadcast.LastError = strings.TrimSpace(reason)
	return s.repo.Update(broadcast)
}

func cloneJSONMap(source models.JSON) models.JSON {
	if source == nil {
		return nil
	}
	result := make(models.JSON, len(source))
	for key, value := range source {
		result[key] = value
	}
	return result
}

func uniqueUintIDs(source []uint) []uint {
	if len(source) == 0 {
		return []uint{}
	}
	result := make([]uint, 0, len(source))
	seen := make(map[uint]struct{}, len(source))
	for _, item := range source {
		if item == 0 {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

func dedupeStrings(source []string) []string {
	if len(source) == 0 {
		return []string{}
	}
	result := make([]string, 0, len(source))
	seen := make(map[string]struct{}, len(source))
	for _, item := range source {
		normalized := strings.TrimSpace(item)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}
