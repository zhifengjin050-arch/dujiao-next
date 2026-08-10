package admin

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/dujiao-next/internal/http/response"
	"github.com/gin-gonic/gin"
)

// GetAdRender ?????????? ad-system
func (h *Handler) GetAdRender(c *gin.Context) {
	slotCode := c.Param("slotCode")
	if slotCode == "" {
		response.Error(c, http.StatusBadRequest, "slot_code is required")
		return
	}

	params := make(map[string]string)
	for _, key := range []string{"tenant", "client", "locale"} {
		if v := c.Query(key); v != "" {
			params[key] = v
		}
	}

	data, err := h.AdProxyService.RenderSlot(c.Request.Context(), slotCode, params)
	if err != nil {
		// ??????????????,??????
		response.Success(c, nil)
		return
	}

	response.Success(c, data)
}

// PostAdImpression ????????? ad-system
func (h *Handler) PostAdImpression(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.AdProxyService.ReportImpression(c.Request.Context(), json.RawMessage(body)); err != nil {
		// ????????????
		response.Success(c, nil)
		return
	}

	response.Success(c, nil)
}
