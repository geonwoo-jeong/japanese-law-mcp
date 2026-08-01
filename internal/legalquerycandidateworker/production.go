package legalquerycandidateworker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprepare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/defaultprofile"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval/evaluators"
)

// Execute は repository の current candidate を exact evaluator v1 で一回評価する。
func Execute(ctx context.Context, input Input) (Handoff, error) {
	prepared, err := loadPreparedEvaluation(ctx, input.RepositoryRoot)
	if err != nil {
		return Handoff{}, wrapFailure(FailureCodePreparedLoad, err)
	}
	evaluate := evaluatePreparedCandidate
	if prepared.tracked != nil {
		evaluate = evaluateTrackedCandidate
	}
	dependencies := Dependencies{
		Load: func(_ context.Context, repositoryRoot string) (PreparedEvaluation, error) {
			if repositoryRoot != prepared.repository {
				return PreparedEvaluation{}, fmt.Errorf("prepared repository identity が一致しません")
			}
			return clonePreparedEvaluation(prepared), nil
		},
		Evaluate: evaluate,
		Accept: func(raw []byte) (bool, error) {
			return acceptCandidateReport(ctx, prepared.repository, raw)
		},
	}
	handoff, err := Run(ctx, input, dependencies)
	if err != nil {
		return Handoff{}, err
	}
	if prepared.tracked == nil {
		return handoff, nil
	}
	if err := verifyTrackedReplay(input.OutputRoot, prepared, handoff); err != nil {
		removeGeneratedOutput(input.OutputRoot)
		return Handoff{}, wrapFailure(FailureCodeTrackedReplay, err)
	}
	return handoff, nil
}

func loadPreparedEvaluation(
	ctx context.Context,
	repositoryRoot string,
) (PreparedEvaluation, error) {
	validator, err := legalquerycandidateprepare.NewReferenceValidator(repositoryRoot)
	if err != nil {
		return PreparedEvaluation{}, err
	}
	current, err := legalquerycandidateeval.LoadCurrentEvaluation(ctx, repositoryRoot, validator)
	if err != nil {
		return PreparedEvaluation{}, err
	}
	prepared := current.Prepared
	if prepared.Request.EvaluatorVersion != evaluators.Version1 {
		return PreparedEvaluation{}, fmt.Errorf("candidate evaluator version が未対応です")
	}
	if current.CurrentResult != nil {
		report, err := legalqueryeval.DecodeStandardReport(ctx, repositoryRoot, current.CurrentReportRaw)
		if err != nil {
			return PreparedEvaluation{}, fmt.Errorf("tracked report が不正です: %w", err)
		}
		acceptanceErr := legalqueryeval.VerifyStandardAcceptance(report)
		passed := current.CurrentResult.Outcome == legalquerycandidateeval.EvaluationOutcomePassed
		if passed != (acceptanceErr == nil) {
			return PreparedEvaluation{}, fmt.Errorf("tracked report と outcome が一致しません")
		}
	}
	return PreparedEvaluation{
		EvaluationID:     prepared.Request.EvaluationID,
		RequestRaw:       bytes.Clone(current.RequestRaw),
		request:          prepared.Request,
		content:          prepared.CandidateContent,
		repository:       repositoryRoot,
		tracked:          cloneEvaluationResult(current.CurrentResult),
		trackedRaw:       bytes.Clone(current.CurrentResultRaw),
		trackedReportRaw: bytes.Clone(current.CurrentReportRaw),
	}, nil
}

func evaluateTrackedCandidate(
	ctx context.Context,
	prepared PreparedEvaluation,
) ([]byte, error) {
	if err := checkRunContext(ctx); err != nil {
		return nil, err
	}
	if prepared.tracked == nil ||
		prepared.EvaluationID != prepared.tracked.EvaluationID ||
		prepared.request.EvaluationID != prepared.EvaluationID ||
		legalquerycandidateeval.RawSHA256(prepared.RequestRaw) !=
			prepared.tracked.RequestSHA256 ||
		legalquerycandidateeval.RawSHA256(prepared.trackedReportRaw) !=
			prepared.tracked.ReportSHA256 {
		return nil, fmt.Errorf("tracked replay identity が一致しません")
	}
	return bytes.Clone(prepared.trackedReportRaw), nil
}

