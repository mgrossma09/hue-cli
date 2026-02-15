package cli

import (
	"bytes"
	"context"
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
	if got := strings.TrimSpace(stdout.String()); got != "abc\tDesk\ttrue" {
		t.Fatalf("stdout = %q", got)
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
