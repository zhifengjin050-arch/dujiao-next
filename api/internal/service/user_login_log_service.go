package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// UserLoginLogService ????????
type UserLoginLogService struct {
	repo repository.UserLoginLogRepository
}

// NewUserLoginLogService ??????????
func NewUserLoginLogService(repo repository.UserLoginLogRepository) *UserLoginLogService {
	return &UserLoginLogService{repo: repo}
}

// RecordUserLoginInput ????????
type RecordUserLoginInput struct {
	UserID      uint
	Email       string
	Status      string
	FailReason  string
	ClientIP    string
	UserAgent   string
	LoginSource string
	RequestID   string
}

// Record ??????
func (s *UserLoginLogService) Record(input RecordUserLoginInput) error {
	if s == nil || s.repo == nil {
		return nil
	}

	email := strings.TrimSpace(input.Email)
	if normalized, err := NormalizeEmail(email); err == nil {
		email = normalized
	}

	status := strings.ToLower(strings.TrimSpace(input.Status))
	if status != constants.LoginLogStatusSuccess {
		status = constants.LoginLogStatusFailed
	}

	failReason := strings.ToLower(strings.TrimSpace(input.FailReason))
	if status == constants.LoginLogStatusSuccess {
		failReason = ""
	} else if failReason == "" {
		failReason = constants.LoginLogFailReasonInternalError
	}

	source := strings.ToLower(strings.TrimSpace(input.LoginSource))
	if source == "" {
		source = constants.LoginLogSourceWeb
	}

	now := time.Now()
	return s.repo.Create(&models.UserLoginLog{
		UserID:      input.UserID,
		Email:       email,
		Status:      status,
		FailReason:  failReason,
		ClientIP:    strings.TrimSpace(input.ClientIP),
		UserAgent:   strings.TrimSpace(input.UserAgent),
		LoginSource: source,
		RequestID:   strings.TrimSpace(input.RequestID),
		CreatedAt:   now,
	})
}

// ListForAdmin ?????????
func (s *UserLoginLogService) ListForAdmin(filter repository.UserLoginLogListFilter) ([]models.UserLoginLog, int64, error) {
	if s == nil || s.repo == nil {
		return []models.UserLoginLog{}, 0, nil
	}
	return s.repo.ListAdmin(filter)
}

// ListByUser ????????????
func (s *UserLoginLogService) ListByUser(userID uint, page, pageSize int) ([]models.UserLoginLog, int64, error) {
	if s == nil || s.repo == nil || userID == 0 {
		return []models.UserLoginLog{}, 0, nil
	}
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return s.repo.ListByUser(userID, page, pageSize)
}
