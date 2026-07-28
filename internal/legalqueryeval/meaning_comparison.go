package legalqueryeval

import (
	"fmt"
	"slices"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
)

type comparisonAssertion struct {
	matched    bool
	applicable bool
}

// MeaningComparison は、意味署名と二つの独立 assertion の比較結果を保持する。
type MeaningComparison struct {
	signatureMatched bool
	evidence         comparisonAssertion
	concept          comparisonAssertion
}

// CompareMeaning は、SOT-ENG-026 の意味署名と独立 assertion を比較する。
func CompareMeaning(
	expected legalquerycorpus.ExpectedMeaning,
	actual legalquery.LegalQueryCandidate,
) (MeaningComparison, error) {
	if err := expected.Validate(); err != nil {
		return MeaningComparison{}, fmt.Errorf("期待意味が有効ではありません: %w", err)
	}
	if err := actual.Validate(); err != nil {
		return MeaningComparison{}, fmt.Errorf("実際の意味候補が有効ではありません: %w", err)
	}

	signatureMatched, err := compareMeaningSignature(expected, actual)
	if err != nil {
		return MeaningComparison{}, err
	}
	if !signatureMatched {
		return MeaningComparison{}, nil
	}

	conceptApplicable := len(expected.ConceptIDs()) > 0
	conceptMatched := false
	if conceptApplicable {
		conceptMatched = slices.Equal(
			expected.ConceptIDs(),
			candidateConceptIDs(actual),
		)
	}

	return MeaningComparison{
		signatureMatched: true,
		evidence: comparisonAssertion{
			matched:    slices.Equal(expected.EvidenceCodes(), actual.EvidenceCodes()),
			applicable: true,
		},
		concept: comparisonAssertion{
			matched:    conceptMatched,
			applicable: conceptApplicable,
		},
	}, nil
}

// SignatureMatched は、provider 非依存の意味署名が一致したかを返す。
func (c MeaningComparison) SignatureMatched() bool {
	return c.signatureMatched
}

// EvidenceAssertion は、根拠コードの一致結果と適用可否を返す。
func (c MeaningComparison) EvidenceAssertion() (matched bool, applicable bool) {
	return c.evidence.matched, c.evidence.applicable
}

// ConceptAssertion は、法概念 ID 集合の一致結果と適用可否を返す。
func (c MeaningComparison) ConceptAssertion() (matched bool, applicable bool) {
	return c.concept.matched, c.concept.applicable
}

// AllMatched は、意味署名と適用された assertion がすべて一致したかを返す。
func (c MeaningComparison) AllMatched() bool {
	return c.signatureMatched &&
		c.evidence.applicable &&
		c.evidence.matched &&
		(!c.concept.applicable || c.concept.matched)
}

func compareMeaningSignature(
	expected legalquerycorpus.ExpectedMeaning,
	actual legalquery.LegalQueryCandidate,
) (bool, error) {
	if !slices.Equal(expected.RequiredPacks(), actual.RequiredPacks()) {
		return false, nil
	}

	expectedSteps := expected.Steps()
	actualSteps := actual.Steps()
	if len(expectedSteps) != len(actualSteps) {
		return false, nil
	}
	for index := range expectedSteps {
		matched, err := compareMeaningStep(expectedSteps[index], actualSteps[index])
		if err != nil {
			return false, fmt.Errorf("意味署名の step[%d] を比較できません: %w", index, err)
		}
		if !matched {
			return false, nil
		}
	}
	return true, nil
}

func compareMeaningStep(
	expected legalquerycorpus.ExpectedStep,
	actual legalquery.LegalQueryCandidateStep,
) (bool, error) {
	if expected.Task() != actual.Task() ||
		expected.Resource() != actual.Resource() ||
		expected.InputKind() != actual.InputKind() {
		return false, nil
	}
	return compareLogicalInput(expected.LogicalInput(), actual.LogicalInput())
}

func candidateConceptIDs(actual legalquery.LegalQueryCandidate) []string {
	sources := actual.ConceptSources()
	identifiers := make([]string, 0, len(sources))
	for _, source := range sources {
		identifiers = append(identifiers, source.ConceptID())
	}
	sort.Strings(identifiers)
	return identifiers
}
