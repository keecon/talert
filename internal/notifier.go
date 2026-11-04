package internal

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/keecon/talert/internal/slack"
)

const notifierShutdownTimeout = 30 * time.Second

type notifier struct {
	config    *Config
	hostname  string
	owners    string
	events    map[string]time.Time
	ch        chan *eventLog
	rc        *resty.Client
	ctx       context.Context
	cancel    context.CancelFunc
	done      chan struct{}
	queueHigh bool
}

func newNotifier(config *Config) *notifier {
	hostname, err := os.Hostname()
	if err != nil {
		log.Println("os hostname error: ", err)
		hostname = "unknown"
	}

	ctx, cancel := context.WithCancel(context.Background())
	v := &notifier{
		ctx: ctx, cancel: cancel, done: make(chan struct{}),
		config:   config,
		hostname: hostname,
		owners:   strings.Join(config.WebhookOwners, ","),
		events:   map[string]time.Time{},
		ch:       make(chan *eventLog, 64),
		rc:       resty.New().SetRetryCount(3).SetRedirectPolicy(resty.RedirectPolicyFunc(func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse })).SetLogger(webhookLogger{}),
	}

	go v.run()
	return v
}

func (n *notifier) run() {
	if n.done != nil {
		defer close(n.done)
	}
	var lastCleanup time.Time
	for evt := range n.ch {
		if n.ctx != nil && n.ctx.Err() != nil {
			log.Printf("notifier stopped before delivery: %d queued events abandoned", len(n.ch)+1)
			return
		}
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

		if err := n.sendEventMessage(evt); err != nil {
			log.Println("send event message error: ", err)
		} else {
			n.events[key] = evt.time
		}
	}
}

func (n *notifier) sendEventMessage(evt *eventLog) error {
	parent := n.ctx
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, 5*time.Second)
	defer cancel()

	log.Printf("send message - %s (%d)\n", evt.message, len(evt.stacktrace))

	resp, err := n.rc.R().
		SetContext(ctx).
		SetHeader("Content-Type", "application/json").
		SetBody(n.newWebhookMessage(evt)).
		Post(n.config.WebhookURL)

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return context.DeadlineExceeded
		}
		return errors.New("webhook request failed")
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("webhook HTTP status %d", resp.StatusCode())
	}
	log.Println("send event message complete: ", resp.StatusCode())
	return nil
}

func (n *notifier) newWebhookMessage(evt *eventLog) *slack.WebhookPayload {
	stacktrace := strings.Join(evt.stacktrace, "\n")
	stacktrace = truncateStack(stacktrace)

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

// Resty errors may contain webhook URLs or response data.
type webhookLogger struct{}

func (webhookLogger) Errorf(string, ...interface{}) {}
func (webhookLogger) Warnf(string, ...interface{})  {}
func (webhookLogger) Debugf(string, ...interface{}) {}

// enqueue is called only by the watcher, which owns the queue producer.
func (n *notifier) enqueue(event *eventLog) {
	if len(n.ch)*100 <= cap(n.ch)*70 {
		n.queueHigh = false
	}
	var canceled <-chan struct{}
	if n.ctx != nil {
		canceled = n.ctx.Done()
	}
	select {
	case n.ch <- event:
		queued := len(n.ch)
		if queued*100 > cap(n.ch)*70 && !n.queueHigh {
			n.queueHigh = true
			log.Printf("notifier queue usage high: %d/%d (>70%%)", queued, cap(n.ch))
		}
	case <-canceled:
		log.Println("notifier shutdown deadline reached; event not queued")
	}
}

func (n *notifier) shutdown(timeout time.Duration) {
	close(n.ch)
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	defer n.cancel()
	select {
	case <-n.done:
	case <-timer.C:
		log.Printf("notifier shutdown deadline reached: %d queued events remaining", len(n.ch))
		n.cancel()
		<-n.done
	}
}
