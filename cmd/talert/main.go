package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

func main() {
	var (
		pattern        string
		level          string
		webhookURL     string
		webhookChannel string
		webhookAppID   string
		webhookOwner   string
	)
	app := &cli.App{
		Name:     "talert",
		Usage:    "read, check and alert from continuously updated files",
		Version:  internal.Version(),
		Compiled: internal.BuildDate(),
		Authors: []*cli.Author{
			{
				Name:  "iwaltgen",
				Email: "iwaltgen@gmail.com",
			},
		},
		EnableBashCompletion: true,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pattern",
				Aliases:     []string{"p"},
				Value:       "^[0-9]{2}:[0-9]{2}:[0-9]{2}.\\d*\\s+([A-Z]+).* : (.+)$",
				Destination: &pattern,
				Usage:       "message pattern in a log line (must need 2 submatch)",
				EnvVars:     []string{"TALERT_PATTERN"},
			},
			&cli.StringFlag{
				Name:        "level",
				Aliases:     []string{"l"},
				Value:       "ERROR",
				Destination: &level,
				Usage:       "notify log level",
				EnvVars:     []string{"TALERT_LEVEL"},
			},
			&cli.StringFlag{
				Name:        "webhook-url",
				Usage:       "slack incoming webhook url",
				Destination: &webhookURL,
				EnvVars:     []string{"TALERT_WEBHOOK_URL"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-channel",
				Usage:       "slack channel",
				Destination: &webhookChannel,
				EnvVars:     []string{"TALERT_WEBHOOK_CHANNEL"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-app-id",
				Usage:       "slack message app id field",
				Destination: &webhookAppID,
				EnvVars:     []string{"TALERT_WEBHOOK_APP_ID"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-owner",
				Usage:       "slack message owner field",
				Destination: &webhookOwner,
				EnvVars:     []string{"TALERT_WEBHOOK_OWNER"},
				Required:    true,
			},
		},

		Action: func(ctx *cli.Context) error {
			filename := ctx.Args().First()
			watcher := internal.NewWatcher()

			return watcher.Watch(filename, &internal.Config{
				Pattern:        pattern,
				Level:          level,
				WebhookAppID:   webhookAppID,
				WebhookURL:     webhookURL,
				WebhookChannel: webhookChannel,
				WebhookOwner:   webhookOwner,
			})
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
