package models

import (
	"strings"

	"github.com/dujiao-next/internal/logger"

	"golang.org/x/crypto/bcrypt"
)

func normalizeBootstrapAdminUsername(username string) string {
	trimmed := strings.TrimSpace(username)
	if trimmed == "" {
		return "admin"
	}
	return trimmed
}

// InitDefaultAdmin ??????????
func InitDefaultAdmin(username, password string) error {
	bootstrapUsername := normalizeBootstrapAdminUsername(username)

	var count int64
	DB.Model(&Admin{}).Count(&count)

	// ???????,?? bootstrap ??????????????
	if count > 0 {
		if err := DB.Model(&Admin{}).Where("username = ?", bootstrapUsername).Update("is_super", true).Error; err != nil {
			logger.Warnw("ensure_default_admin_super_failed", "error", err)
		}
		return nil
	}

	// ???????
	username = bootstrapUsername
	if password == "" {
		password = "admin123"
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	admin := Admin{
		Username:     username,
		PasswordHash: string(hash),
		IsSuper:      true,
	}

	if err := DB.Create(&admin).Error; err != nil {
		return err
	}

	if password == "admin123" {
		logger.Warnw("default_admin_created_with_default_password", "username", username, "password", password)
		logger.Warnw("default_admin_password_change_required", "username", username)
	} else {
		logger.Warnw("default_admin_created", "username", username, "password_hidden", true)
	}

	return nil
}
