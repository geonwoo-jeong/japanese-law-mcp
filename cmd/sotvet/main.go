package main

import (
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/sotvet"
	"golang.org/x/tools/go/analysis/multichecker"
)

func main() {
	multichecker.Main(
		sotvet.DependencyDirectionAnalyzer,
		sotvet.ContextRootAnalyzer,
		sotvet.ProcessExitAnalyzer,
		sotvet.SuppressionPolicyAnalyzer,
	)
}
