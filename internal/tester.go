package internal

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"
)

// Tester is check, test message pattern
type Tester struct {
}

// NewTester creates tester instance.
func NewTester() *Tester {
	return &Tester{}
}

// Test begins tailing the file, check and pass event to the notifier
func (w *Tester) Test(filename string, config *Config) error {
	pattern, err := regexp.Compile(config.Pattern)
	if err != nil {
		return fmt.Errorf("regexp compile error: %w", err)
	}

	reader, err := newReader(filename)
	if err != nil {
		return err
	}

	for v := range reader.lines {
		line := string(v)
		matches := pattern.FindStringSubmatch(line)

		for _, v := range config.Excludes {
			if strings.Contains(line, v) {
				continue
			}
		}

		if 2 < len(matches) && slices.Contains(config.Levels, matches[1]) {
			fmt.Println(line)
		}
	}
	return nil
}

type reader struct {
	file  *os.File
	lines chan []byte
}

func newReader(filename string) (*reader, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file error: %w", err)
	}

	r := &reader{
		file:  file,
		lines: make(chan []byte, 1),
	}
	go r.readlines()

	return r, nil
}

func (r *reader) readlines() {
	defer func() {
		if err := r.file.Close(); err != nil {
			fmt.Println("close file error: ", err)
		}
	}()

	scanner := bufio.NewScanner(r.file)
	for scanner.Scan() {
		r.lines <- scanner.Bytes()
	}

	close(r.lines)
}
