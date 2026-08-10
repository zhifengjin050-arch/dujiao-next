package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// AuthzAuditRecordInput ????????
type AuthzAuditRecordInput struct {
	OperatorAdminID  uint
	OperatorUsername string
	TargetAdminID    *uint
	TargetUsername   string
	Action           string
	Role             string
	Object           string
	Method           string
	RequestID        string
	Detail           models.JSON
}

// AuthzAuditService ??????
type AuthzAuditService struct {
	repo repository.AuthzAuditLogRepository
}

// NewAuthzAuditService ????????
func NewAuthzAuditService(repo repository.AuthzAuditLogRepository) *AuthzAuditService {
	return &AuthzAuditService{repo: repo}
}

// Record ????????
func (s *AuthzAuditService) Record(input AuthzAuditRecordInput) error {
	if s == nil || s.repo == nil {
		return nil
	}
	if input.OperatorAdminID == 0 {
		return nil
	}
	if strings.TrimSpace(input.Action) == "" {
		return nil
	}

	item := &models.AuthzAuditLog{
		OperatorAdminID:  input.OperatorAdminID,
		OperatorUsername: strings.TrimSpace(input.OperatorUsername),
		TargetAdminID:    input.TargetAdminID,
		TargetUsername:   strings.TrimSpace(input.TargetUsername),
		Action:           strings.TrimSpace(input.Action),
		Role:             strings.TrimSpace(input.Role),
		Object:           strings.TrimSpace(input.Object),
		Method:           strings.ToUpper(strings.TrimSpace(input.Method)),
		RequestID:        strings.TrimSpace(input.RequestID),
		DetailJSON:       input.Detail,
		CreatedAt:        time.Now(),
	}
	return s.repo.Create(item)
}

// ListForAdmin ???????????
func (s *AuthzAuditService) ListForAdmin(filter repository.AuthzAuditLogListFilter) ([]models.AuthzAuditLog, int64, error) {
	if s == nil || s.repo == nil {
		return []models.AuthzAuditLog{}, 0, nil
	}
	return s.repo.ListAdmin(filter)
}
