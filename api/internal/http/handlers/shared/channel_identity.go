package shared

import (
	"github.com/dujiao-next/internal/models"

	"github.com/gin-gonic/gin"
)

// BuildChannelIdentityResponse ?? Telegram ?????????
func BuildChannelIdentityResponse(bound, created bool, user *models.User, identity *models.UserOAuthIdentity) gin.H {
	resp := gin.H{
		"bound": bound,
	}
	if identity != nil {
		resp["identity"] = gin.H{
			"provider":         identity.Provider,
			"provider_user_id": identity.ProviderUserID,
			"username":         identity.Username,
			"avatar_url":       identity.AvatarURL,
		}
	}
	if user != nil {
		resp["user"] = gin.H{
			"id":                      user.ID,
			"email":                   user.Email,
			"display_name":            user.DisplayName,
			"status":                  user.Status,
			"locale":                  user.Locale,
			"email_verified":          user.EmailVerifiedAt != nil,
			"password_setup_required": user.PasswordSetupRequired,
		}
	}
	if bound {
		resp["created"] = created
	}
	return resp
}
