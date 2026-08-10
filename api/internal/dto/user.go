package dto

import (
	"time"

	"github.com/dujiao-next/internal/models"
)

// UserProfileResp ??????
type UserProfileResp struct {
	ID                 uint         `json:"id"`
	Email              string       `json:"email"`
	Nickname           string       `json:"nickname"`
	EmailVerifiedAt    *time.Time   `json:"email_verified_at"`
	Locale             string       `json:"locale"`
	MemberLevelID      uint         `json:"member_level_id"`
	TotalRecharged     models.Money `json:"total_recharged"`
	TotalSpent         models.Money `json:"total_spent"`
	EmailChangeMode    string       `json:"email_change_mode,omitempty"`
	PasswordChangeMode string       `json:"password_change_mode,omitempty"`
}

// NewUserProfileResp ? models.User ????????
func NewUserProfileResp(user *models.User, emailMode, passwordMode string) UserProfileResp {
	if user == nil {
		return UserProfileResp{}
	}
	return UserProfileResp{
		ID:                 user.ID,
		Email:              user.Email,
		Nickname:           user.DisplayName,
		EmailVerifiedAt:    user.EmailVerifiedAt,
		Locale:             user.Locale,
		MemberLevelID:      user.MemberLevelID,
		TotalRecharged:     user.TotalRecharged,
		TotalSpent:         user.TotalSpent,
		EmailChangeMode:    emailMode,
		PasswordChangeMode: passwordMode,
	}
	// ??:PasswordHash?PasswordSetupRequired?Status?TokenVersion?TokenInvalidBefore?
	// LastLoginAt?CreatedAt?UpdatedAt?DeletedAt
}

// TelegramBindingResp Telegram ??????
type TelegramBindingResp struct {
	Bound          bool       `json:"bound"`
	Provider       string     `json:"provider,omitempty"`
	ProviderUserID string     `json:"provider_user_id,omitempty"`
	Username       string     `json:"username,omitempty"`
	AvatarURL      string     `json:"avatar_url,omitempty"`
	AuthAt         *time.Time `json:"auth_at,omitempty"`
}

// NewTelegramBindingResp ? models.UserOAuthIdentity ????
func NewTelegramBindingResp(identity *models.UserOAuthIdentity) TelegramBindingResp {
	if identity == nil {
		return TelegramBindingResp{Bound: false}
	}
	return TelegramBindingResp{
		Bound:          true,
		Provider:       identity.Provider,
		ProviderUserID: identity.ProviderUserID,
		Username:       identity.Username,
		AvatarURL:      identity.AvatarURL,
		AuthAt:         identity.AuthAt,
	}
	// ??:ID?UserID?CreatedAt?UpdatedAt
}

// UserAuthBriefResp ??/???????????
type UserAuthBriefResp struct {
	ID              uint       `json:"id"`
	Email           string     `json:"email"`
	Nickname        string     `json:"nickname"`
	EmailVerifiedAt *time.Time `json:"email_verified_at"`
}

// NewUserAuthBriefResp ? models.User ????/??????
func NewUserAuthBriefResp(user *models.User) UserAuthBriefResp {
	return UserAuthBriefResp{
		ID:              user.ID,
		Email:           user.Email,
		Nickname:        user.DisplayName,
		EmailVerifiedAt: user.EmailVerifiedAt,
	}
}
