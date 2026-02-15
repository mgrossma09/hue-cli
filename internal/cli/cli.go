package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mgrossma09/hue-cli/internal/config"
	"github.com/mgrossma09/hue-cli/internal/hue"
)

type App struct {
	ConfigPath string
	Stdout     io.Writer
	Stderr     io.Writer
}

func Run(ctx context.Context, args []string, stdout, stderr io.Writer) error {
	app := App{Stdout: stdout, Stderr: stderr}
	return app.Run(ctx, args)
}

func (a App) Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage(a.Stdout)
		return nil
	}

	cfg, err := config.LoadAndValidate(a.ConfigPath)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	_ = hue.NewClient(cfg, nil)
	return notImplemented(args)
}

func printUsage(w io.Writer) {
	fmt.Fprintln(w, "huectl (Phase 1 scaffold)")
	fmt.Fprintln(w, "Usage:")
	fmt.Fprintln(w, "  huectl lights list")
	fmt.Fprintln(w, "  huectl lights toggle --id <id>")
	fmt.Fprintln(w, "  huectl lights set --id <id> [--on|--off] [--bri <0-100>] [--ct <mireds>] [--xy <x,y>]")
}

func notImplemented(args []string) error {
	return fmt.Errorf("command not implemented in Phase 1: %s", strings.Join(args, " "))
}
