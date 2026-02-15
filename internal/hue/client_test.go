package hue

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	if client.BaseURL != "https://192.168.1.2" {
		t.Fatalf("BaseURL = %q, want %q", client.BaseURL, "https://192.168.1.2")
	}
}

func TestListLights(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clip/v2/resource/light" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Fatalf("method = %q", r.Method)
		}
		if got := r.Header.Get("hue-application-key"); got != "token" {
			t.Fatalf("hue-application-key = %q", got)
		}
		_, _ = io.WriteString(w, `{"data":[{"id":"id-1","metadata":{"name":"Desk"},"on":{"on":true}}]}`)
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	lights, err := client.ListLights(context.Background())
	if err != nil {
		t.Fatalf("ListLights() error = %v", err)
	}
	if len(lights) != 1 {
		t.Fatalf("len(lights) = %d, want 1", len(lights))
	}
	if lights[0].ID != "id-1" || lights[0].Name != "Desk" || !lights[0].On {
		t.Fatalf("unexpected light: %+v", lights[0])
	}
}

func TestUpdateLightPayload(t *testing.T) {
	var got map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clip/v2/resource/light/light-1" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPut {
			t.Fatalf("method = %q", r.Method)
		}
		defer r.Body.Close()
		if err := json.NewDecoder(r.Body).Decode(&got); err != nil {
			t.Fatalf("Decode() error = %v", err)
		}
		_, _ = io.WriteString(w, `{"data":[{"rid":"light-1"}]}`)
	}))
	defer server.Close()

	bri := 55
	ct := 250
	x := &XY{X: 0.12, Y: 0.34}
	on := true

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	err := client.UpdateLight(context.Background(), "light-1", UpdateLightRequest{On: &on, Bri: &bri, CT: &ct, XY: x})
	if err != nil {
		t.Fatalf("UpdateLight() error = %v", err)
	}

	onMap, ok := got["on"].(map[string]any)
	if !ok || onMap["on"] != true {
		t.Fatalf("on payload = %#v", got["on"])
	}
	dimmingMap, ok := got["dimming"].(map[string]any)
	if !ok || dimmingMap["brightness"].(float64) != 55 {
		t.Fatalf("dimming payload = %#v", got["dimming"])
	}
	ctMap, ok := got["color_temperature"].(map[string]any)
	if !ok || ctMap["mirek"].(float64) != 250 {
		t.Fatalf("color_temperature payload = %#v", got["color_temperature"])
	}
	colorMap, ok := got["color"].(map[string]any)
	if !ok {
		t.Fatalf("color payload = %#v", got["color"])
	}
	xyMap, ok := colorMap["xy"].(map[string]any)
	if !ok || xyMap["x"].(float64) != 0.12 || xyMap["y"].(float64) != 0.34 {
		t.Fatalf("xy payload = %#v", colorMap["xy"])
	}
}

func TestToggleLight(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		switch calls {
		case 1:
			if r.Method != http.MethodGet || r.URL.Path != "/clip/v2/resource/light/light-1" {
				t.Fatalf("unexpected first request: %s %s", r.Method, r.URL.Path)
			}
			_, _ = io.WriteString(w, `{"data":[{"id":"light-1","metadata":{"name":"Desk"},"on":{"on":false}}]}`)
		case 2:
			if r.Method != http.MethodPut || r.URL.Path != "/clip/v2/resource/light/light-1" {
				t.Fatalf("unexpected second request: %s %s", r.Method, r.URL.Path)
			}
			defer r.Body.Close()
			body, _ := io.ReadAll(r.Body)
			if string(body) != `{"on":{"on":true}}` {
				t.Fatalf("body = %s", string(body))
			}
			_, _ = io.WriteString(w, `{"data":[{"rid":"light-1"}]}`)
		default:
			t.Fatalf("too many calls: %d", calls)
		}
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	if err := client.ToggleLight(context.Background(), "light-1"); err != nil {
		t.Fatalf("ToggleLight() error = %v", err)
	}
}

func TestUpdateLightNoFields(t *testing.T) {
	client := NewClient(config.Config{BridgeHost: "http://127.0.0.1", APIToken: "token"}, http.DefaultClient)
	err := client.UpdateLight(context.Background(), "id-1", UpdateLightRequest{})
	if err != ErrNoUpdateFields {
		t.Fatalf("UpdateLight() error = %v, want %v", err, ErrNoUpdateFields)
	}
}
