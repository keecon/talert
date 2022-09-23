package internal

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"regexp"
)

// Tester is check, test message pattern
type Tester struct {
	parser
}

// NewTester creates tester instance.
func NewTester() *Tester {
	return &Tester{}
}

// Test begins tailing the file, check and pass event to the notifier
func (p *Tester) Test(filename string, config *Config) (err error) {
	p.config = config
	p.pattern, err = regexp.Compile(config.Pattern)
	if err != nil {
		return fmt.Errorf("regexp compile error: %w", err)
	}

	reader, err := newReader(filename)
	if err != nil {
		return err
	}

	for v := range reader.lines {
		line := string(v)
		if _, _, ok := p.isMatchedAll(string(v)); ok {
			log.Println(line)
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
		close(r.lines)
		if err := r.file.Close(); err != nil {
			log.Println("close file error: ", err)
		}
	}()

	scanner := bufio.NewScanner(r.file)
	for scanner.Scan() {
		r.lines <- scanner.Bytes()
	}
}
