package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/telegramidentity"

	"golang.org/x/crypto/bcrypt"
)

// LoginWithTelegramInput Telegram ????
type LoginWithTelegramInput struct {
	Payload TelegramLoginPayload
	Context context.Context
}

// LoginWithTelegramMiniAppInput Telegram Mini App ????
type LoginWithTelegramMiniAppInput struct {
	InitData string
	Context  context.Context
}

// BindTelegramInput ?? Telegram ??
type BindTelegramInput struct {
	UserID  uint
	Payload TelegramLoginPayload
	Context context.Context
}

// BindTelegramMiniAppInput Telegram Mini App ????
type BindTelegramMiniAppInput struct {
	UserID   uint
	InitData string
	Context  context.Context
}

// TelegramChannelIdentityInput Telegram ??????
type TelegramChannelIdentityInput struct {
	ChannelUserID string
	Username      string
	FirstName     string
	LastName      string
	AvatarURL     string
}

// BindTelegramChannelByEmailCodeInput Telegram ???????????
type BindTelegramChannelByEmailCodeInput struct {
	Identity TelegramChannelIdentityInput
	Email    string
	Code     string
}

// LoginWithTelegram Telegram ??
func (s *UserAuthService) LoginWithTelegram(input LoginWithTelegramInput) (*models.User, string, time.Time, error) {
	if s.telegramAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, "", time.Time{}, ErrTelegramAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.telegramAuthService.VerifyLogin(ctx, input.Payload)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return s.loginWithVerifiedTelegram(verified)
}

// LoginWithTelegramMiniApp Telegram Mini App ??
func (s *UserAuthService) LoginWithTelegramMiniApp(input LoginWithTelegramMiniAppInput) (*models.User, string, time.Time, error) {
	if s.telegramAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, "", time.Time{}, ErrTelegramAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.telegramAuthService.VerifyMiniAppInitData(ctx, input.InitData)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return s.loginWithVerifiedTelegram(verified)
}

