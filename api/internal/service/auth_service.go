package service

import (
	"context"
	"errors"
	"time"

	"github.com/dujiao-next/internal/cache"
	"github.com/dujiao-next/internal/config"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// AuthService ????
type AuthService struct {
	cfg       *config.Config
	adminRepo repository.AdminRepository
}

// NewAuthService ????????
func NewAuthService(cfg *config.Config, adminRepo repository.AdminRepository) *AuthService {
	return &AuthService{
		cfg:       cfg,
		adminRepo: adminRepo,
	}
}

// HashPassword ?? bcrypt ????
func (s *AuthService) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword ????
func (s *AuthService) VerifyPassword(hashedPassword, password string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
}

// ValidatePassword ??????????
func (s *AuthService) ValidatePassword(password string) error {
	if s == nil || s.cfg == nil {
		return nil
	}
	return validatePassword(s.cfg.Security.PasswordPolicy, password)
}

// JWTClaims JWT ??
type JWTClaims struct {
	AdminID      uint   `json:"admin_id"`
	Username     string `json:"username"`
	TokenVersion uint64 `json:"token_version"`
	jwt.RegisteredClaims
}

// GenerateJWT ?? JWT Token
func (s *AuthService) GenerateJWT(admin *models.Admin) (string, time.Time, error) {
	expiresAt := time.Now().Add(time.Duration(s.cfg.JWT.ExpireHours) * time.Hour)

	claims := JWTClaims{
		AdminID:      admin.ID,
		Username:     admin.Username,
		TokenVersion: admin.TokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(s.cfg.JWT.SecretKey))
	if err != nil {
		return "", time.Time{}, err
	}

	return tokenString, expiresAt, nil
}

// ParseJWT ?? JWT Token
func (s *AuthService) ParseJWT(tokenString string) (*JWTClaims, error) {
	parser := newHS256JWTParser()
	token, err := parser.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.JWT.SecretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, errors.New("??? token")
}

// Login ?????
func (s *AuthService) Login(username, password string) (*models.Admin, string, time.Time, error) {
	// ?????
	admin, err := s.adminRepo.GetByUsername(username)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if admin == nil {
		// ???? bcrypt ??????????????????
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummyhashtopreventtimingattacksxxxxxxxxxxxxxxxxxx"), []byte(password))
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	// ????
	if err := s.VerifyPassword(admin.PasswordHash, password); err != nil {
		return nil, "", time.Time{}, ErrInvalidCredentials
	}

	// ?? JWT
	token, expiresAt, err := s.GenerateJWT(admin)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	// ????????
	now := time.Now()
	admin.LastLoginAt = &now
	if err := s.adminRepo.Update(admin); err != nil {
		return nil, "", time.Time{}, err
	}
	_ = cache.SetAdminAuthState(context.Background(), cache.BuildAdminAuthState(admin))

	return admin, token, expiresAt, nil
}

// ChangePassword ???????
func (s *AuthService) ChangePassword(adminID uint, oldPassword, newPassword string) error {
	admin, err := s.adminRepo.GetByID(adminID)
	if err != nil {
		return err
	}
	if admin == nil {
		return ErrNotFound
	}

	if err := s.VerifyPassword(admin.PasswordHash, oldPassword); err != nil {
		return ErrInvalidPassword
	}

	if err := s.ValidatePassword(newPassword); err != nil {
		return err
	}

	hashedPassword, err := s.HashPassword(newPassword)
	if err != nil {
		return err
	}

	admin.PasswordHash = hashedPassword
	now := time.Now()
	admin.TokenVersion++
	admin.TokenInvalidBefore = &now
	if err := s.adminRepo.Update(admin); err != nil {
		return err
	}
	_ = cache.SetAdminAuthState(context.Background(), cache.BuildAdminAuthState(admin))
	return nil
}
