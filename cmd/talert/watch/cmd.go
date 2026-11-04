package watch

import (
	"github.com/urfave/cli/v2"

	"github.com/keecon/talert/internal"
)

var (
	webhookURL        string
	webhookChannel    string
	webhookAppID      string
	webhookTextFormat string
)

// NewCmd creates Command
func NewCmd() *cli.Command {
	return &cli.Command{
		Name:      "watch",
		Usage:     "Watch appended log lines and send matching events to Slack",
		ArgsUsage: "FILE",
		Description: "Watch FILE from its end and process newly appended lines.\n" +
			"Provide required options through flags or their environment variables.\n" +
			"See talert --help for global options and list input rules.\n\n" +
			"Example:\n" +
			"  export TALERT_WEBHOOK_URL='<webhook-url>'\n" +
			"  export TALERT_WEBHOOK_CHANNEL='#alerts'\n" +
			"  export TALERT_WEBHOOK_APP_ID='my-app'\n" +
			"  export TALERT_WEBHOOK_OWNER='alice,bob'\n" +
			"  talert --level ERROR,WARN --exclude 'health check,heartbeat' watch app.log",

		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "webhook-url",
				Usage:       "Slack incoming webhook `URL` (required)",
				Destination: &webhookURL,
				EnvVars:     []string{"TALERT_WEBHOOK_URL"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-channel",
				Usage:       "Slack `CHANNEL` (required)",
				Destination: &webhookChannel,
				EnvVars:     []string{"TALERT_WEBHOOK_CHANNEL"},
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "webhook-app-id",
				Usage:       "Application identifier `APP_ID` shown in alerts (required)",
				Destination: &webhookAppID,
				EnvVars:     []string{"TALERT_WEBHOOK_APP_ID"},
				Required:    true,
			},
			&cli.StringSliceFlag{
				Name:     "webhook-owner",
				Usage:    "Show `OWNER` in alerts (required; repeatable or comma-separated)",
				EnvVars:  []string{"TALERT_WEBHOOK_OWNER"},
				Required: true,
			},
			&cli.StringFlag{
				Name:        "webhook-text-format",
				Usage:       "Alert text `FORMAT`; use two %s placeholders in order: level, message",
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
