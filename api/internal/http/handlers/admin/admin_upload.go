package admin

import (
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/logger"

	"github.com/gin-gonic/gin"
)

// ====================  ????  ====================

// UploadFile ????
func (h *Handler) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		shared.RespondError(c, response.CodeBadRequest, "error.file_missing", nil)
		return
	}
	scene := c.DefaultPostForm("scene", "common")

	// ??????????
	result, err := h.UploadService.SaveFileWithMeta(file, scene)
	if err != nil {
		shared.RespondError(c, response.CodeInternal, "error.upload_failed", err)
		return
	}

	// ??????
	var mediaID uint
	media, err := h.MediaService.RecordMedia(result, scene)
	if err != nil {
		logger.Warnw("upload_record_media_failed", "error", err, "url", result.URL)
	} else if media != nil {
		mediaID = media.ID
	}

	response.Success(c, gin.H{
		"url":      result.URL,
		"filename": result.Filename,
		"size":     result.Size,
		"media_id": mediaID,
	})
}
