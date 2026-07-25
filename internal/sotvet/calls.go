package sotvet

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

const (
	projectModulePath  = "github.com/japanese-law-mcp/japanese-law-mcp"
	productCommandPath = projectModulePath + "/cmd/japanese-law-mcp"
	projectCLIPackage  = projectModulePath + "/internal/cli"
)

func calledFunction(pass *analysis.Pass, call *ast.CallExpr) (*types.Func, bool) {
	return referencedFunction(pass, call.Fun)
}

func referencedFunction(pass *analysis.Pass, expression ast.Expr) (*types.Func, bool) {
	switch expression := expression.(type) {
	case *ast.Ident:
		function, ok := pass.TypesInfo.Uses[expression].(*types.Func)
		return function, ok
	case *ast.SelectorExpr:
		function, ok := pass.TypesInfo.Uses[expression.Sel].(*types.Func)
		return function, ok
	case *ast.ParenExpr:
		return referencedFunction(pass, expression.X)
	default:
		return nil, false
	}
}

func isTestOrGeneratedFile(pass *analysis.Pass, position ast.Node) bool {
	return isTestFile(pass, position) || isGeneratedFile(pass, position)
}

func isTestFile(pass *analysis.Pass, position ast.Node) bool {
	return strings.HasSuffix(normalizedFilename(pass, position), "_test.go")
}

func isGeneratedFile(pass *analysis.Pass, position ast.Node) bool {
	if !strings.HasSuffix(normalizedFilename(pass, position), ".go") {
		return true
	}

	for _, file := range pass.Files {
		if file.Pos() <= position.Pos() && position.End() <= file.End() {
			return ast.IsGenerated(file)
		}
	}
	return false
}

func isDirectCalleeReference(node ast.Node, parents map[ast.Node]ast.Node) bool {
	current := node
	for {
		switch parent := parents[current].(type) {
		case *ast.ParenExpr:
			if parent.X != current {
				return false
			}
			current = parent
		case *ast.CallExpr:
			return parent.Fun == current
		default:
			return false
		}
	}
}

func isSelectorIdentifier(node ast.Node, parents map[ast.Node]ast.Node) bool {
	identifier, ok := node.(*ast.Ident)
	if !ok {
		return false
	}
	selector, ok := parents[node].(*ast.SelectorExpr)
	return ok && selector.Sel == identifier
}

func isProductCommandMain(
	pass *analysis.Pass,
	position ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	return pass.Pkg.Path() == productCommandPath &&
		strings.HasSuffix(normalizedFilename(pass, position), "/main.go") &&
		isCommandMain(pass, position, parents)
}

func isCommandPackage(pass *analysis.Pass) bool {
	return pass.Pkg.Name() == "main" &&
		strings.HasPrefix(pass.Pkg.Path(), projectModulePath+"/cmd/")
}

func isDevelopmentCommand(pass *analysis.Pass) bool {
	return isCommandPackage(pass) &&
		pass.Pkg.Path() != productCommandPath
}

func isCommandMain(
	pass *analysis.Pass,
	position ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	if !isCommandPackage(pass) {
		return false
	}

	for current := position; current != nil; current = parents[current] {
		switch declaration := current.(type) {
		case *ast.FuncLit:
			return false
		case *ast.FuncDecl:
			return declaration.Recv == nil && declaration.Name.Name == "main"
		}
	}
	return false
}

func normalizedFilename(pass *analysis.Pass, position ast.Node) string {
	return filepath.ToSlash(pass.Fset.Position(position.Pos()).Filename)
}

func parentNodes(pass *analysis.Pass) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	for _, file := range pass.Files {
		var stack []ast.Node
		ast.Inspect(file, func(node ast.Node) bool {
			if node == nil {
				stack = stack[:len(stack)-1]
				return false
			}
			if len(stack) > 0 {
				parents[node] = stack[len(stack)-1]
			}
			stack = append(stack, node)
			return true
		})
	}
	return parents
}
