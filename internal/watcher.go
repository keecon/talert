package internal

import (
	"fmt"
	"io"
	"log"
	"math"
	"strings"
	"time"

	"github.com/hpcloud/tail"
)

// Watcher is tailing the file, check and pass event to the notifier
type Watcher struct {
	parser
	file     *tail.Tail
	notifier *notifier
}

// NewWatcher creates watcher instance.
func NewWatcher() *Watcher {
	return &Watcher{}
}

// Watch begins tailing the file, check and pass event to the notifier
func (p *Watcher) Watch(filename string, config *Config) error {
	if err := p.setup(filename, config); err != nil {
		return err
	}
	defer p.file.Cleanup()

	log.Println("watch file: ", filename)
	log.Println("watch level: ", p.config.Levels)
	log.Println("watch hostname: ", p.notifier.hostname)

	urlLength := len(p.config.WebhookURL)
	maskingLength := int(math.Min(float64(urlLength), 20))
	log.Println("webhook url: ", p.config.WebhookURL[:urlLength-maskingLength]+strings.Repeat("*", maskingLength))
	log.Println("webhook channel: ", p.config.WebhookChannel)

	for line := range p.file.Lines {
		if l, m, ok := p.isMatchedAll(line.Text); ok {
			evt := newEvent(l, m, line.Time)

			for evt != nil {
				evt = p.collectEventLog(evt)
			}
		}
	}
	return nil
}

func (p *Watcher) setup(filename string, config *Config) error {
	if err := p.parser.setup(config); err != nil {
		return fmt.Errorf("setup error: %w", err)
	}

	p.config.ReOpen = true
	p.config.MustExist = true
	p.config.Follow = true
	if p.config.Location == nil {
		p.config.Location = &tail.SeekInfo{Whence: io.SeekEnd}
	}
	if p.config.Logger == nil {
		p.config.Logger = tail.DiscardingLogger
	}

	file, err := tail.TailFile(filename, config.Config)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}

	p.file = file
	p.notifier = newNotifier(p.config)
	return nil
}

func (p *Watcher) collectEventLog(evt *eventLog) *eventLog {
	event := evt
	log.Println("starting collect stacktrace: ", event.message)

	for {
		select {
		case line, more := <-p.file.Lines:
			if !more {
				p.notifier.ch <- event
				return nil
			}

			matches := p.pattern.FindStringSubmatch(line.Text)
			if p.isLogLine(matches) {
				p.notifier.ch <- event

				if p.isMatchedLevel(matches[1]) && !p.isMatchedExclude(line.Text) {
					return newEvent(matches[1], matches[2], line.Time)
				}
				return nil
			}

			event.stacktrace = append(event.stacktrace, line.Text)

		case <-time.After(3 * time.Second):
			p.notifier.ch <- event
			return nil
		}
	}
}
