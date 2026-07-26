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
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/contextrootbad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/contextrootgood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/contexttestallowed",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/generatedallowed",
	)
	analysistest.Run(
		t,
		filepath.Join(analysistest.TestData(), "contextroot"),
		ContextRootAnalyzer,
		"github.com/geonwoo-jeong/japanese-law-mcp/cmd/japanese-law-mcp",
		"github.com/geonwoo-jeong/japanese-law-mcp/cmd/quality",
	)
}
