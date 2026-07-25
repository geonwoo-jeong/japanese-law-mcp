package sotvet

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestProcessExitAnalyzer(t *testing.T) {
	t.Parallel()

	analysistest.Run(
		t,
		analysistest.TestData(),
		ProcessExitAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/quality",
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/japanese-law-mcp",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/cli",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/processexitbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/processexitgood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/processtestbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/generatedallowed",
	)
	analysistest.Run(
		t,
		filepath.Join(analysistest.TestData(), "processexit"),
		ProcessExitAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/japanese-law-mcp",
	)
}
