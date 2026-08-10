package service

import (
	"strings"
	"time"

	"github.com/dujiao-next/internal/constants"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// BannerService Banner ????
type BannerService struct {
	repo repository.BannerRepository
}

// NewBannerService ?? Banner ??
func NewBannerService(repo repository.BannerRepository) *BannerService {
	return &BannerService{repo: repo}
}

// BannerInput ??/?? Banner ??
type BannerInput struct {
	Name         string
	Position     string
	TitleJSON    map[string]interface{}
	SubtitleJSON map[string]interface{}
	Image        string
	MobileImage  string
	LinkType     string
	LinkValue    string
	OpenInNewTab *bool
	IsActive     *bool
	StartAt      *time.Time
	EndAt        *time.Time
	SortOrder    int
}

// ListAdmin ???? Banner ??
func (s *BannerService) ListAdmin(position, search string, isActive *bool, page, pageSize int) ([]models.Banner, int64, error) {
	filter := repository.BannerListFilter{
		Page:     page,
		PageSize: pageSize,
		Position: strings.TrimSpace(position),
		Search:   strings.TrimSpace(search),
		IsActive: isActive,
		OrderBy:  "sort_order DESC, created_at DESC",
	}
	return s.repo.List(filter)
}

// ListPublic ???? Banner ??
func (s *BannerService) ListPublic(position string, limit int) ([]models.Banner, error) {
	normalized := normalizeBannerPosition(position)
	return s.repo.ListValidByPosition(normalized, limit, time.Now())
}

// GetByID ?? ID ?? Banner
func (s *BannerService) GetByID(id string) (*models.Banner, error) {
	banner, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if banner == nil {
		return nil, ErrNotFound
	}
	return banner, nil
}

// Create ?? Banner
func (s *BannerService) Create(input BannerInput) (*models.Banner, error) {
	banner, err := buildBannerEntity(input, nil)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Create(banner); err != nil {
		return nil, err
	}
	return banner, nil
}

// Update ?? Banner
func (s *BannerService) Update(id string, input BannerInput) (*models.Banner, error) {
	existing, err := s.repo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrNotFound
	}

	banner, err := buildBannerEntity(input, existing)
	if err != nil {
		return nil, err
	}
	if err := s.repo.Update(banner); err != nil {
		return nil, err
	}
	return banner, nil
}

// Delete ?? Banner
func (s *BannerService) Delete(id string) error {
	banner, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if banner == nil {
		return ErrNotFound
	}
	return s.repo.Delete(id)
}

func buildBannerEntity(input BannerInput, existing *models.Banner) (*models.Banner, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return nil, ErrInvalidBanner
	}
	image := strings.TrimSpace(input.Image)
	if image == "" {
		return nil, ErrInvalidBanner
	}

	position := normalizeBannerPosition(input.Position)
	linkType := normalizeBannerLinkType(input.LinkType)
	if linkType == "" {
		return nil, ErrInvalidBanner
	}

	startAt := input.StartAt
	endAt := input.EndAt
	if startAt != nil && endAt != nil && endAt.Before(*startAt) {
		return nil, ErrInvalidBanner
	}

	linkValue := strings.TrimSpace(input.LinkValue)
	if linkType == constants.BannerLinkTypeNone {
		linkValue = ""
	}
	if linkType != constants.BannerLinkTypeNone && linkValue == "" {
		return nil, ErrInvalidBanner
	}

	if existing == nil {
		entity := &models.Banner{
			Name:         name,
			Position:     position,
			TitleJSON:    normalizeMultiLangJSON(input.TitleJSON),
			SubtitleJSON: normalizeMultiLangJSON(input.SubtitleJSON),
			Image:        image,
			MobileImage:  strings.TrimSpace(input.MobileImage),
			LinkType:     linkType,
			LinkValue:    linkValue,
			StartAt:      startAt,
			EndAt:        endAt,
			SortOrder:    input.SortOrder,
		}
		if input.OpenInNewTab != nil {
			entity.OpenInNewTab = *input.OpenInNewTab
		}
		if input.IsActive != nil {
			entity.IsActive = *input.IsActive
		} else {
			entity.IsActive = true
		}
		return entity, nil
	}

	existing.Name = name
	existing.Position = position
	existing.TitleJSON = normalizeMultiLangJSON(input.TitleJSON)
	existing.SubtitleJSON = normalizeMultiLangJSON(input.SubtitleJSON)
	existing.Image = image
	existing.MobileImage = strings.TrimSpace(input.MobileImage)
	existing.LinkType = linkType
	existing.LinkValue = linkValue
	existing.StartAt = startAt
	existing.EndAt = endAt
	existing.SortOrder = input.SortOrder
	if input.OpenInNewTab != nil {
		existing.OpenInNewTab = *input.OpenInNewTab
	}
	if input.IsActive != nil {
		existing.IsActive = *input.IsActive
	}
	return existing, nil
}

func normalizeBannerPosition(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return constants.BannerPositionHomeHero
	}
	if value == constants.BannerPositionHomeHero {
		return value
	}
	return constants.BannerPositionHomeHero
}

func normalizeBannerLinkType(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	switch value {
	case "", constants.BannerLinkTypeNone:
		return constants.BannerLinkTypeNone
	case constants.BannerLinkTypeInternal:
		return constants.BannerLinkTypeInternal
	case constants.BannerLinkTypeExternal:
		return constants.BannerLinkTypeExternal
	default:
		return ""
	}
}

func normalizeMultiLangJSON(raw map[string]interface{}) models.JSON {
	result := models.JSON{}
	for _, key := range constants.SupportedLocales {
		value, ok := raw[key]
		if !ok {
			result[key] = ""
			continue
		}
		if text, ok := value.(string); ok {
			result[key] = strings.TrimSpace(text)
			continue
		}
		result[key] = ""
	}
	return result
}
