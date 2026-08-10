package admin

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

const protectedSuperAdminUsername = "admin"

type authzCreateAdminPayload struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	IsSuper  *bool  `json:"is_super"`
}

type authzUpdateAdminPayload struct {
	Username *string `json:"username"`
	Password *string `json:"password"`
	IsSuper  *bool   `json:"is_super"`
}

// CreateAuthzAdmin ?????
func (h *Handler) CreateAuthzAdmin(c *gin.Context) {
	var req authzCreateAdminPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	username, err := normalizeAdminUsername(req.Username)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_username_invalid", err)
		return
	}
	password := strings.TrimSpace(req.Password)
	if password == "" {
		shared.RespondError(c, response.CodeBadRequest, "error.password_weak", nil)
		return
	}

	existing, err := h.AdminRepo.GetByUsername(username)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_create_failed", err)
		return
	}
	if existing != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_username_exists", nil)
		return
	}

	if err := h.AuthService.ValidatePassword(password); err != nil {
		if respondAdminPasswordPolicyError(c, err) {
			return
		}
		shared.RespondError(c, response.CodeBadRequest, "error.password_weak", err)
		return
	}

	hash, err := h.AuthService.HashPassword(password)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_create_failed", err)
		return
	}

	isSuper := req.IsSuper != nil && *req.IsSuper
	if strings.EqualFold(username, protectedSuperAdminUsername) {
		isSuper = true
	}

	admin := &models.Admin{
		Username:     username,
		PasswordHash: hash,
		IsSuper:      isSuper,
	}
	if err := h.AdminRepo.Create(admin); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_create_failed", err)
		return
	}

	_ = cache.SetAdminAuthState(c.Request.Context(), cache.BuildAdminAuthState(admin))

	h.recordAuthzAudit(c, service.AuthzAuditRecordInput{
		OperatorAdminID:  c.GetUint("admin_id"),
		OperatorUsername: strings.TrimSpace(c.GetString("username")),
		TargetAdminID:    &admin.ID,
		TargetUsername:   admin.Username,
		Action:           "admin_create",
		RequestID:        strings.TrimSpace(c.GetString("request_id")),
		Detail: models.JSON{
			"target_admin_id": admin.ID,
			"target_username": admin.Username,
			"is_super":        admin.IsSuper,
		},
	})

	logger.Infow("admin_authz_admin_created",
		"operator_admin_id", c.GetUint("admin_id"),
		"target_admin_id", admin.ID,
		"target_username", admin.Username,
		"is_super", admin.IsSuper,
	)

	response.Success(c, admin)
}

// UpdateAuthzAdmin ?????
func (h *Handler) UpdateAuthzAdmin(c *gin.Context) {
	adminID, ok := parseAdminIDParam(c)
	if !ok {
		return
	}

	admin, err := h.AdminRepo.GetByID(adminID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_update_failed", err)
		return
	}
	if admin == nil {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_id_invalid", nil)
		return
	}

	var req authzUpdateAdminPayload
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	updatedFields := make([]string, 0, 3)

	if req.Username != nil {
		normalizedUsername, err := normalizeAdminUsername(*req.Username)
		if err != nil {
			shared.RespondError(c, response.CodeBadRequest, "error.admin_username_invalid", err)
			return
		}
		if normalizedUsername != admin.Username {
			existing, err := h.AdminRepo.GetByUsername(normalizedUsername)
			if err != nil {
				shared.RespondError(c, response.CodeInternal, "error.admin_update_failed", err)
				return
			}
			if existing != nil && existing.ID != admin.ID {
				shared.RespondError(c, response.CodeBadRequest, "error.admin_username_exists", nil)
				return
			}
			admin.Username = normalizedUsername
			updatedFields = append(updatedFields, "username")
		}
	}

	if req.IsSuper != nil {
		nextIsSuper := *req.IsSuper
		if strings.EqualFold(strings.TrimSpace(admin.Username), protectedSuperAdminUsername) {
			nextIsSuper = true
		}
		if admin.IsSuper != nextIsSuper {
			admin.IsSuper = nextIsSuper
			updatedFields = append(updatedFields, "is_super")
		}
	}

	if req.Password != nil {
		password := strings.TrimSpace(*req.Password)
		if password == "" {
			shared.RespondError(c, response.CodeBadRequest, "error.password_weak", nil)
			return
		}
		if err := h.AuthService.ValidatePassword(password); err != nil {
			if respondAdminPasswordPolicyError(c, err) {
				return
			}
			shared.RespondError(c, response.CodeBadRequest, "error.password_weak", err)
			return
		}
		hash, err := h.AuthService.HashPassword(password)
		if err != nil {
			shared.RespondError(c, response.CodeInternal, "error.admin_update_failed", err)
			return
		}
		admin.PasswordHash = hash
		now := time.Now()
		admin.TokenVersion++
		admin.TokenInvalidBefore = &now
		updatedFields = append(updatedFields, "password")
	}

	if len(updatedFields) == 0 {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", nil)
		return
	}

	if err := h.AdminRepo.Update(admin); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_update_failed", err)
		return
	}
	_ = cache.SetAdminAuthState(c.Request.Context(), cache.BuildAdminAuthState(admin))

	sort.Strings(updatedFields)
	if c.GetUint("admin_id") == admin.ID {
		c.Set("admin_is_super", admin.IsSuper)
	}

	h.recordAuthzAudit(c, service.AuthzAuditRecordInput{
		OperatorAdminID:  c.GetUint("admin_id"),
		OperatorUsername: strings.TrimSpace(c.GetString("username")),
		TargetAdminID:    &admin.ID,
		TargetUsername:   admin.Username,
		Action:           "admin_update",
		RequestID:        strings.TrimSpace(c.GetString("request_id")),
		Detail: models.JSON{
			"target_admin_id": admin.ID,
			"target_username": admin.Username,
			"updated_fields":  updatedFields,
			"is_super":        admin.IsSuper,
		},
	})

	logger.Infow("admin_authz_admin_updated",
		"operator_admin_id", c.GetUint("admin_id"),
		"target_admin_id", admin.ID,
		"target_username", admin.Username,
		"updated_fields", updatedFields,
	)

	response.Success(c, admin)
}

