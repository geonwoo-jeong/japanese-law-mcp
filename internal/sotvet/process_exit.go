package sotvet

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// ProcessExitAnalyzer は、SOT-ENG-014 に反する直接終了を検査する。
var ProcessExitAnalyzer = &analysis.Analyzer{
	Name:     "sotprocessexit",
	Doc:      "SOT-ENG-014 に反する os.Exit と log.Fatal 系の呼出しを検出する",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      runProcessExit,
}

func runProcessExit(pass *analysis.Pass) (any, error) {
	tree := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	parents := parentNodes(pass)
	cliExitCodes := allowedCLIExitCodeVariables(pass)
	tree.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(node ast.Node) {
		call := node.(*ast.CallExpr)
		if isGeneratedFile(pass, call) {
			return
		}

		function, ok := calledFunction(pass, call)
		if !ok || function.Pkg() == nil {
			return
		}

		switch function.Pkg().Path() {
		case "os":
			if function.Name() != "Exit" {
				return
			}
			if isProductCommandMain(pass, call, parents) ||
				isDevelopmentCommandMain(pass, call, parents) ||
				isAllowedCLIChildProcessExit(pass, call, cliExitCodes) {
				return
			}
			reportProcessExit(pass, call, function)
		case "log":
			if isFatalFunction(function.Name()) {
				reportProcessExit(pass, call, function)
			}
		}
	})
	tree.Preorder(
		[]ast.Node{(*ast.SelectorExpr)(nil), (*ast.Ident)(nil)},
		func(node ast.Node) {
			if isGeneratedFile(pass, node) ||
				isSelectorIdentifier(node, parents) ||
				isDirectCalleeReference(node, parents) {
				return
			}

			expression, ok := node.(ast.Expr)
			if !ok {
				return
			}
			function, ok := referencedFunction(pass, expression)
			if !ok || function.Pkg() == nil {
				return
			}
			if function.Pkg().Path() == "os" && function.Name() == "Exit" {
				reportProcessExit(pass, expression, function)
				return
			}
			if function.Pkg().Path() == "log" && isFatalFunction(function.Name()) {
				reportProcessExit(pass, expression, function)
			}
		},
	)
	return nil, nil
}

func reportProcessExit(
	pass *analysis.Pass,
	position ast.Node,
	function *types.Func,
) {
	if function.Pkg().Path() == "os" {
		pass.Reportf(
			position.Pos(),
			"SOT-ENG-014: os.Exit は製品または開発用コマンドの main 以外では使用できません",
		)
		return
	}

	pass.Reportf(
		position.Pos(),
		"SOT-ENG-014: log.%s は処理を直接終了するため使用できません",
		function.Name(),
	)
}

func isDevelopmentCommandMain(
	pass *analysis.Pass,
	position ast.Node,
	parents map[ast.Node]ast.Node,
) bool {
	return isDevelopmentCommand(pass) && isCommandMain(pass, position, parents)
}

func isAllowedCLIChildProcessExit(
	pass *analysis.Pass,
	call *ast.CallExpr,
	cliExitCodes map[*types.Var]struct{},
) bool {
	if !isTestFile(pass, call) || len(call.Args) != 1 {
		return false
	}
	if isCLIExecuteExpression(pass, call.Args[0]) {
		return true
	}

	identifier, ok := unparenthesizedExpression(call.Args[0]).(*ast.Ident)
	if !ok {
		return false
	}
	variable, ok := pass.TypesInfo.Uses[identifier].(*types.Var)
	if !ok {
		return false
	}
	_, ok = cliExitCodes[variable]
	return ok
}

type variableWrite struct {
	count      int
	cliExecute bool
	escaped    bool
}

func allowedCLIExitCodeVariables(pass *analysis.Pass) map[*types.Var]struct{} {
	writes := make(map[*types.Var]variableWrite)
	for _, file := range pass.Files {
		ast.Inspect(file, func(node ast.Node) bool {
			switch node := node.(type) {
			case *ast.AssignStmt:
				recordAssignmentWrites(pass, writes, node.Lhs, node.Rhs)
			case *ast.ValueSpec:
				left := make([]ast.Expr, 0, len(node.Names))
				for _, name := range node.Names {
					left = append(left, name)
				}
				recordAssignmentWrites(pass, writes, left, node.Values)
			case *ast.IncDecStmt:
				recordVariableWrite(pass, writes, node.X, nil)
			case *ast.RangeStmt:
				recordVariableWrite(pass, writes, node.Key, nil)
				recordVariableWrite(pass, writes, node.Value, nil)
			case *ast.UnaryExpr:
				if node.Op == token.AND {
					if variable := assignedVariable(pass, node.X); variable != nil {
						state := writes[variable]
						state.escaped = true
						writes[variable] = state
					}
				}
			}
			return true
		})
	}

	allowed := make(map[*types.Var]struct{})
	for variable, write := range writes {
		if write.count == 1 && write.cliExecute && !write.escaped {
			allowed[variable] = struct{}{}
		}
	}
	return allowed
}

func recordAssignmentWrites(
	pass *analysis.Pass,
	writes map[*types.Var]variableWrite,
	left []ast.Expr,
	right []ast.Expr,
) {
	for index, expression := range left {
		var value ast.Expr
		if len(left) == len(right) {
			value = right[index]
		}
		recordVariableWrite(pass, writes, expression, value)
	}
}

func recordVariableWrite(
	pass *analysis.Pass,
	writes map[*types.Var]variableWrite,
	left ast.Expr,
	right ast.Expr,
) {
	variable := assignedVariable(pass, left)
	if variable == nil {
		return
	}

	state := writes[variable]
	state.count++
	state.cliExecute = state.count == 1 && isCLIExecuteExpression(pass, right)
	writes[variable] = state
}

func assignedVariable(pass *analysis.Pass, expression ast.Expr) *types.Var {
	identifier, ok := unparenthesizedExpression(expression).(*ast.Ident)
	if !ok || identifier.Name == "_" {
		return nil
	}
	if variable, ok := pass.TypesInfo.Defs[identifier].(*types.Var); ok {
		return variable
	}
	variable, _ := pass.TypesInfo.Uses[identifier].(*types.Var)
	return variable
}

func isCLIExecuteExpression(pass *analysis.Pass, expression ast.Expr) bool {
	call, ok := unparenthesizedExpression(expression).(*ast.CallExpr)
	if !ok {
		return false
	}
	function, ok := calledFunction(pass, call)
	if !ok || function.Pkg() == nil {
		return false
	}
	signature, ok := function.Type().(*types.Signature)
	return ok &&
		signature.Recv() == nil &&
		function.Pkg().Path() == projectCLIPackage &&
		function.Name() == "Execute"
}

func unparenthesizedExpression(expression ast.Expr) ast.Expr {
	for {
		parentheses, ok := expression.(*ast.ParenExpr)
		if !ok {
			return expression
		}
		expression = parentheses.X
	}
}

func isFatalFunction(name string) bool {
	switch name {
	case "Fatal", "Fatalf", "Fatalln":
		return true
	default:
		return false
	}
}
