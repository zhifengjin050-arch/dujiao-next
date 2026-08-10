package service

import (
	"encoding/binary"
	"fmt"
	"image"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dujiao-next/internal/config"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/google/uuid"
)

var allowedUploadScenes = map[string]struct{}{
	"product":  {},
	"post":     {},
	"banner":   {},
	"editor":   {},
	"common":   {},
	"category": {},
	"telegram": {},
}

// UploadService ??????
type UploadService struct {
	cfg *config.Config
}

// NewUploadService ??????????
func NewUploadService(cfg *config.Config) *UploadService {
	return &UploadService{cfg: cfg}
}

// UploadResult ????(???????)
type UploadResult struct {
	URL      string // ????
	Filename string // ?????
	MimeType string
	Size     int64
	Width    int
	Height   int
}

// SaveFile ???????(????????)
func (s *UploadService) SaveFile(file *multipart.FileHeader, scene string) (string, error) {
	result, err := s.SaveFileWithMeta(file, scene)
	if err != nil {
		return "", err
	}
	return result.URL, nil
}

// SaveFileWithMeta ???????????????
func (s *UploadService) SaveFileWithMeta(file *multipart.FileHeader, scene string) (*UploadResult, error) {
	normalizedScene := normalizeUploadScene(scene)

	// ??????
	if file.Size > s.cfg.Upload.MaxSize {
		return nil, fmt.Errorf("????????(?? %d MB)", s.cfg.Upload.MaxSize/1024/1024)
	}

	// ???????
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if normalizedScene != "telegram" && len(s.cfg.Upload.AllowedExtensions) > 0 {
		if ext == "" || !isAllowedExtension(ext, s.cfg.Upload.AllowedExtensions) {
			return nil, fmt.Errorf("?????????: %s", ext)
		}
	}

	// ??????
	src, err := file.Open()
	if err != nil {
		return nil, err
	}
	defer src.Close()

	// ???????? MIME ??
	buffer := make([]byte, 512)
	_, err = src.Read(buffer)
	if err != nil && err != io.EOF {
		return nil, err
	}
	if _, err := src.Seek(0, 0); err != nil { // ????????
		return nil, err
	}

	contentType := http.DetectContentType(buffer)
	// http.DetectContentType ???? SVG,???????????????
	if ext == ".svg" && isSVGContent(buffer) {
		contentType = "image/svg+xml"
	}
	if normalizedScene != "telegram" && len(s.cfg.Upload.AllowedTypes) > 0 {
		allowed := false
		for _, t := range s.cfg.Upload.AllowedTypes {
			if strings.EqualFold(contentType, t) {
				allowed = true
				break
			}
		}
		if !allowed {
			return nil, fmt.Errorf("????????: %s", contentType)
		}
	}

	var imgWidth, imgHeight int
	if strings.HasPrefix(contentType, "image/") && contentType != "image/svg+xml" {
		if _, err := src.Seek(0, 0); err != nil {
			return nil, err
		}
		width, height, err := decodeImageDimensions(src, contentType)
		if err != nil {
			return nil, err
		}
		imgWidth = width
		imgHeight = height
		if s.cfg.Upload.MaxWidth > 0 && width > s.cfg.Upload.MaxWidth {
			return nil, fmt.Errorf("????????(?? %d)", s.cfg.Upload.MaxWidth)
		}
		if s.cfg.Upload.MaxHeight > 0 && height > s.cfg.Upload.MaxHeight {
			return nil, fmt.Errorf("????????(?? %d)", s.cfg.Upload.MaxHeight)
		}
	}

	// SVG ????:???????????
	if contentType == "image/svg+xml" {
		if _, err := src.Seek(0, 0); err != nil {
			return nil, err
		}
		svgData, err := io.ReadAll(src)
		if err != nil {
			return nil, err
		}
		if err := validateSVGSafety(svgData); err != nil {
			return nil, err
		}
		if _, err := src.Seek(0, 0); err != nil {
			return nil, err
		}
	}

	if _, err := src.Seek(0, 0); err != nil {
		return nil, err
	}
	// ???????
	filename := fmt.Sprintf("%s%s", uuid.New().String(), ext)
	now := time.Now()
	year := now.Format("2006")
	month := now.Format("01")
	savePath := filepath.Join("uploads", normalizedScene, year, month, filename)

	// ????????
	if err := os.MkdirAll(filepath.Dir(savePath), 0755); err != nil {
		return nil, err
	}

	// ????
	dst, err := os.Create(savePath)
	if err != nil {
		return nil, err
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return nil, err
	}

	return &UploadResult{
		URL:      fmt.Sprintf("/uploads/%s/%s/%s/%s", normalizedScene, year, month, filename),
		Filename: file.Filename,
		MimeType: contentType,
		Size:     file.Size,
		Width:    imgWidth,
		Height:   imgHeight,
	}, nil
}

func normalizeUploadScene(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return "common"
	}
	if _, ok := allowedUploadScenes[value]; ok {
		return value
	}
	return "common"
}

