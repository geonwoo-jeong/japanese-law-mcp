package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/releasecheck"
)

const (
	exitSuccess    = 0
	exitValidation = 1
	exitUsage      = 2
)

type notesBuilder func(
	changelog, releaseNotes, tag, repository string,
) ([]byte, error)

func main() {
	os.Exit(execute(
		os.Args[1:],
		os.Stdout,
		os.Stderr,
		releasecheck.BuildPublishNotes,
	))
}

func execute(
	args []string,
	stdout, stderr io.Writer,
	build notesBuilder,
) int {
	flags := flag.NewFlagSet("release-notes", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	var changelog string
	var releaseNotes string
	var tag string
	var repository string
	flags.StringVar(&changelog, "changelog", "", "CHANGELOG.md のパス")
	flags.StringVar(&releaseNotes, "release-notes", "", "リリース契約のパス")
	flags.StringVar(&tag, "tag", "", "v で始まる SemVer tag")
	flags.StringVar(&repository, "repository", "", "SOT を含むリポジトリ")
	if err := flags.Parse(append([]string(nil), args...)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			writeUsage(stdout)
			return exitSuccess
		}
		_, _ = fmt.Fprintln(stderr, "エラー: コマンドライン引数が不正です")
		return exitUsage
	}
	if flags.NArg() != 0 {
		_, _ = fmt.Fprintln(stderr, "エラー: 位置引数は使用できません")
		return exitUsage
	}
	if missingRequiredFlag(changelog, releaseNotes, tag, repository) {
		_, _ = fmt.Fprintln(
			stderr,
			"エラー: --changelog、--release-notes、--tag および --repository は必須です",
		)
		return exitUsage
	}
	if build == nil {
		_, _ = fmt.Fprintln(stderr, "エラー: リリース情報の生成処理がありません")
		return exitValidation
	}
	content, err := build(changelog, releaseNotes, tag, repository)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: %s\n", err)
		return exitValidation
	}
	if _, err := io.Copy(stdout, bytes.NewReader(content)); err != nil {
		_, _ = fmt.Fprintf(stderr, "エラー: 標準出力へ書き込めません: %s\n", err)
		return exitValidation
	}
	return exitSuccess
}

func missingRequiredFlag(values ...string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return true
		}
	}
	return false
}

func writeUsage(output io.Writer) {
	_, _ = fmt.Fprintln(
		output,
		"使用方法: release-notes --changelog <path> --release-notes <path> "+
			"--tag <vSemVer> --repository <path>",
	)
}
