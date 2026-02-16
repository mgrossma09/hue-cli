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
	lights          []hue.Light
	lastListOptions hue.ListLightsOptions
	toggleID        string
	toggleIDs       []string
	updateID        string
	updateIDs       []string
	updateReq       hue.UpdateLightRequest
	updateReqs      []hue.UpdateLightRequest
	listErr         error
	toggleErr       error
	updateErr       error
}

func (f *fakeLightService) ListLights(ctx context.Context) ([]hue.Light, error) {
	_ = ctx
	return f.lights, f.listErr
}

func (f *fakeLightService) ListLightsWithOptions(ctx context.Context, opts hue.ListLightsOptions) ([]hue.Light, error) {
	_ = ctx
	f.lastListOptions = opts
	return f.lights, f.listErr
}

func (f *fakeLightService) ToggleLight(ctx context.Context, id string) error {
	_ = ctx
	f.toggleID = id
	f.toggleIDs = append(f.toggleIDs, id)
	return f.toggleErr
}

func (f *fakeLightService) UpdateLight(ctx context.Context, id string, req hue.UpdateLightRequest) error {
	_ = ctx
	f.updateID = id
	f.updateIDs = append(f.updateIDs, id)
	f.updateReq = req
	f.updateReqs = append(f.updateReqs, req)
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
	if svc.lastListOptions.WithGroup || svc.lastListOptions.WithState {
		t.Fatalf("unexpected list options: %+v", svc.lastListOptions)
	}
}

func TestRunLightsListJSON(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	reachable := true
	bri := 51.0
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true, Reachable: &reachable, Bri: &bri, Color: "yes"}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--json", "--with-state"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	var got []map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if len(got) != 1 || got[0]["id"] != "abc" || got[0]["name"] != "Desk" || got[0]["on"] != true {
		t.Fatalf("unexpected json output: %+v", got)
	}
	if got[0]["color"] != "yes" {
		t.Fatalf("unexpected color value: %+v", got[0]["color"])
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

func TestRunLightsListCSVWithGroup(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", Group: "Kitchen", GroupType: "room", On: true}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv", "--with-group"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if strings.Join(records[0], ",") != "id,name,group,group_type,on" {
		t.Fatalf("header = %v", records[0])
	}
	if strings.Join(records[1], ",") != "abc,Desk,Kitchen,room,true" {
		t.Fatalf("row = %v", records[1])
	}
	if !svc.lastListOptions.WithGroup || svc.lastListOptions.WithState {
		t.Fatalf("unexpected list options: %+v", svc.lastListOptions)
	}
}

func TestRunLightsListCSVWithState(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	reachable := true
	bri := 42.0
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true, Reachable: &reachable, Bri: &bri, Color: "yes"}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv", "--with-state"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if strings.Join(records[0], ",") != "id,name,on,reachable,bri,color" {
		t.Fatalf("header = %v", records[0])
	}
	if strings.Join(records[1], ",") != "abc,Desk,true,true,42,yes" {
		t.Fatalf("row = %v", records[1])
	}
	if svc.lastListOptions.WithGroup || !svc.lastListOptions.WithState {
		t.Fatalf("unexpected list options: %+v", svc.lastListOptions)
	}
}

func TestRunLightsListCSVWideWithMissingFields(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true, Color: "unknown"}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv", "--wide"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if strings.Join(records[0], ",") != "id,name,group,group_type,on,reachable,bri,color" {
		t.Fatalf("header = %v", records[0])
	}
	if strings.Join(records[1], ",") != "abc,Desk,,,true,,,unknown" {
		t.Fatalf("row = %v", records[1])
	}
	if !svc.lastListOptions.WithGroup || !svc.lastListOptions.WithState {
		t.Fatalf("unexpected list options: %+v", svc.lastListOptions)
	}
}

