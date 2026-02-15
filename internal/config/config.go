package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const (
	EnvBridgeHost = "HUE_BRIDGE_HOST"
	EnvAPIToken   = "HUE_API_TOKEN"
)

var (
	ErrMissingBridgeHost = errors.New("missing Hue bridge host")
	ErrMissingAPIToken   = errors.New("missing Hue API token")
)

type Config struct {
	BridgeHost string `json:"bridge_host"`
	APIToken   string `json:"api_token"`
}

func (c Config) Validate() error {
	if c.BridgeHost == "" {
		return ErrMissingBridgeHost
	}
	if c.APIToken == "" {
		return ErrMissingAPIToken
	}
	return nil
}

func (c Config) RedactedToken() string {
	if c.APIToken == "" {
		return ""
	}
	if len(c.APIToken) <= 4 {
		return "****"
	}
	return c.APIToken[:2] + "..." + c.APIToken[len(c.APIToken)-2:]
}

func DefaultPath() (string, error) {
	baseDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(baseDir, "huectl", "config.json"), nil
}

func Load(path string) (Config, error) {
	cfg, err := loadFromFile(path)
	if err != nil {
		return Config{}, err
	}

	if envBridgeHost := os.Getenv(EnvBridgeHost); envBridgeHost != "" {
		cfg.BridgeHost = envBridgeHost
	}
	if envToken := os.Getenv(EnvAPIToken); envToken != "" {
		cfg.APIToken = envToken
	}

	return cfg, nil
}

func LoadAndValidate(path string) (Config, error) {
	cfg, err := Load(path)
	if err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func loadFromFile(path string) (Config, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("read config file %q: %w", path, err)
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", path, err)
	}
	return cfg, nil
}
