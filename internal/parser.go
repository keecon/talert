package internal

import (
	"fmt"
	"regexp"
	"strings"

	"golang.org/x/exp/slices"
)

type parser struct {
	config  *Config
	pattern *regexp.Regexp
}

func (p *parser) setup(config *Config) error {
	pattern, err := regexp.Compile(config.Pattern)
	if err != nil {
		return fmt.Errorf("regexp compile error: %w", err)
	}

	p.config = config
	p.pattern = pattern
	return nil
}

func (p *parser) isMatchedAll(line string) (string, string, bool) {
	matches := p.pattern.FindStringSubmatch(line)
	if !p.isLogLine(matches) || !p.isMatchedLevel(matches[1]) || p.isMatchedExclude(line) {
		return "", "", false
	}

	return matches[1], matches[2], true
}

func (p *parser) isLogLine(matches []string) bool {
	return 2 < len(matches)
}

func (p *parser) isMatchedLevel(level string) bool {
	return slices.Contains(p.config.Levels, level)
}

func (p *parser) isMatchedExclude(line string) bool {
	for _, v := range p.config.Excludes {
		if strings.Contains(line, v) {
			return true
		}
	}
	return false
}