func evaluatePreparedCandidate(
	ctx context.Context,
	prepared PreparedEvaluation,
) ([]byte, error) {
	request := prepared.request
	if request.EvaluationID != prepared.EvaluationID ||
		request.CandidateContentID != prepared.content.CandidateContentID {
		return nil, wrapFailure(
			FailureCodeRequestBinding,
			fmt.Errorf("candidate request と content identity が一致しません"),
		)
	}
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		return nil, wrapFailure(FailureCodeEvaluateBuild, err)
	}
	if err := verifyCandidatePlanningIdentity(candidate, prepared.content); err != nil {
		return nil, wrapFailure(FailureCodeEvaluateBuild, err)
	}
	corpusDirectory := filepath.Join("testdata", "legalquery", request.CorpusVersion)
	corpus, err := legalquerycorpus.Load(ctx, prepared.repository, corpusDirectory)
	if err != nil {
		return nil, wrapFailure(FailureCodeEvaluateBuild, err)
	}
	manifest := corpus.Manifest()
	if manifest.CorpusVersion() != request.CorpusVersion ||
		manifest.HoldoutDigest() != request.HoldoutDigest ||
		!slices.Equal(manifest.HoldoutLeakageGroupDigests(), request.HoldoutLeakageGroupDigests) {
		return nil, wrapFailure(
			FailureCodeEvaluateBuild,
			fmt.Errorf("candidate corpus が request と一致しません"),
		)
	}
	evaluator, err := defaultprofile.NewWithPlanning(candidate)
	if err != nil {
		return nil, wrapFailure(FailureCodeEvaluateBuild, err)
	}
	report, err := evaluator.BuildStandardReport(ctx, corpus, request.BaselineVersion)
	if err != nil {
		return nil, wrapFailure(FailureCodeEvaluateBuild, err)
	}
	profileSet := report.ProfileSet()
	if report.CorpusVersion() != request.CorpusVersion ||
		report.HoldoutDigest() != request.HoldoutDigest ||
		report.BaselineVersion() != request.BaselineVersion ||
		profileSet.ProfileSetID() != prepared.content.ProfileSet.ProfileSetID ||
		profileSet.ProfileSetVersion() != prepared.content.ProfileSet.ProfileSetVersion ||
		profileSet.RankingVersion() != prepared.content.ProfileSet.RankingVersion {
		return nil, wrapFailure(
			FailureCodeReportBinding,
			fmt.Errorf("candidate report identity が request と一致しません"),
		)
	}
	raw, err := json.Marshal(report)
	if err != nil {
		return nil, wrapFailure(
			FailureCodeResultEncode,
			fmt.Errorf("candidate report を canonical JSON にできません: %w", err),
		)
	}
	return append(raw, '\n'), nil
}

func verifyCandidatePlanningIdentity(
	candidate legalquerycandidateprofile.Set,
	content legalquerycandidateeval.CandidateContentManifest,
) error {
	profiles := candidate.Profiles()
	if content.ProfileSet.ProfileSetID != "default" ||
		profiles.ProfileVersion() != content.ProfileSet.ProfileSetVersion ||
		profiles.RankingVersion() != content.ProfileSet.RankingVersion {
		return fmt.Errorf("candidate profile set identity が content manifest と一致しません")
	}
	metadata := candidate.ProfileMetadata()
	if len(metadata) != len(content.ProfileArtifacts) {
		return fmt.Errorf("candidate profile metadata の件数が一致しません")
	}
	for index, profile := range metadata {
		artifact := content.ProfileArtifacts[index]
		if profile.ProfileID() != artifact.ProfileID ||
			profile.ProfileVersion() != artifact.ProfileVersion ||
			profile.SchemaVersion() != artifact.MetadataSchemaVersion ||
			profile.CueSetVersion() != artifact.CueSetVersion ||
			profile.RankingVersion() != content.ProfileSet.RankingVersion {
			return fmt.Errorf("candidate profile metadata が content manifest と一致しません")
		}
	}
	return nil
}

