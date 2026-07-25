package githook

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

func (app *application) materializeTreeSnapshot(
	ctx context.Context,
	treeish, snapshot string,
) (result error) {
	expected, err := app.expectedTreeFiles(ctx, treeish)
	if err != nil {
		return err
	}
	filenames := make([]string, 0, len(expected))
	var requests strings.Builder
	for filename := range expected {
		filenames = append(filenames, filename)
	}
	sort.Strings(filenames)
	for _, filename := range filenames {
		_, _ = fmt.Fprintln(&requests, expected[filename].oid)
	}

	command := gitCommand(
		ctx,
		app.repository,
		strings.NewReader(requests.String()),
		"cat-file",
		"--batch",
	)
	stdout, err := command.StdoutPipe()
	if err != nil {
		return fmt.Errorf("git object reader の標準出力を準備できませんでした: %w", err)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("git object reader を起動できませんでした: %w", err)
	}
	waited := false
	defer func() {
		if waited {
			return
		}
		_ = command.Process.Kill()
		_ = command.Wait()
	}()

	reader := bufio.NewReader(stdout)
	for _, filename := range filenames {
		if err := materializeBatchBlob(reader, snapshot, filename, expected[filename]); err != nil {
			return err
		}
	}
	if extra, readErr := reader.ReadByte(); !errors.Is(readErr, io.EOF) {
		if readErr != nil {
			return fmt.Errorf("git object reader の終了を確認できませんでした: %w", readErr)
		}
		return fmt.Errorf("git object reader が余分な byte 0x%02x を返しました", extra)
	}
	waitErr := command.Wait()
	waited = true
	if waitErr != nil {
		return fmt.Errorf("git object reader が失敗しました: %w", waitErr)
	}
	return app.validateTreeSnapshot(ctx, treeish, snapshot)
}

func materializeBatchBlob(
	reader *bufio.Reader,
	snapshot, filename string,
	expected expectedFile,
) error {
	header, err := reader.ReadString('\n')
	if err != nil {
		return fmt.Errorf("git object header を読み取れませんでした: %s: %w", filename, err)
	}
	fields := strings.Fields(strings.TrimSuffix(header, "\n"))
	if len(fields) != 3 ||
		!strings.EqualFold(fields[0], expected.oid) ||
		fields[1] != "blob" {
		return fmt.Errorf("git object header が不正です: %s: %q", filename, header)
	}
	size, err := strconv.ParseInt(fields[2], 10, 64)
	if err != nil || size < 0 {
		return fmt.Errorf("git object size が不正です: %s: %q", filename, fields[2])
	}
	target := filepath.Join(snapshot, filepath.FromSlash(filename))
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return fmt.Errorf("snapshot の親ディレクトリを作成できませんでした: %w", err)
	}
	mode := os.FileMode(0o640)
	if expected.executable {
		mode = 0o750
	}
	//nolint:gosec // SOT-ENG-021: filename は Git tree 由来かつ snapshot 内への包含を検証済みである。
	file, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, mode)
	if err != nil {
		return fmt.Errorf("snapshot file を作成できませんでした: %s: %w", filename, err)
	}
	_, copyErr := io.CopyN(file, reader, size)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("git blob を snapshot へ書き込めませんでした: %s: %w", filename, copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("snapshot file を閉じられませんでした: %s: %w", filename, closeErr)
	}
	separator, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("git object separator を読み取れませんでした: %s: %w", filename, err)
	}
	if separator != '\n' {
		return fmt.Errorf("git object separator が不正です: %s", filename)
	}
	return nil
}
