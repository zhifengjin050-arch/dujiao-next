package dto

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/dujiao-next/internal/models"
)

func TestUserProfileRespOmitsSensitiveFields(t *testing.T) {
	now := time.Now()
	user := &models.User{
		ID:                    1,
		Email:                 "user@test.com",
		PasswordHash:          "hashed-secret",
		PasswordSetupRequired: true,
		DisplayName:           "Test User",
		Locale:                "zh-CN",
		Status:                "active",
		MemberLevelID:         2,
		TotalRecharged:        newMoney("100.00"),
		TotalSpent:            newMoney("50.00"),
		TokenVersion:          5,
		TokenInvalidBefore:    &now,
		EmailVerifiedAt:       &now,
		LastLoginAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}

	resp := NewUserProfileResp(user, "bind_only", "set_without_old")
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	sensitiveFields := []string{
		"password_hash", "password_setup_required", "token_version",
		"token_invalid_before", "last_login_at", "created_at", "updated_at",
		"status",
	}
	for _, field := range sensitiveFields {
		if strings.Contains(jsonStr, `"`+field+`"`) {
			t.Errorf("sensitive field %q should not appear", field)
		}
	}

	if !strings.Contains(jsonStr, `"email"`) {
		t.Error("email should appear")
	}
	if !strings.Contains(jsonStr, `"nickname"`) {
		t.Error("nickname should appear")
	}
	if resp.Nickname != "Test User" {
		t.Errorf("expected nickname=Test User, got %s", resp.Nickname)
	}
}

func TestUserProfileRespNilSafe(t *testing.T) {
	resp := NewUserProfileResp(nil, "", "")
	if resp.ID != 0 {
		t.Errorf("expected zero ID for nil user, got %d", resp.ID)
	}
}

func TestUserAuthBriefRespFields(t *testing.T) {
	user := &models.User{
		ID:          5,
		Email:       "brief@test.com",
		DisplayName: "Brief",
	}

	resp := NewUserAuthBriefResp(user)
	data, _ := json.Marshal(resp)
	jsonStr := string(data)

	if strings.Contains(jsonStr, `"locale"`) {
		t.Error("locale should not appear in brief response")
	}
	if resp.Email != "brief@test.com" {
		t.Errorf("expected email brief@test.com, got %s", resp.Email)
	}
}
