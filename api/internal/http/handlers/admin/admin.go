package admin

import (
	"errors"
	"log"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/http/handlers/shared"
	"github.com/dujiao-next/internal/http/response"
	"github.com/dujiao-next/internal/i18n"
	"github.com/dujiao-next/internal/service"

	"github.com/gin-gonic/gin"
)

// LoginRequest ????
type LoginRequest struct {
	Username       string                       `json:"username" binding:"required"`
	Password       string                       `json:"password" binding:"required"`
	CaptchaPayload shared.CaptchaPayloadRequest `json:"captcha_payload"`
}

// LoginResponse ????
type LoginResponse struct {
	Token     string                 `json:"token"`
	User      map[string]interface{} `json:"user"`
	ExpiresAt string                 `json:"expires_at"`
}

// AdminLogin ?????
func (h *Handler) AdminLogin(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if h.CaptchaService != nil {
		if captchaErr := h.CaptchaService.Verify(constants.CaptchaSceneLogin, req.CaptchaPayload.ToServicePayload(), c.ClientIP()); captchaErr != nil {
			switch {
			case errors.Is(captchaErr, service.ErrCaptchaRequired):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_required", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaInvalid):
				shared.RespondError(c, response.CodeBadRequest, "error.captcha_invalid", nil)
				return
			case errors.Is(captchaErr, service.ErrCaptchaConfigInvalid):
				shared.RespondError(c, response.CodeInternal, "error.captcha_config_invalid", captchaErr)
				return
			default:
				shared.RespondError(c, response.CodeInternal, "error.captcha_verify_failed", captchaErr)
				return
			}
		}
	}

	admin, token, expiresAt, err := h.AuthService.Login(req.Username, req.Password)
	if err != nil {
		log.Printf("admin login error detail: username=%s error=%v", req.Username, err)
		if errors.Is(err, service.ErrInvalidCredentials) {
			shared.RespondError(c, response.CodeUnauthorized, "error.admin_login_invalid", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.login_failed", err)
		return
	}
	response.Success(c, LoginResponse{
		Token: token,
		User: map[string]interface{}{
			"id":       admin.ID,
			"username": admin.Username,
		},
		ExpiresAt: expiresAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

// UpdatePasswordRequest ??????
type UpdatePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

// UpdateAdminPassword ???????
func (h *Handler) UpdateAdminPassword(c *gin.Context) {
	// ???????? ID
	id, ok := shared.GetAdminID(c)
	if !ok {
		return
	}

	var req UpdatePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		shared.RespondBindError(c, err)
		return
	}

	if err := h.AuthService.ChangePassword(id, req.OldPassword, req.NewPassword); err != nil {
		if errors.Is(err, service.ErrInvalidPassword) {
			shared.RespondError(c, response.CodeBadRequest, "error.password_old_invalid", nil)
			return
		}
		if errors.Is(err, service.ErrWeakPassword) {
			locale := i18n.ResolveLocale(c)
			if perr, ok := err.(interface {
				Key() string
				Args() []interface{}
			}); ok {
				msg := i18n.Sprintf(locale, perr.Key(), perr.Args()...)
				shared.RespondErrorWithMsg(c, response.CodeBadRequest, msg, nil)
				return
			}
			shared.RespondError(c, response.CodeBadRequest, "error.password_weak", nil)
			return
		}
		if errors.Is(err, service.ErrNotFound) {
			shared.RespondError(c, response.CodeNotFound, "error.user_not_found", nil)
			return
		}
		shared.RespondError(c, response.CodeInternal, "error.save_failed", err)
		return
	}

	response.Success(c, nil)
}
