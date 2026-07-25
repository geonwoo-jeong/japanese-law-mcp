package sotvet

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestDependencyDirectionAnalyzer(t *testing.T) {
	t.Parallel()

	analysistest.Run(
		t,
		analysistest.TestData(),
		DependencyDirectionAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/cmd/japanese-law-mcp",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/applicationbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/applicationgood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp/mcpbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/mcp/mcpgood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/model/modelbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/model/modelgood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/source/sourcebad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/source/sourcegood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/transportbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/transport/transportgood",
	)
}
