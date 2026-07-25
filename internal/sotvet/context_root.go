package sotvet

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ContextRootAnalyzer は、SOT-ENG-010 に反する独立コンテキストを検査する。
var ContextRootAnalyzer = &analysis.Analyzer{
	Name:     "sotcontext",
	Doc:      "SOT-ENG-010 に反する context.Background と context.TODO を検出する",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runContextRoot,
}

func runContextRoot(pass *analysis.Pass) (any, error) {
	tree := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	parents := parentNodes(pass)
	tree.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if isTestOrGeneratedFile(pass, call) {
			return
		}

		function, ok := calledFunction(pass, call)
		if !ok || function.Pkg() == nil || function.Pkg().Path() != "context" {
			return
		}
		if function.Name() != "Background" && function.Name() != "TODO" {
			return
		}
		if isCommandRootContext(pass, call, function, parents) {
			return
		}

		reportContextRoot(pass, call, function)
	})
	tree.Preorder(
		[]ast.Node{(*ast.SelectorExpr)(nil), (*ast.Ident)(nil)},
		func(node ast.Node) {
			if isTestOrGeneratedFile(pass, node) ||
				isSelectorIdentifier(node, parents) ||
				isDirectCalleeReference(node, parents) {
				return
			}

			expression, ok := node.(ast.Expr)
			if !ok {
				return
			}
			function, ok := referencedFunction(pass, expression)
			if !ok || function.Pkg() == nil || function.Pkg().Path() != "context" {
				return
			}
			if function.Name() != "Background" && function.Name() != "TODO" {
				return
			}

			reportContextRoot(pass, expression, function)
		},
	)
	return nil, nil
}

func reportContextRoot(
	pass *analysis.Pass,
	position ast.Node,
	function *types.Func,
) {
	pass.Reportf(
		position.Pos(),
		"SOT-ENG-010: 下位処理で context.%s() を生成してはいけません",
		function.Name(),
	)
}

func isCommandRootContext(
	pass *analysis.Pass,
	call *ast.CallExpr,
	function *types.Func,
	parents map[ast.Node]ast.Node,
) bool {
	if function.Name() != "Background" || !isCommandMain(pass, call, parents) {
		return false
	}

	parentCall, ok := parents[call].(*ast.CallExpr)
	if !ok || len(parentCall.Args) == 0 || parentCall.Args[0] != call {
		return false
	}
	parentFunction, ok := calledFunction(pass, parentCall)
	if !ok || parentFunction.Pkg() == nil ||
		parentFunction.Pkg().Path() != "os/signal" ||
		parentFunction.Name() != "NotifyContext" {
		return false
	}
	return true
}
