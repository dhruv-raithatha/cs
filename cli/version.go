package cli

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

// VersionCommand returns the cs version subcommand.
func VersionCommand() *cli.Command {
	return &cli.Command{
		Name:  "version",
		Usage: "Print the cs version",
		Action: func(_ context.Context, _ *cli.Command) error {
			fmt.Println("cs version v0.1.0")
			return nil
		},
	}
}
