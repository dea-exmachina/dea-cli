package auth

import (
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/dea-exmachina/dea-cli/internal/config"
)

// TokenData is the structure stored in ~/.dea/tokens.json.
type TokenData struct {
	WorkspaceToken string    `json:"workspace_token"`
	TokenType      string    `json:"token_type"`
	ExpiresAt      time.Time `json:"expires_at"`
	WorkspaceID    string    `json:"workspace_id"`
	AgentID        string    `json:"agent_id"`
	Endpoint       string    `json:"endpoint"`
}

// TokenStore manages reading and writing the token from disk.
// It implements api.TokenProvider via the GetToken() method.
type TokenStore struct {
	mu   sync.RWMutex
	path string
}

// NewTokenStore creates a TokenStore pointing at ~/.dea/tokens.json.
func NewTokenStore() *TokenStore {
	return &TokenStore{path: config.TokensPath()}
}

// GetToken returns the raw workspace JWT string, or "" if not authenticated.
// Implements api.TokenProvider.
func (s *TokenStore) GetToken() string {
	token := s.Load()
	if token == nil {
		return ""
	}
	return token.WorkspaceToken
}

// Load reads the token from disk. Returns nil if none exists.
func (s *TokenStore) Load() *TokenData {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil
	}

	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil
	}
	return &token
}

// Save writes the token to disk.
func (s *TokenStore) Save(token *TokenData) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.MkdirAll(config.DeaDir(), 0700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(token, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0600)
}

// Clear removes the stored token.
func (s *TokenStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.path)
}
