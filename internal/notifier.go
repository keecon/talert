package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/keecon/talert/internal/slack"
)

type notifier struct {
	app            string
	webhookURL     string
	webhookChannel string
	owner          string
	events         map[string]*eventLog
	ch             chan *eventLog
	rc             *resty.Client
}

func newNotifier(app, url, channel, owner string) *notifier {
	v := &notifier{
		app:            app,
		webhookURL:     url,
		webhookChannel: channel,
		owner:          owner,
		events:         map[string]*eventLog{},
		ch:             make(chan *eventLog, 8),
		rc:             resty.New().SetRetryCount(3),
	}

	go v.run()
	return v
}

func (n *notifier) run() {
	for evt := range n.ch {
		old := n.events[evt.Key()]
		if old != nil {
			if evt.time.Sub(old.time) < 5*time.Minute {
				continue
			}
		}

		n.events[evt.Key()] = evt
		if err := n.sendEventMessage(evt); err != nil {
			fmt.Println("send event message error: ", err)
		}
	}
}

func (n *notifier) sendEventMessage(evt *eventLog) error {
	fmt.Printf("%s: send `%s` (%d)\n", evt.time, evt.message, len(evt.lines))

	statusText := "ok"
	_, err := n.rc.R().
		SetHeader("Content-Type", "application/json").
		SetBody(n.newWebhookMessage(evt)).
		SetResult(&statusText).
		Post(n.webhookURL)

	if err == nil {
		fmt.Println("send event message complete: ", statusText)
	}
	return err
}

func (n *notifier) newWebhookMessage(evt *eventLog) *slack.WebhookPayload {
	return &slack.WebhookPayload{
		Channel:   n.webhookChannel,
		Username:  "TAIL ALERT",
		IconEmoji: ":rotating_light:",
		Attachments: []*slack.Attachment{
			{
				Color: "danger",
				Title: fmt.Sprintf("TAIL ALERT `%s`", evt.level),
				Text:  fmt.Sprintf("ERROR MESSAGE `%s`", evt.message),
				Fields: []*slack.Field{
					{
						Title: "app",
						Value: n.app,
						Short: true,
					},
					{
						Title: "owner",
						Value: n.owner,
						Short: true,
					},
					{
						Title: "call stack",
						Value: strings.Join(evt.lines, "\n"),
						Short: false,
					},
				},
				Footer:     "tail-alert",
				FooterIcon: "https://platform.slack-edge.com/img/default_application_icon.png",
				TS:         time.Now().Unix(),
			},
		},
	}
}