func acceptCandidateReport(ctx context.Context, repositoryRoot string, raw []byte) (bool, error) {
	report, err := legalqueryeval.DecodeStandardReport(ctx, repositoryRoot, raw)
	if err != nil {
		return false, err
	}
	if err := legalqueryeval.VerifyStandardAcceptance(report); err != nil {
		//nolint:nilerr // SOT-ENG-038: 構造上有効な受入基準未達は error ではなく failed outcome へ写像する。
		return false, nil
	}
	return true, nil
}

func clonePreparedEvaluation(prepared PreparedEvaluation) PreparedEvaluation {
	return PreparedEvaluation{
		EvaluationID:     prepared.EvaluationID,
		RequestRaw:       bytes.Clone(prepared.RequestRaw),
		request:          prepared.request,
		content:          prepared.content,
		repository:       prepared.repository,
		tracked:          cloneEvaluationResult(prepared.tracked),
		trackedRaw:       bytes.Clone(prepared.trackedRaw),
		trackedReportRaw: bytes.Clone(prepared.trackedReportRaw),
	}
}

func cloneEvaluationResult(
	result *legalquerycandidateeval.EvaluationResult,
) *legalquerycandidateeval.EvaluationResult {
	if result == nil {
		return nil
	}
	cloned := *result
	return &cloned
}

func verifyTrackedReplay(
	outputRoot string,
	prepared PreparedEvaluation,
	handoff Handoff,
) error {
	if prepared.tracked == nil {
		return nil
	}
	generatedRoot := filepath.Join(outputRoot, prepared.EvaluationID)
	generatedResult, err := readRegularBounded(
		filepath.Join(generatedRoot, "result.json"),
		maximumResultBytes,
	)
	if err != nil || !bytes.Equal(generatedResult, prepared.trackedRaw) {
		return fmt.Errorf("tracked result の replay byte が一致しません")
	}
	generatedReport, err := readRegularBounded(
		filepath.Join(generatedRoot, "report.json"),
		maximumReportBytes,
	)
	if err != nil {
		return err
	}
	if !bytes.Equal(generatedReport, prepared.trackedReportRaw) {
		return fmt.Errorf("tracked report の replay byte が一致しません")
	}
	if handoff.EvaluationID != prepared.tracked.EvaluationID ||
		handoff.Outcome != prepared.tracked.Outcome ||
		handoff.ReportSHA256 != prepared.tracked.ReportSHA256 ||
		handoff.ResultSHA256 != legalquerycandidateeval.RawSHA256(prepared.trackedRaw) {
		return fmt.Errorf("tracked handoff identity が一致しません")
	}
	return nil
}

func readRegularBounded(path string, maximum int) ([]byte, error) {
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() ||
		info.Size() < 1 || info.Size() > int64(maximum) {
		return nil, fmt.Errorf("tracked artifact が通常 file ではありません")
	}
	//nolint:gosec // SOT-ENG-038: Lstat 済みの固定 tracked artifact subtree 内通常 file だけを読む。
	raw, err := os.ReadFile(path)
	if err != nil || int64(len(raw)) != info.Size() {
		return nil, fmt.Errorf("tracked artifact を読めません")
	}
	return raw, nil
}

func removeGeneratedOutput(outputRoot string) {
	info, err := os.Lstat(outputRoot)
	if err == nil && info.Mode()&os.ModeSymlink == 0 && info.IsDir() {
		_ = os.RemoveAll(outputRoot)
	}
}
