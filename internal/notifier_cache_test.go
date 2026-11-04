package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestNotifierDeduplicationAndExpiration(t *testing.T) {
	var mu sync.Mutex
	var received []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		mu.Lock()
		received = append(received, payload.Text)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := &notifier{
		config: &Config{WebhookURL: server.URL, WebhookTextFormat: "%s %s"},
		events: map[string]time.Time{},
		ch:     make(chan *eventLog, 110),
		rc:     resty.New(),
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := newEvent("ERROR", "same", start)
	first.stacktrace = []string{"large stack trace"}
	n.ch <- first
	beforeBoundary := newEvent("ERROR", "same", start.Add(5*time.Minute-time.Nanosecond))
	beforeBoundary.stacktrace = []string{"before boundary"}
	n.ch <- beforeBoundary
	atBoundary := newEvent("ERROR", "same", start.Add(5*time.Minute))
	atBoundary.stacktrace = []string{"at boundary"}
	n.ch <- atBoundary
	for i := 0; i < 100; i++ {
		n.ch <- newEvent("ERROR", fmt.Sprintf("unique-%d", i), start.Add(time.Duration(i+1)*5*time.Minute))
	}
	close(n.ch)
	n.run()

	want := []string{"ERROR same\n```large stack trace```", "ERROR same\n```at boundary```"}
	for i := 0; i < 100; i++ {
		want = append(want, fmt.Sprintf("ERROR unique-%d", i))
	}
	mu.Lock()
	defer mu.Unlock()
	if !reflect.DeepEqual(received, want) {
		t.Errorf("sent events = %q, want %q", received, want)
	}
	if len(n.events) != 1 {
		t.Errorf("cache contains %d entries, want 1", len(n.events))
	}
}

func TestNotifierDoesNotCleanCacheForEveryDuplicate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n := &notifier{
		config: &Config{WebhookURL: server.URL},
		events: map[string]time.Time{"expired": start.Add(-5 * time.Minute)},
		ch:     make(chan *eventLog, 200),
		rc:     resty.New(),
	}
	for i := 0; i < 100; i++ {
		n.ch <- newEvent("ERROR", fmt.Sprintf("unique-%d", i), start)
	}
	for i := 0; i < 100; i++ {
		n.ch <- newEvent("ERROR", "unique-0", start.Add(5*time.Minute-time.Nanosecond))
	}
	close(n.ch)
	n.run()

	if _, ok := n.events["expired"]; !ok {
		t.Error("cache was cleaned before the cleanup interval elapsed")
	}
}
