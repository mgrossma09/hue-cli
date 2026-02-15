package hue

import (
	"context"
	"crypto/tls"
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

func TestNewClient_InsecureTLS(t *testing.T) {
	cfg := config.Config{BridgeHost: "192.168.1.2", APIToken: "token", InsecureTLS: true}
	client := NewClient(cfg, nil)

	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.HTTPClient.Transport)
	}
	if transport.TLSClientConfig == nil {
		t.Fatal("TLSClientConfig is nil")
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false, want true")
	}
}

func TestNewClient_InsecureTLS_ClonesTransport(t *testing.T) {
	base := http.Transport{TLSClientConfig: &tls.Config{}}
	srcClient := &http.Client{Transport: &base}
	cfg := config.Config{BridgeHost: "192.168.1.2", APIToken: "token", InsecureTLS: true}

	client := NewClient(cfg, srcClient)
	if client.HTTPClient == srcClient {
		t.Fatal("client pointer should be cloned")
	}
	transport, ok := client.HTTPClient.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.HTTPClient.Transport)
	}
	if !transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("InsecureSkipVerify = false, want true")
	}
	if base.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("source transport was mutated")
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
		_, _ = io.WriteString(w, `{"data":[{"id":"id-1","metadata":{"name":"Desk"},"on":{"on":true},"dimming":{"brightness":66.6},"status":"connected"}]}`)
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
	if lights[0].Bri != nil || lights[0].Reachable != nil || lights[0].Group != "" || lights[0].GroupType != "" {
		t.Fatalf("default list unexpectedly included optional fields: %+v", lights[0])
	}
}

