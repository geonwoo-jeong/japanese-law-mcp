package sotvet

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestSuppressionPolicyAnalyzer(t *testing.T) {
	t.Parallel()

	analysistest.Run(
		t,
		analysistest.TestData(),
		SuppressionPolicyAnalyzer,
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/suppressionbad",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/suppressiongood",
		"github.com/japanese-law-mcp/japanese-law-mcp/internal/suppressiongenerated",
	)
}
