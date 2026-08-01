package legalquerycandidateworker

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

const verificationDeterministicReplay = "candidate-evaluation-deterministic-replay"

func TestRunは同じ入力を同じ二Fileへ再現する(t *testing.T) {
	t.Parallel()

	requestRaw := []byte("{\"request\":\"synthetic\"}\n")
	dependencies := syntheticDependencies(requestRaw, true)
	firstRoot := filepath.Join(t.TempDir(), "output")
	secondRoot := filepath.Join(t.TempDir(), "output")

	first, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: firstRoot}, dependencies)
	if err != nil {
		t.Fatalf("%s: 一回目を実行できません: %v", verificationDeterministicReplay, err)
	}
	second, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: secondRoot}, dependencies)
	if err != nil {
		t.Fatalf("%s: replay を実行できません: %v", verificationDeterministicReplay, err)
	}
	if first != second || first.Outcome != legalquerycandidateeval.EvaluationOutcomePassed {
		t.Fatalf("%s: handoff が一致しません: first=%+v second=%+v", verificationDeterministicReplay, first, second)
	}
	for _, name := range []string{"report.json", "result.json"} {
		firstRaw := mustReadWorkerFile(t, firstRoot, first.EvaluationID, name)
		secondRaw := mustReadWorkerFile(t, secondRoot, second.EvaluationID, name)
		if !bytes.Equal(firstRaw, secondRaw) {
			t.Fatalf("%s: %s の replay byte が一致しません", verificationDeterministicReplay, name)
		}
	}
	assertClosedOutput(t, firstRoot, first.EvaluationID)
	resultRaw := mustReadWorkerFile(t, firstRoot, first.EvaluationID, "result.json")
	result, err := legalquerycandidateeval.DecodeEvaluationResult(resultRaw)
	if err != nil {
		t.Fatalf("result を decode できません: %v", err)
	}
	if result.RequestSHA256 != legalquerycandidateeval.RawSHA256(requestRaw) ||
		result.ReportSHA256 != first.ReportSHA256 ||
		legalquerycandidateeval.RawSHA256(resultRaw) != first.ResultSHA256 {
		t.Fatal("result または handoff の digest binding が一致しません")
	}
}

func TestRunはAcceptance未達を有効なFailedHandoffにする(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "output")
	handoff, err := Run(
		context.Background(),
		Input{RepositoryRoot: ".", OutputRoot: root},
		syntheticDependencies([]byte("synthetic request\n"), false),
	)
	if err != nil {
		t.Fatalf("failed handoff を構成できません: %v", err)
	}
	if handoff.Outcome != legalquerycandidateeval.EvaluationOutcomeFailed {
		t.Fatalf("outcome=%q, want failed", handoff.Outcome)
	}
	assertClosedOutput(t, root, handoff.EvaluationID)
}

func TestRunはLoaderが検証したProductionPayloadをEvaluatorへ保持する(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "output")
	evaluationID := "evaluation-sha256-" + strings.Repeat("b", 64)
	requestRaw := []byte("synthetic request\n")
	dependencies := Dependencies{
		Load: func(context.Context, string) (PreparedEvaluation, error) {
			return PreparedEvaluation{
				EvaluationID: evaluationID,
				RequestRaw:   append([]byte(nil), requestRaw...),
				content: legalquerycandidateeval.CandidateContentManifest{
					CandidateContentID: "candidate-content-sha256-" + strings.Repeat("c", 64),
				},
				repository: ".",
			}, nil
		},
		Evaluate: func(_ context.Context, prepared PreparedEvaluation) ([]byte, error) {
			if prepared.content.CandidateContentID == "" || prepared.repository != "." {
				return nil, errors.New("production payload was lost")
			}
			return []byte("{\"report\":\"payload\"}\n"), nil
		},
		Accept: func([]byte) (bool, error) { return true, nil },
	}
	if _, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: root}, dependencies); err != nil {
		t.Fatalf("loader の production payload が evaluator まで保持されません: %v", err)
	}
}

