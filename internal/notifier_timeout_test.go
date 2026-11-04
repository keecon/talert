package internal

import (
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
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
	t.Parallel()
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

type deadlineObservation struct {
	deadline time.Time
	observed time.Time
}

type deadlineTransport struct {
	observations chan deadlineObservation
}

func (t deadlineTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	deadline, _ := req.Context().Deadline()
	t.observations <- deadlineObservation{deadline: deadline, observed: time.Now()}
	return http.DefaultTransport.RoundTrip(req)
}

func TestNotifierTimeoutIncludesRetryAndWait(t *testing.T) {
	t.Parallel()
	attempts := make(chan int, 2)
	release := make(chan struct{})
	var count atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt := count.Add(1)
		attempts <- int(attempt)
		if attempt == 1 {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)

	observations := make(chan deadlineObservation, 4)
	n := newNotifier(&Config{WebhookURL: server.URL})
	defer close(n.ch)
	n.rc.SetTransport(deadlineTransport{observations: observations})
	n.rc.SetRetryWaitTime(time.Second).SetRetryMaxWaitTime(time.Second)

	done := make(chan error, 1)
	go func() { done <- n.sendEventMessage(newEvent("ERROR", "retry", time.Now())) }()

	var first, second deadlineObservation
	select {
	case first = <-observations:
	case <-time.After(3 * time.Second):
		t.Fatal("first request did not start")
	}
	if first.deadline.IsZero() || first.deadline.Sub(first.observed) > 5*time.Second {
		t.Fatalf("first request deadline is not within 5 seconds: %v", first.deadline.Sub(first.observed))
	}
	select {
	case second = <-observations:
	case <-time.After(4 * time.Second):
		t.Fatal("retry did not start")
	}
	if !second.deadline.Equal(first.deadline) {
		t.Fatal("retry did not retain the original delivery deadline")
	}
	if second.observed.Sub(first.observed) < time.Second {
		t.Fatal("retry did not include the configured wait")
	}
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
			t.Fatalf("delivery error = %v, want deadline exceeded", err)
		}
	case <-time.After(7 * time.Second):
		t.Fatal("delivery did not stop after its deadline")
	}
	if len(attempts) != 2 {
		t.Fatalf("server received %d attempts, want 2", len(attempts))
	}
}
