package test

import (
	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

// NewCmd creates Command
func NewCmd() *cli.Command {
	return &cli.Command{
		Name:      "test",
		Usage:     "Show matching log lines without sending notifications",
		ArgsUsage: "FILE",
		Description: "Read FILE once, show lines matching the pattern and filters, then exit.\n" +
			"See talert --help for global options and list input rules.\n\n" +
			"Example:\n" +
			"  talert --pattern '^(ERROR|WARN) (.*)$' --level ERROR,WARN test app.log",

		Action: func(ctx *cli.Context) error {
			filename := ctx.Args().First()
			tester := internal.NewTester()

			return tester.Test(filename, &internal.Config{
				Pattern:  ctx.String("pattern"),
				Levels:   ctx.StringSlice("level"),
				Excludes: ctx.StringSlice("exclude"),
			})
		},
	}
}
