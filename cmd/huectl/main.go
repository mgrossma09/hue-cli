package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mgrossma09/hue-cli/internal/cli"
)

func main() {
	if err := cli.Run(context.Background(), os.Args[1:], os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
