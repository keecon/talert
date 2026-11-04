package internal

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

// Tester checks log patterns against a file.
type Tester struct{ parser }

// NewTester creates a tester instance.
func NewTester() *Tester { return &Tester{} }

// Test prints matching log lines and reports read errors.
func (p *Tester) Test(filename string, config *Config) error {
	if err := p.setup(config); err != nil {
		return fmt.Errorf("setup error: %w", err)
	}
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("open file error: %w", err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Println("close file error: ", err)
		}
	}()
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadString('\n')
		hasLine := len(line) > 0
		line = strings.TrimSuffix(strings.TrimSuffix(line, "\n"), "\r")
		if hasLine {
			if _, _, ok := p.isMatchedAll(line); ok {
				log.Println(line)
			}
		}
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("read file error: %w", err)
		}
	}
}
