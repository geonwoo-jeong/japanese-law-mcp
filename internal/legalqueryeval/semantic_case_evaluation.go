package legalqueryeval

import (
	"fmt"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

// MeaningEvaluation は、一件の expected meaning に対する一致結果を保持する。
type MeaningEvaluation struct {
	meaningID            string
	matchedCandidateRank int
	signatureMatched     bool
	evidence             comparisonAssertion
	concept              comparisonAssertion
}

// MeaningID は、fixture 内の評価用識別子を返す。
func (e MeaningEvaluation) MeaningID() string {
	return e.meaningID
}

// MatchedCandidateRank は、一致した候補の一始まり順位を返す。一致しない場合は 0 を返す。
func (e MeaningEvaluation) MatchedCandidateRank() int {
	return e.matchedCandidateRank
}

// SignatureMatched は、意味署名の一致有無を返す。
func (e MeaningEvaluation) SignatureMatched() bool {
	return e.signatureMatched
}

// EvidenceAssertion は、根拠 assertion の一致結果と適用可否を返す。
func (e MeaningEvaluation) EvidenceAssertion() (matched bool, applicable bool) {
	return e.evidence.matched, e.evidence.applicable
}

// ConceptAssertion は、法概念 assertion の一致結果と適用可否を返す。
func (e MeaningEvaluation) ConceptAssertion() (matched bool, applicable bool) {
	return e.concept.matched, e.concept.applicable
}

// SemanticCaseEvaluation は、semantic fixture 一件に対する評価結果を保持する。
type SemanticCaseEvaluation struct {
	caseID                    string
	categoryIDs               []string
	coverageIDs               []string
	expectedKind              legalquerycorpus.SemanticExpectedKind
	requestErrorMatched       bool
	planDecisionMatched       bool
	reasonCodesMatched        bool
	selectedMeaningIDsMatched bool
	primaryTop1Matched        bool
	primaryTop2Matched        bool
	highConfidence            comparisonAssertion
	meanings                  []MeaningEvaluation
}

// CaseID は、評価した fixture ID を返す。
func (e SemanticCaseEvaluation) CaseID() string {
	return e.caseID
}

// CategoryIDs は、manifest 順の category ID を返す。
func (e SemanticCaseEvaluation) CategoryIDs() []string {
	return append([]string{}, e.categoryIDs...)
}

// CoverageIDs は、coverage ID の複製を返す。
func (e SemanticCaseEvaluation) CoverageIDs() []string {
	return append([]string{}, e.coverageIDs...)
}

// ExpectedKind は、semantic expectation variant を返す。
func (e SemanticCaseEvaluation) ExpectedKind() legalquerycorpus.SemanticExpectedKind {
	return e.expectedKind
}

// RequestErrorMatched は、request_error の code/field が一致したか返す。
func (e SemanticCaseEvaluation) RequestErrorMatched() bool {
	return e.requestErrorMatched
}

// PlanOutcomeMatched は、decision、reasonCodes および selection がすべて一致したか返す。
func (e SemanticCaseEvaluation) PlanOutcomeMatched() bool {
	return e.planDecisionMatched &&
		e.reasonCodesMatched &&
		e.selectedMeaningIDsMatched
}

// PrimaryTop1Matched は、主正解が順位一位か返す。
func (e SemanticCaseEvaluation) PrimaryTop1Matched() bool {
	return e.primaryTop1Matched
}

// PrimaryTop2Matched は、主正解が上位二候補に含まれるか返す。
func (e SemanticCaseEvaluation) PrimaryTop2Matched() bool {
	return e.primaryTop2Matched
}

// HighConfidencePrecision は、高確信度 precision の一致結果と適用可否を返す。
func (e SemanticCaseEvaluation) HighConfidencePrecision() (matched bool, applicable bool) {
	return e.highConfidence.matched, e.highConfidence.applicable
}

// Meanings は、expected meaning ごとの評価結果を返す。
func (e SemanticCaseEvaluation) Meanings() []MeaningEvaluation {
	return append([]MeaningEvaluation{}, e.meanings...)
}

// EvaluateSemanticPlanCase は、plan 期待値を実際の plan と比較する。
func EvaluateSemanticPlanCase(
	semanticCase legalquerycorpus.SemanticCase,
	actualPlan legalquery.LegalQueryPlan,
) (SemanticCaseEvaluation, error) {
	if err := semanticCase.Validate(); err != nil {
		return SemanticCaseEvaluation{}, fmt.Errorf("semantic case が有効ではありません: %w", err)
	}
	expectedPlan, ok := semanticCase.Expected().(legalquerycorpus.ExpectedPlan)
	if !ok {
		return SemanticCaseEvaluation{}, fmt.Errorf("semantic case の expected.kind は plan でなければなりません")
	}
	if err := actualPlan.Validate(); err != nil {
		return SemanticCaseEvaluation{}, fmt.Errorf("actual plan が有効ではありません: %w", err)
	}

	meanings, err := evaluateExpectedMeanings(expectedPlan.Meanings(), actualPlan.RankedCandidates())
	if err != nil {
		return SemanticCaseEvaluation{}, err
	}
	selectedMatched, err := compareSelectedMeaningIDs(expectedPlan, actualPlan)
	if err != nil {
		return SemanticCaseEvaluation{}, err
	}

	top1Matched, top2Matched, highConfidence := evaluatePrimaryRanking(meanings, actualPlan)
	return SemanticCaseEvaluation{
		caseID:                    semanticCase.CaseID(),
		categoryIDs:               semanticCase.CategoryIDs(),
		coverageIDs:               semanticCase.CoverageIDs(),
		expectedKind:              legalquerycorpus.SemanticExpectedKindPlan,
		planDecisionMatched:       actualPlan.Decision() == expectedPlan.Decision(),
		reasonCodesMatched:        slices.Equal(actualPlan.ReasonCodes(), expectedPlan.ReasonCodes()),
		selectedMeaningIDsMatched: selectedMatched,
		primaryTop1Matched:        top1Matched,
		primaryTop2Matched:        top2Matched,
		highConfidence:            highConfidence,
		meanings:                  meanings,
	}, nil
}

// EvaluateSemanticRequestErrorCase は、request_error 期待値を実際の入力エラーと比較する。
func EvaluateSemanticRequestErrorCase(
	semanticCase legalquerycorpus.SemanticCase,
	argumentError legalquery.ArgumentError,
) (SemanticCaseEvaluation, error) {
	if err := semanticCase.Validate(); err != nil {
		return SemanticCaseEvaluation{}, fmt.Errorf("semantic case が有効ではありません: %w", err)
	}
	expectedError, ok := semanticCase.Expected().(legalquerycorpus.ExpectedRequestError)
	if !ok {
		return SemanticCaseEvaluation{}, fmt.Errorf("semantic case の expected.kind は request_error でなければなりません")
	}
	if err := argumentError.Validate(); err != nil {
		return SemanticCaseEvaluation{}, fmt.Errorf("actual request error が有効ではありません: %w", err)
	}

	return SemanticCaseEvaluation{
		caseID:       semanticCase.CaseID(),
		categoryIDs:  semanticCase.CategoryIDs(),
		coverageIDs:  semanticCase.CoverageIDs(),
		expectedKind: legalquerycorpus.SemanticExpectedKindRequestError,
		requestErrorMatched: argumentError.Code() == expectedError.ErrorCode() &&
			argumentError.Field() == string(expectedError.Field()),
	}, nil
}

func evaluateExpectedMeanings(
	expected []legalquerycorpus.ExpectedMeaning,
	ranked []legalquery.LegalQueryCandidate,
) ([]MeaningEvaluation, error) {
	values := make([]MeaningEvaluation, 0, len(expected))
	for _, expectedMeaning := range expected {
		value, err := evaluateExpectedMeaning(expectedMeaning, ranked)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
	}
	return values, nil
}

func evaluateExpectedMeaning(
	expected legalquerycorpus.ExpectedMeaning,
	ranked []legalquery.LegalQueryCandidate,
) (MeaningEvaluation, error) {
	result := MeaningEvaluation{meaningID: expected.MeaningID()}
	for index, candidate := range ranked {
		comparison, err := CompareMeaning(expected, candidate)
		if err != nil {
			return MeaningEvaluation{}, fmt.Errorf("meaning %q の比較に失敗しました: %w", expected.MeaningID(), err)
		}
		if !comparison.SignatureMatched() {
			continue
		}
		evidenceMatched, evidenceApplicable := comparison.EvidenceAssertion()
		conceptMatched, conceptApplicable := comparison.ConceptAssertion()
		result.matchedCandidateRank = index + 1
		result.signatureMatched = true
		result.evidence = comparisonAssertion{
			matched:    evidenceMatched,
			applicable: evidenceApplicable,
		}
		result.concept = comparisonAssertion{
			matched:    conceptMatched,
			applicable: conceptApplicable,
		}
		return result, nil
	}
	return result, nil
}

func compareSelectedMeaningIDs(
	expected legalquerycorpus.ExpectedPlan,
	actual legalquery.LegalQueryPlan,
) (bool, error) {
	candidates := make(map[string]legalquery.LegalQueryCandidate, len(actual.RankedCandidates()))
	for _, candidate := range actual.RankedCandidates() {
		candidates[candidate.CandidateID()] = candidate
	}

	actualMeaningIDs := make([]string, 0, len(actual.Selected()))
	for _, selection := range actual.Selected() {
		candidate, exists := candidates[selection.CandidateID()]
		if !exists {
			return false, fmt.Errorf("selected candidate %q が rankedCandidates に存在しません", selection.CandidateID())
		}
		matchedMeaningID, matched, err := matchedExpectedMeaningID(expected.Meanings(), candidate)
		if err != nil {
			return false, err
		}
		if !matched {
			return false, nil
		}
		actualMeaningIDs = append(actualMeaningIDs, matchedMeaningID)
	}
	return slices.Equal(actualMeaningIDs, expected.SelectedMeaningIDs()), nil
}

func matchedExpectedMeaningID(
	expected []legalquerycorpus.ExpectedMeaning,
	candidate legalquery.LegalQueryCandidate,
) (string, bool, error) {
	matchedIDs := make([]string, 0, 1)
	for _, meaning := range expected {
		comparison, err := CompareMeaning(meaning, candidate)
		if err != nil {
			return "", false, fmt.Errorf("selection の意味照合に失敗しました: %w", err)
		}
		if comparison.SignatureMatched() {
			matchedIDs = append(matchedIDs, meaning.MeaningID())
		}
	}
	switch len(matchedIDs) {
	case 0:
		return "", false, nil
	case 1:
		return matchedIDs[0], true, nil
	default:
		return "", false, fmt.Errorf("selection が複数の expected meaning へ一致しました")
	}
}

func evaluatePrimaryRanking(
	meanings []MeaningEvaluation,
	actual legalquery.LegalQueryPlan,
) (top1Matched bool, top2Matched bool, highConfidence comparisonAssertion) {
	if len(meanings) == 0 {
		return false, false, comparisonAssertion{}
	}

	primaryRank := meanings[0].MatchedCandidateRank()
	top1Matched = primaryRank == 1
	top2Matched = primaryRank == 1 || primaryRank == 2

	ranked := actual.RankedCandidates()
	if len(ranked) == 0 || ranked[0].Confidence() != legalquery.ConfidenceHigh {
		return top1Matched, top2Matched, comparisonAssertion{}
	}
	return top1Matched, top2Matched, comparisonAssertion{
		matched:    top1Matched,
		applicable: true,
	}
}
