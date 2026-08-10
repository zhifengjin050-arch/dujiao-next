package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	responseMsgSuccess = "success"
)

// Response ??????
type Response struct {
	StatusCode int         `json:"status_code"` // ?????
	Msg        string      `json:"msg"`         // ????
	Data       interface{} `json:"data"`        // ????
}

// PageResponse ??????
type PageResponse struct {
	StatusCode int         `json:"status_code"`
	Msg        string      `json:"msg"`
	Data       interface{} `json:"data"`
	Pagination Pagination  `json:"pagination"`
}

// ChannelResponse ?? API ?????
type ChannelResponse struct {
	StatusCode int         `json:"status_code"`
	Msg        string      `json:"msg"`
	Data       interface{} `json:"data,omitempty"`
	ErrorCode  string      `json:"error_code,omitempty"`
	RequestID  string      `json:"request_id,omitempty"`
}

// Pagination ????
type Pagination struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	Total     int64 `json:"total"`
	TotalPage int64 `json:"total_page"`
}

// NormalizePage ???????,????????????
func NormalizePage(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return page, pageSize
}

// BuildPagination ?????????
func BuildPagination(page, pageSize int, total int64) Pagination {
	page, pageSize = NormalizePage(page, pageSize)
	return Pagination{
		Page:      page,
		PageSize:  pageSize,
		Total:     total,
		TotalPage: (total + int64(pageSize) - 1) / int64(pageSize),
	}
}

// Success ????
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		StatusCode: 0,
		Msg:        responseMsgSuccess,
		Data:       data,
	})
}

// SuccessWithPage ??????
func SuccessWithPage(c *gin.Context, data interface{}, pagination Pagination) {
	c.JSON(http.StatusOK, PageResponse{
		StatusCode: 0,
		Msg:        responseMsgSuccess,
		Data:       data,
		Pagination: pagination,
	})
}

// Error ????
func Error(c *gin.Context, statusCode int, msg string) {
	c.JSON(http.StatusOK, Response{
		StatusCode: statusCode,
		Msg:        msg,
		Data:       attachRequestID(c, nil),
	})
}

// Unauthorized 401??
func Unauthorized(c *gin.Context, msg string) {
	Error(c, CodeUnauthorized, msg)
}

// Forbidden 403??
func Forbidden(c *gin.Context, msg string) {
	Error(c, CodeForbidden, msg)
}

// ChannelSuccess ?? API ?????
func ChannelSuccess(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, ChannelResponse{
		StatusCode: CodeOK,
		Msg:        responseMsgSuccess,
		Data:       data,
		RequestID:  currentRequestID(c),
	})
}

// ChannelError ?? API ?????
func ChannelError(c *gin.Context, httpStatus, statusCode int, msg, errorCode string) {
	c.JSON(httpStatus, ChannelResponse{
		StatusCode: statusCode,
		Msg:        msg,
		ErrorCode:  errorCode,
		RequestID:  currentRequestID(c),
	})
}

func attachRequestID(c *gin.Context, data interface{}) interface{} {
	requestID := currentRequestID(c)
	if requestID == "" {
		return data
	}
	if data == nil {
		return gin.H{"request_id": requestID}
	}
	switch v := data.(type) {
	case gin.H:
		if _, ok := v["request_id"]; !ok {
			v["request_id"] = requestID
		}
		return v
	case map[string]interface{}:
		if _, ok := v["request_id"]; !ok {
			v["request_id"] = requestID
		}
		return v
	default:
		return gin.H{
			"request_id": requestID,
			"data":       data,
		}
	}
}

func currentRequestID(c *gin.Context) string {
	if c == nil {
		return ""
	}
	if value, ok := c.Get("request_id"); ok {
		if id, ok := value.(string); ok {
			return id
		}
	}
	return ""
}
