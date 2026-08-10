package service

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"

	"github.com/dujiao-next/internal/crypto"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/upstream"
)

// ChannelClientService ?????????
type ChannelClientService struct {
	repo      repository.ChannelClientRepository
	encKey    []byte // AES-256 ??
	secretKey string // ????(????)
}

// NewChannelClientService ?????????
func NewChannelClientService(repo repository.ChannelClientRepository, appSecretKey string) *ChannelClientService {
	return &ChannelClientService{
		repo:      repo,
		encKey:    crypto.DeriveKey(appSecretKey),
		secretKey: appSecretKey,
	}
}

// ChannelClientResponse ???????(??? secret)
type ChannelClientResponse struct {
	ID            uint   `json:"id"`
	Name          string `json:"name"`
	ChannelType   string `json:"channel_type"`
	ChannelKey    string `json:"channel_key"`
	ChannelSecret string `json:"channel_secret"`
	BotToken      string `json:"bot_token"`
	BotTokenSet   bool   `json:"bot_token_set"`
	CallbackURL   string `json:"callback_url"`
	Description   string `json:"description"`
	Status        int    `json:"status"`
}

// CreateChannelClient ???????
func (s *ChannelClientService) CreateChannelClient(name, channelType, description, botToken, callbackURL string) (*ChannelClientResponse, error) {
	// ???? key (32 bytes = 64 hex chars)
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return nil, fmt.Errorf("generate channel key: %w", err)
	}
	channelKey := hex.EncodeToString(keyBytes)

	// ???? secret (32 bytes = 64 hex chars)
	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate channel secret: %w", err)
	}
	plainSecret := hex.EncodeToString(secretBytes)

	// ?? secret ??
	encryptedSecret, err := crypto.Encrypt(s.encKey, plainSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt channel secret: %w", err)
	}

	client := &models.ChannelClient{
		Name:          name,
		ChannelType:   channelType,
		ChannelKey:    channelKey,
		ChannelSecret: encryptedSecret,
		CallbackURL:   callbackURL,
		Status:        1,
		Description:   description,
	}

	// ?? bot_token(????)
	if botToken != "" {
		encryptedToken, err := crypto.Encrypt(s.encKey, botToken)
		if err != nil {
			return nil, fmt.Errorf("encrypt bot token: %w", err)
		}
		client.BotToken = encryptedToken
	}

	if err := s.repo.Create(client); err != nil {
		return nil, err
	}

	return &ChannelClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		ChannelType:   client.ChannelType,
		ChannelKey:    client.ChannelKey,
		ChannelSecret: plainSecret,
		BotToken:      maskBotToken(botToken),
		BotTokenSet:   botToken != "",
		CallbackURL:   client.CallbackURL,
		Description:   client.Description,
		Status:        client.Status,
	}, nil
}

// GetChannelClient ???????
func (s *ChannelClientService) GetChannelClient(id uint) (*models.ChannelClient, error) {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrChannelClientNotFound
	}
	return client, nil
}

// ListChannelClients ?????????
func (s *ChannelClientService) ListChannelClients() ([]models.ChannelClient, error) {
	return s.repo.FindAll()
}

// GetChannelClientDetail ?????????(??? secret)
func (s *ChannelClientService) GetChannelClientDetail(id uint) (*ChannelClientResponse, error) {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrChannelClientNotFound
	}

	plainSecret, err := crypto.Decrypt(s.encKey, client.ChannelSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt channel secret: %w", err)
	}

	resp := &ChannelClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		ChannelType:   client.ChannelType,
		ChannelKey:    client.ChannelKey,
		ChannelSecret: plainSecret,
		BotTokenSet:   client.BotToken != "",
		CallbackURL:   client.CallbackURL,
		Description:   client.Description,
		Status:        client.Status,
	}

	if client.BotToken != "" {
		plainToken, err := crypto.Decrypt(s.encKey, client.BotToken)
		if err == nil {
			resp.BotToken = maskBotToken(plainToken)
		}
	}

	return resp, nil
}

// ListChannelClientDetails ?????????(??? secret)
func (s *ChannelClientService) ListChannelClientDetails() ([]ChannelClientResponse, error) {
	clients, err := s.repo.FindAll()
	if err != nil {
		return nil, err
	}
	result := make([]ChannelClientResponse, 0, len(clients))
	for _, c := range clients {
		plainSecret, decErr := crypto.Decrypt(s.encKey, c.ChannelSecret)
		if decErr != nil {
			plainSecret = ""
		}
		resp := ChannelClientResponse{
			ID:            c.ID,
			Name:          c.Name,
			ChannelType:   c.ChannelType,
			ChannelKey:    c.ChannelKey,
			ChannelSecret: plainSecret,
			BotTokenSet:   c.BotToken != "",
			CallbackURL:   c.CallbackURL,
			Description:   c.Description,
			Status:        c.Status,
		}
		if c.BotToken != "" {
			plainToken, decErr := crypto.Decrypt(s.encKey, c.BotToken)
			if decErr == nil {
				resp.BotToken = maskBotToken(plainToken)
			}
		}
		result = append(result, resp)
	}
	return result, nil
}

