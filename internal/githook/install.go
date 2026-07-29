package githook

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func (app *application) install(ctx context.Context) error {
	localValue, configured, err := app.localHooksPath(ctx)
	if err != nil {
		return err
	}
	if configured {
		if localValue != hooksPath {
			return fmt.Errorf("別の local core.hooksPath が設定されています: %s", localValue)
		}
		if err := app.checkHookFiles(); err != nil {
			return err
		}
		if err := app.warmUp(ctx, app.repository); err != nil {
			return fmt.Errorf("固定済み検証ツールを再準備できませんでした: %w", err)
		}
		_, _ = fmt.Fprintln(app.stdout, "リポジトリ固有の Git hook と検証ツールは有効です。")
		return nil
	}

	inherited, err := app.effectiveHooksPath(ctx)
	if err != nil {
		return fmt.Errorf("継承する core.hooksPath を確認できませんでした: %w", err)
	}
	if err := app.checkHookFiles(); err != nil {
		return err
	}
	if err := app.warmUp(ctx, app.repository); err != nil {
		return fmt.Errorf("固定済み検証ツールを準備できませんでした: %w", err)
	}
	if err := app.setLocalHooksPath(ctx); err != nil {
		return err
	}
	if inherited != "" {
		_, _ = fmt.Fprintf(
			app.stdout,
			"local core.hooksPath=%s を設定し、継承していた %s をこのリポジトリで上書きしました。\n",
			hooksPath,
			inherited,
		)
	}
	_, _ = fmt.Fprintf(app.stdout, "Git hook を %s から実行するよう設定しました。\n", hooksPath)
	return nil
}

func (app *application) uninstall(ctx context.Context) error {
	value, configured, err := app.localHooksPath(ctx)
	if err != nil {
		return err
	}
	if !configured {
		_, _ = fmt.Fprintln(app.stdout, "このリポジトリには local core.hooksPath がありません。")
		return nil
	}
	if value != hooksPath {
		return fmt.Errorf("管理対象外の local core.hooksPath は削除しません: %s", value)
	}

	command := gitCommand(ctx, app.repository, nil,
		"config", "--local", "--unset", "core.hooksPath",
	)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("local core.hooksPath を削除できませんでした: %w: %s", err, output)
	}
	_, _ = fmt.Fprintln(app.stdout, "local core.hooksPath を削除しました。継承設定が再び有効になります。")
	return nil
}

func (app *application) check(ctx context.Context) error {
	value, configured, err := app.localHooksPath(ctx)
	if err != nil {
		return err
	}
	if !configured {
		return errors.New("local core.hooksPath が設定されていません")
	}
	if value != hooksPath {
		return fmt.Errorf("local core.hooksPath が %s ではありません: %s", hooksPath, value)
	}
	if err := app.checkHookFiles(); err != nil {
		return err
	}
	if err := checkHookCachePaths(ctx, app.repository); err != nil {
		return err
	}
	_, _ = fmt.Fprintln(app.stdout, "リポジトリ固有の Git hook 設定は有効です。")
	return nil
}

func (app *application) localHooksPath(ctx context.Context) (string, bool, error) {
	command := gitCommand(ctx, app.repository, nil,
		"config", "--local", "--get", "core.hooksPath",
	)
	output, err := command.Output()
	if err != nil {
		if exitCode(err) == 1 {
			return "", false, nil
		}
		return "", false, fmt.Errorf("local core.hooksPath を取得できませんでした: %w", err)
	}
	return stringWithoutLineEnding(output), true, nil
}

func (app *application) effectiveHooksPath(ctx context.Context) (string, error) {
	command := gitCommand(ctx, app.repository, nil,
		"config", "--get", "core.hooksPath",
	)
	output, err := command.Output()
	if err != nil {
		if exitCode(err) == 1 {
			return "", nil
		}
		return "", err
	}
	return stringWithoutLineEnding(output), nil
}

func (app *application) setLocalHooksPath(ctx context.Context) error {
	command := gitCommand(ctx, app.repository, nil,
		"config", "--local", "core.hooksPath", hooksPath,
	)
	if output, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("local core.hooksPath を設定できませんでした: %w: %s", err, output)
	}
	return nil
}

func (app *application) checkHookFiles() error {
	hookDirectory := filepath.Join(app.repository, hooksPath)
	info, err := os.Lstat(hookDirectory)
	if err != nil {
		return fmt.Errorf("%s を確認できませんでした: %w", hookDirectory, err)
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("%s は実体のあるディレクトリではありません", hookDirectory)
	}
	if err := ensureContainedDirectory(app.repository, hookDirectory); err != nil {
		return err
	}
	for _, name := range []string{"manage", "pre-commit", "pre-push"} {
		target := filepath.Join(app.repository, hooksPath, name)
		info, err := os.Lstat(target)
		if err != nil {
			return fmt.Errorf("%s を確認できませんでした: %w", target, err)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("%s は通常ファイルではありません", target)
		}
		if hasPOSIXPermissionBits() && info.Mode().Perm()&0o111 == 0 {
			return fmt.Errorf("%s に実行権限がありません", target)
		}
		//nolint:gosec // SOT-ENG-027: target は包含確認済みの固定 hook 名だけから構築する。
		content, err := os.ReadFile(target)
		if err != nil {
			return fmt.Errorf("%s の本文を確認できませんでした: %w", target, err)
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(content)); actual != expectedHookSHA256(name) {
			return fmt.Errorf("%s の本文が管理対象の Git hook と一致しません", target)
		}
	}
	return nil
}

