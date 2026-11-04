package internal

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hpcloud/tail"
)

func TestWatchReturnsTailError(t *testing.T) {
	filename := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(filename, nil, 0600); err != nil {
		t.Fatal(err)
	}
	err := NewWatcher().Watch(filename, &Config{Config: tail.Config{Location: &tail.SeekInfo{Offset: -1, Whence: io.SeekStart}}, Pattern: "(.*)(.*)"})
	if err == nil || !strings.Contains(err.Error(), "Seek error") {
		t.Fatalf("expected tail seek error, got %v", err)
	}
}

type signalWriter struct {
	text  string
	ready chan struct{}
}

func (w signalWriter) Write(p []byte) (int, error) {
	if strings.Contains(string(p), w.text) {
		select {
		case w.ready <- struct{}{}:
		default:
		}
	}
	return len(p), nil
}

func TestWatchReopensFileAndStopsNormally(t *testing.T) {
	messages := make(chan string, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Text string `json:"text"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid payload", http.StatusBadRequest)
			return
		}
		messages <- payload.Text
	}))
	defer server.Close()
	filename := filepath.Join(t.TempDir(), "test.log")
	if err := os.WriteFile(filename, nil, 0600); err != nil {
		t.Fatal(err)
	}
	ready := make(chan struct{}, 1)
	reading := make(chan struct{}, 1)
	reopened := make(chan struct{}, 1)
	oldWriter := log.Writer()
	log.SetOutput(io.MultiWriter(signalWriter{text: "watch file:", ready: ready}, signalWriter{text: "starting collect stacktrace:", ready: reading}))
	defer log.SetOutput(oldWriter)
	watcher := NewWatcher()
	result := make(chan error, 1)
	go func() {
		result <- watcher.Watch(filename, &Config{Config: tail.Config{Poll: true, Location: &tail.SeekInfo{Whence: io.SeekStart}, Logger: log.New(signalWriter{text: "Successfully reopened", ready: reopened}, "", 0)}, Pattern: "^([A-Z]+) (.*)$", Levels: []string{"ERROR"}, WebhookURL: server.URL, WebhookTextFormat: "%s %s"})
	}()
	select {
	case <-ready:
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not start")
	}
	defer func() { _ = watcher.file.Stop() }()
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.WriteString("ERROR test\n"); err != nil {
		t.Fatal(err)
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-reading:
	case <-time.After(5 * time.Second):
		t.Fatal("initial line was not read")
	}
	if err := os.Rename(filename, filename+".old"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filename, nil, 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-reopened:
	case <-time.After(5 * time.Second):
		t.Fatal("file was not reopened")
	}
	if err := os.WriteFile(filename, []byte("ERROR reopened\nINFO flush\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for {
		select {
		case message := <-messages:
			if message == "ERROR reopened" {
				goto received
			}
		case <-time.After(5 * time.Second):
			t.Fatal("reopened file's event was not delivered")
		}
	}
received:
	if err := watcher.file.Stop(); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("normal stop: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("watch did not stop")
	}
}
