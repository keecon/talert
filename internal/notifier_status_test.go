package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/go-resty/resty/v2"
)

func TestNotifierFailedDeliveryDoesNotSuppressDuplicate(t *testing.T) {
	for _, status := range []int{http.StatusBadRequest, http.StatusInternalServerError} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				requests++
				if requests == 1 {
					w.WriteHeader(status)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			n := &notifier{config: &Config{WebhookURL: server.URL}, events: map[string]time.Time{}, ch: make(chan *eventLog, 3), rc: resty.New()}
			now := time.Now()
			for i := 0; i < 3; i++ {
				n.ch <- newEvent("ERROR", "same", now)
			}
			close(n.ch)
			n.run()
			if requests != 2 {
				t.Errorf("requests = %d, want 2 (failure, success, suppressed)", requests)
			}
		})
	}
}

func TestNotifierErrorsDoNotExposeWebhookSecrets(t *testing.T) {
	const secret = "private-webhook-token-abc"
	var logs bytes.Buffer
	oldOutput := log.Writer()
	log.SetOutput(&logs)
	defer log.SetOutput(oldOutput)
	stderr, err := os.CreateTemp(t.TempDir(), "resty-log")
	if err != nil {
		t.Fatal(err)
	}
	defer stderr.Close()
	oldStderr := os.Stderr
	os.Stderr = stderr
	defer func() { os.Stderr = oldStderr }()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/ok", http.StatusFound)
			return
		}
		if r.URL.Path == "/ok" {
			w.WriteHeader(http.StatusOK)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/server-error") {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			w.WriteHeader(http.StatusForbidden)
		}
		fmt.Fprint(w, secret)
	}))
	n := newNotifier(&Config{WebhookURL: server.URL + "/" + secret})
	defer close(n.ch)
	n.rc.SetRetryWaitTime(time.Millisecond).SetRetryMaxWaitTime(time.Millisecond)
	event := newEvent("ERROR", "message", time.Now())
	if err := n.sendEventMessage(event); err == nil {
		t.Error("403 response was accepted")
	}
	n.config.WebhookURL = server.URL + "/server-error/" + secret
	if err := n.sendEventMessage(event); err == nil {
		t.Error("500 response was accepted")
	}
	n.config.WebhookURL = server.URL + "/redirect?token=" + secret
	if err := n.sendEventMessage(event); err == nil {
		t.Error("redirect response was accepted")
	}
	server.Close()
	n.config.WebhookURL = server.URL + "/" + secret
	if err := n.sendEventMessage(event); err == nil {
		t.Error("connection failure was accepted")
	} else if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), server.URL) {
		t.Error("returned error exposed the webhook URL or token")
	}
	data, err := os.ReadFile(stderr.Name())
	if err != nil {
		t.Fatal(err)
	}
	combined := logs.String() + string(data)
	if strings.Contains(combined, secret) || strings.Contains(combined, server.URL) {
		t.Error("logs exposed the webhook URL or token")
	}
	if strings.Contains(combined, "send event message complete") {
		t.Error("failed request logged success")
	}
}

func TestNotifierQueueBuffersBurst(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	received := make(chan struct{}, 65)
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if count.Add(1) == 1 {
			close(started)
			<-release
		}
		received <- struct{}{}
	}))
	defer server.Close()
	var releaseOnce sync.Once
	defer releaseOnce.Do(func() { close(release) })
	n := newNotifier(&Config{WebhookURL: server.URL})
	defer close(n.ch)
	if cap(n.ch) != 64 {
		t.Errorf("queue capacity = %d, want 64", cap(n.ch))
	}
	n.ch <- newEvent("ERROR", "first", time.Now())
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("first request did not start")
	}
	for i := 0; i < 64; i++ {
		select {
		case n.ch <- newEvent("ERROR", fmt.Sprint(i), time.Now()):
		default:
			t.Fatal("queue blocked during burst")
		}
	}
	releaseOnce.Do(func() { close(release) })
	for i := 0; i < 65; i++ {
		select {
		case <-received:
		case <-time.After(5 * time.Second):
			t.Fatalf("received %d of 65 events", i)
		}
	}
}

func TestNotifierExpiredSuccessFailureAndRetry(t *testing.T) {
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
		attempt := len(received)
		mu.Unlock()
		if attempt == 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	n := &notifier{
		config: &Config{WebhookURL: server.URL, WebhookTextFormat: "%s %s"},
		events: map[string]time.Time{},
		ch:     make(chan *eventLog, 4),
		rc:     resty.New(),
	}
	start := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	for i, offset := range []time.Duration{0, 5 * time.Minute, 5*time.Minute + time.Nanosecond, 5*time.Minute + 2*time.Nanosecond} {
		event := newEvent("ERROR", "same", start.Add(offset))
		event.stacktrace = []string{fmt.Sprintf("attempt-%d", i+1)}
		n.ch <- event
	}
	close(n.ch)
	n.run()

	mu.Lock()
	defer mu.Unlock()
	want := []string{"ERROR same\n```attempt-1```", "ERROR same\n```attempt-2```", "ERROR same\n```attempt-3```"}
	if !reflect.DeepEqual(received, want) {
		t.Errorf("received %q, want %q", received, want)
	}
	if got := n.events["ERROR same"]; !got.Equal(start.Add(5*time.Minute + time.Nanosecond)) {
		t.Errorf("last successful event time = %v", got)
	}
}