func TestRunLightsListCSVWithGroupNoGroup(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{{ID: "abc", Name: "Desk", On: true}}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv", "--with-group"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if strings.Join(records[1], ",") != "abc,Desk,,,true" {
		t.Fatalf("row = %v", records[1])
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
	if !strings.Contains(output, "--with-group") || !strings.Contains(output, "--with-state") || !strings.Contains(output, "--wide") {
		t.Fatalf("stdout missing phase3 options, got %q", output)
	}
	if !strings.Contains(output, "--group") {
		t.Fatalf("stdout missing --group option, got %q", output)
	}
}

func TestRunLightsListGroupFilter(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{lights: []hue.Light{
		{ID: "a", Name: "Desk", Group: "Office", GroupType: "room", On: true},
		{ID: "b", Name: "Lamp", Group: "Kitchen", GroupType: "room", On: true},
	}}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "list", "--csv", "--group", "kitchen"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	reader := csv.NewReader(bytes.NewReader(stdout.Bytes()))
	records, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("record count = %d, want 2", len(records))
	}
	if strings.Join(records[1], ",") != "b,Lamp,Kitchen,room,true" {
		t.Fatalf("row = %v", records[1])
	}
	if !svc.lastListOptions.WithGroup {
		t.Fatalf("--group should imply withGroup, got %+v", svc.lastListOptions)
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

func TestRunLightsHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "huectl lights list") {
		t.Fatalf("stdout missing lights usage, got %q", output)
	}
}

func TestRunLightsHelpShort(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "-h"})
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
	if len(svc.toggleIDs) != 1 {
		t.Fatalf("toggleIDs len = %d, want 1", len(svc.toggleIDs))
	}
}

func TestRunLightsToggleHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "huectl lights toggle (--id <id> | --group <group> [--name <name>])") {
		t.Fatalf("stdout missing toggle usage, got %q", got)
	}
}

func TestRunLightsToggleHelpShort(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "-h"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("stdout missing usage, got %q", got)
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
	if len(svc.updateIDs) != 1 {
		t.Fatalf("updateIDs len = %d, want 1", len(svc.updateIDs))
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

func TestRunLightsToggleGroupAll(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{
		lights: []hue.Light{
			{ID: "a", Name: "Desk", Group: "Office"},
			{ID: "b", Name: "Lamp", Group: "Office"},
			{ID: "c", Name: "Porch", Group: "Outside"},
		},
	}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--group", "office"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(svc.toggleIDs) != 2 || svc.toggleIDs[0] != "a" || svc.toggleIDs[1] != "b" {
		t.Fatalf("toggleIDs = %v, want [a b]", svc.toggleIDs)
	}
	if !svc.lastListOptions.WithGroup {
		t.Fatalf("expected group-aware list options, got %+v", svc.lastListOptions)
	}
}

func TestRunLightsToggleGroupName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{
		lights: []hue.Light{
			{ID: "a", Name: "Desk", Group: "Office"},
			{ID: "b", Name: "Lamp", Group: "Office"},
		},
	}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--group", "office", "--name", "lamp"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(svc.toggleIDs) != 1 || svc.toggleIDs[0] != "b" {
		t.Fatalf("toggleIDs = %v, want [b]", svc.toggleIDs)
	}
}

func TestRunLightsToggleGroupNameRequiresGroup(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--name", "lamp"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsToggleMutuallyExclusiveTargetFlags(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "toggle", "--id", "x", "--group", "office"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsSetGroupAll(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{
		lights: []hue.Light{
			{ID: "a", Name: "Desk", Group: "Office"},
			{ID: "b", Name: "Lamp", Group: "Office"},
			{ID: "c", Name: "Porch", Group: "Outside"},
		},
	}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--group", "office", "--off"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(svc.updateIDs) != 2 || svc.updateIDs[0] != "a" || svc.updateIDs[1] != "b" {
		t.Fatalf("updateIDs = %v, want [a b]", svc.updateIDs)
	}
}

