package logger

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLogFilePathDefaultDir(t *testing.T) {
	tmpDir := t.TempDir()
	oldWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("get wd failed: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldWD)
	})
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir failed: %v", err)
	}

	got, err := resolveLogFilePath(Options{})
	if err != nil {
		t.Fatalf("resolve default log path failed: %v", err)
	}

	realTmpDir, err := filepath.EvalSymlinks(tmpDir)
	if err != nil {
		t.Fatalf("resolve tmp dir symlink failed: %v", err)
	}
	realGot, err := filepath.EvalSymlinks(filepath.Dir(got))
	if err != nil {
		t.Fatalf("resolve got dir symlink failed: %v", err)
	}
	expectedDir := filepath.Join(realTmpDir, defaultLogDirName)
	if realGot != expectedDir {
		t.Fatalf("unexpected log dir: got=%s expected=%s", realGot, expectedDir)
	}
	if filepath.Base(got) != defaultLogFilename {
		t.Fatalf("unexpected log filename: %s", filepath.Base(got))
	}
	if _, err := os.Stat(filepath.Dir(got)); err != nil {
		t.Fatalf("expected log dir to be created: %v", err)
	}
}

func TestNewReleaseWritesToConfiguredFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Options{
		Dir:      tmpDir,
		Filename: "release.log",
	}
	log := New("release", cfg)
	log.Info("release-log-test")
	_ = log.Sync()

	content, err := os.ReadFile(filepath.Join(tmpDir, "release.log"))
	if err != nil {
		t.Fatalf("read release log failed: %v", err)
	}
	if !strings.Contains(string(content), "release-log-test") {
		t.Fatalf("expected log content to contain message, got=%s", string(content))
	}
}

func TestNewDebugDoesNotWriteFile(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := Options{
		Dir:      tmpDir,
		Filename: "debug.log",
	}
	log := New("debug", cfg)
	log.Info("debug-log-test")
	_ = log.Sync()

	if _, err := os.Stat(filepath.Join(tmpDir, "debug.log")); !os.IsNotExist(err) {
		t.Fatalf("debug mode should not create log file")
	}
}
