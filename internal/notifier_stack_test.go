package internal

import (
	"strings"
	"testing"
	"time"
)

func TestWebhookPreserves32KiBStack(t *testing.T) {
	stack := strings.Repeat("x", 32*1024-len("Caused by: root")) + "Caused by: root"
	evt := newEvent("ERROR", "failure", time.Now())
	evt.stacktrace = []string{stack}
	n := &notifier{config: &Config{WebhookTextFormat: "%s %s"}}
	if got := n.newWebhookMessage(evt).Text; !strings.Contains(got, stack) {
		t.Fatal("webhook truncated a stack within the 32 KiB limit")
	}
}
