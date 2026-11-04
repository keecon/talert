package internal

import (
	"bytes"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hpcloud/tail"
)

func TestWatchStartupDoesNotLogWebhookURL(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(filename, nil, 0600); err != nil {
		t.Fatal(err)
	}
	var logs bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(previous)

	const webhookURL = "https://example.invalid/secret-path-that-is-longer-than-twenty-characters"
	watcher := NewWatcher()
	_ = watcher.Watch(filename, &Config{
		Config:  tail.Config{Location: &tail.SeekInfo{Offset: -1, Whence: io.SeekStart}},
		Pattern: "(.*)(.*)", WebhookURL: webhookURL,
	})
	close(watcher.notifier.ch)
	if !strings.Contains(logs.String(), "watch file:") {
		t.Fatal("watch startup log was not recorded")
	}
	if !strings.Contains(logs.String(), "https://example.invalid/[REDACTED]aracters") {
		t.Fatal("masked webhook destination was not recorded")
	}
	if strings.Contains(logs.String(), webhookURL) || strings.Contains(logs.String(), "secret-path") {
		t.Fatal("watch startup log exposed webhook URL or secret path")
	}
}

func TestMaskedWebhookURL(t *testing.T) {
	for _, tc := range []struct{ raw, want string }{
		{"https://example.invalid/services/secret/12345678", "https://example.invalid/[REDACTED]12345678"},
		{"https://user:password@example.invalid/12345678?token=secret#secret", "https://example.invalid/[REDACTED]"},
		{"https://example.invalid/abc", "https://example.invalid/[REDACTED]"},
		{"https://example.invalid/%zz", "[REDACTED]"},
	} {
		if got := maskedWebhookURL(tc.raw); got != tc.want {
			t.Errorf("masked destination = %q, want %q", got, tc.want)
		}
	}
}