func (s *UserAuthService) loginWithVerifiedTelegram(verified *TelegramIdentityVerified) (*models.User, string, time.Time, error) {
	identity, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	var user *models.User
	if identity != nil {
		user, err = s.getActiveUserByID(identity.UserID)
		if err != nil {
			return nil, "", time.Time{}, err
		}
		identityChanged := applyTelegramIdentity(verified, identity)
		if identityChanged {
			identity.UpdatedAt = time.Now()
			if err := s.userOAuthIdentityRepo.Update(identity); err != nil {
				return nil, "", time.Time{}, err
			}
		}
	} else {
		user, err = s.findOrCreateTelegramUser(verified)
		if err != nil {
			return nil, "", time.Time{}, err
		}
		identity = &models.UserOAuthIdentity{
			UserID:         user.ID,
			Provider:       verified.Provider,
			ProviderUserID: verified.ProviderUserID,
			Username:       verified.Username,
			AvatarURL:      verified.AvatarURL,
			AuthAt:         &verified.AuthAt,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.userOAuthIdentityRepo.Create(identity); err != nil {
			existing, getErr := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
			if getErr != nil {
				return nil, "", time.Time{}, err
			}
			if existing == nil {
				return nil, "", time.Time{}, err
			}
			identity = existing
			user, err = s.getActiveUserByID(existing.UserID)
			if err != nil {
				return nil, "", time.Time{}, err
			}
		}
	}

	token, expiresAt, err := s.GenerateUserJWT(user, 0)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	user.UpdatedAt = now
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", time.Time{}, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))
	return user, token, expiresAt, nil
}

// BindTelegram ?? Telegram
func (s *UserAuthService) BindTelegram(input BindTelegramInput) (*models.UserOAuthIdentity, error) {
	if input.UserID == 0 {
		return nil, ErrNotFound
	}
	if s.telegramAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, ErrTelegramAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.telegramAuthService.VerifyLogin(ctx, input.Payload)
	if err != nil {
		return nil, err
	}
	return s.bindVerifiedTelegram(input.UserID, verified)
}

// BindTelegramMiniApp ??????? Telegram Mini App ??
func (s *UserAuthService) BindTelegramMiniApp(input BindTelegramMiniAppInput) (*models.UserOAuthIdentity, error) {
	if input.UserID == 0 {
		return nil, ErrNotFound
	}
	if s.telegramAuthService == nil || s.userOAuthIdentityRepo == nil {
		return nil, ErrTelegramAuthConfigInvalid
	}
	ctx := input.Context
	if ctx == nil {
		ctx = context.Background()
	}
	verified, err := s.telegramAuthService.VerifyMiniAppInitData(ctx, input.InitData)
	if err != nil {
		return nil, err
	}
	return s.bindVerifiedTelegram(input.UserID, verified)
}

func (s *UserAuthService) bindVerifiedTelegram(userID uint, verified *TelegramIdentityVerified) (*models.UserOAuthIdentity, error) {
	if _, err := s.getActiveUserByID(userID); err != nil {
		return nil, err
	}

	occupied, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, err
	}
	if occupied != nil && occupied.UserID != userID {
		return nil, ErrUserOAuthIdentityExists
	}

	current, err := s.userOAuthIdentityRepo.GetByUserProvider(userID, verified.Provider)
	if err != nil {
		return nil, err
	}
	if current != nil && current.ProviderUserID != verified.ProviderUserID {
		return nil, ErrUserOAuthAlreadyBound
	}
	if current == nil {
		current = &models.UserOAuthIdentity{
			UserID:         userID,
			Provider:       verified.Provider,
			ProviderUserID: verified.ProviderUserID,
			Username:       verified.Username,
			AvatarURL:      verified.AvatarURL,
			AuthAt:         &verified.AuthAt,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		if err := s.userOAuthIdentityRepo.Create(current); err != nil {
			return nil, err
		}
		return current, nil
	}

	if applyTelegramIdentity(verified, current) {
		current.UpdatedAt = time.Now()
		if err := s.userOAuthIdentityRepo.Update(current); err != nil {
			return nil, err
		}
	}
	return current, nil
}

// UnbindTelegram ?? Telegram
func (s *UserAuthService) UnbindTelegram(userID uint) error {
	if userID == 0 {
		return ErrNotFound
	}
	if s.userOAuthIdentityRepo == nil {
		return ErrTelegramAuthConfigInvalid
	}
	user, err := s.getActiveUserByID(userID)
	if err != nil {
		return err
	}
	mode, err := s.ResolveEmailChangeMode(user)
	if err != nil {
		return err
	}
	if mode == EmailChangeModeBindOnly {
		return ErrTelegramUnbindRequiresEmail
	}
	identity, err := s.userOAuthIdentityRepo.GetByUserProvider(userID, constants.UserOAuthProviderTelegram)
	if err != nil {
		return err
	}
	if identity == nil {
		return ErrUserOAuthNotBound
	}
	return s.userOAuthIdentityRepo.DeleteByID(identity.ID)
}

// GetTelegramBinding ?? Telegram ??
func (s *UserAuthService) GetTelegramBinding(userID uint) (*models.UserOAuthIdentity, error) {
	if userID == 0 {
		return nil, ErrNotFound
	}
	if s.userOAuthIdentityRepo == nil {
		return nil, ErrTelegramAuthConfigInvalid
	}
	return s.userOAuthIdentityRepo.GetByUserProvider(userID, constants.UserOAuthProviderTelegram)
}

// ResolveTelegramChannelIdentity ?? Telegram ????
func (s *UserAuthService) ResolveTelegramChannelIdentity(input TelegramChannelIdentityInput) (*models.User, *models.UserOAuthIdentity, error) {
	verified, err := normalizeTelegramChannelIdentityInput(input)
	if err != nil {
		return nil, nil, err
	}
	return s.resolveTelegramChannelIdentity(verified)
}

// ProvisionTelegramChannelIdentity ?? Telegram ????
func (s *UserAuthService) ProvisionTelegramChannelIdentity(input TelegramChannelIdentityInput) (*models.User, *models.UserOAuthIdentity, bool, error) {
	verified, err := normalizeTelegramChannelIdentityInput(input)
	if err != nil {
		return nil, nil, false, err
	}
	return s.provisionTelegramChannelIdentity(verified)
}

// BindTelegramChannelByEmailCode ????????? Telegram ?????????
func (s *UserAuthService) BindTelegramChannelByEmailCode(input BindTelegramChannelByEmailCodeInput) (*models.User, *models.UserOAuthIdentity, uint, error) {
	verified, err := normalizeTelegramChannelIdentityInput(input.Identity)
	if err != nil {
		return nil, nil, 0, err
	}
	if s.userOAuthIdentityRepo == nil || s.userRepo == nil || s.codeRepo == nil {
		return nil, nil, 0, ErrTelegramAuthConfigInvalid
	}

	email, err := normalizeEmail(input.Email)
	if err != nil {
		return nil, nil, 0, err
	}
	if _, err := s.verifyCode(email, constants.VerifyPurposeTelegramBind, input.Code); err != nil {
		return nil, nil, 0, err
	}

	targetUser, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, nil, 0, err
	}
	if targetUser == nil {
		return nil, nil, 0, ErrNotFound
	}
	if strings.ToLower(strings.TrimSpace(targetUser.Status)) != constants.UserStatusActive {
		return nil, nil, 0, ErrUserDisabled
	}

	return s.bindTelegramIdentityToUser(targetUser, verified)
}

