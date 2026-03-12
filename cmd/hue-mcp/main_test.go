package main

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mgrossma09/hue-cli/internal/config"
	"github.com/mgrossma09/hue-cli/internal/hue"
)

// newGroupTestServer returns an httptest server that serves a small fixture:
// two lights in two rooms. light-1 "Desk" in "Kitchen", light-2 "Lamp" in "Office".
func newGroupTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clip/v2/resource/light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"light-1","owner":{"rid":"device-1","rtype":"device"},"metadata":{"name":"Desk"},"on":{"on":true}},
				{"id":"light-2","owner":{"rid":"device-2","rtype":"device"},"metadata":{"name":"Lamp"},"on":{"on":false}}
			]}`)
		case "/clip/v2/resource/grouped_light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"gl-1","type":"grouped_light","owner":{"rid":"room-1","rtype":"room"}},
				{"id":"gl-2","type":"grouped_light","owner":{"rid":"room-2","rtype":"room"}}
			]}`)
		case "/clip/v2/resource/room":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"room-1","type":"room","metadata":{"name":"Kitchen"},"services":[{"rid":"gl-1","rtype":"grouped_light"}],"children":[{"rid":"device-1","rtype":"device"}]},
				{"id":"room-2","type":"room","metadata":{"name":"Office"},"services":[{"rid":"gl-2","rtype":"grouped_light"}],"children":[{"rid":"device-2","rtype":"device"}]}
			]}`)
		case "/clip/v2/resource/zone":
			_, _ = io.WriteString(w, `{"data":[]}`)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
}

func TestResolveLightIDs_DirectID(t *testing.T) {
	// When id is provided, no API calls are made.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	ids, err := resolveLightIDs(context.Background(), client, "abc-123", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "abc-123" {
		t.Fatalf("ids = %v, want [abc-123]", ids)
	}
}

func TestResolveLightIDs_NoIDNoGroup(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to %s", r.URL.Path)
	}))
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	_, err := resolveLightIDs(context.Background(), client, "", "", "")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolveLightIDs_ByGroup(t *testing.T) {
	srv := newGroupTestServer(t)
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	ids, err := resolveLightIDs(context.Background(), client, "", "kitchen", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "light-1" {
		t.Fatalf("ids = %v, want [light-1]", ids)
	}
}

func TestResolveLightIDs_ByGroupCaseInsensitive(t *testing.T) {
	srv := newGroupTestServer(t)
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	ids, err := resolveLightIDs(context.Background(), client, "", "OFFICE", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "light-2" {
		t.Fatalf("ids = %v, want [light-2]", ids)
	}
}

func TestResolveLightIDs_ByGroupAndName(t *testing.T) {
	srv := newGroupTestServer(t)
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	ids, err := resolveLightIDs(context.Background(), client, "", "Kitchen", "desk")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(ids) != 1 || ids[0] != "light-1" {
		t.Fatalf("ids = %v, want [light-1]", ids)
	}
}

func TestResolveLightIDs_ByGroupAndName_NoMatch(t *testing.T) {
	srv := newGroupTestServer(t)
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	_, err := resolveLightIDs(context.Background(), client, "", "Kitchen", "nonexistent")
	if err == nil {
		t.Fatal("expected error for unmatched name filter, got nil")
	}
}

func TestResolveLightIDs_GroupNotFound(t *testing.T) {
	srv := newGroupTestServer(t)
	defer srv.Close()

	client := hue.NewClient(config.Config{BridgeHost: srv.URL, APIToken: "token"}, srv.Client())
	_, err := resolveLightIDs(context.Background(), client, "", "Basement", "")
	if err == nil {
		t.Fatal("expected error for unknown group, got nil")
	}
}
