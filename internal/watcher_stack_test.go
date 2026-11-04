package internal

import (
	"fmt"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/hpcloud/tail"
)

func stackTestWatcher(t *testing.T) (*Watcher, chan *tail.Line) {
	t.Helper()
	lines := make(chan *tail.Line, 1)
	p := &Watcher{file: &tail.Tail{Lines: lines}, notifier: &notifier{ch: make(chan *eventLog, 2)}}
	if err := p.parser.setup(&Config{Pattern: `^(ERROR|INFO) (.*)$`, Levels: []string{"ERROR"}}); err != nil {
		t.Fatal(err)
	}
	return p, lines
}

func TestStackLimitPreservesNextEvent(t *testing.T) {
	for _, line := range []string{strings.Repeat("가", 2000), ""} {
		t.Run(fmt.Sprintf("line-bytes-%d", len(line)), func(t *testing.T) {
			p, lines := stackTestWatcher(t)
			done := make(chan struct{})
			go func() {
				p.collectEventLog(newEvent("ERROR", "first", time.Now()))
				close(done)
			}()
			go func() {
				for i := 0; i < maxStackBytes+2; i++ {
					select {
					case lines <- &tail.Line{Text: line}:
					case <-done:
						return
					}
				}
			}()
			select {
			case event := <-p.notifier.ch:
				stack := strings.Join(event.stacktrace, "\n")
				if len(stack) > maxStackBytes || !strings.Contains(stack, "[truncated]") || !utf8.ValidString(stack) {
					t.Fatalf("invalid truncated stack: bytes=%d validUTF8=%t", len(stack), utf8.ValidString(stack))
				}
			case <-time.After(time.Second):
				t.Fatal("size limit did not release event")
			}
		})
	}
}

func TestStackLimitLeavesNextLogLine(t *testing.T) {
	p, lines := stackTestWatcher(t)
	lines <- &tail.Line{Text: strings.Repeat("x", maxStackBytes+1)}
	go func() { lines <- &tail.Line{Text: "ERROR next"} }()
	p.collectEventLog(newEvent("ERROR", "first", time.Now()))
	first := <-p.notifier.ch
	if first.message != "first" || !strings.Contains(strings.Join(first.stacktrace, "\n"), "[truncated]") {
		t.Fatal("first event was not truncated")
	}
	select {
	case line := <-lines:
		level, message, ok := p.isMatchedAll(line.Text)
		if !ok || level != "ERROR" || message != "next" {
			t.Fatalf("next event was not preserved: %q", line.Text)
		}
		close(lines)
		p.collectEventLog(newEvent(level, message, time.Now()))
		if next := <-p.notifier.ch; next.message != "next" || len(next.stacktrace) != 0 {
			t.Fatal("next event was not sent independently")
		}
	case <-time.After(time.Second):
		t.Fatal("next log line was lost")
	}
}

func TestStackCollectionHasOverallDeadline(t *testing.T) {
	p, lines := stackTestWatcher(t)
	done := make(chan struct{})
	go func() {
		defer close(done)
		p.collectEventLogUntil(newEvent("ERROR", "first", time.Now()), time.After(350*time.Millisecond))
	}()
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case lines <- &tail.Line{Text: "trace"}:
				case <-done:
					return
				}
			case <-done:
				return
			}
		}
	}()
	select {
	case <-done:
		if got := <-p.notifier.ch; got.message != "first" {
			t.Fatalf("got event %q", got.message)
		}
	case <-time.After(time.Second):
		t.Fatal("continuous lines prevented the overall deadline")
	}
}

func TestStackCollectionEntryPointHasTenSecondDeadline(t *testing.T) {
	t.Parallel()
	p, lines := stackTestWatcher(t)
	done := make(chan struct{})
	stop := make(chan struct{})
	defer close(stop)
	started := time.Now()
	go func() {
		defer close(done)
		p.collectEventLog(newEvent("ERROR", "first", time.Now()))
	}()
	go func() {
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				select {
				case lines <- &tail.Line{Text: "trace"}:
				case <-done:
					return
				case <-stop:
					return
				}
			case <-done:
				return
			case <-stop:
				return
			}
		}
	}()
	select {
	case <-done:
		if elapsed := time.Since(started); elapsed < 9500*time.Millisecond {
			t.Fatalf("collection ended too early after %s", elapsed)
		}
		if event := <-p.notifier.ch; event.message != "first" {
			t.Fatalf("got event %q", event.message)
		}
	case <-time.After(11 * time.Second):
		t.Fatal("continuous lines prevented the 10-second deadline")
	}
}

func TestStackCollectionKeepsNormalEventBoundary(t *testing.T) {
	p, lines := stackTestWatcher(t)
	go func() {
		lines <- &tail.Line{Text: "stack line"}
		lines <- &tail.Line{Text: "ERROR next", Time: time.Now()}
	}()
	next := p.collectEventLog(newEvent("ERROR", "first", time.Now()))
	first := <-p.notifier.ch
	if first.message != "first" || strings.Join(first.stacktrace, "\n") != "stack line" || next == nil || next.message != "next" {
		t.Fatal("normal event boundary was not preserved")
	}
}
