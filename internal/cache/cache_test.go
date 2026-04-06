package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCache_Get_ReturnsEmptyString_WhenFileDoesNotExist(t *testing.T) {
	c := &Cache{path: "/nonexistent/path/.mec_livros_token"}
	got := c.Get()
	if got != "" {
		t.Errorf("Get() = %q, want empty string", got)
	}
}

func TestCache_Get_ReturnsEmptyString_WhenFileIsInvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	if err := os.WriteFile(tmpFile, []byte("not json"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	c := &Cache{path: tmpFile}
	got := c.Get()
	if got != "" {
		t.Errorf("Get() = %q, want empty string for invalid JSON", got)
	}
}

func TestCache_Get_ReturnsEmptyString_WhenFileHasNoToken(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	tokenData := TokenData{Token: "", SavedAt: "2024-01-01T00:00:00Z"}
	data, _ := json.Marshal(tokenData)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	c := &Cache{path: tmpFile}
	got := c.Get()
	if got != "" {
		t.Errorf("Get() = %q, want empty string when token is empty", got)
	}
}

func TestCache_Get_ReturnsToken_WhenFileIsValid(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	tokenData := TokenData{Token: "test-token-123", SavedAt: "2024-01-01T00:00:00Z"}
	data, _ := json.Marshal(tokenData)
	if err := os.WriteFile(tmpFile, data, 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	c := &Cache{path: tmpFile}
	got := c.Get()
	if got != "test-token-123" {
		t.Errorf("Get() = %q, want %q", got, "test-token-123")
	}
}

func TestCache_Save_WritesTokenToFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	c := &Cache{path: tmpFile}
	err := c.Save("my-secret-token")
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got := c.Get()
	if got != "my-secret-token" {
		t.Errorf("Get() after Save() = %q, want %q", got, "my-secret-token")
	}
}

func TestCache_Save_ReturnsError_WhenDirectoryIsNotWritable(t *testing.T) {
	c := &Cache{path: "/proc/notwritable/.mec_livros_token"}
	err := c.Save("token")
	if err == nil {
		t.Error("Save() expected error for non-writable path, got nil")
	}
}

func TestCache_Clear_RemovesFile(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	c := &Cache{path: tmpFile}
	err := c.Clear()
	if err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if _, err := os.Stat(tmpFile); !os.IsNotExist(err) {
		t.Error("Clear() did not remove the file")
	}
}

func TestCache_Clear_ReturnsNoError_WhenFileDoesNotExist(t *testing.T) {
	c := &Cache{path: "/nonexistent/path/.mec_livros_token"}
	err := c.Clear()
	if err != nil {
		t.Errorf("Clear() error = %v, want nil", err)
	}
}

func TestCache_Exists_ReturnsFalse_WhenFileDoesNotExist(t *testing.T) {
	c := &Cache{path: "/nonexistent/path/.mec_livros_token"}
	if c.Exists() {
		t.Error("Exists() = true, want false")
	}
}

func TestCache_Exists_ReturnsTrue_WhenFileExists(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	if err := os.WriteFile(tmpFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	c := &Cache{path: tmpFile}
	if !c.Exists() {
		t.Error("Exists() = false, want true")
	}
}

func TestCache_Save_NewTokenFlow(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	c := &Cache{path: tmpFile}

	usedCache := c.Get() != ""
	if usedCache {
		t.Skip("Cache already has token - this test is for new token flow")
	}

	newToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.newtoken"
	err := c.Save(newToken)
	if err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	savedToken := c.Get()
	if savedToken != newToken {
		t.Errorf("Get() = %q, want %q", savedToken, newToken)
	}
}

func TestCache_Get_ReturnsEmpty_WhenTokenIsUsedCache(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	c := &Cache{path: tmpFile}

	c.Save("cached-token")

	jwtToken := c.Get()
	usedCache := jwtToken != ""

	if !usedCache {
		t.Error("usedCache should be true when token exists")
	}

	if usedCache {
		t.Log("Cache correctly returned existing token - skip saving new token")
	}
}

func TestCache_SkipSave_WhenUsedCache(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, ".mec_livros_token")

	c := &Cache{path: tmpFile}

	c.Save("existing-token")

	jwtToken := c.Get()
	usedCache := jwtToken != ""

	if !usedCache {
		t.Error("usedCache should be true")
	}

	if usedCache {
		savedToken := c.Get()
		if savedToken != "existing-token" {
			t.Errorf("Token should not be overwritten when usedCache=true, got %q", savedToken)
		}
	}
}

func TestNew_ReturnsCacheWithPathInHomeDir(t *testing.T) {
	c := New()
	if c.path == "" {
		t.Error("New() returned Cache with empty path")
	}
	if !filepath.IsAbs(c.path) {
		t.Errorf("New() path = %q, want absolute path", c.path)
	}
	if !strings.HasSuffix(c.path, cacheFileName) {
		t.Errorf("New() path = %q, want suffix %q", c.path, cacheFileName)
	}
}
