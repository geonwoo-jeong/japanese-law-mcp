package main

import (
	"context"
	"os"

	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp/testfixture"
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/source/testfixture"
	_ "github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/stdio/testfixture"
)

func main() {
	_ = context.Background()

	if false {
		os.Exit(0)
	}
}