func TestRunLightsSetGroupName(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{
		lights: []hue.Light{
			{ID: "a", Name: "Desk", Group: "Office"},
			{ID: "b", Name: "Lamp", Group: "Office"},
		},
	}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--group", "office", "--name", "desk", "--on"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(svc.updateIDs) != 1 || svc.updateIDs[0] != "a" {
		t.Fatalf("updateIDs = %v, want [a]", svc.updateIDs)
	}
}

func TestRunLightsSetGroupNameRequiresGroup(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--name", "desk", "--on"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsSetMutuallyExclusiveTargetFlags(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--id", "x", "--group", "office", "--on"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunLightsSetHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "huectl lights set (--id <id> | --group <group> [--name <name>])") {
		t.Fatalf("stdout missing set usage, got %q", output)
	}
	if !strings.Contains(output, "--bri") || !strings.Contains(output, "--ct") || !strings.Contains(output, "--xy") || !strings.Contains(output, "--group") || !strings.Contains(output, "--name") {
		t.Fatalf("stdout missing set options, got %q", output)
	}
}

func TestRunLightsSetHelpShort(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	svc := &fakeLightService{}
	app := newTestApp(t, svc, stdout, stderr)

	err := app.Run(context.Background(), []string{"lights", "set", "-h"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := stdout.String(); !strings.Contains(got, "Usage:") {
		t.Fatalf("stdout missing usage, got %q", got)
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

func TestRunGetToken(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{
		Stdout: stdout,
		Stderr: stderr,
		GetToken: func(ctx context.Context, bridgeHost string) (string, error) {
			_ = ctx
			if bridgeHost != "192.168.1.2" {
				t.Fatalf("bridgeHost = %q", bridgeHost)
			}
			return "token-123", nil
		},
	}

	err := app.Run(context.Background(), []string{"get-token", "--bridge-host", "192.168.1.2"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if got := strings.TrimSpace(stdout.String()); got != "token-123" {
		t.Fatalf("stdout = %q, want token-123", got)
	}
	if got := strings.TrimSpace(stderr.String()); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRunGetTokenRequiresBridgeHost(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{Stdout: stdout, Stderr: stderr}

	err := app.Run(context.Background(), []string{"get-token"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrUsage) {
		t.Fatalf("error = %v", err)
	}
}

func TestRunGetTokenHelp(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{Stdout: stdout, Stderr: stderr}

	err := app.Run(context.Background(), []string{"get-token", "--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "huectl get-token --bridge-host <host>") {
		t.Fatalf("stdout missing usage, got %q", output)
	}
	if !strings.Contains(output, "press the Hue Bridge button") {
		t.Fatalf("stdout missing hint, got %q", output)
	}
}

func TestRunGetTokenLinkButtonHintOnNoToken(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{
		Stdout: stdout,
		Stderr: stderr,
		GetToken: func(ctx context.Context, bridgeHost string) (string, error) {
			_ = ctx
			_ = bridgeHost
			return "", hue.ErrNoToken
		},
	}

	err := app.Run(context.Background(), []string{"get-token", "--bridge-host", "192.168.1.2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, hue.ErrNoToken) {
		t.Fatalf("error = %v", err)
	}
	if got := stderr.String(); !strings.Contains(got, "within 30 seconds") {
		t.Fatalf("stderr missing hint, got %q", got)
	}
}

func TestRunHelpIncludesGetTokenHint(t *testing.T) {
	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	app := App{Stdout: stdout, Stderr: stderr}

	err := app.Run(context.Background(), []string{"--help"})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	output := stdout.String()
	if !strings.Contains(output, "huectl get-token --bridge-host <host>") {
		t.Fatalf("stdout missing get-token usage, got %q", output)
	}
	if !strings.Contains(output, "press the Hue Bridge button") {
		t.Fatalf("stdout missing bridge button hint, got %q", output)
	}
}
