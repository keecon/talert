package main

import (
	"log"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

func main() {
	var (
		pattern           string
		webhookURL        string
		webhookChannel    string
		webhookAppID      string
		webhookTextFormat string
	)

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
			&cli.StringFlag{
				Name:        "webhook-url",
				Usage:       "slack incoming webhook url",
				Destination: &webhookURL,
				EnvVars:     []string{"TALERT_WEBHOOK_URL"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-channel",
				Usage:       "slack incoming webhook channel",
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
			&cli.StringSliceFlag{
				Name:     "webhook-owner",
				Usage:    "slack message owner field",
				EnvVars:  []string{"TALERT_WEBHOOK_OWNER"},
				Required: true,
			},
			&cli.StringFlag{
				Name:        "webhook-text-format",
				Usage:       "slack message text format (must need 3 string placeholder)",
				Value:       "ALERT LEVEL `%s` :fire: MESSAGE `%s`\n```%s```",
				Destination: &webhookTextFormat,
				EnvVars:     []string{"TALERT_WEBHOOK_TEXT_FORMAT"},
			},
		},

		Action: func(ctx *cli.Context) error {
			filename := ctx.Args().First()
			watcher := internal.NewWatcher()

			return watcher.Watch(filename, &internal.Config{
				Pattern:           pattern,
				Levels:            ctx.StringSlice("level"),
				WebhookAppID:      webhookAppID,
				WebhookURL:        webhookURL,
				WebhookChannel:    webhookChannel,
				WebhookOwners:     ctx.StringSlice("webhook-owner"),
				WebhookTextFormat: webhookTextFormat,
			})
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
