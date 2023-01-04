package internal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParserMatchedAll(t *testing.T) {
	type record struct {
		name     string
		pattern  string
		levels   []string
		excludes []string
		text     string
		level    string
		message  string
		success  bool
	}
	// https://regoio.herokuapp.com/
	// https://regexr.com/
	dataset := []record{
		{
			name:    "Logback",
			pattern: "^.+\\s+([A-Z]+).* : (.+)$",
			levels:  []string{"ERROR"},
			text:    "2022-09-23 15:02:03 ERROR com.keecon.pkg.name : test message",
			level:   "ERROR",
			message: "test message",
			success: true,
		},
		{
			name:    "Nginx",
			pattern: "^[0-9]{4}/[0-9]{2}/[0-9]{2} [0-9]{2}:[0-9]{2}:[0-9]{2} \\[([a-z]+)\\] (.*)$",
			levels:  []string{"crit"},
			text:    "2022/09/26 11:35:21 [crit] test message",
			level:   "crit",
			message: "test message",
			success: true,
		},
		{
			name:    "Zap",
			pattern: "^.*\"level\":\"([a-z]+)\".*\"msg\":\"([\\w\\s]+)\".*$",
			levels:  []string{"error"},
			text:    "{\"level\":\"error\",\"time\":\"09-26T11:22:43.744\",\"logger\":\"test\",\"msg\":\"test message\"}",
			level:   "error",
			message: "test message",
			success: true,
		},
	}

	newParser := func(t *testing.T, v record) *parser {
		parser := &parser{}
		err := parser.setup(&Config{
			Pattern:  v.pattern,
			Levels:   v.levels,
			Excludes: v.excludes,
		})
		assert.NoError(t, err)
		return parser
	}

	for _, v := range dataset {
		t.Run(v.name, func(t *testing.T) {
			// given
			parser := newParser(t, v)

			// when
			lv, msg, ok := parser.isMatchedAll(v.text)

			// then
			assert.Equal(t, v.success, ok)
			assert.Equal(t, v.level, lv)
			assert.Equal(t, v.message, msg)
		})
	}
}