func (s *UserAuthService) resolveTelegramChannelIdentity(verified *TelegramIdentityVerified) (*models.User, *models.UserOAuthIdentity, error) {
	if verified == nil {
		return nil, nil, ErrTelegramAuthPayloadInvalid
	}
	if s.userOAuthIdentityRepo == nil || s.userRepo == nil {
		return nil, nil, ErrTelegramAuthConfigInvalid
	}

	identity, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, nil, err
	}
	if identity == nil {
		return nil, nil, nil
	}

	user, err := s.getActiveUserByID(identity.UserID)
	if err != nil {
		return nil, nil, err
	}
	if applyTelegramIdentity(verified, identity) {
		identity.UpdatedAt = time.Now()
		if err := s.userOAuthIdentityRepo.Update(identity); err != nil {
			return nil, nil, err
		}
	}
	return user, identity, nil
}

func (s *UserAuthService) provisionTelegramChannelIdentity(verified *TelegramIdentityVerified) (*models.User, *models.UserOAuthIdentity, bool, error) {
	if verified == nil {
		return nil, nil, false, ErrTelegramAuthPayloadInvalid
	}
	if s.userOAuthIdentityRepo == nil || s.userRepo == nil {
		return nil, nil, false, ErrTelegramAuthConfigInvalid
	}

	user, identity, err := s.resolveTelegramChannelIdentity(verified)
	if err != nil {
		return nil, nil, false, err
	}
	if identity != nil {
		return user, identity, false, nil
	}

	placeholderUser, err := s.userRepo.GetByEmail(telegramidentity.BuildPlaceholderEmail(verified.ProviderUserID))
	if err != nil {
		return nil, nil, false, err
	}
	created := placeholderUser == nil

	user, err = s.findOrCreateTelegramUser(verified)
	if err != nil {
		return nil, nil, false, err
	}

	identity, err = s.userOAuthIdentityRepo.GetByUserProvider(user.ID, verified.Provider)
	if err != nil {
		return nil, nil, false, err
	}
	if identity != nil {
		if identity.ProviderUserID != verified.ProviderUserID {
			return nil, nil, false, ErrUserOAuthAlreadyBound
		}
		if applyTelegramIdentity(verified, identity) {
			identity.UpdatedAt = time.Now()
			if err := s.userOAuthIdentityRepo.Update(identity); err != nil {
				return nil, nil, false, err
			}
		}
		return user, identity, created, nil
	}

	identity = &models.UserOAuthIdentity{
		UserID:         user.ID,
		Provider:       verified.Provider,
		ProviderUserID: verified.ProviderUserID,
		Username:       verified.Username,
		AvatarURL:      verified.AvatarURL,
		AuthAt:         &verified.AuthAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.userOAuthIdentityRepo.Create(identity); err != nil {
		existing, getErr := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
		if getErr != nil {
			return nil, nil, false, err
		}
		if existing == nil {
			return nil, nil, false, err
		}
		identity = existing
		user, err = s.getActiveUserByID(existing.UserID)
		if err != nil {
			return nil, nil, false, err
		}
		return user, identity, false, nil
	}

	return user, identity, created, nil
}

func (s *UserAuthService) bindTelegramIdentityToUser(targetUser *models.User, verified *TelegramIdentityVerified) (*models.User, *models.UserOAuthIdentity, uint, error) {
	if targetUser == nil || verified == nil {
		return nil, nil, 0, ErrNotFound
	}
	if s.userOAuthIdentityRepo == nil {
		return nil, nil, 0, ErrTelegramAuthConfigInvalid
	}

	current, err := s.userOAuthIdentityRepo.GetByUserProvider(targetUser.ID, verified.Provider)
	if err != nil {
		return nil, nil, 0, err
	}
	if current != nil && current.ProviderUserID != verified.ProviderUserID {
		return nil, nil, 0, ErrUserOAuthAlreadyBound
	}

	occupied, err := s.userOAuthIdentityRepo.GetByProviderUserID(verified.Provider, verified.ProviderUserID)
	if err != nil {
		return nil, nil, 0, err
	}
	if occupied != nil && occupied.UserID == targetUser.ID {
		if applyTelegramIdentity(verified, occupied) {
			occupied.UpdatedAt = time.Now()
			if err := s.userOAuthIdentityRepo.Update(occupied); err != nil {
				return nil, nil, 0, err
			}
		}
		return targetUser, occupied, 0, nil
	}

	if occupied != nil {
		previousUser, err := s.userRepo.GetByID(occupied.UserID)
		if err != nil {
			return nil, nil, 0, err
		}
		if previousUser == nil || !telegramidentity.IsPlaceholderEmail(previousUser.Email) {
			return nil, nil, 0, ErrUserOAuthIdentityExists
		}

		previousUserID := occupied.UserID
		occupied.UserID = targetUser.ID
		applyTelegramIdentity(verified, occupied)
		occupied.UpdatedAt = time.Now()
		if err := s.userOAuthIdentityRepo.Update(occupied); err != nil {
			return nil, nil, 0, err
		}
		return targetUser, occupied, previousUserID, nil
	}

	identity := &models.UserOAuthIdentity{
		UserID:         targetUser.ID,
		Provider:       verified.Provider,
		ProviderUserID: verified.ProviderUserID,
		Username:       verified.Username,
		AvatarURL:      verified.AvatarURL,
		AuthAt:         &verified.AuthAt,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	if err := s.userOAuthIdentityRepo.Create(identity); err != nil {
		return nil, nil, 0, err
	}
	return targetUser, identity, 0, nil
}

func (s *UserAuthService) getActiveUserByID(userID uint) (*models.User, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrNotFound
	}
	if strings.ToLower(strings.TrimSpace(user.Status)) != constants.UserStatusActive {
		return nil, ErrUserDisabled
	}
	return user, nil
}

func (s *UserAuthService) findOrCreateTelegramUser(verified *TelegramIdentityVerified) (*models.User, error) {
	if verified == nil {
		return nil, ErrTelegramAuthPayloadInvalid
	}
	email := telegramidentity.BuildPlaceholderEmail(verified.ProviderUserID)
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if user != nil {
		if strings.ToLower(strings.TrimSpace(user.Status)) != constants.UserStatusActive {
			return nil, ErrUserDisabled
		}
		return user, nil
	}
	if s.settingService != nil {
		registrationEnabled, err := s.settingService.GetRegistrationEnabled(true)
		if err != nil {
			return nil, err
		}
		if !registrationEnabled {
			return nil, ErrRegistrationDisabled
		}
	}

	randomSuffix, err := randomNumericCode(16)
	if err != nil {
		return nil, err
	}
	passwordSeed := fmt.Sprintf("tg_%s_%s", verified.ProviderUserID, randomSuffix)
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(passwordSeed), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	user = &models.User{
		Email:                 email,
		PasswordHash:          string(hashedPassword),
		PasswordSetupRequired: true,
		DisplayName:           telegramidentity.ResolveDisplayName(verified.ProviderUserID, verified.Username, verified.FirstName, verified.LastName),
		Status:                constants.UserStatusActive,
		LastLoginAt:           &now,
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}
	// ????????
	if s.memberLevelSvc != nil {
		_ = s.memberLevelSvc.AssignDefaultLevel(user.ID)
	}
	return user, nil
}

func applyTelegramIdentity(verified *TelegramIdentityVerified, identity *models.UserOAuthIdentity) bool {
	if verified == nil || identity == nil {
		return false
	}
	changed := false
	if identity.Provider == "" {
		identity.Provider = verified.Provider
		changed = true
	}
	if identity.ProviderUserID == "" {
		identity.ProviderUserID = verified.ProviderUserID
		changed = true
	}
	if identity.Username != verified.Username {
		identity.Username = verified.Username
		changed = true
	}
	if identity.AvatarURL != verified.AvatarURL {
		identity.AvatarURL = verified.AvatarURL
		changed = true
	}
	if identity.AuthAt == nil || !identity.AuthAt.Equal(verified.AuthAt) {
		authAt := verified.AuthAt
		identity.AuthAt = &authAt
		changed = true
	}
	return changed
}

func normalizeTelegramChannelIdentityInput(input TelegramChannelIdentityInput) (*TelegramIdentityVerified, error) {
	providerUserID := strings.TrimSpace(input.ChannelUserID)
	if providerUserID == "" {
		return nil, ErrTelegramAuthPayloadInvalid
	}
	return &TelegramIdentityVerified{
		Provider:       constants.UserOAuthProviderTelegram,
		ProviderUserID: providerUserID,
		Username:       strings.TrimSpace(input.Username),
		AvatarURL:      strings.TrimSpace(input.AvatarURL),
		FirstName:      strings.TrimSpace(input.FirstName),
		LastName:       strings.TrimSpace(input.LastName),
		AuthAt:         time.Now(),
	}, nil
}
