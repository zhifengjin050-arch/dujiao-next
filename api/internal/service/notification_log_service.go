package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

const (
	notificationLogStatusSuccess = "success"
	notificationLogStatusFailed  = "failed"
)

// NotificationLogRecordInput ????????
type NotificationLogRecordInput struct {
	EventType    string
	BizType      string
	BizID        uint
	Channel      string
	Recipient    string
	Locale       string
	Title        string
	Body         string
	Status       string
	ErrorMessage string
	IsTest       bool
	Variables    models.JSON
}

// NotificationLogService ??????
type NotificationLogService struct {
	repo repository.NotificationLogRepository
}

// NewNotificationLogService ????????
func NewNotificationLogService(repo repository.NotificationLogRepository) *NotificationLogService {
	return &NotificationLogService{repo: repo}
}

// Record ????????
func (s *NotificationLogService) Record(input NotificationLogRecordInput) error {
	if s == nil || s.repo == nil {
		return nil
	}

	channel := strings.ToLower(strings.TrimSpace(input.Channel))
	recipient := strings.TrimSpace(input.Recipient)
	if channel == "" || recipient == "" {
		return nil
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != notificationLogStatusSuccess {
		status = notificationLogStatusFailed
	}

	item := &models.NotificationLog{
		EventType:     strings.ToLower(strings.TrimSpace(input.EventType)),
		BizType:       strings.ToLower(strings.TrimSpace(input.BizType)),
		BizID:         input.BizID,
		Channel:       channel,
		Recipient:     recipient,
		Locale:        strings.TrimSpace(input.Locale),
		Title:         strings.TrimSpace(input.Title),
		Body:          strings.TrimSpace(input.Body),
		Status:        status,
		ErrorMessage:  strings.TrimSpace(input.ErrorMessage),
		IsTest:        input.IsTest,
		VariablesJSON: cloneNotificationLogJSON(input.Variables),
		CreatedAt:     time.Now(),
	}
	return s.repo.Create(item)
}

// ListForAdmin ?????????
func (s *NotificationLogService) ListForAdmin(filter repository.NotificationLogListFilter) ([]models.NotificationLog, int64, error) {
	if s == nil || s.repo == nil {
		return []models.NotificationLog{}, 0, nil
	}
	return s.repo.ListAdmin(filter)
}

func cloneNotificationLogJSON(data models.JSON) models.JSON {
	if len(data) == 0 {
		return models.JSON{}
	}
	result := make(models.JSON, len(data))
	for key, value := range data {
		result[key] = value
	}
	return result
}
