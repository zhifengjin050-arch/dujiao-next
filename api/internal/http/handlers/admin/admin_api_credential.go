package admin

import (
	"errors"
	"strconv"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/repository"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetApiCredentials ?? API ????
func (h *Handler) GetApiCredentials(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	status := c.Query("status")
	search := c.Query("search")
	userID, _ := shared.ParseQueryUint(c.Query("user_id"), false)

	creds, total, err := h.ApiCredentialService.List(repository.ApiCredentialListFilter{
		Status: status,
		UserID: userID,
		Search: search,
		Pagination: repository.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.api_credential_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, creds, pagination)
}

// GetApiCredential ?? API ????
func (h *Handler) GetApiCredential(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	cred, err := h.ApiCredentialService.GetByID(id)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.api_credential_fetch_failed", err)
		return
	}
	if cred == nil {
		shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
		return
	}

	response.Success(c, cred)
}

// ApproveApiCredential ???? API ??
func (h *Handler) ApproveApiCredential(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	cred, _, err := h.ApiCredentialService.Approve(id)
	if err != nil {
		if errors.Is(err, service.ErrApiCredentialNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.api_credential_approve_failed", err)
		return
	}

	response.Success(c, gin.H{
		"credential": cred,
		"approved":   true,
	})
}

// RejectApiCredentialRequest ??????
type RejectApiCredentialRequest struct {
	Reason string `json:"reason" binding:"required"`
}

// RejectApiCredential ???? API ??
func (h *Handler) RejectApiCredential(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var req RejectApiCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.ApiCredentialService.Reject(id, req.Reason); err != nil {
		if errors.Is(err, service.ErrApiCredentialNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.api_credential_reject_failed", err)
		return
	}

	response.Success(c, gin.H{"rejected": true})
}

// UpdateApiCredentialStatusRequest ????????
type UpdateApiCredentialStatusRequest struct {
	IsActive bool `json:"is_active"`
}

// UpdateApiCredentialStatus ??/?? API ??
func (h *Handler) UpdateApiCredentialStatus(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var req UpdateApiCredentialStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.ApiCredentialService.SetActive(id, req.IsActive); err != nil {
		if errors.Is(err, service.ErrApiCredentialNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.api_credential_not_found", nil)
			return
		}
		if errors.Is(err, service.ErrApiCredentialNotApproved) {
			shared.RespondError(c, response.CodeBadRequest, "error.api_credential_not_approved", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.api_credential_update_failed", err)
		return
	}

	response.Success(c, gin.H{"updated": true})
}

// DeleteApiCredential ?? API ??
func (h *Handler) DeleteApiCredential(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	if err := h.ApiCredentialService.Delete(id); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.api_credential_delete_failed", err)
		return
	}

	response.Success(c, gin.H{"deleted": true})
}
