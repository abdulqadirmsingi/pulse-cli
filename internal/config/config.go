package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const AppVersion = "0.2.6"

type Config struct {
	DataDir string // ~/.devpulse/
	DBPath  string // ~/.devpulse/pulse.db
}

func Load() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("finding home directory: %w", err)
	}

	dataDir := filepath.Join(home, ".devpulse")
	return &Config{
		DataDir: dataDir,
		DBPath:  filepath.Join(dataDir, "pulse.db"),
	}, nil
}

func (c *Config) EnsureDir() error {
	return os.MkdirAll(c.DataDir, 0755)
}