func isAllowedExtension(ext string, allowed []string) bool {
	for _, allowedExt := range allowed {
		normalized := strings.ToLower(strings.TrimSpace(allowedExt))
		if normalized == "" {
			continue
		}
		if !strings.HasPrefix(normalized, ".") {
			normalized = "." + normalized
		}
		if strings.EqualFold(ext, normalized) {
			return true
		}
	}
	return false
}

func decodeImageDimensions(src io.ReadSeeker, contentType string) (int, int, error) {
	if strings.EqualFold(contentType, "image/webp") {
		width, height, err := decodeWebPDimensions(src)
		if err != nil {
			return 0, 0, fmt.Errorf("???? WebP ??: %w", err)
		}
		return width, height, nil
	}

	if _, err := src.Seek(0, 0); err != nil {
		return 0, 0, err
	}
	cfg, _, err := image.DecodeConfig(src)
	if err != nil {
		return 0, 0, fmt.Errorf("??????: %w", err)
	}
	return cfg.Width, cfg.Height, nil
}

// isSVGContent ??????????? SVG
func isSVGContent(buf []byte) bool {
	content := strings.TrimSpace(string(buf))
	// SVG ????? XML ??? <svg ????
	return strings.HasPrefix(content, "<?xml") ||
		strings.HasPrefix(content, "<svg") ||
		strings.Contains(content, "<svg")
}

// validateSVGSafety ?? SVG ?????,?????????
func validateSVGSafety(data []byte) error {
	content := strings.ToLower(string(data))
	// ??????
	if strings.Contains(content, "<script") {
		return fmt.Errorf("SVG ??????? <script> ??")
	}
	// ????????(onclick, onload, onerror ?)
	dangerousAttrs := []string{
		"onload", "onclick", "onerror", "onmouseover", "onmouseout",
		"onmousemove", "onfocus", "onblur", "onchange", "onsubmit",
		"onanimationstart", "onanimationend", "onanimationiteration",
	}
	for _, attr := range dangerousAttrs {
		if strings.Contains(content, attr+"=") || strings.Contains(content, attr+" =") {
			return fmt.Errorf("SVG ?????????????: %s", attr)
		}
	}
	// ?? javascript: ??
	if strings.Contains(content, "javascript:") {
		return fmt.Errorf("SVG ??????? javascript: ??")
	}
	// ?? data: URI(????? CSP)
	if strings.Contains(content, "data:text/html") || strings.Contains(content, "data:application") {
		return fmt.Errorf("SVG ?????????? data: URI")
	}
	// ?? foreignObject(??? HTML)
	if strings.Contains(content, "<foreignobject") {
		return fmt.Errorf("SVG ??????? <foreignObject> ??")
	}
	return nil
}

func decodeWebPDimensions(src io.ReadSeeker) (int, int, error) {
	if _, err := src.Seek(0, 0); err != nil {
		return 0, 0, err
	}

	header := make([]byte, 12)
	if _, err := io.ReadFull(src, header); err != nil {
		return 0, 0, err
	}
	if string(header[0:4]) != "RIFF" || string(header[8:12]) != "WEBP" {
		return 0, 0, fmt.Errorf("??? WebP ???")
	}

	for {
		chunkHeader := make([]byte, 8)
		if _, err := io.ReadFull(src, chunkHeader); err != nil {
			return 0, 0, err
		}
		chunkType := string(chunkHeader[0:4])
		chunkSize := int(binary.LittleEndian.Uint32(chunkHeader[4:8]))
		if chunkSize < 0 {
			return 0, 0, fmt.Errorf("??? WebP chunk")
		}

		data := make([]byte, chunkSize)
		if _, err := io.ReadFull(src, data); err != nil {
			return 0, 0, err
		}

		if chunkType == "VP8X" {
			if len(data) < 10 {
				return 0, 0, fmt.Errorf("VP8X chunk ????")
			}
			width := 1 + int(data[4]) + int(data[5])<<8 + int(data[6])<<16
			height := 1 + int(data[7]) + int(data[8])<<8 + int(data[9])<<16
			return width, height, nil
		}
		if chunkType == "VP8 " {
			if len(data) < 10 {
				return 0, 0, fmt.Errorf("VP8 chunk ????")
			}
			width := int(binary.LittleEndian.Uint16(data[6:8]) & 0x3FFF)
			height := int(binary.LittleEndian.Uint16(data[8:10]) & 0x3FFF)
			return width, height, nil
		}
		if chunkType == "VP8L" {
			if len(data) < 5 {
				return 0, 0, fmt.Errorf("VP8L chunk ????")
			}
			if data[0] != 0x2f {
				return 0, 0, fmt.Errorf("VP8L ????")
			}
			bits := binary.LittleEndian.Uint32(data[1:5])
			width := int(bits&0x3FFF) + 1
			height := int((bits>>14)&0x3FFF) + 1
			return width, height, nil
		}

		if chunkSize%2 == 1 {
			if _, err := src.Seek(1, io.SeekCurrent); err != nil {
				return 0, 0, err
			}
		}
	}
}
