package legalquerycandidateeval

import (
	"bytes"
	"errors"
	"fmt"
	"slices"
)

type CurrentReadinessState string

const (
	CurrentReadinessReady CurrentReadinessState = "ready"
	CurrentReadinessStale CurrentReadinessState = "stale"
)

type StaleReason string

const (
	StaleReasonCandidateContentDrift   StaleReason = "candidate_content_drift"
	StaleReasonReviewSOTLifecycleDrift StaleReason = "review_sot_lifecycle_drift"
	StaleReasonReviewSOTDigestDrift    StaleReason = "review_sot_digest_drift"
	StaleReasonCurrentEvaluatorDrift   StaleReason = "current_evaluator_drift"
)

var staleReasonOrder = [...]StaleReason{
	StaleReasonCandidateContentDrift,
	StaleReasonReviewSOTLifecycleDrift,
	StaleReasonReviewSOTDigestDrift,
	StaleReasonCurrentEvaluatorDrift,
}

// CurrentEvaluationInspection は、完全性を検証した current と導出済み readiness を保持する。
type CurrentEvaluationInspection struct {
	evaluation CurrentEvaluation
	state      CurrentReadinessState
	reasons    []StaleReason
}

// Evaluation は、完全性を検証した current evaluation を値として返す。
func (i CurrentEvaluationInspection) Evaluation() CurrentEvaluation {
	return cloneCurrentEvaluation(i.evaluation)
}

// ReadinessState は、同じ snapshot から導出した readiness を返す。
func (i CurrentEvaluationInspection) ReadinessState() CurrentReadinessState {
	return i.state
}

// StaleReasons は、閉じた固定順の stale 理由を複製して返す。
func (i CurrentEvaluationInspection) StaleReasons() []StaleReason {
	return append([]StaleReason(nil), i.reasons...)
}

func readyInspection(evaluation CurrentEvaluation) CurrentEvaluationInspection {
	return CurrentEvaluationInspection{
		evaluation: cloneCurrentEvaluation(evaluation),
		state:      CurrentReadinessReady,
	}
}

func staleInspection(
	evaluation CurrentEvaluation,
	reasons []StaleReason,
) (CurrentEvaluationInspection, error) {
	normalized, err := normalizeStaleReasons(reasons)
	if err != nil {
		return CurrentEvaluationInspection{}, err
	}
	if len(normalized) == 0 {
		return CurrentEvaluationInspection{}, fmt.Errorf("候補評価 stale reason がありません")
	}
	return CurrentEvaluationInspection{
		evaluation: cloneCurrentEvaluation(evaluation),
		state:      CurrentReadinessStale,
		reasons:    normalized,
	}, nil
}

func cloneCurrentEvaluation(source CurrentEvaluation) CurrentEvaluation {
	cloned := source
	cloned.Prepared = clonePreparedCurrent(source.Prepared)
	cloned.RequestRaw = bytes.Clone(source.RequestRaw)
	cloned.History = make([]ConsumedEvaluation, len(source.History))
	for index, consumed := range source.History {
		cloned.History[index] = ConsumedEvaluation{
			Request: cloneEvaluationRequest(consumed.Request),
			Result:  consumed.Result,
		}
	}
	if source.CurrentResult != nil {
		currentResult := *source.CurrentResult
		cloned.CurrentResult = &currentResult
	}
	cloned.CurrentResultRaw = bytes.Clone(source.CurrentResultRaw)
	cloned.CurrentReportRaw = bytes.Clone(source.CurrentReportRaw)
	return cloned
}

func clonePreparedCurrent(source PreparedCurrent) PreparedCurrent {
	cloned := source
	cloned.Request = cloneEvaluationRequest(source.Request)
	cloned.CandidateContent = cloneCandidateContentManifest(source.CandidateContent)
	cloned.ReviewAttestations = make([]ReviewAttestation, len(source.ReviewAttestations))
	for index, attestation := range source.ReviewAttestations {
		cloned.ReviewAttestations[index] = cloneReviewAttestation(attestation)
	}
	return cloned
}

