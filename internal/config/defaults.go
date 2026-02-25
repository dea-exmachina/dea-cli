package config

import (
	"os"
	"path/filepath"
)

const (
	DefaultEndpoint       = "https://hehldpjqlxhshdqqadng.supabase.co/functions/v1"
	DefaultTimeoutSeconds = 30
	DefaultProject        = "workspace-runtime"
)

// DeaDir returns the ~/.dea directory path.
func DeaDir() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ".dea"
	}
	return filepath.Join(home, ".dea")
}

// ConfigPath returns the path to ~/.dea/config.toml.
func ConfigPath() string {
	return filepath.Join(DeaDir(), "config.toml")
}

// TokensPath returns the path to ~/.dea/tokens.json.
func TokensPath() string {
	return filepath.Join(DeaDir(), "tokens.json")
}

// QueuePath returns the path to ~/.dea/queue.json.
func QueuePath() string {
	return filepath.Join(DeaDir(), "queue.json")
}
