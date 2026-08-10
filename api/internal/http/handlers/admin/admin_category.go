package admin

import (
	"errors"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAdminCategories ?????? (Admin)
func (h *Handler) GetAdminCategories(c *gin.Context) {
	categories, err := h.CategoryService.List()
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.category_fetch_failed", err)
		return
	}

	response.Success(c, categories)
}

// ====================  ????  ====================

// CreateCategoryRequest ??????
type CreateCategoryRequest struct {
	ParentID  uint                   `json:"parent_id"`
	Slug      string                 `json:"slug" binding:"required"`
	NameJSON  map[string]interface{} `json:"name" binding:"required"`
	Icon      string                 `json:"icon"`
	SortOrder int                    `json:"sort_order"`
}

// CreateCategory ????
func (h *Handler) CreateCategory(c *gin.Context) {
	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	category, err := h.CategoryService.Create(service.CreateCategoryInput{
		ParentID:  req.ParentID,
		Slug:      req.Slug,
		NameJSON:  req.NameJSON,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		if errors.Is(err, service.ErrSlugExists) {
			shared.RespondError(c, response.CodeBadRequest, "error.slug_exists", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryParentInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.category_parent_invalid", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.category_create_failed", err)
		return
	}

	response.Success(c, category)
}

// UpdateCategory ????
func (h *Handler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")

	var req CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	category, err := h.CategoryService.Update(id, service.CreateCategoryInput{
		ParentID:  req.ParentID,
		Slug:      req.Slug,
		NameJSON:  req.NameJSON,
		Icon:      req.Icon,
		SortOrder: req.SortOrder,
	})
	if err != nil {
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.category_not_found", nil)
			return
		}
		if errors.Is(err, service.ErrSlugExists) {
			shared.RespondError(c, response.CodeBadRequest, "error.slug_used", nil)
			return
		}
		if errors.Is(err, service.ErrCategoryParentInvalid) {
			shared.RespondError(c, response.CodeBadRequest, "error.category_parent_invalid", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.category_update_failed", err)
		return
	}

	response.Success(c, category)
}

// DeleteCategory ????(???)
func (h *Handler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	if err := h.CategoryService.Delete(id); err != nil {
		if errors.Is(err, service.ErrCategoryInUse) {
			shared.RespondError(c, response.CodeBadRequest, "error.category_in_use", nil)
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.category_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.category_delete_failed", err)
		return
	}

	response.Success(c, nil)
}
