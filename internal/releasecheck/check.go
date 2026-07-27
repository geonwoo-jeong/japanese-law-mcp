// Package releasecheck は、公式ローカル配布物のリリース契約を検証する。
package releasecheck

import (
	"context"
	"fmt"
	"strings"
)

const projectName = "japanese-law-mcp"

// Request は、一回のリリース検証に必要な入力を保持する。
type Request struct {
	ReleaseNotes string
	Tag          string
	Repository   string
	Dist         string
	Commit       string
	TargetOS     string
	TargetArch   string
}

// ValidateRequest は、フラグ間の組み合わせを外部ファイルへアクセスせず検証する。
func ValidateRequest(request Request) error {
	if strings.TrimSpace(request.ReleaseNotes) == "" ||
		strings.TrimSpace(request.Tag) == "" ||
		strings.TrimSpace(request.Repository) == "" {
		return fmt.Errorf("--release-notes、--tag および --repository は必須です")
	}

	hasDist := strings.TrimSpace(request.Dist) != ""
	hasCommit := strings.TrimSpace(request.Commit) != ""
	if hasDist != hasCommit {
		return fmt.Errorf("--dist と --commit は同じ組み合わせで指定してください")
	}

	hasTargetOS := strings.TrimSpace(request.TargetOS) != ""
	hasTargetArch := strings.TrimSpace(request.TargetArch) != ""
	if hasTargetOS != hasTargetArch {
		return fmt.Errorf("--target-os と --target-arch は同じ組み合わせで指定してください")
	}
	if hasTargetOS && !hasDist {
		return fmt.Errorf("target の組み合わせは --dist と --commit とともに指定してください")
	}
	if hasTargetOS {
		if _, exists := findReleaseTarget(
			request.TargetOS,
			request.TargetArch,
			strings.TrimPrefix(request.Tag, "v"),
		); !exists {
			return fmt.Errorf(
				"target の組み合わせ %s/%s は公式配布対象ではありません",
				request.TargetOS,
				request.TargetArch,
			)
		}
	}
	return nil
}

// Check は、SOT-DEL-004、SOT-DEL-007、SOT-DEL-010、SOT-DEL-011、
// SOT-DEL-012、SOT-DEL-014、SOT-IF-019 および SOT-ENG-020 のリリース条件を検証する。
func Check(ctx context.Context, request Request) error {
	if ctx == nil {
		return fmt.Errorf("検証コンテキストがありません")
	}
	if err := ValidateRequest(request); err != nil {
		return err
	}
	if err := validateReleaseNotes(
		request.ReleaseNotes,
		request.Tag,
		request.Repository,
	); err != nil {
		return err
	}
	if request.Dist == "" {
		return nil
	}
	if err := validateDistribution(
		request.Dist,
		request.Tag,
		request.Commit,
	); err != nil {
		return err
	}
	if request.TargetOS == "" {
		return nil
	}

	version := strings.TrimPrefix(request.Tag, "v")
	target, _ := findReleaseTarget(request.TargetOS, request.TargetArch, version)
	archivePath := target.archivePath(request.Dist)
	return smokeTarget(ctx, archivePath, target, version)
}
