package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
)

const (
	EnvBridgeHost  = "HUE_BRIDGE_HOST"
	EnvAPIToken    = "HUE_API_TOKEN"
	EnvInsecureTLS = "HUE_INSECURE_TLS"
)

var (
	ErrMissingBridgeHost = errors.New("missing Hue bridge host")
	ErrMissingAPIToken   = errors.New("missing Hue API token")
)

type Config struct {
	BridgeHost  string `json:"bridge_host"`
	APIToken    string `json:"api_token"`
	InsecureTLS bool   `json:"insecure_tls"`
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

func legacyDefaultPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(homeDir, ".config", "huectl", "config.json"), nil
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
	if envInsecureTLS := os.Getenv(EnvInsecureTLS); envInsecureTLS != "" {
		insecureTLS, err := strconv.ParseBool(envInsecureTLS)
		if err != nil {
			return Config{}, fmt.Errorf("parse %s: %w", EnvInsecureTLS, err)
		}
		cfg.InsecureTLS = insecureTLS
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
	usedDefaultPath := false
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return Config{}, err
		}
		usedDefaultPath = true
	}

	fallbackPath := ""
	if usedDefaultPath {
		legacyPath, err := legacyDefaultPath()
		if err != nil {
			return Config{}, err
		}
		fallbackPath = legacyPath
	}

	content, loadedPath, err := readConfigFile(path, fallbackPath)
	if err != nil {
		return Config{}, err
	}
	if content == nil {
		return Config{}, nil
	}

	var cfg Config
	if err := json.Unmarshal(content, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config file %q: %w", loadedPath, err)
	}
	return cfg, nil
}

func readConfigFile(primaryPath, fallbackPath string) ([]byte, string, error) {
	content, err := os.ReadFile(primaryPath)
	if err == nil {
		return content, primaryPath, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, "", fmt.Errorf("read config file %q: %w", primaryPath, err)
	}

	if fallbackPath != "" && fallbackPath != primaryPath {
		fallbackContent, fallbackErr := os.ReadFile(fallbackPath)
		if fallbackErr == nil {
			return fallbackContent, fallbackPath, nil
		}
		if !errors.Is(fallbackErr, os.ErrNotExist) {
			return nil, "", fmt.Errorf("read config file %q: %w", fallbackPath, fallbackErr)
		}
	}

	return nil, "", nil
}
