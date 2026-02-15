package hue

import (
	"testing"

	"github.com/mgrossma09/hue-cli/internal/config"
)

func TestNewClient_DefaultHTTPClient(t *testing.T) {
	cfg := config.Config{BridgeHost: "192.168.1.2", APIToken: "token"}
	client := NewClient(cfg, nil)
	if client.HTTPClient == nil {
		t.Fatal("HTTPClient is nil")
	}
	if client.BridgeHost != cfg.BridgeHost {
		t.Fatalf("BridgeHost = %q, want %q", client.BridgeHost, cfg.BridgeHost)
	}
	if client.APIToken != cfg.APIToken {
		t.Fatalf("APIToken = %q, want %q", client.APIToken, cfg.APIToken)
	}
}
