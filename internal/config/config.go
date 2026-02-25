package config

import (
	"os"

	"github.com/BurntSushi/toml"
)

// Config holds the dea CLI configuration loaded from ~/.dea/config.toml.
type Config struct {
	Endpoint       string `toml:"endpoint"`
	DefaultProject string `toml:"default_project"`
	TimeoutSeconds int    `toml:"timeout_seconds"`
}

// Load reads the config from ~/.dea/config.toml. Returns defaults if the file
// does not exist.
func Load() (*Config, error) {
	cfg := &Config{
		Endpoint:       DefaultEndpoint,
		DefaultProject: DefaultProject,
		TimeoutSeconds: DefaultTimeoutSeconds,
	}

	path := ConfigPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	if _, err := toml.DecodeFile(path, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// Save writes the config to ~/.dea/config.toml.
func Save(cfg *Config) error {
	if err := os.MkdirAll(DeaDir(), 0700); err != nil {
		return err
	}

	f, err := os.Create(ConfigPath())
	if err != nil {
		return err
	}
	defer f.Close()

	return toml.NewEncoder(f).Encode(cfg)
}
