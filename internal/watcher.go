package internal

import (
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/hpcloud/tail"
	"golang.org/x/exp/slices"
)

// Watcher is tailing the file, check and pass event to the notifier
type Watcher struct {
	config   *Config
	pattern  *regexp.Regexp
	file     *tail.Tail
	notifier *notifier
}

// NewWatcher creates watcher instance.
func NewWatcher() *Watcher {
	return &Watcher{}
}

// Watch begins tailing the file, check and pass event to the notifier
func (w *Watcher) Watch(filename string, config *Config) error {
	if err := w.setup(filename, config); err != nil {
		return err
	}
	defer w.file.Cleanup()

	fmt.Printf("start watch `%s`\n", filename)
	fmt.Printf("watch levels `%s`\n", w.config.Levels)
	fmt.Printf("watch hostname `%s`\n", w.notifier.hostname)

	urlLength := len(w.config.WebhookURL)
	maskingLength := int(math.Min(float64(urlLength), 20))
	fmt.Printf("webhook url `%s%s`\n", w.config.WebhookURL[:urlLength-maskingLength], strings.Repeat("*", maskingLength))

	for line := range w.file.Lines {
		matches := w.pattern.FindStringSubmatch(line.Text)

		for _, v := range w.config.Excludes {
			if strings.Contains(line.Text, v) {
				continue
			}
		}

		if w.isStartLogLine(matches) {
			evt := newEvent(matches, line.Time)

			for evt != nil && slices.Contains(config.Levels, evt.level) {
				evt = w.collectEventLog(evt)
			}
		}
	}
	return nil
}

func (w *Watcher) setup(filename string, config *Config) (err error) {
	w.config = config

	w.config.ReOpen = true
	w.config.MustExist = true
	w.config.Follow = true
	if w.config.Location == nil {
		w.config.Location = &tail.SeekInfo{Whence: os.SEEK_END}
	}
	if w.config.Logger == nil {
		w.config.Logger = tail.DiscardingLogger
	}

	w.pattern, err = regexp.Compile(w.config.Pattern)
	if err != nil {
		return fmt.Errorf("regexp compile error: %w", err)
	}

	w.file, err = tail.TailFile(filename, config.Config)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}

	w.notifier = newNotifier(w.config)
	return nil
}

func (w *Watcher) isStartLogLine(matches []string) bool {
	return 2 < len(matches)
}

func (w *Watcher) collectEventLog(evt *eventLog) *eventLog {
	event := evt
	fmt.Printf("%s: collect start `%s`\n", event.time, event.message)

	for {
		select {
		case line, more := <-w.file.Lines:
			if !more {
				w.notifier.ch <- event
				return nil
			}

			matches := w.pattern.FindStringSubmatch(line.Text)

			if w.isStartLogLine(matches) {
				w.notifier.ch <- event

				return newEvent(matches, line.Time)
			}

			event.lines = append(evt.lines, line.Text)

		case <-time.After(3 * time.Second):
			w.notifier.ch <- event
			return nil
		}
	}
}