// ResetChannelClientSecret ??????? Secret
func (s *ChannelClientService) ResetChannelClientSecret(id uint) (*ChannelClientResponse, error) {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrChannelClientNotFound
	}

	secretBytes := make([]byte, 32)
	if _, err := rand.Read(secretBytes); err != nil {
		return nil, fmt.Errorf("generate channel secret: %w", err)
	}
	plainSecret := hex.EncodeToString(secretBytes)

	encryptedSecret, err := crypto.Encrypt(s.encKey, plainSecret)
	if err != nil {
		return nil, fmt.Errorf("encrypt channel secret: %w", err)
	}

	client.ChannelSecret = encryptedSecret
	if err := s.repo.Update(client); err != nil {
		return nil, err
	}

	resp := &ChannelClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		ChannelType:   client.ChannelType,
		ChannelKey:    client.ChannelKey,
		ChannelSecret: plainSecret,
		BotTokenSet:   client.BotToken != "",
		CallbackURL:   client.CallbackURL,
		Description:   client.Description,
		Status:        client.Status,
	}
	if client.BotToken != "" {
		plainToken, decErr := crypto.Decrypt(s.encKey, client.BotToken)
		if decErr == nil {
			resp.BotToken = maskBotToken(plainToken)
		}
	}
	return resp, nil
}

// DeleteChannelClient ???????(???)
func (s *ChannelClientService) DeleteChannelClient(id uint) error {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrChannelClientNotFound
	}
	return s.repo.Delete(client)
}

// UpdateChannelClientStatus ?????????
func (s *ChannelClientService) UpdateChannelClientStatus(id uint, status int) error {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return err
	}
	if client == nil {
		return ErrChannelClientNotFound
	}
	client.Status = status
	return s.repo.Update(client)
}

// UpdateChannelClient ?????????(??????bot_token)
func (s *ChannelClientService) UpdateChannelClient(id uint, name, description string, botToken *string, callbackURL *string) (*ChannelClientResponse, error) {
	client, err := s.repo.FindByID(id)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrChannelClientNotFound
	}

	if name != "" {
		client.Name = name
	}
	client.Description = description
	if callbackURL != nil {
		client.CallbackURL = *callbackURL
	}

	// botToken ? nil ?????;? nil ???(????????)
	if botToken != nil {
		if *botToken != "" {
			encryptedToken, err := crypto.Encrypt(s.encKey, *botToken)
			if err != nil {
				return nil, fmt.Errorf("encrypt bot token: %w", err)
			}
			client.BotToken = encryptedToken
		} else {
			client.BotToken = ""
		}
	}

	if err := s.repo.Update(client); err != nil {
		return nil, err
	}

	plainSecret, err := crypto.Decrypt(s.encKey, client.ChannelSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt channel secret: %w", err)
	}

	resp := &ChannelClientResponse{
		ID:            client.ID,
		Name:          client.Name,
		ChannelType:   client.ChannelType,
		ChannelKey:    client.ChannelKey,
		ChannelSecret: plainSecret,
		BotTokenSet:   client.BotToken != "",
		CallbackURL:   client.CallbackURL,
		Description:   client.Description,
		Status:        client.Status,
	}
	if client.BotToken != "" {
		plainToken, decErr := crypto.Decrypt(s.encKey, client.BotToken)
		if decErr == nil {
			resp.BotToken = maskBotToken(plainToken)
		}
	}
	return resp, nil
}

// DecryptBotToken ???????? Bot Token(? Channel API ??)
func (s *ChannelClientService) DecryptBotToken(client *models.ChannelClient) (string, error) {
	if client.BotToken == "" {
		return "", nil
	}
	return crypto.Decrypt(s.encKey, client.BotToken)
}

// DecryptChannelSecret ???????? ChannelSecret
func (s *ChannelClientService) DecryptChannelSecret(client *models.ChannelClient) (string, error) {
	if client.ChannelSecret == "" {
		return "", nil
	}
	return crypto.Decrypt(s.encKey, client.ChannelSecret)
}

// VerifyChannelSignature ??????
// ?? upstream/signer.go ? HMAC-SHA256 ????
func (s *ChannelClientService) VerifyChannelSignature(key, signature string, timestamp int64, method, path string, body []byte) (*models.ChannelClient, error) {
	// ?????
	if !upstream.IsTimestampValid(timestamp) {
		return nil, ErrChannelTimestampExpired
	}

	// ?????
	client, err := s.repo.FindByChannelKey(key)
	if err != nil {
		return nil, err
	}
	if client == nil {
		return nil, ErrChannelClientNotFound
	}
	if client.Status != 1 {
		return nil, ErrChannelClientDisabled
	}

	// ?? secret
	plainSecret, err := crypto.Decrypt(s.encKey, client.ChannelSecret)
	if err != nil {
		return nil, fmt.Errorf("decrypt channel secret: %w", err)
	}

	// ????(?? upstream.Verify)
	if !upstream.Verify(plainSecret, method, path, signature, timestamp, body) {
		return nil, ErrChannelSignatureInvalid
	}

	return client, nil
}
