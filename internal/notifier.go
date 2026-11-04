package internal

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/keecon/talert/internal/slack"
)

type notifier struct {
	config   *Config
	hostname string
	owners   string
	events   map[string]time.Time
	ch       chan *eventLog
	rc       *resty.Client
}

func newNotifier(config *Config) *notifier {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("os hostname error: ", err)
		hostname = "unknown"
	}

	v := &notifier{
		config:   config,
		hostname: hostname,
		owners:   strings.Join(config.WebhookOwners, ","),
		events:   map[string]time.Time{},
		ch:       make(chan *eventLog, 8),
		rc:       resty.New().SetRetryCount(3),
	}

	go v.run()
	return v
}

func (n *notifier) run() {
	var lastCleanup time.Time
	for evt := range n.ch {
		if lastCleanup.IsZero() {
			lastCleanup = evt.time
		}
		if evt.time.Sub(lastCleanup) >= 5*time.Minute {
			for key, timestamp := range n.events {
				if evt.time.Sub(timestamp) >= 5*time.Minute {
					delete(n.events, key)
				}
			}
			lastCleanup = evt.time
		}
		key := evt.Key()
		if old, ok := n.events[key]; ok && evt.time.Sub(old) < 5*time.Minute {
			continue
		}

		n.events[key] = evt.time
		if err := n.sendEventMessage(evt); err != nil {
			log.Println("send event message error: ", err)
		}
	}
}

func (n *notifier) sendEventMessage(evt *eventLog) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Printf("send message - %s (%d)\n", evt.message, len(evt.stacktrace))

	statusText := "ok"
	_, err := n.rc.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(n.newWebhookMessage(evt)).
		SetResult(&statusText).
		Post(n.config.WebhookURL)

	if err == nil {
		log.Println("send event message complete: ", statusText)
	}
	return err
}

func (n *notifier) newWebhookMessage(evt *eventLog) *slack.WebhookPayload {
	stacktrace := strings.Join(evt.stacktrace, "\n")
	length := len(stacktrace)
	if 3800 < length {
		stacktrace = stacktrace[:3800] + "..."
	}

	var stacktraceText string
	if 0 < len(stacktrace) {
		stacktraceText = fmt.Sprintf("\n```%s```", stacktrace)
	}

	return &slack.WebhookPayload{
		Channel:   n.config.WebhookChannel,
		Username:  "TAIL ALERT",
		IconEmoji: ":rotating_light:",
		Text:      fmt.Sprintf(n.config.WebhookTextFormat, evt.level, evt.message) + stacktraceText,
		Attachments: []*slack.Attachment{
			{
				Color: "danger",
				Fields: []*slack.Field{
					{
						Title: "app",
						Value: n.config.WebhookAppID,
						Short: true,
					},
					{
						Title: "level",
						Value: evt.level,
						Short: true,
					},
					{
						Title: "hostname",
						Value: n.hostname,
						Short: true,
					},
					{
						Title: "owner",
						Value: n.owners,
						Short: true,
					},
				},
				Footer:     "tail-alert",
				FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
				TS:         time.Now().Unix(),
			},
		},
	}
}
