package service

import (
	"testing"

	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/telegramidentity"
)

func TestBuildTelegramPlaceholderEmail(t *testing.T) {
	email := telegramidentity.BuildPlaceholderEmail("123456")
	if email != "telegram_123456@login.local" {
		t.Fatalf("unexpected placeholder email: %s", email)
	}
}

func TestIsTelegramPlaceholderEmail(t *testing.T) {
	cases := []struct {
		name     string
		email    string
		expected bool
	}{
		{name: "valid", email: "telegram_123@login.local", expected: true},
		{name: "valid uppercase", email: "Telegram_123@Login.Local", expected: true},
		{name: "invalid prefix", email: "tg_123@login.local", expected: false},
		{name: "invalid domain", email: "telegram_123@example.com", expected: false},
		{name: "empty", email: "", expected: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := telegramidentity.IsPlaceholderEmail(tc.email)
			if actual != tc.expected {
				t.Fatalf("isTelegramPlaceholderEmail(%q) = %v, want %v", tc.email, actual, tc.expected)
			}
		})
	}
}

func TestResolvePasswordChangeMode(t *testing.T) {
	svc := &UserAuthService{}

	mode, err := svc.ResolvePasswordChangeMode(&models.User{
		Email:                 "telegram_1@login.local",
		PasswordSetupRequired: true,
	})
	if err != nil {
		t.Fatalf("ResolvePasswordChangeMode returned error: %v", err)
	}
	if mode != PasswordChangeModeSetWithoutOld {
		t.Fatalf("unexpected mode for telegram placeholder user: %s", mode)
	}

	mode, err = svc.ResolvePasswordChangeMode(&models.User{
		Email:                 "user@example.com",
		PasswordSetupRequired: false,
	})
	if err != nil {
		t.Fatalf("ResolvePasswordChangeMode returned error: %v", err)
	}
	if mode != PasswordChangeModeChangeWithOld {
		t.Fatalf("unexpected mode for normal user: %s", mode)
	}
}
