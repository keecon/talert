package internal

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestNotifierDeduplicationAndExpiration(t *testing.T) {
	var received int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		received++
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	n := &notifier{
		config: &Config{WebhookURL: server.URL},
		events: map[string]time.Time{},
		ch:     make(chan *eventLog, 110),
		rc:     resty.New(),
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	first := newEvent("ERROR", "same", start)
	first.stacktrace = []string{"large stack trace"}
	n.ch <- first
	n.ch <- newEvent("ERROR", "same", start.Add(5*time.Minute-time.Nanosecond))
	n.ch <- newEvent("ERROR", "same", start.Add(5*time.Minute))
	for i := 0; i < 100; i++ {
		n.ch <- newEvent("ERROR", fmt.Sprintf("unique-%d", i), start.Add(time.Duration(i+1)*5*time.Minute))
	}
	close(n.ch)
	n.run()

	if received != 102 {
		t.Errorf("sent %d events, want 102", received)
	}
	if len(n.events) != 1 {
		t.Errorf("cache contains %d entries, want 1", len(n.events))
	}
}
