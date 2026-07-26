package sotvet

import (
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const modulePath = "github.com/geonwoo-jeong/japanese-law-mcp"

type dependencyRule struct {
	owner  string
	denied []string
}

func configuredDependencyRules() []dependencyRule {
	return []dependencyRule{
		{
			owner: modulePath + "/internal/model",
			denied: []string{
				"encoding/xml",
				"net/http",
				"github.com/modelcontextprotocol/go-sdk",
				modulePath + "/internal/mcp",
				modulePath + "/internal/source",
				modulePath + "/internal/transport",
			},
		},
		{
			owner: modulePath + "/internal/application",
			denied: []string{
				"encoding/xml",
				"net/http",
				"github.com/modelcontextprotocol/go-sdk",
				modulePath + "/internal/cli",
				modulePath + "/internal/config",
				modulePath + "/internal/mcp",
				modulePath + "/internal/source",
				modulePath + "/internal/transport",
			},
		},
		{
			// SOT-ARCH-002 の transport は MCP トランスポート境界であり、情報源の HTTP クライアント共通層ではない。
			owner: modulePath + "/internal/source",
			denied: []string{
				"github.com/modelcontextprotocol/go-sdk",
				modulePath + "/internal/mcp",
				modulePath + "/internal/transport",
			},
		},
		{
			owner: modulePath + "/internal/mcp",
			denied: []string{
				modulePath + "/internal/source",
				modulePath + "/internal/transport",
			},
		},
		{
			owner: modulePath + "/internal/transport",
			denied: []string{
				modulePath + "/internal/source",
			},
		},
	}
}

// DependencyDirectionAnalyzer は、SOT-ARCH-007 の禁止依存を検査する。
var DependencyDirectionAnalyzer = &analysis.Analyzer{
	Name: "sotdependency",
	Doc:  "SOT-ARCH-007 に反するパッケージ依存を検出する",
	Run:  runDependencyDirection,
}

func runDependencyDirection(pass *analysis.Pass) (any, error) {
	rule, ok := dependencyRuleFor(pass.Pkg.Path())
	if !ok {
		return nil, nil
	}

	for _, file := range pass.Files {
		for _, imported := range file.Imports {
			importPath, err := strconv.Unquote(imported.Path.Value)
			if err != nil {
				continue
			}
			for _, denied := range rule.denied {
				if packageIsWithin(importPath, denied) {
					pass.Reportf(
						imported.Path.Pos(),
						"SOT-ARCH-007: %s から %s への依存は禁止されています",
						pass.Pkg.Path(),
						importPath,
					)
					break
				}
			}
		}
	}

	return nil, nil
}

func dependencyRuleFor(packagePath string) (dependencyRule, bool) {
	for _, rule := range configuredDependencyRules() {
		if packageIsWithin(packagePath, rule.owner) {
			return rule, true
		}
	}
	return dependencyRule{}, false
}

func packageIsWithin(packagePath, parent string) bool {
	return packagePath == parent || strings.HasPrefix(packagePath, parent+"/")
}
