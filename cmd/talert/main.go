package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/cmd/talert/test"
	"github.com/keecon/talert/cmd/talert/watch"
	"github.com/keecon/talert/internal"
)

func main() {
	app := &cli.App{
		Name:      "talert",
		Usage:     "Filter logs and send matching events to Slack",
		ArgsUsage: "FILE",
		Description: "Place global options before the command.\n\n" +
			"List options (--level, --exclude, --webhook-owner):\n" +
			"  Repeat the option or separate values with commas.\n" +
			"  Environment variables also use commas.\n" +
			"  Leading and trailing whitespace is trimmed.",
		Version:  internal.Version(),
		Compiled: internal.BuildDate(),
		Authors: []*cli.Author{
			{
				Name:  "iwaltgen",
				Email: "iwaltgen@gmail.com",
			},
		},

		Suggest:                true,
		UseShortOptionHandling: true,
		EnableBashCompletion:   true,

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "pattern",
				Aliases: []string{"p"},
				Value:   "^.+\\s+([A-Z]+).* : (.+)$",
				Usage:   "Parse log lines using `REGEXP`; group 1 = level, group 2 = message (at least 2 groups required)",
				EnvVars: []string{"TALERT_PATTERN"},
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Aliases: []string{"e"},
				Usage:   "Skip lines containing any `TEXT` substring (case-sensitive)",
				EnvVars: []string{"TALERT_EXCLUDE"},
			},
			&cli.StringSliceFlag{
				Name:    "level",
				Aliases: []string{"l"},
				Value:   cli.NewStringSlice("ERROR"),
				Usage:   "Match any specified log `LEVEL` (case-sensitive)",
				EnvVars: []string{"TALERT_LEVEL"},
			},
		},

		Commands: cli.Commands{
			watch.NewCmd(),
			test.NewCmd(),
		},

		Before: func(ctx *cli.Context) error {
			log.SetFlags(0)
			return nil
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
