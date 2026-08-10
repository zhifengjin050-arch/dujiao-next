package service

import (
	"context"
	"errors"
	"testing"

	"github.com/dujiao-next/internal/constants"
	cryptoutil "github.com/dujiao-next/internal/crypto"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"gorm.io/gorm"
)

type telegramBroadcastRepoStub struct {
	items  map[uint]*models.TelegramBroadcast
	nextID uint
}

func (r *telegramBroadcastRepoStub) Create(broadcast *models.TelegramBroadcast) error {
	if broadcast == nil {
		return nil
	}
	if r.items == nil {
		r.items = map[uint]*models.TelegramBroadcast{}
	}
	r.nextID++
	broadcast.ID = r.nextID
	copyValue := *broadcast
	r.items[broadcast.ID] = &copyValue
	return nil
}

func (r *telegramBroadcastRepoStub) GetByID(id uint) (*models.TelegramBroadcast, error) {
	if item, ok := r.items[id]; ok {
		copyValue := *item
		return &copyValue, nil
	}
	return nil, nil
}

func (r *telegramBroadcastRepoStub) List(filter repository.TelegramBroadcastListFilter) ([]models.TelegramBroadcast, int64, error) {
	items := make([]models.TelegramBroadcast, 0, len(r.items))
	for _, item := range r.items {
		items = append(items, *item)
	}
	return items, int64(len(items)), nil
}

func (r *telegramBroadcastRepoStub) Update(broadcast *models.TelegramBroadcast) error {
	if broadcast == nil {
		return nil
	}
	copyValue := *broadcast
	r.items[broadcast.ID] = &copyValue
	return nil
}

type telegramUserRepoStub struct {
	items []repository.TelegramUserListItem
}

func (r *telegramUserRepoStub) GetByProviderUserID(provider, providerUserID string) (*models.UserOAuthIdentity, error) {
	return nil, nil
}

func (r *telegramUserRepoStub) GetByUserProvider(userID uint, provider string) (*models.UserOAuthIdentity, error) {
	return nil, nil
}

func (r *telegramUserRepoStub) ListByUserID(userID uint) ([]models.UserOAuthIdentity, error) {
	return []models.UserOAuthIdentity{}, nil
}

func (r *telegramUserRepoStub) ListTelegramUsers(filter repository.TelegramUserListFilter) ([]repository.TelegramUserListItem, int64, error) {
	result := make([]repository.TelegramUserListItem, 0, len(r.items))
	allowed := map[uint]struct{}{}
	if len(filter.UserIDs) > 0 {
		for _, id := range filter.UserIDs {
			allowed[id] = struct{}{}
		}
	}
	for _, item := range r.items {
		if len(allowed) > 0 {
			if _, ok := allowed[item.UserID]; !ok {
				continue
			}
		}
		result = append(result, item)
	}
	return result, int64(len(result)), nil
}

func (r *telegramUserRepoStub) Create(identity *models.UserOAuthIdentity) error {
	return nil
}

func (r *telegramUserRepoStub) Update(identity *models.UserOAuthIdentity) error {
	return nil
}

func (r *telegramUserRepoStub) DeleteByID(id uint) error {
	return nil
}

func (r *telegramUserRepoStub) WithTx(tx *gorm.DB) *repository.GormUserOAuthIdentityRepository {
	return nil
}

type channelClientRepoStub struct {
	client *models.ChannelClient
}

func (r *channelClientRepoStub) Create(client *models.ChannelClient) error {
	r.client = client
	return nil
}

func (r *channelClientRepoStub) FindByID(id uint) (*models.ChannelClient, error) {
	if r.client != nil && r.client.ID == id {
		return r.client, nil
	}
	return nil, nil
}

func (r *channelClientRepoStub) FindByChannelKey(key string) (*models.ChannelClient, error) {
	return nil, nil
}

func (r *channelClientRepoStub) FindActiveByChannelType(channelType string) (*models.ChannelClient, error) {
	if r.client == nil || r.client.ChannelType != channelType || r.client.Status != 1 {
		return nil, nil
	}
	return r.client, nil
}

func (r *channelClientRepoStub) FindAll() ([]models.ChannelClient, error) {
	if r.client == nil {
		return []models.ChannelClient{}, nil
	}
	return []models.ChannelClient{*r.client}, nil
}

