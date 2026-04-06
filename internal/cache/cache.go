package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const cacheFileName = ".mec_livros_token"

// Cache manages JWT token persistence
type Cache struct {
	path string
}

// TokenData represents cached token with metadata
type TokenData struct {
	Token   string `json:"token"`
	SavedAt string `json:"saved_at"`
}

func New() *Cache {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	return &Cache{
		path: filepath.Join(homeDir, cacheFileName),
	}
}

func (c *Cache) Get() string {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return ""
	}

	var tokenData TokenData
	if err := json.Unmarshal(data, &tokenData); err != nil {
		return ""
	}

	return tokenData.Token
}

func (c *Cache) Save(token string) error {
	tokenData := TokenData{
		Token:   token,
		SavedAt: time.Now().Format(time.RFC3339),
	}

	data, err := json.Marshal(tokenData)
	if err != nil {
		return fmt.Errorf("marshal token: %w", err)
	}

	if err := os.WriteFile(c.path, data, 0o600); err != nil {
		return fmt.Errorf("write cache file: %w", err)
	}

	return nil
}

func (c *Cache) Clear() error {
	err := os.Remove(c.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove cache file: %w", err)
	}

	return nil
}

func (c *Cache) Exists() bool {
	_, err := os.Stat(c.path)
	return err == nil
}
