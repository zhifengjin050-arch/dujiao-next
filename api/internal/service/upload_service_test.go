package service

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dujiao-next/internal/config"
)

func TestUploadServiceSaveFileAllowsArchiveForTelegramScene(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	cfg := &config.Config{}
	cfg.Upload.MaxSize = 10 * 1024 * 1024
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png"}
	cfg.Upload.AllowedExtensions = []string{".jpg", ".png"}
	service := NewUploadService(cfg)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "demo.zip")
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write([]byte("fake zip content")); err != nil {
		t.Fatalf("write form content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}

	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("read form failed: %v", err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected one file, got %d", len(files))
	}

	savedPath, err := service.SaveFile(files[0], "telegram")
	if err != nil {
		t.Fatalf("save file failed: %v", err)
	}
	if filepath.Ext(savedPath) != ".zip" {
		t.Fatalf("expected .zip saved path, got %s", savedPath)
	}
	if _, err := os.Stat(filepath.Join(tempDir, strings.TrimPrefix(savedPath, "/"))); err != nil {
		t.Fatalf("saved file not found: %v", err)
	}
}

func TestUploadServiceSaveFileSVG(t *testing.T) {
	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("chdir temp dir failed: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWD)
	}()

	cfg := &config.Config{}
	cfg.Upload.MaxSize = 10 * 1024 * 1024
	cfg.Upload.AllowedTypes = []string{"image/jpeg", "image/png", "image/svg+xml"}
	cfg.Upload.AllowedExtensions = []string{".jpg", ".png", ".svg"}
	svc := NewUploadService(cfg)

	safeSVG := `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><circle cx="50" cy="50" r="40" fill="red"/></svg>`

	t.Run("safe SVG upload succeeds", func(t *testing.T) {
		fh := createMultipartFile(t, "icon.svg", []byte(safeSVG))
		path, err := svc.SaveFile(fh, "common")
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if filepath.Ext(path) != ".svg" {
			t.Fatalf("expected .svg extension, got %s", path)
		}
		if _, err := os.Stat(filepath.Join(tempDir, strings.TrimPrefix(path, "/"))); err != nil {
			t.Fatalf("saved file not found: %v", err)
		}
	})

	t.Run("SVG with script tag is rejected", func(t *testing.T) {
		malicious := `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`
		fh := createMultipartFile(t, "bad.svg", []byte(malicious))
		_, err := svc.SaveFile(fh, "common")
		if err == nil {
			t.Fatal("expected error for SVG with script tag")
		}
		if !strings.Contains(err.Error(), "<script>") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("SVG with event handler is rejected", func(t *testing.T) {
		malicious := `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><circle cx="50" cy="50" r="40"/></svg>`
		fh := createMultipartFile(t, "bad2.svg", []byte(malicious))
		_, err := svc.SaveFile(fh, "common")
		if err == nil {
			t.Fatal("expected error for SVG with event handler")
		}
		if !strings.Contains(err.Error(), "onload") {
			t.Fatalf("unexpected error message: %v", err)
		}
	})

	t.Run("SVG with javascript protocol is rejected", func(t *testing.T) {
		malicious := `<svg xmlns="http://www.w3.org/2000/svg"><a href="javascript:alert(1)"><circle cx="50" cy="50" r="40"/></a></svg>`
		fh := createMultipartFile(t, "bad3.svg", []byte(malicious))
		_, err := svc.SaveFile(fh, "common")
		if err == nil {
			t.Fatal("expected error for SVG with javascript protocol")
		}
	})

	t.Run("SVG with foreignObject is rejected", func(t *testing.T) {
		malicious := `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><body xmlns="http://www.w3.org/1999/xhtml"><div>hello</div></body></foreignObject></svg>`
		fh := createMultipartFile(t, "bad4.svg", []byte(malicious))
		_, err := svc.SaveFile(fh, "common")
		if err == nil {
			t.Fatal("expected error for SVG with foreignObject")
		}
	})

	t.Run("SVG with XML declaration", func(t *testing.T) {
		xmlSVG := `<?xml version="1.0" encoding="UTF-8"?><svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100"><rect width="100" height="100" fill="blue"/></svg>`
		fh := createMultipartFile(t, "xml.svg", []byte(xmlSVG))
		path, err := svc.SaveFile(fh, "common")
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if filepath.Ext(path) != ".svg" {
			t.Fatalf("expected .svg extension, got %s", path)
		}
	})
}

func TestIsSVGContent(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect bool
	}{
		{"svg tag", `<svg xmlns="http://www.w3.org/2000/svg"></svg>`, true},
		{"xml declaration", `<?xml version="1.0"?><svg></svg>`, true},
		{"not svg", `<html><body></body></html>`, false},
		{"plain text", `hello world`, false},
		{"empty", ``, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isSVGContent([]byte(tt.input))
			if got != tt.expect {
				t.Errorf("isSVGContent(%q) = %v, want %v", tt.input, got, tt.expect)
			}
		})
	}
}

func TestValidateSVGSafety(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"safe svg", `<svg xmlns="http://www.w3.org/2000/svg"><circle cx="50" cy="50" r="40"/></svg>`, false},
		{"script tag", `<svg><script>alert(1)</script></svg>`, true},
		{"onclick", `<svg onclick="alert(1)"></svg>`, true},
		{"javascript href", `<svg><a href="javascript:void(0)"></a></svg>`, true},
		{"data uri html", `<svg><image href="data:text/html,<h1>hi</h1>"/></svg>`, true},
		{"foreignObject", `<svg><foreignObject></foreignObject></svg>`, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSVGSafety([]byte(tt.input))
			if (err != nil) != tt.wantErr {
				t.Errorf("validateSVGSafety() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// createMultipartFile ????:????? multipart ??
func createMultipartFile(t *testing.T, filename string, content []byte) *multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("create form file failed: %v", err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatalf("write form content failed: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer failed: %v", err)
	}
	reader := multipart.NewReader(&body, writer.Boundary())
	form, err := reader.ReadForm(1024 * 1024)
	if err != nil {
		t.Fatalf("read form failed: %v", err)
	}
	files := form.File["file"]
	if len(files) != 1 {
		t.Fatalf("expected one file, got %d", len(files))
	}
	return files[0]
}
