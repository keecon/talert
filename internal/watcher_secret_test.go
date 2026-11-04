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
	if strings.Contains(logs.String(), "example.invalid") || strings.Contains(logs.String(), "secret-path") {
		t.Fatal("watch startup log exposed webhook URL or secret path")
	}
}
