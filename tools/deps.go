//go:build tools
// +build tools

package tools

import (
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/magefile/mage"
	_ "github.com/mfridman/tparse"
	_ "golang.org/x/tools/cmd/stringer"
)
