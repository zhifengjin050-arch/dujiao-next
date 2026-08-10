package service

import (
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/dujiao-next/internal/logger"
	"github.com/dujiao-next/internal/models"
	"github.com/dujiao-next/internal/repository"
)

// MediaService ??????
type MediaService struct {
	repo repository.MediaRepository
}

// NewMediaService ????????
func NewMediaService(repo repository.MediaRepository) *MediaService {
	return &MediaService{repo: repo}
}

// List ????
func (s *MediaService) List(scene, search string, page, pageSize int) ([]models.Media, int64, error) {
	return s.repo.List(repository.MediaListFilter{
		Page:     page,
		PageSize: pageSize,
		Scene:    scene,
		Search:   search,
	})
}

// RecordMedia ??????????(???????)
func (s *MediaService) RecordMedia(result *UploadResult, scene string) (*models.Media, error) {
	// ???????(??????)
	existing, err := s.repo.GetByPath(result.URL)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	// ??????????????(?????)
	name := result.Filename
	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[:idx]
	}

	media := &models.Media{
		Name:     name,
		Filename: result.Filename,
		Path:     result.URL,
		MimeType: result.MimeType,
		Size:     result.Size,
		Scene:    scene,
		Width:    result.Width,
		Height:   result.Height,
	}
	if err := s.repo.Create(media); err != nil {
		return nil, err
	}
	return media, nil
}

// RecordLocalFile ???????????????(??????????)
// localPath ??? /uploads/upstream/uuid.jpg,scene ? "upstream"
func (s *MediaService) RecordLocalFile(localPath, scene string) {
	// ??
	existing, _ := s.repo.GetByPath(localPath)
	if existing != nil {
		return
	}

	// ??????:????? /
	diskPath := strings.TrimPrefix(localPath, "/")
	fi, err := os.Stat(diskPath)
	if err != nil {
		return
	}

	filename := filepath.Base(localPath)
	name := filename
	if idx := strings.LastIndex(name, "."); idx > 0 {
		name = name[:idx]
	}

	// ?? MIME ??
	mimeType := "application/octet-stream"
	if f, err := os.Open(diskPath); err == nil {
		buf := make([]byte, 512)
		if n, _ := f.Read(buf); n > 0 {
			mimeType = http.DetectContentType(buf[:n])
		}
		f.Close()
	}

	// ????????
	var width, height int
	if strings.HasPrefix(mimeType, "image/") && mimeType != "image/svg+xml" {
		if f, err := os.Open(diskPath); err == nil {
			if cfg, _, err := image.DecodeConfig(f); err == nil {
				width = cfg.Width
				height = cfg.Height
			}
			f.Close()
		}
	}

	media := &models.Media{
		Name:     name,
		Filename: filename,
		Path:     localPath,
		MimeType: mimeType,
		Size:     fi.Size(),
		Scene:    scene,
		Width:    width,
		Height:   height,
	}
	if err := s.repo.Create(media); err != nil {
		logger.Warnw("media_record_local_file_failed", "path", localPath, "error", err)
	}
}

// Rename ?????
func (s *MediaService) Rename(id uint, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrMediaNameEmpty
	}
	media, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if media == nil {
		return ErrMediaNotFound
	}
	media.Name = name
	return s.repo.Update(media)
}

// Delete ????(????????????)
func (s *MediaService) Delete(id uint) error {
	media, err := s.repo.GetByID(id)
	if err != nil {
		return err
	}
	if media == nil {
		return ErrMediaNotFound
	}
	if err := s.repo.Delete(id); err != nil {
		return err
	}
	// ??????(Path ??? /uploads/product/2026/04/uuid.jpg)
	diskPath := strings.TrimPrefix(media.Path, "/")
	if err := os.Remove(diskPath); err != nil && !os.IsNotExist(err) {
		logger.Warnw("media_delete_file_failed", "id", id, "path", diskPath, "error", err)
	}
	return nil
}
