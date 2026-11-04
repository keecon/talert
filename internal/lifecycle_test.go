package internal

import (
	"bytes"
	"context"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/hpcloud/tail"
)

func TestParserRejectsMissingCaptures(t *testing.T) {
	for _, pattern := range []string{"ERROR.*", "(ERROR).*"} {
		if err := (&parser{}).setup(&Config{Pattern: pattern}); err == nil {
			t.Errorf("accepted %q", pattern)
		}
	}
}

func TestTesterReadsLongLinesAndReportsOpenErrors(t *testing.T) {
	var output bytes.Buffer
	old := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(old)
	file := filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(file, []byte(strings.Repeat("x", 70000)+"\nERROR after\r\nERROR last"), 0600); err != nil {
		t.Fatal(err)
	}
	tester := NewTester()
	cfg := &Config{Pattern: `^(ERROR) (.*)$`, Levels: []string{"ERROR"}}
	if err := tester.Test(file, cfg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "ERROR after") || !strings.Contains(output.String(), "ERROR last") {
		t.Fatal("long line hid next event")
	}
	if err := tester.Test(file+"missing", cfg); err == nil {
		t.Fatal("missing open error")
	}
	if err := tester.Test(t.TempDir(), cfg); err == nil {
		t.Fatal("missing read error")
	}
}

func TestNotifierShutdownDrains(t *testing.T) {
	received := make(chan struct{}, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { received <- struct{}{} }))
	defer server.Close()
	n := newNotifier(&Config{WebhookURL: server.URL, WebhookTextFormat: "%s %s"})
	n.enqueue(newEvent("ERROR", "one", time.Now()))
	n.enqueue(newEvent("ERROR", "two", time.Now()))
	n.shutdown(time.Second)
	if len(received) != 2 {
		t.Fatalf("delivered %d events", len(received))
	}
}

func TestNotifierQueueWarningCrossing(t *testing.T) {
	var output bytes.Buffer
	old := log.Writer()
	log.SetOutput(&output)
	defer log.SetOutput(old)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	n := &notifier{ch: make(chan *eventLog, 64), ctx: ctx}
	for i := 0; i < 44; i++ {
		n.enqueue(newEvent("ERROR", "one", time.Now()))
	}
	if output.Len() != 0 {
		t.Fatal("warned below threshold")
	}
	n.enqueue(newEvent("ERROR", "one", time.Now()))
	n.enqueue(newEvent("ERROR", "one", time.Now()))
	if strings.Count(output.String(), "queue usage high") != 1 {
		t.Fatal("missing or repeated queue warning")
	}
	for len(n.ch) > 0 {
		<-n.ch
	}
	for i := 0; i < 45; i++ {
		n.enqueue(newEvent("ERROR", "one", time.Now()))
	}
	if strings.Count(output.String(), "queue usage high") != 2 {
		t.Fatal("new threshold crossing not logged")
	}
}

func TestNotifierShutdownCancelsBlockedRequest(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		select {
		case <-r.Context().Done():
		case <-release:
		}
	}))
	defer server.Close()
	defer close(release)
	n := newNotifier(&Config{WebhookURL: server.URL, WebhookTextFormat: "%s %s"})
	n.enqueue(newEvent("ERROR", "one", time.Now()))
	<-started
	start := time.Now()
	n.shutdown(50 * time.Millisecond)
	if time.Since(start) > time.Second {
		t.Fatal("shutdown did not cancel HTTP request")
	}
}

