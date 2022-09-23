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
		Name:     "tail alert",
		Usage:    "read, check and alert from continuously updated files",
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
				Usage:   "message pattern in a log line (must need 2 submatch)",
				EnvVars: []string{"TALERT_PATTERN"},
			},
			&cli.StringSliceFlag{
				Name:    "exclude",
				Aliases: []string{"e"},
				Usage:   "exclude message word",
				EnvVars: []string{"TALERT_EXCLUDE"},
			},
			&cli.StringSliceFlag{
				Name:    "level",
				Aliases: []string{"l"},
				Value:   cli.NewStringSlice("ERROR"),
				Usage:   "notify log level",
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
