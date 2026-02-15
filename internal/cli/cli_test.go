package cli

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/mgrossma09/hue-cli/internal/config"
	"github.com/mgrossma09/hue-cli/internal/hue"
)

type fakeLightService struct {
	lights    []hue.Light
	toggleID  string
	updateID  string
	updateReq hue.UpdateLightRequest
	listErr   error
	toggleErr error
	updateErr error
}

func (f *fakeLightService) ListLights(ctx context.Context) ([]hue.Light, error) {
	_ = ctx
	return f.lights, f.listErr
}

func (f *fakeLightService) ToggleLight(ctx context.Context, id string) error {
	_ = ctx
	f.toggleID = id
	return f.toggleErr
}

func (f *fakeLightService) UpdateLight(ctx context.Context, id string, req hue.UpdateLightRequest) error {
	_ = ctx
	f.updateID = id
	f.updateReq = req
	return f.updateErr
}

func newTestApp(t *testing.T, svc *fakeLightService, stdout, stderr *bytes.Buffer) App {
	t.Helper()
	t.Setenv(config.EnvBridgeHost, "192.168.1.2")
	t.Setenv(config.EnvAPIToken, "token")

	return App{
		Stdout: stdout,
		Stderr: stderr,
		NewClient: func(cfg config.Config) lightService {
			if cfg.BridgeHost == "" || cfg.APIToken == "" {
				t.Fatalf("missing cfg values: %+v", cfg)
			}
			return svc
		},
	}
}

func TestRunLightsList(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2; output=%q", len(lines), stdout.String())
	}
	if got := strings.Fields(lines[0]); len(got) != 3 || got[0] != "ID" || got[1] != "NAME" || got[2] != "ON" {
		t.Fatalf("header fields = %#v", got)
	}
	if got := strings.Fields(lines[1]); len(got) != 3 || got[0] != "abc" || got[1] != "Desk" || got[2] != "true" {
		t.Fatalf("row fields = %#v", got)
	}
}

func TestRunLightsListJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--json"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var got []hue.Light
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got[0].ID != "abc" || got[0].Name != "Desk" || !got[0].On {
		t.Fatalf("unexpected json output: %+v", got)
	}
}

func TestRunLightsListCSV(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{
		{ID: "abc", Name: "Desk", On: true},
		{ID: "def", Name: "Kitchen", On: false},
	}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("record count = %d, want 3", len(records))
	}
	if strings.Join(records[0], ",") != "id,name,on" {
		t.Fatalf("header = %v", records[0])
	}
	if strings.Join(records[1], ",") != "abc,Desk,true" {
		t.Fatalf("first row = %v", records[1])
	}
	if strings.Join(records[2], ",") != "def,Kitchen,false" {
		t.Fatalf("second row = %v", records[2])
	}
}

func TestRunLightsListFormatMutuallyExclusive(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--json", "--csv"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsListHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "huectl lights list [--json|--csv]") {
		t.Fatalf("stdout missing list usage, got %q", output)
	}
	if !strings.Contains(output, "--json") || !strings.Contains(output, "--csv") {
		t.Fatalf("stdout missing options, got %q", output)
	}
}

func TestRunLightsListHelpShort(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "-h"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("stdout missing usage, got %q", got)
	}
}

func TestRunHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("stdout missing usage, got %q", got)
	}
}

func TestRunHelpWithoutConfig(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{Stdout: stdout, Stderr: stderr}

	err := app.Run(context.Background(), []string{"-h"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("stdout missing usage, got %q", got)
	}
}

func TestRunLightsToggle(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--id", "light-1"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if svc.toggleID != "light-1" {
		t.Fatalf("toggleID = %q", svc.toggleID)
	}
}

func TestRunLightsSet(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--id", "light-1", "--off", "--bri", "42", "--ct", "250", "--xy", "0.1,0.2"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if svc.updateID != "light-1" {
		t.Fatalf("updateID = %q", svc.updateID)
	}
	if svc.updateReq.On == nil || *svc.updateReq.On {
		t.Fatalf("On = %#v", svc.updateReq.On)
	}
	if svc.updateReq.Bri == nil || *svc.updateReq.Bri != 42 {
		t.Fatalf("Bri = %#v", svc.updateReq.Bri)
	}
	if svc.updateReq.CT == nil || *svc.updateReq.CT != 250 {
		t.Fatalf("CT = %#v", svc.updateReq.CT)
	}
	if svc.updateReq.XY == nil || svc.updateReq.XY.X != 0.1 || svc.updateReq.XY.Y != 0.2 {
		t.Fatalf("XY = %#v", svc.updateReq.XY)
	}
}

func TestRunLightsSetMutuallyExclusive(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--id", "light-1", "--on", "--off"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsSetBriRange(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--id", "light-1", "--bri", "101"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrInvalidRange) {
		t.Fatalf("error = %v", err)
	}
}
