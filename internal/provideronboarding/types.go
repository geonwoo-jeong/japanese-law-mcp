package provideronboarding

import (
	"context"
	"errors"
	"io"
)

const (
	canonicalSchemaPath  = "conformance/provider-capability.schema.json"
	canonicalLoaderPath  = "internal/providerconformance/loader.go"
	canonicalCommandPath = "cmd/provider-onboarding-fit/main.go"
)

// ErrInvalidBaseRef は、指定値を比較元 commit として使用できないことを示す。
var ErrInvalidBaseRef = errors.New("--base-ref を commit として解決できません")

// Options は、検査対象の source snapshot と Git 比較条件を表す。
type Options struct {
	Repository         string
	GitRepository      string
	BaseRef            string
	HeadRef            string
	IncludeIndex       bool
	IncludeWorkingTree bool
	IncludeUntracked   bool
	Stdout             io.Writer
	Stderr             io.Writer
}

type matrixRow struct {
	providerID    string
	implementedBy string
	status        string
}

type dependencies struct {
	load func(string) ([]matrixRow, error)
	test func(context.Context, string, io.Writer, io.Writer) error
}

type comparison struct {
	baseCommit string
	headCommit string
	mergeBase  string
}

type changeSources struct {
	index       bool
	workingTree bool
	untracked   bool
}
