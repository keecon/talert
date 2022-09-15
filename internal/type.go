package internal

import (
	"fmt"
	"time"

	"github.com/hpcloud/tail"
)

// Config is used to specify how a file must be tailed.
type Config struct {
	tail.Config
	// Pattern is the log level and message pattern in a log line.
	Pattern string
	// Levels if match log level then notify alert.
	Levels []string
	// WebhookURL is slack incoming webhook URL.
	WebhookURL string
	// WebhookChannel is slack incoming webhook channel.
	WebhookChannel string
	// WebhookAppID is webhook message field.
	WebhookAppID string
	// WebhookOwners is webhook message field.
	WebhookOwners []string
	// WebhookTextFormat is webhook message text format.
	WebhookTextFormat string
}

type eventLog struct {
	level   string
	message string
	lines   []string
	time    time.Time
}

func newEvent(tokens []string, time time.Time) *eventLog {
	return &eventLog{
		level:   tokens[1],
		message: tokens[2],
		time:    time,
	}
}

func (e *eventLog) Key() string {
	return fmt.Sprintf("%s %s", e.level, e.message)
}
