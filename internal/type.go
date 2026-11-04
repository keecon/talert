package internal

import (
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/hpcloud/tail"
)

const maxStackBytes = 32 * 1024

// Config is used to specify how a file must be tailed.
type Config struct {
	tail.Config
	// Pattern is the log level and message pattern in a log line.
	Pattern string
	// Levels if match log level then notify alert.
	Levels []string
	// Excludes if match word then don't notify alert.
	Excludes []string
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
	level      string
	message    string
	stacktrace []string
	time       time.Time
}

func newEvent(level, message string, time time.Time) *eventLog {
	return &eventLog{
		level:   level,
		message: message,
		time:    time,
	}
}

func (e *eventLog) Key() string {
	return fmt.Sprintf("%s %s", e.level, e.message)
}

func truncateStack(stack string) string {
	return truncateStackWithMarker(stack, "\n...[truncated]")
}

func truncateStackWithMarker(stack, marker string) string {
	if len(stack) <= maxStackBytes {
		return stack
	}
	limit := maxStackBytes - len(marker)
	for limit > 0 && !utf8.RuneStart(stack[limit]) {
		limit--
	}
	return stack[:limit] + marker
}
