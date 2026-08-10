package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
	"gorm.io/gorm"
)

var (
	ErrApiCredentialExists       = errors.New("api credential already exists for this user")
	ErrApiCredentialNotFound     = errors.New("api credential not found")
	ErrApiCredentialNotApproved  = errors.New("api credential is not approved")
	ErrApiCredentialPendingExist = errors.New("pending application already exists")
)

// ApiCredentialService API ????
type ApiCredentialService struct {
	credRepo repository.ApiCredentialRepository
}

// NewApiCredentialService ??????
func NewApiCredentialService(credRepo repository.ApiCredentialRepository) *ApiCredentialService {
	return &ApiCredentialService{credRepo: credRepo}
}

// Apply ???? API ????
func (s *ApiCredentialService) Apply(userID uint) (*models.ApiCredential, error) {
	existing, err := s.credRepo.GetAnyByUserID(userID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		if existing.DeletedAt.Valid {
			if err := resetApiCredentialForReapply(existing); err != nil {
				return nil, err
			}
			if err := s.credRepo.UpdateAny(existing); err != nil {
				return nil, err
			}
			return existing, nil
		}

		switch existing.Status {
		case constants.ApiCredentialStatusPendingReview:
			return nil, ErrApiCredentialPendingExist
		case constants.ApiCredentialStatusApproved:
			return nil, ErrApiCredentialExists
		case constants.ApiCredentialStatusRejected:
			// ??????,????????????
			if err := resetApiCredentialForReapply(existing); err != nil {
				return nil, err
			}
			if err := s.credRepo.Update(existing); err != nil {
				return nil, err
			}
			return existing, nil
		case constants.ApiCredentialStatusDisabled:
			return nil, ErrApiCredentialExists
		}
	}

	apiKey, err := generateRandomHex(32)
	if err != nil {
		return nil, err
	}
	cred := &models.ApiCredential{
		UserID: userID,
		ApiKey: apiKey,
		Status: constants.ApiCredentialStatusPendingReview,
	}
	if err := s.credRepo.Create(cred); err != nil {
		return nil, err
	}
	return cred, nil
}

func resetApiCredentialForReapply(cred *models.ApiCredential) error {
	apiKey, err := generateRandomHex(32)
	if err != nil {
		return err
	}
	cred.ApiKey = apiKey
	cred.ApiSecret = ""
	cred.Status = constants.ApiCredentialStatusPendingReview
	cred.RejectReason = ""
	cred.ApprovedAt = nil
	cred.LastUsedAt = nil
	cred.IsActive = false
	cred.DeletedAt = gorm.DeletedAt{}
	return nil
}

// Approve admin ????
func (s *ApiCredentialService) Approve(id uint) (*models.ApiCredential, string, error) {
	cred, err := s.credRepo.GetByID(id)
	if err != nil {
		return nil, "", err
	}
	if cred == nil {
		return nil, "", ErrApiCredentialNotFound
	}

	apiKey, err := generateRandomHex(32)
	if err != nil {
		return nil, "", err
	}
	apiSecret, err := generateRandomHex(64)
	if err != nil {
		return nil, "", err
	}

	now := time.Now()
	cred.ApiKey = apiKey
	cred.ApiSecret = apiSecret
	cred.Status = constants.ApiCredentialStatusApproved
	cred.ApprovedAt = &now
	cred.IsActive = true
	cred.RejectReason = ""

	if err := s.credRepo.Update(cred); err != nil {
		return nil, "", err
	}

	return cred, apiSecret, nil
}

// Reject admin ????
func (s *ApiCredentialService) Reject(id uint, reason string) error {
	cred, err := s.credRepo.GetByID(id)
	if err != nil {
		return err
	}
	if cred == nil {
		return ErrApiCredentialNotFound
	}

	cred.Status = constants.ApiCredentialStatusRejected
	cred.RejectReason = reason
	return s.credRepo.Update(cred)
}

// SetActive ??/??
func (s *ApiCredentialService) SetActive(id uint, active bool) error {
	cred, err := s.credRepo.GetByID(id)
	if err != nil {
		return err
	}
	if cred == nil {
		return ErrApiCredentialNotFound
	}
	if cred.Status != constants.ApiCredentialStatusApproved {
		return ErrApiCredentialNotApproved
	}

	cred.IsActive = active
	return s.credRepo.Update(cred)
}

// SetActiveByUserID ??????/??
func (s *ApiCredentialService) SetActiveByUserID(userID uint, active bool) error {
	cred, err := s.credRepo.GetByUserID(userID)
	if err != nil {
		return err
	}
	if cred == nil {
		return ErrApiCredentialNotFound
	}
	if cred.Status != constants.ApiCredentialStatusApproved {
		return ErrApiCredentialNotApproved
	}

	cred.IsActive = active
	return s.credRepo.Update(cred)
}

// Regenerate ???? Secret
func (s *ApiCredentialService) Regenerate(id uint) (string, error) {
	cred, err := s.credRepo.GetByID(id)
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", ErrApiCredentialNotFound
	}
	if cred.Status != constants.ApiCredentialStatusApproved {
		return "", ErrApiCredentialNotApproved
	}

	newSecret, err := generateRandomHex(64)
	if err != nil {
		return "", err
	}

	cred.ApiSecret = newSecret
	if err := s.credRepo.Update(cred); err != nil {
		return "", err
	}
	return newSecret, nil
}

// RegenerateByUserID ?????? Secret
func (s *ApiCredentialService) RegenerateByUserID(userID uint) (string, error) {
	cred, err := s.credRepo.GetByUserID(userID)
	if err != nil {
		return "", err
	}
	if cred == nil {
		return "", ErrApiCredentialNotFound
	}
	return s.Regenerate(cred.ID)
}

// GetByUserID ???????
func (s *ApiCredentialService) GetByUserID(userID uint) (*models.ApiCredential, error) {
	return s.credRepo.GetByUserID(userID)
}

// GetByID ?? ID ????
func (s *ApiCredentialService) GetByID(id uint) (*models.ApiCredential, error) {
	return s.credRepo.GetByID(id)
}

// List ????
func (s *ApiCredentialService) List(filter repository.ApiCredentialListFilter) ([]models.ApiCredential, int64, error) {
	return s.credRepo.List(filter)
}

// Delete ????
func (s *ApiCredentialService) Delete(id uint) error {
	return s.credRepo.Delete(id)
}

func generateRandomHex(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
