package public

import (
	"errors"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetMyApiCredential ????? API ????
func (h *Handler) GetMyApiCredential(c *gin.Context) {
	userID, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	cred, err := h.ApiCredentialService.GetByUserID(userID)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.api_credential_fetch_failed", err)
		return
	}

	if cred == nil {
		response.Success(c, gin.H{"status": "none"})
		return
	}

	result := gin.H{
		"id":         cred.ID,
		"status":     cred.Status,
		"is_active":  cred.IsActive,
		"created_at": cred.CreatedAt,
	}

	if cred.Status == constants.ApiCredentialStatusRejected {
		result["reject_reason"] = cred.RejectReason
	}

	if cred.Status == constants.ApiCredentialStatusApproved {
		result["api_key"] = cred.ApiKey
		result["approved_at"] = cred.ApprovedAt
		result["last_used_at"] = cred.LastUsedAt
		// Secret ? 4 ?(????)
		if len(cred.ApiSecret) >= 4 {
			result["api_secret_tail"] = cred.ApiSecret[len(cred.ApiSecret)-4:]
		}
	}

	response.Success(c, result)
}

// ApplyApiCredential ?? API ????
func (h *Handler) ApplyApiCredential(c *gin.Context) {
	userID, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	cred, err := h.ApiCredentialService.Apply(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApiCredentialExists):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, "API credential already exists", nil)
		case errors.Is(err, service.ErrApiCredentialPendingExist):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, "Application is pending review", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.api_credential_apply_failed", err)
		}
		return
	}

	response.Success(c, gin.H{
		"id":     cred.ID,
		"status": cred.Status,
	})
}

// RegenerateMyApiCredential ???? Secret
func (h *Handler) RegenerateMyApiCredential(c *gin.Context) {
	userID, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	newSecret, err := h.ApiCredentialService.RegenerateByUserID(userID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrApiCredentialNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
		case errors.Is(err, service.ErrApiCredentialNotApproved):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, "API credential is not approved", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.api_credential_regenerate_failed", err)
		}
		return
	}

	response.Success(c, gin.H{
		"api_secret": newSecret,
	})
}

// UpdateMyApiCredentialStatusRequest ????????
type UpdateMyApiCredentialStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateMyApiCredentialStatus ??/???????
func (h *Handler) UpdateMyApiCredentialStatus(c *gin.Context) {
	userID, ok := shared.GetUserID(c)
	if !ok {
		return
	}

	var req UpdateMyApiCredentialStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.ApiCredentialService.SetActiveByUserID(userID, req.IsActive); err != nil {
		switch {
		case errors.Is(err, service.ErrApiCredentialNotFound):
			shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
		case errors.Is(err, service.ErrApiCredentialNotApproved):
			shared.RespondErrorWithMsg(c, response.CodeBadRequest, "API credential is not approved", nil)
		default:
			shared.RespondError(c, response.CodeInternal, "error.api_credential_update_failed", err)
		}
		return
	}

	response.Success(c, gin.H{"updated": true})
}
