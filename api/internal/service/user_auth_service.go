package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"net/mail"
	"strings"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// UserAuthService ??????
type UserAuthService struct {
	cfg                   *config.Config
	userRepo              repository.UserRepository
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository
	codeRepo              repository.EmailVerifyCodeRepository
	settingService        *SettingService
	emailService          *EmailService
	telegramAuthService   *TelegramAuthService
	memberLevelSvc        *MemberLevelService
}

// SetMemberLevelService ????????
func (s *UserAuthService) SetMemberLevelService(svc *MemberLevelService) {
	s.memberLevelSvc = svc
}

// NewUserAuthService ????????
func NewUserAuthService(
	cfg *config.Config,
	userRepo repository.UserRepository,
	userOAuthIdentityRepo repository.UserOAuthIdentityRepository,
	codeRepo repository.EmailVerifyCodeRepository,
	settingService *SettingService,
	emailService *EmailService,
	telegramAuthService *TelegramAuthService,
) *UserAuthService {
	return &UserAuthService{
		cfg:                   cfg,
		userRepo:              userRepo,
		userOAuthIdentityRepo: userOAuthIdentityRepo,
		codeRepo:              codeRepo,
		settingService:        settingService,
		emailService:          emailService,
		telegramAuthService:   telegramAuthService,
	}
}

// UserJWTClaims ?? JWT ??
type UserJWTClaims struct {
	UserID       uint   `json:"user_id"`
	Email        string `json:"email"`
	TokenVersion uint64 `json:"token_version"`
	jwt.RegisteredClaims
}

const (
	// EmailChangeModeBindOnly ????????????(?? Telegram ??????)
	EmailChangeModeBindOnly = "bind_only"
	// EmailChangeModeChangeWithOldAndNew ??????? + ???????
	EmailChangeModeChangeWithOldAndNew = "change_with_old_and_new"
	// PasswordChangeModeSetWithoutOld ????????,??????
	PasswordChangeModeSetWithoutOld = "set_without_old"
	// PasswordChangeModeChangeWithOld ??????,?????
	PasswordChangeModeChangeWithOld = "change_with_old"
)

// GenerateUserJWT ???? JWT Token
func (s *UserAuthService) GenerateUserJWT(user *models.User, expireHours int) (string, time.Time, error) {
	resolvedHours := expireHours
	if resolvedHours <= 0 {
		resolvedHours = resolveUserJWTExpireHours(s.cfg.UserJWT)
	}
	expiresAt := time.Now().Add(time.Duration(resolvedHours) * time.Hour)
	claims := UserJWTClaims{
		UserID:       user.ID,
		Email:        user.Email,
		TokenVersion: user.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.UserJWT.SecretKey))
	if err != nil {
		return "", time.Time{}, err
	}
	return tokenString, expiresAt, nil
}

// ParseUserJWT ???? JWT Token
func (s *UserAuthService) ParseUserJWT(tokenString string) (*UserJWTClaims, error) {
	parser := newHS256JWTParser()
	claims := &UserJWTClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.UserJWT.SecretKey), nil
	})
	if err != nil {
		return nil, err
	}
	if claims, ok := token.Claims.(*UserJWTClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("??? token")
}

// SendVerifyCode ???????
func (s *UserAuthService) SendVerifyCode(email, purpose, locale string) error {
	if s.emailService == nil {
		return ErrEmailServiceNotConfigured
	}
	normalized, err := normalizeEmail(email)
	if err != nil {
		return err
	}
	if !isVerifyPurposeSupported(purpose) {
		return ErrInvalidVerifyPurpose
	}

	if purpose == constants.VerifyPurposeRegister {
		exist, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if exist != nil {
			return ErrEmailExists
		}
	}

	if purpose == constants.VerifyPurposeReset {
		user, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrNotFound
		}
		if strings.TrimSpace(user.Locale) != "" {
			locale = user.Locale
		}
	}

	if purpose == constants.VerifyPurposeTelegramBind {
		user, err := s.userRepo.GetByEmail(normalized)
		if err != nil {
			return err
		}
		if user == nil {
			return ErrNotFound
		}
		if strings.TrimSpace(user.Locale) != "" {
			locale = user.Locale
		}
	}

	return s.sendVerifyCode(normalized, strings.ToLower(purpose), locale)
}