func TestRunは失敗時に部分Outputを残さない(t *testing.T) {
	t.Parallel()

	base := syntheticDependencies([]byte("synthetic request\n"), true)
	cases := map[string]struct {
		mutate   func(Dependencies) Dependencies
		wantCode int
	}{
		"load": {
			mutate: func(value Dependencies) Dependencies {
				value.Load = func(context.Context, string) (PreparedEvaluation, error) {
					return PreparedEvaluation{}, errors.New("load failure")
				}
				return value
			},
			wantCode: FailureCodePreparedLoad,
		},
		"identity": {
			mutate: func(value Dependencies) Dependencies {
				value.Load = func(context.Context, string) (PreparedEvaluation, error) {
					return PreparedEvaluation{EvaluationID: "invalid", RequestRaw: []byte("request")}, nil
				}
				return value
			},
			wantCode: FailureCodeRequestBinding,
		},
		"evaluate": {
			mutate: func(value Dependencies) Dependencies {
				value.Evaluate = func(context.Context, PreparedEvaluation) ([]byte, error) {
					return nil, errors.New("evaluation failure")
				}
				return value
			},
			wantCode: FailureCodeEvaluateBuild,
		},
		"accept": {
			mutate: func(value Dependencies) Dependencies {
				value.Accept = func([]byte) (bool, error) { return false, errors.New("accept failure") }
				return value
			},
			wantCode: FailureCodeAccept,
		},
	}
	for name, mutate := range cases {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			root := filepath.Join(t.TempDir(), "output")
			_, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: root}, mutate.mutate(base))
			if err == nil {
				t.Fatalf("%s failure を受理しました", name)
			}
			if code := FailureExitCode(err); code != mutate.wantCode {
				t.Fatalf("%s failure code=%d, want %d", name, code, mutate.wantCode)
			}
			if _, err := os.Lstat(root); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("%s failure 後に output が残りました: %v", name, err)
			}
		})
	}
}

func TestRunは既存Handoffを上書きしない(t *testing.T) {
	t.Parallel()

	root := filepath.Join(t.TempDir(), "output")
	dependencies := syntheticDependencies([]byte("synthetic request\n"), true)
	first, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: root}, dependencies)
	if err != nil {
		t.Fatalf("一回目を実行できません: %v", err)
	}
	before := mustReadWorkerFile(t, root, first.EvaluationID, "result.json")
	if _, err := Run(context.Background(), Input{RepositoryRoot: ".", OutputRoot: root}, dependencies); err == nil {
		t.Fatal("既存 handoff の上書きを受理しました")
	}
	after := mustReadWorkerFile(t, root, first.EvaluationID, "result.json")
	if !bytes.Equal(before, after) {
		t.Fatal("既存 handoff byte が変更されました")
	}
}

func syntheticDependencies(requestRaw []byte, accepted bool) Dependencies {
	evaluationID := "evaluation-sha256-" + strings.Repeat("a", 64)
	return Dependencies{
		Load: func(context.Context, string) (PreparedEvaluation, error) {
			return PreparedEvaluation{
				EvaluationID: evaluationID,
				RequestRaw:   append([]byte(nil), requestRaw...),
			}, nil
		},
		Evaluate: func(_ context.Context, prepared PreparedEvaluation) ([]byte, error) {
			if len(prepared.RequestRaw) > 0 {
				prepared.RequestRaw[0] ^= 1
			}
			return []byte("{\"report\":\"synthetic\"}\n"), nil
		},
		Accept: func([]byte) (bool, error) { return accepted, nil },
	}
}

func assertClosedOutput(t *testing.T, root, evaluationID string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil || len(entries) != 1 || entries[0].Name() != evaluationID || !entries[0].IsDir() {
		t.Fatalf("output root が一 evaluation directory に閉じていません: entries=%v err=%v", entries, err)
	}
	children, err := os.ReadDir(filepath.Join(root, evaluationID))
	if err != nil || len(children) != 2 || children[0].Name() != "report.json" || children[1].Name() != "result.json" {
		t.Fatalf("handoff が report/result 二 file に閉じていません: entries=%v err=%v", children, err)
	}
	for _, child := range children {
		info, infoErr := child.Info()
		if infoErr != nil || !info.Mode().IsRegular() || child.Type()&os.ModeSymlink != 0 {
			t.Fatalf("handoff entry が通常 file ではありません: %s", child.Name())
		}
	}
}

func mustReadWorkerFile(t *testing.T, root, evaluationID, name string) []byte {
	t.Helper()
	//nolint:gosec // SOT-ENG-038: t.TempDir 配下の固定 synthetic handoff file だけを読む。
	raw, err := os.ReadFile(filepath.Join(root, evaluationID, name))
	if err != nil {
		t.Fatalf("worker output を読めません: %v", err)
	}
	return raw
}
