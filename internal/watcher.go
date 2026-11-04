package internal

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"
	"unicode/utf8"

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

	log.Println("webhook channel: ", p.config.WebhookChannel)

	for line := range p.file.Lines {
		if l, m, ok := p.isMatchedAll(line.Text); ok {
			evt := newEvent(l, m, line.Time)

			for evt != nil {
				evt = p.collectEventLog(evt)
			}
		}
	}
	return p.file.Wait()
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
	deadline := time.NewTimer(10 * time.Second)
	defer deadline.Stop()
	return p.collectEventLogUntil(evt, deadline.C)
}

func (p *Watcher) collectEventLogUntil(event *eventLog, deadline <-chan time.Time) *eventLog {
	log.Println("starting collect stacktrace: ", event.message)
	stackSize := 0

	for {
		select {
		case <-deadline:
			p.notifier.ch <- event
			return nil
		default:
		}
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

			separator := 0
			if len(event.stacktrace) > 0 {
				separator = 1
			}
			if stackSize+separator+len(line.Text) > 3800 {
				const marker = "\n...[truncated]"
				stack := strings.Join(event.stacktrace, "\n")
				if separator > 0 {
					stack += "\n"
				}
				limit := 3800 - len(marker)
				if len(stack) < limit {
					n := min(len(line.Text), limit-len(stack))
					for n > 0 && n < len(line.Text) && !utf8.RuneStart(line.Text[n]) {
						n--
					}
					stack += line.Text[:n]
				} else if len(stack) > limit {
					for limit > 0 && !utf8.RuneStart(stack[limit]) {
						limit--
					}
					stack = stack[:limit]
				}
				event.stacktrace = []string{stack + marker}
				p.notifier.ch <- event
				return nil
			}
			event.stacktrace = append(event.stacktrace, strings.Clone(line.Text))
			stackSize += separator + len(line.Text)

		case <-time.After(3 * time.Second):
			p.notifier.ch <- event
			return nil
		case <-deadline:
			p.notifier.ch <- event
			return nil
		}
	}
}
