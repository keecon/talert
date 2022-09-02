package internal

import (
	"fmt"
	"os"
	"regexp"
	"time"

	"github.com/hpcloud/tail"
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
	fmt.Printf("watch level `%s`\n", w.config.Level)
	fmt.Printf("webhook url `%s`\n", w.config.WebhookURL)

	for line := range w.file.Lines {
		matches := w.pattern.FindStringSubmatch(line.Text)

		if w.isStartLogLine(matches) {
			evt := newEvent(matches, line.Time)

			for evt != nil && evt.level == config.Level {
				evt = w.collectEventLog(evt)
			}
		}
	}
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

	w.notifier = newNotifier(
		w.config.WebhookAppID,
		w.config.WebhookURL,
		w.config.WebhookChannel,
		w.config.WebhookOwner,
	)
	return nil
}