func TestListLightsWithOptions_GroupAndState(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clip/v2/resource/light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"light-1","owner":{"rid":"device-1","rtype":"device"},"metadata":{"name":"Desk"},"on":{"on":true},"dimming":{"brightness":65.2},"status":"connected"},
				{"id":"light-2","owner":{"rid":"device-2","rtype":"device"},"metadata":{"name":"Lamp"},"on":{"on":false},"status":"disconnected"},
				{"id":"light-3","metadata":{"name":"Porch"},"on":{"on":true}}
			]}`)
		case "/clip/v2/resource/zigbee_connectivity":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/grouped_light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"grouped-room-1","type":"grouped_light","owner":{"rid":"room-1","rtype":"room"}},
				{"id":"grouped-zone-1","type":"grouped_light","owner":{"rid":"zone-1","rtype":"zone"}},
				{"id":"grouped-room-2","type":"grouped_light","owner":{"rid":"room-2","rtype":"room"}}
			]}`)
		case "/clip/v2/resource/room":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"room-1","type":"room","metadata":{"name":"Kitchen"},"services":[{"rid":"grouped-room-1","rtype":"grouped_light"}],"children":[{"rid":"device-1","rtype":"device"}]},
				{"id":"room-2","type":"room","metadata":{"name":"Office"},"services":[{"rid":"grouped-room-2","rtype":"grouped_light"}],"children":[{"rid":"device-1","rtype":"device"},{"rid":"light-3","rtype":"light"}]}
			]}`)
		case "/clip/v2/resource/zone":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"zone-1","type":"zone","metadata":{"name":"Upstairs"},"services":[{"rid":"grouped-zone-1","rtype":"grouped_light"}],"children":[{"rid":"device-2","rtype":"device"}]}
			]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	lights, err := client.ListLightsWithOptions(context.Background(), ListLightsOptions{WithGroup: true, WithState: true})
	if err != nil {
		t.Fatalf("ListLightsWithOptions() error = %v", err)
	}
	if len(lights) != 3 {
		t.Fatalf("len(lights) = %d, want 3", len(lights))
	}

	byID := map[string]Light{}
	for _, light := range lights {
		byID[light.ID] = light
	}

	light1 := byID["light-1"]
	if light1.Group != "Kitchen" || light1.GroupType != "room" {
		t.Fatalf("light-1 group = (%q,%q), want (Kitchen,room)", light1.Group, light1.GroupType)
	}
	if light1.Reachable == nil || !*light1.Reachable {
		t.Fatalf("light-1 reachable = %#v, want true", light1.Reachable)
	}
	if light1.Bri == nil || *light1.Bri != 65.2 {
		t.Fatalf("light-1 bri = %#v, want 65.2", light1.Bri)
	}

	light2 := byID["light-2"]
	if light2.Group != "Upstairs" || light2.GroupType != "zone" {
		t.Fatalf("light-2 group = (%q,%q), want (Upstairs,zone)", light2.Group, light2.GroupType)
	}
	if light2.Reachable == nil || *light2.Reachable {
		t.Fatalf("light-2 reachable = %#v, want false", light2.Reachable)
	}
	if light2.Bri != nil {
		t.Fatalf("light-2 bri = %#v, want nil", light2.Bri)
	}

	light3 := byID["light-3"]
	if light3.Group != "Office" || light3.GroupType != "room" {
		t.Fatalf("light-3 group = (%q,%q), want (Office,room)", light3.Group, light3.GroupType)
	}
	if light3.Reachable != nil {
		t.Fatalf("light-3 reachable = %#v, want nil", light3.Reachable)
	}
}

func TestListLightsWithOptions_GroupFromDeviceChildren(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clip/v2/resource/light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"light-1","owner":{"rid":"device-1","rtype":"device"},"metadata":{"name":"Desk"},"on":{"on":true}}
			]}`)
		case "/clip/v2/resource/zigbee_connectivity":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/grouped_light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"grouped-room-1","type":"grouped_light","owner":{"rid":"room-1","rtype":"room"}}
			]}`)
		case "/clip/v2/resource/room":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"room-1","type":"room","metadata":{"name":"Kitchen"},"services":[{"rid":"grouped-room-1","rtype":"grouped_light"}],"children":[{"rid":"device-1","rtype":"device"}]}
			]}`)
		case "/clip/v2/resource/zone":
			_, _ = io.WriteString(w, `{"data":[]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	lights, err := client.ListLightsWithOptions(context.Background(), ListLightsOptions{WithGroup: true})
	if err != nil {
		t.Fatalf("ListLightsWithOptions() error = %v", err)
	}
	if len(lights) != 1 {
		t.Fatalf("len(lights) = %d, want 1", len(lights))
	}
	if lights[0].Group != "Kitchen" || lights[0].GroupType != "room" {
		t.Fatalf("unexpected group data: %+v", lights[0])
	}
}

func TestListLightsWithOptions_ReachableFromZigbeeConnectivity(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clip/v2/resource/light":
			_, _ = io.WriteString(w, `{"data":[
				{"id":"light-1","owner":{"rid":"device-1","rtype":"device"},"metadata":{"name":"Desk"},"on":{"on":true},"dimming":{"brightness":40.0}},
				{"id":"light-2","owner":{"rid":"device-2","rtype":"device"},"metadata":{"name":"Lamp"},"on":{"on":false}}
			]}`)
		case "/clip/v2/resource/zigbee_connectivity":
			_, _ = io.WriteString(w, `{"data":[
				{"owner":{"rid":"device-1","rtype":"device"},"status":"connected"},
				{"owner":{"rid":"device-2","rtype":"device"},"status":"disconnected"}
			]}`)
		case "/clip/v2/resource/grouped_light":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/room":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/zone":
			_, _ = io.WriteString(w, `{"data":[]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	lights, err := client.ListLightsWithOptions(context.Background(), ListLightsOptions{WithState: true})
	if err != nil {
		t.Fatalf("ListLightsWithOptions() error = %v", err)
	}
	if len(lights) != 2 {
		t.Fatalf("len(lights) = %d, want 2", len(lights))
	}
	byID := map[string]Light{}
	for _, light := range lights {
		byID[light.ID] = light
	}

	light1 := byID["light-1"]
	if light1.Reachable == nil || !*light1.Reachable {
		t.Fatalf("light-1 reachable = %#v, want true", light1.Reachable)
	}
	light2 := byID["light-2"]
	if light2.Reachable == nil || *light2.Reachable {
		t.Fatalf("light-2 reachable = %#v, want false", light2.Reachable)
	}
}

func TestListLightsWithOptions_NoGroupFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/clip/v2/resource/light":
			_, _ = io.WriteString(w, `{"data":[{"id":"light-1","metadata":{"name":"Desk"},"on":{"on":true}}]}`)
		case "/clip/v2/resource/grouped_light":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/room":
			_, _ = io.WriteString(w, `{"data":[]}`)
		case "/clip/v2/resource/zone":
			_, _ = io.WriteString(w, `{"data":[]}`)
		default:
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	client := NewClient(config.Config{BridgeHost: server.URL, APIToken: "token"}, server.Client())
	lights, err := client.ListLightsWithOptions(context.Background(), ListLightsOptions{WithGroup: true})
	if err != nil {
		t.Fatalf("ListLightsWithOptions() error = %v", err)
	}
	if len(lights) != 1 {
		t.Fatalf("len(lights) = %d, want 1", len(lights))
	}
	if lights[0].Group != "" || lights[0].GroupType != "" {
		t.Fatalf("unexpected group info: %+v", lights[0])
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
