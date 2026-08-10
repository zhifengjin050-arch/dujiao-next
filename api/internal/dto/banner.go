package dto

import (
	"github.com/dujiao-next/internal/models"
)

// BannerResp ?? Banner ??
type BannerResp struct {
	ID           uint        `json:"id"`
	Position     string      `json:"position"`
	Title        models.JSON `json:"title"`
	Subtitle     models.JSON `json:"subtitle"`
	Image        string      `json:"image"`
	MobileImage  string      `json:"mobile_image,omitempty"`
	LinkType     string      `json:"link_type"`
	LinkValue    string      `json:"link_value,omitempty"`
	OpenInNewTab bool        `json:"open_in_new_tab"`
}

// NewBannerResp ? models.Banner ????
func NewBannerResp(b *models.Banner) BannerResp {
	return BannerResp{
		ID:           b.ID,
		Position:     b.Position,
		Title:        b.TitleJSON,
		Subtitle:     b.SubtitleJSON,
		Image:        b.Image,
		MobileImage:  b.MobileImage,
		LinkType:     b.LinkType,
		LinkValue:    b.LinkValue,
		OpenInNewTab: b.OpenInNewTab,
	}
	// ??:Name(????)?IsActive?StartAt?EndAt?SortOrder?CreatedAt?UpdatedAt
}

// NewBannerRespList ???? Banner ??
func NewBannerRespList(banners []models.Banner) []BannerResp {
	result := make([]BannerResp, 0, len(banners))
	for i := range banners {
		result = append(result, NewBannerResp(&banners[i]))
	}
	return result
}
