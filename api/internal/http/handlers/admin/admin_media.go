package admin

import (
	"strconv"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"

	"github.com/gin-gonic/gin"
)

// ====================  ????  ====================

// GetAdminMedia ????
func (h *Handler) GetAdminMedia(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	scene := c.Query("scene")
	search := c.Query("search")

	items, total, err := h.MediaService.List(scene, search, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.internal", err)
		return
	}

	response.Success(c, gin.H{
		"items": items,
		"total": total,
	})
}

// UpdateMedia ??????(???)
func (h *Handler) UpdateMedia(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.invalid_id", nil)
		return
	}

	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.invalid_params", nil)
		return
	}

	if err := h.MediaService.Rename(uint(id), req.Name); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.internal", err)
		return
	}

	response.Success(c, nil)
}

// DeleteMedia ????
func (h *Handler) DeleteMedia(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.invalid_id", nil)
		return
	}

	if err := h.MediaService.Delete(uint(id)); err != nil {
		shared.RespondError(c, response.CodeInternal, "error.internal", err)
		return
	}

	response.Success(c, nil)
}