// Register ????
func (s *UserAuthService) Register(email, password, code string, agreementAccepted bool, emailVerificationEnabled bool) (*models.User, string, time.Time, error) {
	if !agreementAccepted {
		return nil, "", time.Time{}, ErrAgreementRequired
	}
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if err := validatePassword(s.cfg.Security.PasswordPolicy, password); err != nil {
		return nil, "", time.Time{}, err
	}

	exist, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if exist != nil {
		return nil, "", time.Time{}, ErrEmailExists
	}

	if emailVerificationEnabled {
		if _, err := s.verifyCode(normalized, constants.VerifyPurposeRegister, code); err != nil {
			return nil, "", time.Time{}, err
		}
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	now := time.Now()
	nickname := resolveNicknameFromEmail(normalized)
	user := &models.User{
		Email:           normalized,
		PasswordHash:    string(hashedPassword),
		DisplayName:     nickname,
		Status:          constants.UserStatusActive,
		EmailVerifiedAt: &now,
		CreatedAt:       now,
		UpdatedAt:       now,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, "", time.Time{}, err
	}

	token, expiresAt, err := s.GenerateUserJWT(user, 0)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", time.Time{}, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))

	// ????????(??????? Update ??,?????)
	if s.memberLevelSvc != nil {
		_ = s.memberLevelSvc.AssignDefaultLevel(user.ID)
	}

	return user, token, expiresAt, nil
}

// Login ????
func (s *UserAuthService) Login(email, password string) (*models.User, string, time.Time, error) {
	return s.LoginWithRememberMe(email, password, false)
}

// LoginWithRememberMe ????(?????)
func (s *UserAuthService) LoginWithRememberMe(email, password string, rememberMe bool) (*models.User, string, time.Time, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	user, err := s.userRepo.GetByEmail(normalized)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if user == nil {
		// ???? bcrypt ?????????????????
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashtopreventtimingattacksxxxxxxxxxxxxxxxxxx"), []byte(password))
		return nil, "", time.Time{}, ErrInvalidCredentials
	}
	if strings.ToLower(user.Status) != constants.UserStatusActive {
		return nil, "", time.Time{}, ErrUserDisabled
	}
	if user.EmailVerifiedAt == nil {
		return nil, "", time.Time{}, ErrEmailNotVerified
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	expireHours := resolveUserJWTExpireHours(s.cfg.UserJWT)
	if rememberMe {
		expireHours = resolveRememberMeExpireHours(s.cfg.UserJWT)
	}
	token, expiresAt, err := s.GenerateUserJWT(user, expireHours)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	now := time.Now()
	user.LastLoginAt = &now
	if err := s.userRepo.Update(user); err != nil {
		return nil, "", time.Time{}, err
	}
	_ = cache.SetUserAuthState(context.Background(), cache.BuildUserAuthState(user))

	return user, token, expiresAt, nil
}

func (s *UserAuthService) verifyCode(email, purpose, code string) (*models.EmailVerifyCode, error) {
	record, err := s.codeRepo.GetLatest(email, purpose)
	if err != nil {
		return nil, err
	}
	if record == nil {
		return nil, ErrVerifyCodeInvalid
	}
	if record.VerifiedAt != nil {
		return nil, ErrVerifyCodeInvalid
	}

	now := time.Now()
	if record.ExpiresAt.Before(now) {
		return nil, ErrVerifyCodeExpired
	}

	maxAttempts := resolveMaxAttempts(s.cfg.Email.VerifyCode)
	if maxAttempts > 0 && record.AttemptCount >= maxAttempts {
		return nil, ErrVerifyCodeAttemptsExceeded
	}

	if strings.TrimSpace(record.Code) != strings.TrimSpace(code) {
		_ = s.codeRepo.IncrementAttempt(record.ID)
		return nil, ErrVerifyCodeInvalid
	}

	if err := s.codeRepo.MarkVerified(record.ID, now); err != nil {
		return nil, err
	}
	return record, nil
}

func (s *UserAuthService) sendVerifyCode(email, purpose, locale string) error {
	latest, err := s.codeRepo.GetLatest(email, purpose)
	if err != nil {
		return err
	}
	now := time.Now()
	if latest != nil {
		interval := time.Duration(resolveSendIntervalSeconds(s.cfg.Email.VerifyCode)) * time.Second
		if !latest.SentAt.IsZero() && now.Sub(latest.SentAt) < interval {
			return ErrVerifyCodeTooFrequent
		}
	}

	code, err := randomNumericCode(resolveCodeLength(s.cfg.Email.VerifyCode))
	if err != nil {
		return err
	}

	record := &models.EmailVerifyCode{
		Email:     email,
		Purpose:   strings.ToLower(purpose),
		Code:      code,
		ExpiresAt: now.Add(time.Duration(resolveExpireMinutes(s.cfg.Email.VerifyCode)) * time.Minute),
		SentAt:    now,
		CreatedAt: now,
	}
	if err := s.emailService.SendVerifyCode(email, code, purpose, locale); err != nil {
		return err
	}

	if err := s.codeRepo.Create(record); err != nil {
		return err
	}

	return nil
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", ErrInvalidEmail
	}
	if _, err := mail.ParseAddress(normalized); err != nil {
		return "", ErrInvalidEmail
	}
	return normalized, nil
}

// NormalizeEmail ??????
func NormalizeEmail(email string) (string, error) {
	return normalizeEmail(email)
}

func isVerifyPurposeSupported(purpose string) bool {
	switch strings.ToLower(strings.TrimSpace(purpose)) {
	case constants.VerifyPurposeRegister, constants.VerifyPurposeReset, constants.VerifyPurposeTelegramBind, constants.VerifyPurposeChangeEmailOld, constants.VerifyPurposeChangeEmailNew:
		return true
	default:
		return false
	}
}

func resolveUserJWTExpireHours(cfg config.JWTConfig) int {
	if cfg.ExpireHours <= 0 {
		return 24
	}
	return cfg.ExpireHours
}

func resolveRememberMeExpireHours(cfg config.JWTConfig) int {
	if cfg.RememberMeExpireHours <= 0 {
		return resolveUserJWTExpireHours(cfg)
	}
	return cfg.RememberMeExpireHours
}

func resolveNicknameFromEmail(email string) string {
	parts := strings.SplitN(email, "@", 2)
	if len(parts) == 2 && strings.TrimSpace(parts[0]) != "" {
		return strings.TrimSpace(parts[0])
	}
	return email
}

func resolveExpireMinutes(cfg config.VerifyCodeConfig) int {
	if cfg.ExpireMinutes <= 0 {
		return 10
	}
	return cfg.ExpireMinutes
}

func resolveSendIntervalSeconds(cfg config.VerifyCodeConfig) int {
	if cfg.SendIntervalSeconds <= 0 {
		return 60
	}
	return cfg.SendIntervalSeconds
}

func resolveMaxAttempts(cfg config.VerifyCodeConfig) int {
	if cfg.MaxAttempts <= 0 {
		return 5
	}
	return cfg.MaxAttempts
}

func resolveCodeLength(cfg config.VerifyCodeConfig) int {
	if cfg.Length < 4 || cfg.Length > 10 {
		return 6
	}
	return cfg.Length
}

func randomNumericCode(length int) (string, error) {
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(10))
		if err != nil {
			return "", err
		}
		b.WriteString(fmt.Sprintf("%d", n.Int64()))
	}
	return b.String(), nil
}