// DeleteAuthzAdmin ?????
func (h *Handler) DeleteAuthzAdmin(c *gin.Context) {
	adminID, ok := parseAdminIDParam(c)
	if !ok {
		return
	}

	admin, err := h.AdminRepo.GetByID(adminID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_delete_failed", err)
		return
	}
	if admin == nil {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_id_invalid", nil)
		return
	}
	if c.GetUint("admin_id") == adminID {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_delete_self_forbidden", nil)
		return
	}
	if strings.EqualFold(strings.TrimSpace(admin.Username), protectedSuperAdminUsername) {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_delete_protected", nil)
		return
	}

	count, err := h.AdminRepo.Count()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_delete_failed", err)
		return
	}
	if count <= 1 {
		shared.RespondError(c, response.CodeBadRequest, "error.admin_delete_last_forbidden", nil)
		return
	}

	if err := h.AuthzService.SetAdminRoles(adminID, []string{}); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_delete_failed", err)
		return
	}
	if err := h.AdminRepo.Delete(adminID); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.admin_delete_failed", err)
		return
	}
	_ = cache.DelAdminAuthState(c.Request.Context(), adminID)

	h.recordAuthzAudit(c, service.AuthzAuditRecordInput{
		OperatorAdminID:  c.GetUint("admin_id"),
		OperatorUsername: strings.TrimSpace(c.GetString("username")),
		TargetAdminID:    &adminID,
		TargetUsername:   admin.Username,
		Action:           "admin_delete",
		RequestID:        strings.TrimSpace(c.GetString("request_id")),
		Detail: models.JSON{
			"target_admin_id": adminID,
			"target_username": admin.Username,
		},
	})

	logger.Infow("admin_authz_admin_deleted",
		"operator_admin_id", c.GetUint("admin_id"),
		"target_admin_id", adminID,
		"target_username", admin.Username,
	)

	response.Success(c, nil)
}

func normalizeAdminUsername(username string) (string, error) {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "", fmt.Errorf("username is required")
	}
	if strings.ContainsAny(trimmed, " \t\r\n") {
		return "", fmt.Errorf("username contains whitespace")
	}
	length := len([]rune(trimmed))
	if length < 3 || length > 64 {
		return "", fmt.Errorf("username length out of range")
	}
	return trimmed, nil
}

func respondAdminPasswordPolicyError(c *gin.Context, err error) bool {
	if err == nil || !errors.Is(err, service.ErrWeakPassword) {
		return false
	}
	if perr, ok := err.(interface {
		Key() string
		Args() []interface{}
	}); ok {
		msg := i18n.Sprintf(i18n.ResolveLocale(c), perr.Key(), perr.Args()...)
		shared.RespondErrorWithMsg(c, response.CodeBadRequest, msg, nil)
		return true
	}
	shared.RespondError(c, response.CodeBadRequest, "error.password_weak", nil)
	return true
}
