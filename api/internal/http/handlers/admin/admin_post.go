package admin

import (
	"errors"
	"strconv"

	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// GetAdminPosts ?????? (Admin)
func (h *Handler) GetAdminPosts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	page, pageSize = shared.NormalizePagination(page, pageSize)
	postType := c.Query("type")
	search := c.Query("search")

	posts, total, err := h.PostService.ListAdmin(postType, search, page, pageSize)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.post_fetch_failed", err)
		return
	}

	pagination := response.BuildPagination(page, pageSize, total)
	response.SuccessWithPage(c, posts, pagination)
}

// ====================  ????  ====================

// CreatePostRequest ??????
type CreatePostRequest struct {
	Slug        string                 `json:"slug" binding:"required"`
	Type        string                 `json:"type" binding:"required"` // blog ? notice
	TitleJSON   map[string]interface{} `json:"title" binding:"required"`
	SummaryJSON map[string]interface{} `json:"summary"`
	ContentJSON map[string]interface{} `json:"content"`
	Thumbnail   string                 `json:"thumbnail"`
	IsPublished *bool                  `json:"is_published"`
}

// CreatePost ????
func (h *Handler) CreatePost(c *gin.Context) {
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	post, err := h.PostService.Create(service.CreatePostInput{
		Slug:        req.Slug,
		Type:        req.Type,
		TitleJSON:   req.TitleJSON,
		SummaryJSON: req.SummaryJSON,
		ContentJSON: req.ContentJSON,
		Thumbnail:   req.Thumbnail,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidPostType) {
			shared.RespondError(c, response.CodeBadRequest, "error.post_type_invalid", nil)
			return
		}
		if errors.Is(err, service.ErrSlugExists) {
			shared.RespondError(c, response.CodeBadRequest, "error.slug_exists", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.post_create_failed", err)
		return
	}

	response.Success(c, post)
}

// UpdatePost ????
func (h *Handler) UpdatePost(c *gin.Context) {
	id := c.Param("id")

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	post, err := h.PostService.Update(id, service.CreatePostInput{
		Slug:        req.Slug,
		Type:        req.Type,
		TitleJSON:   req.TitleJSON,
		SummaryJSON: req.SummaryJSON,
		ContentJSON: req.ContentJSON,
		Thumbnail:   req.Thumbnail,
		IsPublished: req.IsPublished,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidPostType) {
			shared.RespondError(c, response.CodeBadRequest, "error.post_type_invalid", nil)
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.post_not_found", nil)
			return
		}
		if errors.Is(err, service.ErrSlugExists) {
			shared.RespondError(c, response.CodeBadRequest, "error.slug_used", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.post_update_failed", err)
		return
	}

	response.Success(c, post)
}

// DeletePost ????(???)
func (h *Handler) DeletePost(c *gin.Context) {
	id := c.Param("id")

	if err := h.PostService.Delete(id); err != nil {
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.post_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.post_delete_failed", err)
		return
	}

	response.Success(c, nil)
}
