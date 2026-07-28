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
		"github.com/geonwoo-jeong/japanese-law-mcp/cmd/japanese-law-mcp",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/applicationbad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/applicationgood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp/mcpbad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/mcp/mcpgood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/model/modelbad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/model/modelgood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/queryprofilebad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/queryprofilegood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/sourcebad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/source/sourcegood",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/transportbad",
		"github.com/geonwoo-jeong/japanese-law-mcp/internal/transport/transportgood",
	)
}
