package cli

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"

	"github.com/mgrossma09/hue-cli/internal/config"
	"github.com/mgrossma09/hue-cli/internal/hue"
)

var (
	ErrUsage        = errors.New("invalid command usage")
	ErrInvalidRange = errors.New("flag value out of range")
)

type lightService interface {
	ListLights(ctx context.Context) ([]hue.Light, error)
	ListLightsWithOptions(ctx context.Context, opts hue.ListLightsOptions) ([]hue.Light, error)
	ToggleLight(ctx context.Context, id string) error
	UpdateLight(ctx context.Context, id string, req hue.UpdateLightRequest) error
}

type App struct {
	ConfigPath string
	Stdout     io.Writer
	Stderr     io.Writer
	NewClient  func(cfg config.Config) lightService
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	app := App{Stdout: stdout, Stderr: stderr}
	return app.Run(ctx, args)
}

func (a App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printUsage(a.Stdout)
		return nil
	}

	cfg, err := config.LoadAndValidate(a.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	newClient := a.NewClient
	if newClient == nil {
		newClient = func(cfg config.Config) lightService {
			return hue.NewClient(cfg, nil)
		}
	}
	client := newClient(cfg)

	switch args[0] {
	case "lights":
		return a.runLights(ctx, client, args[1:])
	default:
		printUsage(a.Stderr)
		return fmt.Errorf("%w: unknown command %q", ErrUsage, args[0])
	}
}

func (a App) runLights(ctx context.Context, client lightService, args []string) error {
	if len(args) == 0 || isHelpArg(args[0]) {
		printLightsUsage(a.Stdout)
		return nil
	}

	switch args[0] {
	case "list":
		return runLightsList(ctx, a.Stdout, client, args[1:])
	case "toggle":
		return runLightsToggle(ctx, a.Stdout, client, args[1:])
	case "set":
		return runLightsSet(ctx, a.Stdout, client, args[1:])
	default:
		return fmt.Errorf("%w: unknown lights subcommand %q", ErrUsage, args[0])
	}
}

func runLightsList(ctx context.Context, stdout io.Writer, client lightService, args []string) error {
	fs := flag.NewFlagSet("lights list", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	jsonOut := fs.Bool("json", false, "Output as JSON")
	csvOut := fs.Bool("csv", false, "Output as CSV")
	withGroup := fs.Bool("with-group", false, "Include room/zone group metadata")
	withState := fs.Bool("with-state", false, "Include additional state fields")
	wide := fs.Bool("wide", false, "Equivalent to --with-group --with-state")
	groupFilter := fs.String("group", "", "Filter lights by room/zone name")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLightsListUsage(stdout)
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: lights list does not take positional arguments", ErrUsage)
	}
	if *jsonOut && *csvOut {
		return fmt.Errorf("%w: --json and --csv are mutually exclusive", ErrUsage)
	}
	if *wide {
		*withGroup = true
		*withState = true
	}
	if *groupFilter != "" {
		*withGroup = true
	}

	lights, err := client.ListLightsWithOptions(ctx, hue.ListLightsOptions{
		WithGroup: *withGroup,
		WithState: *withState,
	})
	if err != nil {
		return err
	}
	if *groupFilter != "" {
		lights = filterLightsByGroup(lights, *groupFilter)
	}

	if *jsonOut {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(buildListJSONRows(lights, *withGroup, *withState))
	}
	if *csvOut {
		writer := csv.NewWriter(stdout)
		header := listCSVColumns(*withGroup, *withState)
		if err := writer.Write(header); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, light := range lights {
			row := renderCSVRow(light, *withGroup, *withState)
			if err := writer.Write(row); err != nil {
				return fmt.Errorf("write csv row: %w", err)
			}
		}
		writer.Flush()
		if err := writer.Error(); err != nil {
			return fmt.Errorf("flush csv: %w", err)
		}
		return nil
	}

	if len(lights) == 0 {
		fmt.Fprintln(stdout, "no lights found")
		return nil
	}

	tw := tabwriter.NewWriter(stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, strings.ToUpper(strings.Join(listCSVColumns(*withGroup, *withState), "\t")))
	for _, light := range lights {
		fmt.Fprintln(tw, strings.Join(renderCSVRow(light, *withGroup, *withState), "\t"))
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush table output: %w", err)
	}

	return nil
}

func printLightsListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights list [--json|--csv] [--with-group] [--with-state] [--wide] [--group <name>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --json  Output lights as JSON")
	fmt.Fprintln(w, "  --csv   Output lights as CSV")
	fmt.Fprintln(w, "  --with-group  Include group and group_type fields")
	fmt.Fprintln(w, "  --with-state  Include reachable, bri, and color fields")
	fmt.Fprintln(w, "  --wide  Include both group metadata and extra state")
	fmt.Fprintln(w, "  --group  Filter by room/zone name (implies --with-group)")
}

func printLightsUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights list [--json|--csv] [--with-group] [--with-state] [--wide] [--group <name>]")
	fmt.Fprintln(w, "  huectl lights toggle (--id <id> | --group <group> [--name <name>])")
	fmt.Fprintln(w, "  huectl lights set (--id <id> | --group <group> [--name <name>]) [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]")
}

func listCSVColumns(withGroup, withState bool) []string {
	switch {
	case withGroup && withState:
		return []string{"id", "name", "group", "group_type", "on", "reachable", "bri", "color"}
	case withGroup:
		return []string{"id", "name", "group", "group_type", "on"}
	case withState:
		return []string{"id", "name", "on", "reachable", "bri", "color"}
	default:
		return []string{"id", "name", "on"}
	}
}

func renderCSVRow(light hue.Light, withGroup, withState bool) []string {
	row := []string{light.ID, light.Name}
	if withGroup {
		row = append(row, light.Group, light.GroupType)
	}
	row = append(row, strconv.FormatBool(light.On))
	if withState {
		reachable := ""
		if light.Reachable != nil {
			reachable = strconv.FormatBool(*light.Reachable)
		}
		row = append(row, reachable, hue.FormatBrightness(light.Bri), light.Color)
	}
	return row
}

func buildListJSONRows(lights []hue.Light, withGroup, withState bool) []map[string]any {
	rows := make([]map[string]any, 0, len(lights))
	for _, light := range lights {
		row := map[string]any{
			"id":   light.ID,
			"name": light.Name,
			"on":   light.On,
		}
		if withGroup {
			row["group"] = light.Group
			row["group_type"] = light.GroupType
		}
		if withState {
			if light.Reachable != nil {
				row["reachable"] = *light.Reachable
			} else {
				row["reachable"] = ""
			}
			row["bri"] = hue.FormatBrightness(light.Bri)
			row["color"] = light.Color
		}
		rows = append(rows, row)
	}
	return rows
}

func filterLightsByGroup(lights []hue.Light, group string) []hue.Light {
	filtered := make([]hue.Light, 0, len(lights))
	for _, light := range lights {
		if strings.EqualFold(light.Group, group) {
			filtered = append(filtered, light)
		}
	}
	return filtered
}

func runLightsToggle(ctx context.Context, stdout io.Writer, client lightService, args []string) error {
	fs := flag.NewFlagSet("lights toggle", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "Hue light ID")
	group := fs.String("group", "", "Room/zone name")
	name := fs.String("name", "", "Light name within group")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLightsToggleUsage(stdout)
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", ErrUsage)
	}
	if *id != "" && (*group != "" || *name != "") {
		return fmt.Errorf("%w: --id is mutually exclusive with --group/--name", ErrUsage)
	}
	if *name != "" && *group == "" {
		return fmt.Errorf("%w: --name requires --group", ErrUsage)
	}
	if *id == "" && *group == "" {
		return fmt.Errorf("%w: one of --id or --group is required", ErrUsage)
	}

	targetIDs, err := resolveLightTargetIDs(ctx, client, *id, *group, *name)
	if err != nil {
		return err
	}
	for _, targetID := range targetIDs {
		if err := client.ToggleLight(ctx, targetID); err != nil {
			return fmt.Errorf("toggle %s: %w", targetID, err)
		}
	}
	if *id != "" || *name != "" {
		fmt.Fprintf(stdout, "toggled %d light(s)\n", len(targetIDs))
		return nil
	}
	fmt.Fprintf(stdout, "toggled %d light(s) in group %s\n", len(targetIDs), *group)
	return nil
}

func printLightsToggleUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights toggle (--id <id> | --group <group> [--name <name>])")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --id     Hue light ID")
	fmt.Fprintln(w, "  --group  Room/zone name")
	fmt.Fprintln(w, "  --name   Light name within group (requires --group)")
}

