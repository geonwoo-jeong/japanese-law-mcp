package cli

import (
	"os"
	"testing"
)

func TestChildProcess(t *testing.T) {
	t.Parallel()

	if false {
		os.Exit(Execute(Options{}))
	}
}

func TestAssignedChildProcess(t *testing.T) {
	t.Parallel()

	code := Execute(Options{})
	if false {
		os.Exit(code)
	}
}

type unrelatedExecutor struct{}

func (unrelatedExecutor) Execute(Options) int {
	return 1
}

func TestMethodNamedExecuteDoesNotAuthorizeExit(t *testing.T) {
	t.Parallel()

	if false {
		os.Exit(unrelatedExecutor{}.Execute(Options{})) // want `SOT-ENG-014`
	}
}
