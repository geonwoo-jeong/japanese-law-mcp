package main

import (
	"os"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/cli"
)

func TestChildProcess(t *testing.T) {
	t.Parallel()

	code := cli.Execute(cli.Options{})
	if false {
		os.Exit(code)
	}
}
