package config

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func TestLoad_MissingFileIsAllowed(t *testing.T) {
	t.Setenv(EnvBridgeHost, "")
	t.Setenv(EnvAPIToken, "")
	t.Setenv(EnvInsecureTLS, "")

	cfg, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BridgeHost != "" || cfg.APIToken != "" {
		t.Fatalf("expected empty config, got %+v", cfg)
	}
}

func TestLoad_FileValues(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"bridge_host":"10.0.0.2","api_token":"from-file","insecure_tls":true}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv(EnvBridgeHost, "")
	t.Setenv(EnvAPIToken, "")
	t.Setenv(EnvInsecureTLS, "")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BridgeHost != "10.0.0.2" {
		t.Fatalf("BridgeHost = %q, want %q", cfg.BridgeHost, "10.0.0.2")
	}
	if cfg.APIToken != "from-file" {
		t.Fatalf("APIToken = %q, want %q", cfg.APIToken, "from-file")
	}
	if !cfg.InsecureTLS {
		t.Fatalf("InsecureTLS = %v, want true", cfg.InsecureTLS)
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	data := []byte(`{"bridge_host":"10.0.0.2","api_token":"from-file","insecure_tls":false}`)
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	t.Setenv(EnvBridgeHost, "10.0.0.99")
	t.Setenv(EnvAPIToken, "from-env")
	t.Setenv(EnvInsecureTLS, "true")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.BridgeHost != "10.0.0.99" {
		t.Fatalf("BridgeHost = %q, want %q", cfg.BridgeHost, "10.0.0.99")
	}
	if cfg.APIToken != "from-env" {
		t.Fatalf("APIToken = %q, want %q", cfg.APIToken, "from-env")
	}
	if !cfg.InsecureTLS {
		t.Fatalf("InsecureTLS = %v, want true", cfg.InsecureTLS)
	}
}

func TestLoadAndValidate(t *testing.T) {
	t.Setenv(EnvBridgeHost, "192.168.1.10")
	t.Setenv(EnvAPIToken, "abc123")
	t.Setenv(EnvInsecureTLS, "false")

	cfg, err := LoadAndValidate(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("LoadAndValidate() error = %v", err)
	}
	if cfg.BridgeHost != "192.168.1.10" || cfg.APIToken != "abc123" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoad_InvalidInsecureTLSEnv(t *testing.T) {
	t.Setenv(EnvInsecureTLS, "not-a-bool")

	_, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err == nil {
		t.Fatal("Load() expected error")
	}
	if !errors.Is(err, strconv.ErrSyntax) {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestConfigValidateErrors(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want error
	}{
		{name: "missing host", cfg: Config{APIToken: "abc"}, want: ErrMissingBridgeHost},
		{name: "missing token", cfg: Config{BridgeHost: "host"}, want: ErrMissingAPIToken},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if err != tt.want {
				t.Fatalf("Validate() error = %v, want %v", err, tt.want)
			}
		})
	}
}