func TestStackTimeoutMarksPartialStack(t *testing.T) {
	p, lines := stackTestWatcher(t)
	deadline := make(chan time.Time, 1)
	lines <- &tail.Line{Text: "first frame"}
	done := make(chan struct{})
	go func() { p.collectEventLogUntil(newEvent("ERROR", "timeout", time.Now()), deadline); close(done) }()
	// Once the line is consumed, force the overall deadline.
	for len(lines) > 0 {
		time.Sleep(time.Millisecond)
	}
	deadline <- time.Now()
	select {
	case event := <-p.notifier.ch:
		stack := strings.Join(event.stacktrace, "\n")
		if !strings.Contains(stack, "first frame") || !strings.Contains(stack, "[collection timeout]") {
			t.Fatalf("unexpected stack: %q", stack)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout event not queued")
	}
	<-done
}

func TestStackTruncationPreservesUTF8(t *testing.T) {
	stack := strings.Repeat("가", maxStackBytes)
	got := truncateStack(stack)
	if len(got) > maxStackBytes || !utf8.ValidString(got) || !strings.HasSuffix(got, "[truncated]") {
		t.Fatal("invalid stack truncation")
	}
	marker := "\n...[collection timeout]"
	got = truncateStackWithMarker(stack+marker, marker)
	if len(got) > maxStackBytes || !utf8.ValidString(got) || !strings.HasSuffix(got, marker) {
		t.Fatal("timeout marker lost")
	}
}

func TestWatchWaitsForFinalDelivery(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release }))
	defer server.Close()
	defer once.Do(func() { close(release) })
	file := filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(file, []byte("ERROR final\nINFO boundary\n"), 0600); err != nil {
		t.Fatal(err)
	}
	watcher := NewWatcher()
	result := make(chan error, 1)
	go func() {
		result <- watcher.Watch(file, &Config{Config: tail.Config{Location: &tail.SeekInfo{Whence: io.SeekStart}}, Pattern: `^(ERROR|INFO) (.*)$`, Levels: []string{"ERROR"}, WebhookURL: server.URL, WebhookTextFormat: "%s %s"})
	}()
	select {
	case <-started:
	case <-time.After(time.Second):
		t.Fatal("webhook not started")
	}
	stopped := make(chan error, 1)
	go func() { stopped <- watcher.file.Stop() }()
	select {
	case err := <-stopped:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("tail did not stop")
	}
	select {
	case <-result:
		t.Fatal("watch returned before final delivery completed")
	case <-time.After(30 * time.Millisecond):
	}
	once.Do(func() { close(release) })
	select {
	case err := <-result:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(time.Second):
		t.Fatal("watch did not finish")
	}
	select {
	case <-watcher.notifier.done:
	default:
		t.Fatal("notifier left running")
	}
}

func TestWatchSignalHelper(t *testing.T) {
	file := os.Getenv("TALERT_TEST_SIGNAL_FILE")
	if file == "" {
		t.Skip("subprocess helper")
	}
	err := NewWatcher().Watch(file, &Config{Config: tail.Config{Location: &tail.SeekInfo{Whence: io.SeekStart}}, Pattern: `^(ERROR|INFO) (.*)$`, Levels: []string{"ERROR"}, WebhookURL: os.Getenv("TALERT_TEST_SIGNAL_URL"), WebhookTextFormat: "%s %s"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestWatchSIGTERMDrains(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SIGTERM requires Unix")
	}
	started := make(chan struct{})
	release := make(chan struct{})
	var once sync.Once
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(started); <-release }))
	defer server.Close()
	defer once.Do(func() { close(release) })
	file := filepath.Join(t.TempDir(), "log")
	if err := os.WriteFile(file, []byte("ERROR final\nINFO boundary\n"), 0600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestWatchSignalHelper$")
	cmd.Env = append(os.Environ(), "TALERT_TEST_SIGNAL_FILE="+file, "TALERT_TEST_SIGNAL_URL="+server.URL)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() { result <- cmd.Wait() }()
	select {
	case <-started:
	case <-ctx.Done():
		t.Fatal("child did not send webhook")
	}
	if err := cmd.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	select {
	case <-result:
		t.Fatal("SIGTERM did not wait for delivery")
	case <-time.After(30 * time.Millisecond):
	}
	once.Do(func() { close(release) })
	select {
	case err := <-result:
		if err != nil {
			t.Fatalf("child failed: %v\n%s", err, output.String())
		}
	case <-ctx.Done():
		t.Fatal("SIGTERM shutdown stalled")
	}
}
