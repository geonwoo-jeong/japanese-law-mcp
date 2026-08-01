// Package legalquerycandidateworker は、候補評価を閉じた二 file の handoff へ変換する。
package legalquerycandidateworker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

const (
	maximumRequestBytes = 256 << 10
	maximumReportBytes  = 4 << 20
	maximumResultBytes  = 256 << 10
)

var evaluationIDPattern = regexp.MustCompile(`^evaluation-sha256-[0-9a-f]{64}$`)

// Input は worker が使用できる固定 repository と output root だけを保持する。
type Input struct {
	RepositoryRoot string
	OutputRoot     string
}

// PreparedEvaluation は検証済み request の公開可能な identity と原 byte を保持する。
type PreparedEvaluation struct {
	EvaluationID     string
	RequestRaw       []byte
	request          legalquerycandidateeval.EvaluationRequest
	content          legalquerycandidateeval.CandidateContentManifest
	repository       string
	tracked          *legalquerycandidateeval.EvaluationResult
	trackedRaw       []byte
	trackedReportRaw []byte
}

// Dependencies は loader、evaluator、受入判定の順序を固定する。
type Dependencies struct {
	Load     func(context.Context, string) (PreparedEvaluation, error)
	Evaluate func(context.Context, PreparedEvaluation) ([]byte, error)
	Accept   func([]byte) (bool, error)
}

// Handoff は log へ出せる artifact identity だけを保持する。
type Handoff struct {
	EvaluationID string
	Outcome      string
	ReportSHA256 string
	ResultSHA256 string
}

// Run は評価を一回実行し、report/result の二 file を exclusive create する。
func Run(ctx context.Context, input Input, dependencies Dependencies) (Handoff, error) {
	if err := validateRunBoundary(ctx, input, dependencies); err != nil {
		return Handoff{}, wrapFailure(FailureCodeUnknown, err)
	}
	prepared, err := dependencies.Load(ctx, input.RepositoryRoot)
	if err != nil {
		return Handoff{}, wrapFailure(
			FailureCodePreparedLoad,
			fmt.Errorf("候補評価 request を準備できません: %w", err),
		)
	}
	if !evaluationIDPattern.MatchString(prepared.EvaluationID) ||
		len(prepared.RequestRaw) == 0 || len(prepared.RequestRaw) > maximumRequestBytes {
		return Handoff{}, wrapFailure(
			FailureCodeRequestBinding,
			fmt.Errorf("候補評価 request identity が不正です"),
		)
	}
	requestRaw := bytes.Clone(prepared.RequestRaw)
	evaluationInput := clonePreparedEvaluation(prepared)
	evaluationInput.RequestRaw = bytes.Clone(requestRaw)
	reportRaw, err := dependencies.Evaluate(ctx, evaluationInput)
	if err != nil {
		return Handoff{}, wrapFailure(
			FailureCodeEvaluateBuild,
			fmt.Errorf("候補評価 report を構成できません: %w", err),
		)
	}
	if len(reportRaw) == 0 || len(reportRaw) > maximumReportBytes {
		return Handoff{}, wrapFailure(
			FailureCodeEvaluateBuild,
			fmt.Errorf("候補評価 report の size が上限外です"),
		)
	}
	reportRaw = bytes.Clone(reportRaw)
	accepted, err := dependencies.Accept(bytes.Clone(reportRaw))
	if err != nil {
		return Handoff{}, wrapFailure(
			FailureCodeAccept,
			fmt.Errorf("候補評価の受入判定を完了できません: %w", err),
		)
	}
	outcome := legalquerycandidateeval.EvaluationOutcomeFailed
	if accepted {
		outcome = legalquerycandidateeval.EvaluationOutcomePassed
	}
	result := legalquerycandidateeval.EvaluationResult{
		ArtifactKind:  legalquerycandidateeval.ArtifactKindEvaluationResult,
		SchemaVersion: legalquerycandidateeval.SchemaVersionV2,
		EvaluationID:  prepared.EvaluationID,
		RequestSHA256: legalquerycandidateeval.RawSHA256(requestRaw),
		Outcome:       outcome,
		ReportSHA256:  legalquerycandidateeval.RawSHA256(reportRaw),
	}
	if prepared.request.EvaluationID != "" {
		result, err = legalquerycandidateeval.NewEvaluationResult(
			prepared.request,
			requestRaw,
			reportRaw,
			outcome,
		)
		if err != nil {
			return Handoff{}, wrapFailure(
				FailureCodeResultBind,
				fmt.Errorf("候補評価 result を request へ結合できません: %w", err),
			)
		}
	}
	resultRaw, err := legalquerycandidateeval.MarshalCanonicalJSON(result)
	if err != nil {
		return Handoff{}, wrapFailure(FailureCodeResultEncode, err)
	}
	if len(resultRaw) > maximumResultBytes {
		return Handoff{}, wrapFailure(
			FailureCodeResultEncode,
			fmt.Errorf("候補評価 result の size が上限外です"),
		)
	}
	if _, err := legalquerycandidateeval.DecodeEvaluationResult(resultRaw); err != nil {
		return Handoff{}, wrapFailure(
			FailureCodeResultDecode,
			fmt.Errorf("候補評価 result が不正です: %w", err),
		)
	}
	if err := checkRunContext(ctx); err != nil {
		return Handoff{}, wrapFailure(FailureCodeHandoffWrite, err)
	}
	if err := writeClosedHandoff(input.OutputRoot, prepared.EvaluationID, reportRaw, resultRaw); err != nil {
		return Handoff{}, wrapFailure(FailureCodeHandoffWrite, err)
	}
	return Handoff{
		EvaluationID: prepared.EvaluationID,
		Outcome:      outcome,
		ReportSHA256: result.ReportSHA256,
		ResultSHA256: legalquerycandidateeval.RawSHA256(resultRaw),
	}, nil
}

