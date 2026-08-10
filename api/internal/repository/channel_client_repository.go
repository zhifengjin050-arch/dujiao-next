package repository

import (
	"errors"

	"github.com/dujiao-next/internal/models"

	"gorm.io/gorm"
)

// ChannelClientRepository ???????????
type ChannelClientRepository interface {
	Create(client *models.ChannelClient) error
	FindByID(id uint) (*models.ChannelClient, error)
	FindByChannelKey(key string) (*models.ChannelClient, error)
	FindActiveByChannelType(channelType string) (*models.ChannelClient, error)
	FindAll() ([]models.ChannelClient, error)
	Update(client *models.ChannelClient) error
	Delete(client *models.ChannelClient) error
}

// GormChannelClientRepository GORM ??
type GormChannelClientRepository struct {
	db *gorm.DB
}

// NewChannelClientRepository ?????????
func NewChannelClientRepository(db *gorm.DB) *GormChannelClientRepository {
	return &GormChannelClientRepository{db: db}
}

// Create ???????
func (r *GormChannelClientRepository) Create(client *models.ChannelClient) error {
	return r.db.Create(client).Error
}

// FindByID ?? ID ??
func (r *GormChannelClientRepository) FindByID(id uint) (*models.ChannelClient, error) {
	var client models.ChannelClient
	if err := r.db.First(&client, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindByChannelKey ?? channel_key ??
func (r *GormChannelClientRepository) FindByChannelKey(key string) (*models.ChannelClient, error) {
	var client models.ChannelClient
	if err := r.db.Where("channel_key = ?", key).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindActiveByChannelType ?? channel_type ??????????(????)
func (r *GormChannelClientRepository) FindActiveByChannelType(channelType string) (*models.ChannelClient, error) {
	var client models.ChannelClient
	if err := r.db.Where("channel_type = ? AND status = 1", channelType).First(&client).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &client, nil
}

// FindAll ?????????
func (r *GormChannelClientRepository) FindAll() ([]models.ChannelClient, error) {
	var clients []models.ChannelClient
	if err := r.db.Order("created_at DESC").Find(&clients).Error; err != nil {
		return nil, err
	}
	return clients, nil
}

// Update ???????
func (r *GormChannelClientRepository) Update(client *models.ChannelClient) error {
	return r.db.Save(client).Error
}

// Delete ???????(???)
func (r *GormChannelClientRepository) Delete(client *models.ChannelClient) error {
	return r.db.Delete(client).Error
}
