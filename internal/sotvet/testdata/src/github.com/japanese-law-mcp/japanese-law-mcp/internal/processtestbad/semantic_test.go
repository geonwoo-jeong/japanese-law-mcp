package processtestbad

import (
	"os"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/cli"
)

func Execute() int {
	return 1
}

func TestOtherExecuteDoesNotAuthorizeExit(t *testing.T) {
	t.Parallel()

	if false {
		os.Exit(Execute()) // want `SOT-ENG-014`
	}
}

func TestReassignedExitCode(t *testing.T) {
	t.Parallel()

	code := cli.Execute(cli.Options{})
	code = 1
	if false {
		os.Exit(code) // want `SOT-ENG-014`
	}
}

func TestEscapedExitCode(t *testing.T) {
	t.Parallel()

	code := cli.Execute(cli.Options{})
	touch(&code)
	if false {
		os.Exit(code) // want `SOT-ENG-014`
	}
}

func touch(*int) {}
