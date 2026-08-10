package dto

import (
	"time"

	"github.com/dujiao-next/internal/models"
)

// PostResp ??/??????
type PostResp struct {
	ID          uint        `json:"id"`
	Slug        string      `json:"slug"`
	Type        string      `json:"type"`
	Title       models.JSON `json:"title"`
	Summary     models.JSON `json:"summary"`
	Content     models.JSON `json:"content"`
	Thumbnail   string      `json:"thumbnail,omitempty"`
	PublishedAt *time.Time  `json:"published_at"`
}

// NewPostResp ? models.Post ????
func NewPostResp(p *models.Post) PostResp {
	return PostResp{
		ID:          p.ID,
		Slug:        p.Slug,
		Type:        p.Type,
		Title:       p.TitleJSON,
		Summary:     p.SummaryJSON,
		Content:     p.ContentJSON,
		Thumbnail:   p.Thumbnail,
		PublishedAt: p.PublishedAt,
	}
	// ??:IsPublished(????)?CreatedAt
}

// NewPostRespList ????????
func NewPostRespList(posts []models.Post) []PostResp {
	result := make([]PostResp, 0, len(posts))
	for i := range posts {
		result = append(result, NewPostResp(&posts[i]))
	}
	return result
}
