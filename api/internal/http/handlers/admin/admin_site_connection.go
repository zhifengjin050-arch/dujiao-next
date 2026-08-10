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

// GetSiteConnections ????????
func (h *Handler) GetSiteConnections(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)

	conns, total, err := h.SiteConnectionService.List(repository.SiteConnectionListFilter{
		Pagination: repository.Pagination{
			Page:     page,
			PageSize: pageSize,
		},
	})
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.connection_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, conns, pagination)
}

// GetSiteConnection ????????
func (h *Handler) GetSiteConnection(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	conn, err := h.SiteConnectionService.GetByID(id)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.connection_fetch_failed", err)
		return
	}
	if conn == nil {
		shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
		return
	}

	response.Success(c, conn)
}

// CreateSiteConnection ??????
func (h *Handler) CreateSiteConnection(c *gin.Context) {
	var input service.CreateConnectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	conn, err := h.SiteConnectionService.Create(input)
	if err != nil {
		if errors.Is(err, service.ErrConnectionInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.connection_invalid", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.connection_create_failed", err)
		return
	}

	response.Success(c, conn)
}

// UpdateSiteConnection ??????
func (h *Handler) UpdateSiteConnection(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var input service.UpdateConnectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	conn, err := h.SiteConnectionService.Update(id, input)
	if err != nil {
		if errors.Is(err, service.ErrConnectionNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.connection_update_failed", err)
		return
	}

	response.Success(c, conn)
}

// DeleteSiteConnection ??????
func (h *Handler) DeleteSiteConnection(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	if err := h.SiteConnectionService.Delete(id); err != nil {
		if errors.Is(err, service.ErrConnectionNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.connection_delete_failed", err)
		return
	}

	response.Success(c, gin.H{"deleted": true})
}

// PingSiteConnection ??????
func (h *Handler) PingSiteConnection(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	result, err := h.SiteConnectionService.Ping(id)
	if err != nil {
		if errors.Is(err, service.ErrConnectionNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
			return
		}
		shared.RespondErrorWithMsg(c, response.CodeInternal, err.Error(), err)
		return
	}

	response.Success(c, result)
}

// ReapplyConnectionMarkup ??????????????????
func (h *Handler) ReapplyConnectionMarkup(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	count, err := h.ProductMappingService.ReapplyMarkup(id)
	if err != nil {
		if errors.Is(err, service.ErrConnectionNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.reapply_markup_failed", err)
		return
	}

	response.Success(c, gin.H{"updated_products": count})
}

// UpdateSiteConnectionStatusRequest ????????
type UpdateSiteConnectionStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

// UpdateSiteConnectionStatus ??????
func (h *Handler) UpdateSiteConnectionStatus(c *gin.Context) {
	id, err := shared.ParseParamUint(c, "id")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.bad_request", err)
		return
	}

	var req UpdateSiteConnectionStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.SiteConnectionService.SetStatus(id, req.Status); err != nil {
		if errors.Is(err, service.ErrConnectionNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.connection_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.connection_update_failed", err)
		return
	}

	response.Success(c, gin.H{"updated": true})
}
