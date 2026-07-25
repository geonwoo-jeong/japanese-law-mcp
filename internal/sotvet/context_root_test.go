package sotvet

import (
	"path/filepath"
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestContextRootAnalyzer(t *testing.T) {
	t.Parallel()

	analysistest.Run(
		t,
		analysistest.TestData(),
		ContextRootAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/contextrootbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/contextrootgood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/contexttestallowed",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/generatedallowed",
	)
	analysistest.Run(
		t,
		filepath.Join(analysistest.TestData(), "contextroot"),
		ContextRootAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/japanese-law-mcp",
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/quality",
	)
}
