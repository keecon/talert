package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type logChannel chan string

func (c logChannel) Write(p []byte) (int, error) {
	select {
	case c <- string(p):
	default:
	}
	return len(p), nil
}

func TestNotifierTimeoutContinuesAndSuppressesDuplicates(t *testing.T) {
	requests := make(chan string, 4)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return
		}
		requests <- payload.Text
		if payload.Text == "ERROR first" {
			select {
			case <-release:
			case <-r.Context().Done():
			}
		}
	}))
	defer server.Close()
	defer close(release)

	logs := make(logChannel, 20)
	originalOutput := log.Writer()
	log.SetOutput(logs)
	defer log.SetOutput(originalOutput)

	n := newNotifier(&Config{WebhookURL: server.URL, WebhookTextFormat: "%s %s"})
	defer close(n.ch)
	now := time.Now()
	n.ch <- newEvent("ERROR", "first", now)
	n.ch <- newEvent("ERROR", "first", now)
	n.ch <- newEvent("ERROR", "second", now)

	select {
	case got := <-requests:
		if got != "ERROR first" {
			t.Fatalf("first request = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("first request did not arrive")
	}
	select {
	case got := <-requests:
		if got != "ERROR second" {
			t.Fatalf("next request = %q; duplicate was sent", got)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("next event was not sent after the request stalled")
	}

	foundTimeout := false
	for len(logs) > 0 {
		line := <-logs
		if strings.Contains(line, "send event message error:") && strings.Contains(line, "context deadline exceeded") {
			foundTimeout = true
		}
	}
	if !foundTimeout {
		t.Fatal("timeout error was not logged before the next event")
	}
}