func (r *channelClientRepoStub) Update(client *models.ChannelClient) error {
	r.client = client
	return nil
}

func (r *channelClientRepoStub) Delete(client *models.ChannelClient) error {
	r.client = nil
	return nil
}

type telegramSenderStub struct {
	failures map[string]error
	calls    []TelegramSendOptions
}

func (s *telegramSenderStub) SendWithBotToken(ctx context.Context, botToken string, options TelegramSendOptions) error {
	s.calls = append(s.calls, options)
	if err, ok := s.failures[options.ChatID]; ok {
		return err
	}
	return nil
}

func TestTelegramBroadcastServiceCreateBroadcastValidation(t *testing.T) {
	service := &TelegramBroadcastService{
		repo:                  &telegramBroadcastRepoStub{},
		userOAuthIdentityRepo: &telegramUserRepoStub{},
		channelClientRepo:     &channelClientRepoStub{},
		channelClientService:  NewChannelClientService(&channelClientRepoStub{}, "test-secret"),
		telegramSender:        &telegramSenderStub{},
	}

	_, err := service.CreateBroadcast(context.Background(), TelegramBroadcastCreateInput{
		Title:         "",
		RecipientType: constants.TelegramBroadcastRecipientTypeAll,
		MessageHTML:   "<b>hello</b>",
	})
	if !errors.Is(err, ErrTelegramBroadcastInvalid) {
		t.Fatalf("expected invalid broadcast error, got %v", err)
	}

	_, err = service.CreateBroadcast(context.Background(), TelegramBroadcastCreateInput{
		Title:         "Demo",
		RecipientType: constants.TelegramBroadcastRecipientTypeAll,
		MessageHTML:   "<b>hello</b>",
	})
	if !errors.Is(err, ErrTelegramBotTokenUnavailable) {
		t.Fatalf("expected token unavailable error, got %v", err)
	}
}

func TestTelegramBroadcastServiceProcessBroadcastUpdatesStats(t *testing.T) {
	repo := &telegramBroadcastRepoStub{
		items: map[uint]*models.TelegramBroadcast{
			1: {
				ID:               1,
				Title:            "Demo",
				RecipientType:    constants.TelegramBroadcastRecipientTypeSpecific,
				RecipientChatIDs: models.StringArray{"1001", "1002"},
				RecipientCount:   2,
				Status:           constants.TelegramBroadcastStatusPending,
				MessageHTML:      "<b>Hello</b>",
			},
		},
		nextID: 1,
	}
	encryptedToken, err := cryptoutil.Encrypt(cryptoutil.DeriveKey("test-secret"), "bot-token")
	if err != nil {
		t.Fatalf("encrypt token failed: %v", err)
	}
	channelRepo := &channelClientRepoStub{
		client: &models.ChannelClient{
			ID:          1,
			ChannelType: "telegram_bot",
			BotToken:    encryptedToken,
			Status:      1,
		},
	}
	sender := &telegramSenderStub{
		failures: map[string]error{
			"1002": errors.New("send failed"),
		},
	}
	service := &TelegramBroadcastService{
		repo:                  repo,
		userOAuthIdentityRepo: &telegramUserRepoStub{},
		channelClientRepo:     channelRepo,
		channelClientService:  NewChannelClientService(channelRepo, "test-secret"),
		telegramSender:        sender,
	}

	if err := service.ProcessBroadcast(context.Background(), 1); err != nil {
		t.Fatalf("process broadcast failed: %v", err)
	}

	updated, err := repo.GetByID(1)
	if err != nil {
		t.Fatalf("get updated broadcast failed: %v", err)
	}
	if updated == nil {
		t.Fatalf("expected updated broadcast")
	}
	if updated.SuccessCount != 1 || updated.FailedCount != 1 {
		t.Fatalf("unexpected stats: success=%d failed=%d", updated.SuccessCount, updated.FailedCount)
	}
	if updated.Status != constants.TelegramBroadcastStatusCompleted {
		t.Fatalf("expected completed status, got %s", updated.Status)
	}
	if len(sender.calls) != 2 {
		t.Fatalf("expected 2 send calls, got %d", len(sender.calls))
	}
}