func cloneEvaluationRequest(source EvaluationRequest) EvaluationRequest {
	cloned := source
	cloned.HoldoutLeakageGroupDigests = slices.Clone(source.HoldoutLeakageGroupDigests)
	cloned.RequiredReviewSOTs = slices.Clone(source.RequiredReviewSOTs)
	cloned.ReviewAttestations = slices.Clone(source.ReviewAttestations)
	return cloned
}

func cloneCandidateContentManifest(source CandidateContentManifest) CandidateContentManifest {
	cloned := source
	cloned.ProfileArtifacts = slices.Clone(source.ProfileArtifacts)
	cloned.LexiconArtifacts = make([]LexiconArtifact, len(source.LexiconArtifacts))
	for index, artifact := range source.LexiconArtifacts {
		cloned.LexiconArtifacts[index] = artifact
		cloned.LexiconArtifacts[index].Files = slices.Clone(artifact.Files)
	}
	cloned.Composition.Components = slices.Clone(source.Composition.Components)
	cloned.SemanticSourceSet.GoDebugSettings = slices.Clone(source.SemanticSourceSet.GoDebugSettings)
	cloned.SemanticSourceSet.BuildTags = slices.Clone(source.SemanticSourceSet.BuildTags)
	cloned.SemanticSourceSet.Files = slices.Clone(source.SemanticSourceSet.Files)
	cloned.SemanticSourceSet.ModuleDependencies = slices.Clone(
		source.SemanticSourceSet.ModuleDependencies,
	)
	return cloned
}

func cloneReviewAttestation(source ReviewAttestation) ReviewAttestation {
	cloned := source
	cloned.ReviewedSOTs = slices.Clone(source.ReviewedSOTs)
	cloned.CriterionScores = slices.Clone(source.CriterionScores)
	return cloned
}

type CurrentStaleError struct {
	reasons []StaleReason
}

func (e *CurrentStaleError) Error() string {
	return "候補評価 current は stale です"
}

func (e *CurrentStaleError) ReadinessState() CurrentReadinessState {
	return CurrentReadinessStale
}

func (e *CurrentStaleError) Reasons() []StaleReason {
	if e == nil {
		return nil
	}
	return append([]StaleReason(nil), e.reasons...)
}

func NewCurrentStaleError(reasons ...StaleReason) error {
	normalized, err := normalizeStaleReasons(reasons)
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return nil
	}
	return &CurrentStaleError{reasons: normalized}
}

func AppendStaleReasons(
	current []StaleReason,
	additional ...StaleReason,
) ([]StaleReason, error) {
	merged := append(append([]StaleReason(nil), current...), additional...)
	return normalizeStaleReasons(merged)
}

func StaleReasonsFromError(err error) ([]StaleReason, bool) {
	var stale *CurrentStaleError
	if !errors.As(err, &stale) {
		return nil, false
	}
	return stale.Reasons(), true
}

func IsCurrentStale(err error) bool {
	_, ok := StaleReasonsFromError(err)
	return ok
}

func normalizeStaleReasons(reasons []StaleReason) ([]StaleReason, error) {
	seen := make(map[StaleReason]struct{}, len(reasons))
	for _, candidate := range reasons {
		if !slices.Contains(staleReasonOrder[:], candidate) {
			return nil, fmt.Errorf("候補評価 stale reason が閉じた集合にありません")
		}
	}
	normalized := make([]StaleReason, 0, len(staleReasonOrder))
	for _, reason := range staleReasonOrder {
		for _, candidate := range reasons {
			if candidate != reason {
				continue
			}
			if _, exists := seen[reason]; exists {
				break
			}
			seen[reason] = struct{}{}
			normalized = append(normalized, reason)
			break
		}
	}
	return normalized, nil
}

func EqualStaleReasons(left, right []StaleReason) bool {
	normalizedLeft, leftErr := normalizeStaleReasons(left)
	normalizedRight, rightErr := normalizeStaleReasons(right)
	return leftErr == nil && rightErr == nil && slices.Equal(normalizedLeft, normalizedRight)
}