func validateRunBoundary(ctx context.Context, input Input, dependencies Dependencies) error {
	if err := checkRunContext(ctx); err != nil {
		return err
	}
	if input.RepositoryRoot == "" || input.OutputRoot == "" {
		return fmt.Errorf("候補評価の repository または output root が指定されていません")
	}
	if dependencies.Load == nil || dependencies.Evaluate == nil || dependencies.Accept == nil {
		return fmt.Errorf("候補評価 dependency が初期化されていません")
	}
	return nil
}

func checkRunContext(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("候補評価 context は nil にできません")
	}
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("候補評価が取り消されました: %w", err)
	}
	return nil
}

func writeClosedHandoff(outputRoot, evaluationID string, reportRaw, resultRaw []byte) error {
	absoluteOutput, err := filepath.Abs(outputRoot)
	if err != nil || filepath.Clean(absoluteOutput) != absoluteOutput {
		return fmt.Errorf("候補評価 output root を解決できません")
	}
	if _, err := os.Lstat(absoluteOutput); !os.IsNotExist(err) {
		if err == nil {
			return fmt.Errorf("候補評価 output root は既に存在します")
		}
		return fmt.Errorf("候補評価 output root を検査できません: %w", err)
	}
	parent := filepath.Dir(absoluteOutput)
	if err := os.MkdirAll(parent, 0o700); err != nil {
		return fmt.Errorf("候補評価 output parent を作成できません: %w", err)
	}
	parentInfo, err := os.Lstat(parent)
	if err != nil || parentInfo.Mode()&os.ModeSymlink != 0 || !parentInfo.IsDir() {
		return fmt.Errorf("候補評価 output parent が通常 directory ではありません")
	}
	staging, err := os.MkdirTemp(parent, ".candidate-handoff-")
	if err != nil {
		return fmt.Errorf("候補評価 staging directory を作成できません: %w", err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	evaluationRoot := filepath.Join(staging, evaluationID)
	if err := os.Mkdir(evaluationRoot, 0o700); err != nil {
		return fmt.Errorf("候補評価 staging root を作成できません: %w", err)
	}
	if err := writeExclusiveFile(filepath.Join(evaluationRoot, "report.json"), reportRaw); err != nil {
		return err
	}
	if err := writeExclusiveFile(filepath.Join(evaluationRoot, "result.json"), resultRaw); err != nil {
		return err
	}
	if err := os.Mkdir(absoluteOutput, 0o700); err != nil {
		return fmt.Errorf("候補評価 output root を exclusive create できません: %w", err)
	}
	if err := os.Rename(evaluationRoot, filepath.Join(absoluteOutput, evaluationID)); err != nil {
		_ = os.Remove(absoluteOutput)
		return fmt.Errorf("候補評価 handoff を確定できません: %w", err)
	}
	return nil
}

func writeExclusiveFile(path string, raw []byte) error {
	//nolint:gosec // SOT-ENG-038: 新規 staging root 内の固定 report/result 名だけを exclusive create する。
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return fmt.Errorf("候補評価 handoff file を作成できません: %w", err)
	}
	writeErr := error(nil)
	if _, err := io.Copy(file, bytes.NewReader(raw)); err != nil {
		writeErr = err
	} else if err := file.Sync(); err != nil {
		writeErr = err
	}
	if err := file.Close(); writeErr == nil {
		writeErr = err
	}
	if writeErr != nil {
		return fmt.Errorf("候補評価 handoff file を書き込めません: %w", writeErr)
	}
	return nil
}
