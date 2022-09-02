package internal

import (
	"fmt"
	"time"

	"github.com/hpcloud/tail"
)

// Config is used to specify how a file must be tailed.
type Config struct {
	tail.Config
	// Pattern is the error level and message pattern in a log line.
	Pattern string
	// Level if match error level then notify alert.
	Level string
	// WebhookURL is slack incoming webhook URL.
	WebhookURL string
	// WebhookChannel is slack incoming webhook channel.
	WebhookChannel string
	// WebhookAppID is webhook message field.
	WebhookAppID string
	// WebhookOwner is webhook message field.
	WebhookOwner string
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
