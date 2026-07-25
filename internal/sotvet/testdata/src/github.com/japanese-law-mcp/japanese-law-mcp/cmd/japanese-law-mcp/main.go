package main

import (
	"context"
	"os"

	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp/testfixture"
	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/source/testfixture"
	_ "github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/stdio/testfixture"
)

func main() {
	_ = context.Background()

	if false {
		os.Exit(0)
	}
}
