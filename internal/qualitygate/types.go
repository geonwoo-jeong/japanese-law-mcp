// Package qualitygate は、SOT-ENG-020 の決定的な品質ゲートを実行する。
package qualitygate

import (
	"fmt"
	"io"
)

// Profile は、品質ゲートの実行範囲を表す。
type Profile string

const (
	// ProfilePreCommit は、ステージ済み内容に対する高速かつオフラインの検査である。
	ProfilePreCommit Profile = "pre-commit"
	// ProfilePrePush は、外部の脆弱性 DB を除く全検査である。
	ProfilePrePush Profile = "pre-push"
	// ProfileCI は、外部の脆弱性 DB と Git 全履歴を含む全検査である。
	ProfileCI Profile = "ci"
)

// ParseProfile は、CLI のプロファイル名を厳密に解釈する。
func ParseProfile(value string) (Profile, error) {
	profile := Profile(value)
	switch profile {
	case ProfilePreCommit, ProfilePrePush, ProfileCI:
		return profile, nil
	default:
		return "", fmt.Errorf("品質ゲートのプロファイルが不正です: %q", value)
	}
}

// Options は、検査対象と Git メタデータの境界を指定する。
type Options struct {
	Profile       Profile
	Repository    string
	GitRepository string
	GitRanges     []string
}

type planInput struct {
	profile       Profile
	repository    string
	snapshot      string
	changedPaths  []string
	goFormatPaths []string
	gitRanges     []string
}

type outputValidator func([]byte) error

type commandSpec struct {
	path               string
	args               []string
	dir                string
	goCommand          bool
	network            bool
	goFlags            string
	preserveGitIndex   bool
	preserveGitObjects bool
	isolateGitConfig   bool
	validateOutput     outputValidator
}

type step struct {
	key     string
	name    string
	sotID   string
	command *commandSpec
	check   func() error
}

type commandExecutor interface {
	run(commandSpec) ([]byte, []byte, error)
}

func commandStep(key, name, sotID string, command commandSpec) step {
	return step{
		key:     key,
		name:    name,
		sotID:   sotID,
		command: &command,
	}
}

func internalStep(key, name, sotID string, check func() error) step {
	return step{
		key:   key,
		name:  name,
		sotID: sotID,
		check: check,
	}
}

func normalizeWriter(writer io.Writer) io.Writer {
	if writer == nil {
		return io.Discard
	}
	return writer
}
