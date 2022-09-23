package internal

import (
	"fmt"
	"log"
	"math"
	"os"
	"regexp"
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

func (p *Watcher) setup(filename string, config *Config) (err error) {
	p.config = config

	p.config.ReOpen = true
	p.config.MustExist = true
	p.config.Follow = true
	if p.config.Location == nil {
		p.config.Location = &tail.SeekInfo{Whence: os.SEEK_END}
	}
	if p.config.Logger == nil {
		p.config.Logger = tail.DiscardingLogger
	}

	p.pattern, err = regexp.Compile(p.config.Pattern)
	if err != nil {
		return fmt.Errorf("regexp compile error: %w", err)
	}

	p.file, err = tail.TailFile(filename, config.Config)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}

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
