package githook

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const maximumPushInputLine = 1024 * 1024

type pushUpdate struct {
	localRef  string
	localOID  string
	remoteOID string
}

type tipPlan struct {
	oid    string
	ranges []string
}

func (app *application) prePush(ctx context.Context) error {
	updates, err := parsePushUpdates(app.stdin)
	if err != nil {
		return err
	}
	plans, err := app.planPush(ctx, updates)
	if err != nil {
		return err
	}
	for _, plan := range plans {
		if err := app.checkTip(ctx, plan); err != nil {
			return err
		}
	}
	return nil
}

func parsePushUpdates(input io.Reader) ([]pushUpdate, error) {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 4096), maximumPushInputLine)
	var updates []pushUpdate
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		fields := strings.Fields(scanner.Text())
		if len(fields) != 4 {
			return nil, fmt.Errorf("pre-push 標準入力の %d 行目は 4 項目ではありません", lineNumber)
		}
		if err := validateOIDPair(fields[1], fields[3]); err != nil {
			return nil, fmt.Errorf("pre-push 標準入力の %d 行目: %w", lineNumber, err)
		}
		updates = append(updates, pushUpdate{
			localRef:  fields[0],
			localOID:  strings.ToLower(fields[1]),
			remoteOID: strings.ToLower(fields[3]),
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("pre-push 標準入力を読み取れませんでした: %w", err)
	}
	return updates, nil
}

func validateOIDPair(localOID, remoteOID string) error {
	if !validOID(localOID) {
		return fmt.Errorf("local object ID の形式が不正です: %s", localOID)
	}
	if !validOID(remoteOID) {
		return fmt.Errorf("remote object ID の形式が不正です: %s", remoteOID)
	}
	if len(localOID) != len(remoteOID) {
		return errors.New("local と remote の object ID 長が一致しません")
	}
	return nil
}

func validOID(value string) bool {
	if len(value) != 40 && len(value) != 64 {
		return false
	}
	for _, character := range value {
		if !isHexCharacter(character) {
			return false
		}
	}
	return true
}

func isHexCharacter(character rune) bool {
	return '0' <= character && character <= '9' ||
		'a' <= character && character <= 'f' ||
		'A' <= character && character <= 'F'
}

func allZeroOID(value string) bool {
	return strings.Trim(value, "0") == ""
}

func (app *application) planPush(
	ctx context.Context,
	updates []pushUpdate,
) ([]tipPlan, error) {
	plans := make([]tipPlan, 0, len(updates))
	planIndexes := make(map[string]int)
	resolvedObjects := make(map[string]string)
	for _, update := range updates {
		if allZeroOID(update.localOID) {
			continue
		}
		localCommit, err := app.resolveCommit(ctx, update.localOID, resolvedObjects)
		if err != nil {
			return nil, fmt.Errorf("%s の local commit が不正です: %w", update.localRef, err)
		}
		remoteCommit := ""
		if !allZeroOID(update.remoteOID) {
			remoteCommit, _ = app.resolveCommit(ctx, update.remoteOID, resolvedObjects)
		}
		plans, planIndexes = addUpdatePlan(
			plans,
			planIndexes,
			localCommit,
			remoteCommit,
		)
	}
	return plans, nil
}

func (app *application) resolveCommit(
	ctx context.Context,
	oid string,
	resolved map[string]string,
) (string, error) {
	if commit, ok := resolved[oid]; ok {
		if commit == "" {
			return "", fmt.Errorf("object %s は commit へ解決できません", oid)
		}
		return commit, nil
	}
	command := gitCommand(ctx, app.repository, nil, "rev-parse", "--verify", oid+"^{commit}")
	output, err := command.Output()
	if err != nil {
		resolved[oid] = ""
		return "", fmt.Errorf("object %s は commit へ解決できません", oid)
	}
	commit := strings.ToLower(stringWithoutLineEnding(output))
	if !validOID(commit) || len(commit) != len(oid) {
		resolved[oid] = ""
		return "", fmt.Errorf("object %s の peeled commit が不正です", oid)
	}
	resolved[oid] = commit
	return commit, nil
}

func addUpdatePlan(
	plans []tipPlan,
	indexes map[string]int,
	localCommit, remoteCommit string,
) ([]tipPlan, map[string]int) {
	index, found := indexes[localCommit]
	if !found {
		index = len(plans)
		indexes[localCommit] = index
		plans = append(plans, tipPlan{oid: localCommit})
	}
	gitRange := localCommit
	if remoteCommit != "" {
		gitRange = remoteCommit + ".." + localCommit
	}
	if !containsString(plans[index].ranges, gitRange) {
		plans[index].ranges = append(plans[index].ranges, gitRange)
	}
	return plans, indexes
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func (app *application) checkTip(ctx context.Context, plan tipPlan) (result error) {
	directory, err := os.MkdirTemp("", "japanese-law-mcp-pre-push-")
	if err != nil {
		return fmt.Errorf("commit snapshot 用の一時ディレクトリを作成できませんでした: %w", err)
	}
	defer func() {
		if cleanupErr := os.RemoveAll(directory); result == nil && cleanupErr != nil {
			result = fmt.Errorf("commit snapshot を削除できませんでした: %w", cleanupErr)
		}
	}()

	snapshot := filepath.Join(directory, "snapshot")
	if err := os.Mkdir(snapshot, 0o700); err != nil {
		return err
	}
	if err := app.materializeTreeSnapshot(ctx, plan.oid, snapshot); err != nil {
		return err
	}
	return app.qualityGate(
		ctx,
		"pre-push",
		snapshot,
		app.repository,
		"",
		append([]string(nil), plan.ranges...),
		app.stdout,
		app.stderr,
	)
}
