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
