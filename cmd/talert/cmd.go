package main

import (
	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

func newWatchCmd() *cli.Command {
	var (
		webhookURL        string
		webhookChannel    string
		webhookAppID      string
		webhookTextFormat string
	)

	return &cli.Command{
		Name:  "watch",
		Usage: "read, check and alert from continuously updated files",

		Flags: []cli.Flag{
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
				Usage:       "slack message text format (must need 2 string placeholder)",
				Value:       "ALERT LEVEL `%s` :fire: MESSAGE `%s`",
				Destination: &webhookTextFormat,
				EnvVars:     []string{"TALERT_WEBHOOK_TEXT_FORMAT"},
			},
		},

		Action: func(ctx *cli.Context) error {
			filename := ctx.Args().First()
			watcher := internal.NewWatcher()

			return watcher.Watch(filename, &internal.Config{
				Pattern:           ctx.String("pattern"),
				Levels:            ctx.StringSlice("level"),
				Excludes:          ctx.StringSlice("exclude"),
				WebhookAppID:      webhookAppID,
				WebhookURL:        webhookURL,
				WebhookChannel:    webhookChannel,
				WebhookOwners:     ctx.StringSlice("webhook-owner"),
				WebhookTextFormat: webhookTextFormat,
			})
		},
	}
}

func newTestCmd() *cli.Command {
	return &cli.Command{
		Name:  "test",
		Usage: "check message pattern in file",

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