func expectedHookSHA256(name string) string {
	switch name {
	case "manage":
		return "40e7f2266074d1f5150bc478d865353beaabb1fb36f964d08fe131624f4bf57b"
	case "pre-commit":
		return "05034f60bc3dd71e3031f4b5144b9e42a747b1b05f3489daa344977a54a6365e"
	case "pre-push":
		return "b541e3561789fb739a0d1a1ed9bce8f8ee0bb5355d76a24676e04b8773acefb3"
	default:
		return ""
	}
}

func ensureContainedDirectory(repository, directory string) error {
	resolvedRepository, err := filepath.EvalSymlinks(repository)
	if err != nil {
		return fmt.Errorf("リポジトリの実体を解決できませんでした: %w", err)
	}
	resolvedDirectory, err := filepath.EvalSymlinks(directory)
	if err != nil {
		return fmt.Errorf("hook ディレクトリの実体を解決できませんでした: %w", err)
	}
	expected := filepath.Join(resolvedRepository, hooksPath)
	if resolvedDirectory != expected {
		return fmt.Errorf("hook ディレクトリがリポジトリ外を指しています: %s", resolvedDirectory)
	}
	return nil
}

func warmUpTools(ctx context.Context, repository string) (result error) {
	caches, err := prepareHookCachePaths(ctx, repository)
	if err != nil {
		return err
	}
	directory, err := os.MkdirTemp("", "japanese-law-mcp-hook-install-")
	if err != nil {
		return err
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); result == nil && cleanupErr != nil {
			result = cleanupErr
		}
	}()
	directory, err = filepath.Abs(directory)
	if err != nil {
		return fmt.Errorf("tool warm-up の一時ディレクトリを絶対パスにできませんでした: %w", err)
	}

	environment := environmentWithHookCaches(
		controlledGoEnvironment(os.Environ(), true),
		caches,
	)
	downloads := moduleDownloads(repository, directory)
	for _, download := range downloads {
		if copyErr := copyModuleFiles(download); copyErr != nil {
			return fmt.Errorf("%s の module 定義を一時領域へ複製できませんでした: %w", download.name, copyErr)
		}
	}
	for _, download := range downloads {
		//nolint:gosec // SOT-ENG-027: 実行ファイルと download 対象は固定済み一覧だけから構築する。
		command := exec.CommandContext(ctx, "go", download.arguments...)
		command.Dir = repository
		command.Env = environment
		if output, runErr := command.CombinedOutput(); runErr != nil {
			return fmt.Errorf("%s の依存モジュールを準備できませんでした: %w: %s", download.name, runErr, output)
		}
	}

	for _, build := range toolBuilds(directory) {
		//nolint:gosec // SOT-ENG-027: 実行ファイルと build 対象は固定済み一覧だけから構築する。
		command := exec.CommandContext(ctx, "go", build.arguments...)
		command.Dir = repository
		command.Env = environment
		if output, runErr := command.CombinedOutput(); runErr != nil {
			return fmt.Errorf("%s の build に失敗しました: %w: %s", build.name, runErr, output)
		}
	}
	return nil
}

type moduleDownload struct {
	name             string
	sourceModfile    string
	temporaryModfile string
	arguments        []string
}

func moduleDownloads(repository, directory string) []moduleDownload {
	return []moduleDownload{
		newModuleDownload(repository, directory, "go.mod", "root"),
		newModuleDownload(repository, directory, "tools/go.mod", "tools"),
		newModuleDownload(repository, directory, "tools/gitleaks/go.mod", "gitleaks"),
	}
}

func newModuleDownload(repository, directory, name, temporaryName string) moduleDownload {
	temporaryModfile := filepath.Join(directory, temporaryName+".mod")
	return moduleDownload{
		name:             name,
		sourceModfile:    filepath.Join(repository, filepath.FromSlash(name)),
		temporaryModfile: temporaryModfile,
		arguments: []string{
			"mod",
			"download",
			"-modfile=" + temporaryModfile,
			"all",
		},
	}
}

func copyModuleFiles(download moduleDownload) error {
	files := []struct {
		source string
		target string
	}{
		{source: download.sourceModfile, target: download.temporaryModfile},
		{
			source: moduleSumFile(download.sourceModfile),
			target: moduleSumFile(download.temporaryModfile),
		},
	}
	for _, file := range files {
		content, err := os.ReadFile(file.source)
		if err != nil {
			return fmt.Errorf("%s を読み取れませんでした: %w", file.source, err)
		}
		//nolint:gosec // SOT-ENG-027: 書込先は private な一時領域内の固定 module ファイル名に限定する。
		if err := os.WriteFile(file.target, content, 0o600); err != nil {
			return fmt.Errorf("%s を書き込めませんでした: %w", file.target, err)
		}
	}
	return nil
}

func moduleSumFile(modfile string) string {
	return strings.TrimSuffix(modfile, filepath.Ext(modfile)) + ".sum"
}

type toolBuild struct {
	name      string
	arguments []string
}

func toolBuilds(directory string) []toolBuild {
	return []toolBuild{
		newToolBuild(directory, "quality-gate", "", "./cmd/quality-gate"),
		newToolBuild(
			directory,
			"golangci-lint",
			"tools/go.mod",
			"github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
		),
		newToolBuild(
			directory,
			"actionlint",
			"tools/go.mod",
			"github.com/rhysd/actionlint/cmd/actionlint",
		),
		newToolBuild(
			directory,
			"gitleaks",
			"tools/gitleaks/go.mod",
			"github.com/zricethezav/gitleaks/v8",
		),
	}
}

func newToolBuild(directory, name, modfile, packagePath string) toolBuild {
	arguments := []string{"build"}
	if modfile != "" {
		arguments = append(arguments, "-modfile="+modfile)
	}
	arguments = append(arguments, "-o", filepath.Join(directory, name), packagePath)
	return toolBuild{name: name, arguments: arguments}
}