func runLightsSet(ctx context.Context, stdout io.Writer, client lightService, args []string) error {
	fs := flag.NewFlagSet("lights set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "Hue light ID")
	group := fs.String("group", "", "Room/zone name")
	name := fs.String("name", "", "Light name within group")
	on := fs.Bool("on", false, "Turn light on")
	off := fs.Bool("off", false, "Turn light off")
	bri := fs.Int("bri", -1, "Brightness 0-100")
	ct := fs.Int("ct", -1, "Color temperature in mireks")
	xy := fs.String("xy", "", "XY color coordinates as x,y")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printLightsSetUsage(stdout)
			return nil
		}
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", ErrUsage)
	}
	if *id != "" && (*group != "" || *name != "") {
		return fmt.Errorf("%w: --id is mutually exclusive with --group/--name", ErrUsage)
	}
	if *name != "" && *group == "" {
		return fmt.Errorf("%w: --name requires --group", ErrUsage)
	}
	if *id == "" && *group == "" {
		return fmt.Errorf("%w: one of --id or --group is required", ErrUsage)
	}
	if *on && *off {
		return fmt.Errorf("%w: --on and --off are mutually exclusive", ErrUsage)
	}

	var req hue.UpdateLightRequest
	if *on {
		v := true
		req.On = &v
	}
	if *off {
		v := false
		req.On = &v
	}
	if *bri >= 0 {
		if *bri > 100 {
			return fmt.Errorf("%w: --bri must be between 0 and 100", ErrInvalidRange)
		}
		v := *bri
		req.Bri = &v
	}
	if *ct >= 0 {
		if *ct == 0 {
			return fmt.Errorf("%w: --ct must be greater than 0", ErrInvalidRange)
		}
		v := *ct
		req.CT = &v
	}
	if *xy != "" {
		parsedXY, err := parseXY(*xy)
		if err != nil {
			return err
		}
		req.XY = parsedXY
	}

	if req.On == nil && req.Bri == nil && req.CT == nil && req.XY == nil {
		return fmt.Errorf("%w: at least one field must be provided", ErrUsage)
	}

	targetIDs, err := resolveLightTargetIDs(ctx, client, *id, *group, *name)
	if err != nil {
		return err
	}
	for _, targetID := range targetIDs {
		if err := client.UpdateLight(ctx, targetID, req); err != nil {
			return fmt.Errorf("set %s: %w", targetID, err)
		}
	}
	if *id != "" || *name != "" {
		fmt.Fprintf(stdout, "updated %d light(s)\n", len(targetIDs))
		return nil
	}
	fmt.Fprintf(stdout, "updated %d light(s) in group %s\n", len(targetIDs), *group)
	return nil
}

func printLightsSetUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights set (--id <id> | --group <group> [--name <name>]) [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --id    Hue light ID")
	fmt.Fprintln(w, "  --group Room/zone name")
	fmt.Fprintln(w, "  --name  Light name within group (requires --group)")
	fmt.Fprintln(w, "  --on    Turn light on")
	fmt.Fprintln(w, "  --off   Turn light off")
	fmt.Fprintln(w, "  --bri   Brightness 0-100")
	fmt.Fprintln(w, "  --ct    Color temperature in mireks")
	fmt.Fprintln(w, "  --xy    XY color coordinates formatted as x,y")
}

func resolveLightTargetIDs(ctx context.Context, client lightService, id, group, name string) ([]string, error) {
	if id != "" {
		return []string{id}, nil
	}
	lights, err := client.ListLightsWithOptions(ctx, hue.ListLightsOptions{WithGroup: true})
	if err != nil {
		return nil, err
	}

	matches := make([]hue.Light, 0, len(lights))
	for _, light := range lights {
		if !strings.EqualFold(light.Group, group) {
			continue
		}
		if name != "" && !strings.EqualFold(light.Name, name) {
			continue
		}
		matches = append(matches, light)
	}
	if len(matches) == 0 {
		if name != "" {
			return nil, fmt.Errorf("%w: no light found for --group %q and --name %q", ErrUsage, group, name)
		}
		return nil, fmt.Errorf("%w: no lights found for --group %q", ErrUsage, group)
	}
	if name != "" && len(matches) > 1 {
		return nil, fmt.Errorf("%w: multiple lights matched --group %q and --name %q; use --id", ErrUsage, group, name)
	}

	ids := make([]string, 0, len(matches))
	for _, light := range matches {
		ids = append(ids, light.ID)
	}
	return ids, nil
}

func parseXY(raw string) (*hue.XY, error) {
	parts := strings.Split(raw, ",")
	if len(parts) != 2 {
		return nil, fmt.Errorf("%w: --xy must be formatted as x,y", ErrUsage)
	}

	x, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid --xy x value", ErrUsage)
	}
	y, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid --xy y value", ErrUsage)
	}
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return nil, fmt.Errorf("%w: --xy values must be between 0 and 1", ErrInvalidRange)
	}

	return &hue.XY{X: x, Y: y}, nil
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "huectl - control Philips Hue lights via Hue API v2")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights list")
	fmt.Fprintln(w, "  huectl lights toggle (--id <id> | --group <group> [--name <name>])")
	fmt.Fprintln(w, "  huectl lights set (--id <id> | --group <group> [--name <name>]) [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]")
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}
