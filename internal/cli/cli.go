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
	if len(args) == 0 {
		return fmt.Errorf("%w: missing lights subcommand", ErrUsage)
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

	lights, err := client.ListLights(ctx)
	if err != nil {
		return err
	}

	if *jsonOut {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		return encoder.Encode(lights)
	}
	if *csvOut {
		writer := csv.NewWriter(stdout)
		if err := writer.Write([]string{"id", "name", "on"}); err != nil {
			return fmt.Errorf("write csv header: %w", err)
		}
		for _, light := range lights {
			row := []string{light.ID, light.Name, strconv.FormatBool(light.On)}
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
	fmt.Fprintln(tw, "ID\tNAME\tON")
	for _, light := range lights {
		fmt.Fprintf(tw, "%s\t%s\t%t\n", light.ID, light.Name, light.On)
	}
	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flush table output: %w", err)
	}

	return nil
}

func printLightsListUsage(w io.Writer) {
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights list [--json|--csv]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Options:")
	fmt.Fprintln(w, "  --json  Output lights as JSON")
	fmt.Fprintln(w, "  --csv   Output lights as CSV")
}

func runLightsToggle(ctx context.Context, stdout io.Writer, client lightService, args []string) error {
	fs := flag.NewFlagSet("lights toggle", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "Hue light ID")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", ErrUsage)
	}
	if *id == "" {
		return fmt.Errorf("%w: --id is required", ErrUsage)
	}

	if err := client.ToggleLight(ctx, *id); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "toggled light %s\n", *id)
	return nil
}

func runLightsSet(ctx context.Context, stdout io.Writer, client lightService, args []string) error {
	fs := flag.NewFlagSet("lights set", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	id := fs.String("id", "", "Hue light ID")
	on := fs.Bool("on", false, "Turn light on")
	off := fs.Bool("off", false, "Turn light off")
	bri := fs.Int("bri", -1, "Brightness 0-100")
	ct := fs.Int("ct", -1, "Color temperature in mireks")
	xy := fs.String("xy", "", "XY color coordinates as x,y")

	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("%w: %v", ErrUsage, err)
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("%w: unexpected positional arguments", ErrUsage)
	}
	if *id == "" {
		return fmt.Errorf("%w: --id is required", ErrUsage)
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

	if err := client.UpdateLight(ctx, *id, req); err != nil {
		return err
	}
	fmt.Fprintf(stdout, "updated light %s\n", *id)
	return nil
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
	fmt.Fprintln(w, "  huectl lights toggle --id <id>")
	fmt.Fprintln(w, "  huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]")
}

func isHelpArg(arg string) bool {
	return arg == "-h" || arg == "--help"
}
