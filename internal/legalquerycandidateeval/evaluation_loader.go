package legalquerycandidateeval

import (
	"bytes"
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryartifact"
)

// CurrentEvaluation は未評価 current と、同じ入口で検証した消費済み履歴を束ねる。
type CurrentEvaluation struct {
	Prepared         PreparedCurrent
	RequestRaw       []byte
	History          []ConsumedEvaluation
	CurrentResult    *EvaluationResult
	CurrentResultRaw []byte
	CurrentReportRaw []byte
}

// LoadCurrentEvaluation は preparation と tracked replay の両状態を閉じて検証する。
func LoadCurrentEvaluation(
	ctx context.Context,
	repositoryRoot string,
	referenceValidator ReferenceValidator,
) (CurrentEvaluation, error) {
	if err := checkContext(ctx); err != nil {
		return CurrentEvaluation{}, err
	}
	if referenceValidator == nil {
		return CurrentEvaluation{}, fmt.Errorf("外部参照 validator が指定されていません")
	}
	repository, err := legalqueryartifact.OpenRepository(repositoryRoot)
	if err != nil {
		return CurrentEvaluation{}, fmt.Errorf("candidate evaluation repository を開けません: %w", err)
	}
	defer func() { _ = repository.Close() }()
	root, err := openCandidateEvaluationRoot(repository)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	defer func() { _ = root.Close() }()
	return loadCurrentEvaluationFromRoot(ctx, repository, root, referenceValidator)
}

func loadCurrentEvaluationFromRoot(
	ctx context.Context,
	repository *legalqueryartifact.Repository,
	root *legalqueryartifact.Repository,
	referenceValidator ReferenceValidator,
) (CurrentEvaluation, error) {
	layout, err := validateRootEntries(root)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	schema, err := loadSchema(root)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	pointer, err := loadPointer(ctx, root, schema)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	artifacts, err := loadPreparationArtifacts(ctx, root, schema, layout, false)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	if err := validatePreparationBindings(artifacts); err != nil {
		return CurrentEvaluation{}, err
	}
	current, exists := artifacts.requests[pointer.EvaluationID]
	if !exists {
		return CurrentEvaluation{}, fmt.Errorf("current evaluation request が存在しません")
	}
	results, err := loadTrackedResults(ctx, root, schema, layout)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	history, currentResult, currentResultRaw, err := bindTrackedResults(
		pointer.EvaluationID,
		artifacts.requests,
		results,
	)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	// 未評価の current だけを現在の source と照合する。評価済みの
	// current は不変な request/result/report の replay であり、後続候補の
	// source への変更に結び付けない。
	if currentResult == nil {
		if err := validateExternalReferences(
			ctx,
			current.document,
			artifacts,
			referenceValidator,
		); err != nil {
			return CurrentEvaluation{}, err
		}
	}
	reports, err := validateTrackedReportBindings(repository, root, artifacts.requests, results, layout)
	if err != nil {
		return CurrentEvaluation{}, err
	}
	if err := CheckEvaluationPreflight(current.document, history); err != nil {
		return CurrentEvaluation{}, err
	}
	return CurrentEvaluation{
		Prepared:         prepareCurrent(pointer, current.document, artifacts),
		RequestRaw:       bytes.Clone(current.raw),
		History:          append([]ConsumedEvaluation(nil), history...),
		CurrentResult:    currentResult,
		CurrentResultRaw: bytes.Clone(currentResultRaw),
		CurrentReportRaw: bytes.Clone(reports[pointer.EvaluationID]),
	}, nil
}

func loadTrackedResults(
	ctx context.Context,
	root *legalqueryartifact.Repository,
	schema SchemaV2,
	layout preparationRootLayout,
) (map[string]loadedArtifact[EvaluationResult], error) {
	if !layout.resultsPresent {
		return map[string]loadedArtifact[EvaluationResult]{}, nil
	}
	return loadArtifactDirectory(
		ctx,
		root,
		"results",
		schema,
		maximumResultFiles,
		maximumResultTotalBytes,
		maximumResultBytes,
		evaluationIDPattern,
		DecodeEvaluationResult,
		func(value EvaluationResult) string { return value.EvaluationID },
	)
}

func bindTrackedResults(
	currentID string,
	requests map[string]loadedArtifact[EvaluationRequest],
	results map[string]loadedArtifact[EvaluationResult],
) ([]ConsumedEvaluation, *EvaluationResult, []byte, error) {
	history := make([]ConsumedEvaluation, 0, len(results))
	var currentResult *EvaluationResult
	var currentResultRaw []byte
	for _, evaluationID := range sortedKeys(results) {
		result := results[evaluationID]
		request, exists := requests[evaluationID]
		if !exists || result.document.RequestSHA256 != request.digest {
			return nil, nil, nil, fmt.Errorf("tracked result の request binding が一致しません")
		}
		consumed := ConsumedEvaluation{Request: request.document, Result: result.document}
		if err := validateConsumedEvaluation(consumed); err != nil {
			return nil, nil, nil, err
		}
		history = append(history, consumed)
		if evaluationID == currentID {
			value := result.document
			currentResult = &value
			currentResultRaw = bytes.Clone(result.raw)
		}
	}
	return history, currentResult, currentResultRaw, nil
}
