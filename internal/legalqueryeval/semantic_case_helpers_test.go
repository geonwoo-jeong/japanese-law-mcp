package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func mustExpectedMeaningWithID(
	t *testing.T,
	meaningID string,
	evidenceCodes []legalquery.EvidenceCode,
	conceptIDs []string,
	requiredPacks []string,
	steps ...comparisonStepFixture,
) legalquerycorpus.ExpectedMeaning {
	t.Helper()

	expectedSteps := make([]legalquerycorpus.ExpectedStep, 0, len(steps))
	for _, step := range steps {
		expectedStep, err := legalquerycorpus.NewExpectedStep(
			legalquerycorpus.ExpectedStepValues{
				Task:         step.task,
				Resource:     step.resource,
				InputKind:    step.inputKind,
				LogicalInput: step.logicalInput,
			},
		)
		if err != nil {
			t.Fatalf("試験用 ExpectedStep を作成できません: %v", err)
		}
		expectedSteps = append(expectedSteps, expectedStep)
	}

	meaning, err := legalquerycorpus.NewExpectedMeaning(
		legalquerycorpus.ExpectedMeaningValues{
			MeaningID:     meaningID,
			EvidenceCodes: evidenceCodes,
			ConceptIDs:    conceptIDs,
			RequiredPacks: requiredPacks,
			Steps:         expectedSteps,
		},
	)
	if err != nil {
		t.Fatalf("試験用 ExpectedMeaning を作成できません: %v", err)
	}
	return meaning
}

func mustSemanticPlanCase(
	t *testing.T,
	caseID string,
	coverageIDs []string,
	query string,
	enabledPacks []string,
	expected legalquerycorpus.ExpectedPlan,
) legalquerycorpus.SemanticCase {
	t.Helper()

	request, err := legalquerycorpus.NewRequest(legalquerycorpus.RequestValues{
		Query: query,
	})
	if err != nil {
		t.Fatalf("試験用 corpus request を作成できません: %v", err)
	}
	semanticCase, err := legalquerycorpus.NewSemanticCase(
		legalquerycorpus.SemanticCaseValues{
			ArtifactKind:   legalquerycorpus.ArtifactKindSemanticCase,
			SchemaVersion:  1,
			CaseID:         caseID,
			LeakageGroupID: caseID + "-group",
			CoverageIDs:    coverageIDs,
			EnabledPacks:   enabledPacks,
			Request:        request,
			Expected:       expected,
		},
	)
	if err != nil {
		t.Fatalf("試験用 SemanticCase を作成できません: %v", err)
	}
	return semanticCase
}

func mustSemanticRequestErrorCase(
	t *testing.T,
	caseID string,
	coverageIDs []string,
	query string,
	field legalquerycorpus.RequestErrorField,
) legalquerycorpus.SemanticCase {
	t.Helper()

	request, err := legalquerycorpus.NewRequest(legalquerycorpus.RequestValues{
		Query: query,
	})
	if err != nil {
		t.Fatalf("試験用 corpus request を作成できません: %v", err)
	}
	expected, err := legalquerycorpus.NewExpectedRequestError(
		legalquerycorpus.ExpectedRequestErrorValues{
			ErrorCode: model.ErrorCodeInvalidArgument,
			Field:     field,
		},
	)
	if err != nil {
		t.Fatalf("試験用 ExpectedRequestError を作成できません: %v", err)
	}
	semanticCase, err := legalquerycorpus.NewSemanticCase(
		legalquerycorpus.SemanticCaseValues{
			ArtifactKind:   legalquerycorpus.ArtifactKindSemanticCase,
			SchemaVersion:  1,
			CaseID:         caseID,
			LeakageGroupID: caseID + "-group",
			CoverageIDs:    coverageIDs,
			EnabledPacks:   nil,
			Request:        request,
			Expected:       expected,
		},
	)
	if err != nil {
		t.Fatalf("試験用 SemanticCase を作成できません: %v", err)
	}
	return semanticCase
}

func mustExpectedPlan(
	t *testing.T,
	decision legalquery.PlanDecision,
	reasonCodes []legalquery.ReasonCode,
	meanings []legalquerycorpus.ExpectedMeaning,
	selectedMeaningIDs []string,
) legalquerycorpus.ExpectedPlan {
	t.Helper()

	plan, err := legalquerycorpus.NewExpectedPlan(
		legalquerycorpus.ExpectedPlanValues{
			Decision:           decision,
			ReasonCodes:        reasonCodes,
			Meanings:           meanings,
			SelectedMeaningIDs: selectedMeaningIDs,
		},
	)
	if err != nil {
		t.Fatalf("試験用 ExpectedPlan を作成できません: %v", err)
	}
	return plan
}

func mustPlan(
	t *testing.T,
	profileVersion string,
	decision legalquery.PlanDecision,
	ranked []legalquery.LegalQueryCandidate,
	selectedIDs []string,
	reasonCodes []legalquery.ReasonCode,
	limitPerAttempt int,
) legalquery.LegalQueryPlan {
	t.Helper()

	selected := make([]legalquery.LegalQueryPlanSelection, 0, len(selectedIDs))
	candidates := make(map[string]legalquery.LegalQueryCandidate, len(ranked))
	for _, candidate := range ranked {
		candidates[candidate.CandidateID()] = candidate
	}
	for _, candidateID := range selectedIDs {
		candidate, exists := candidates[candidateID]
		if !exists {
			t.Fatalf("試験用 selection が未知の candidateId を参照しました: %s", candidateID)
		}
		availability := legalquery.SelectionAvailabilityAvailable
		if len(candidate.RequiredPacks()) > 0 {
			availability = legalquery.SelectionAvailabilityPackDisabled
		}
		selection, err := legalquery.NewLegalQueryPlanSelection(
			legalquery.LegalQueryPlanSelectionValues{
				CandidateID:   candidateID,
				Availability:  availability,
				RequiredPacks: candidate.RequiredPacks(),
			},
		)
		if err != nil {
			t.Fatalf("試験用 selection を作成できません: %v", err)
		}
		selected = append(selected, selection)
	}

	plan, err := legalquery.NewLegalQueryPlan(legalquery.LegalQueryPlanValues{
		ProfileVersion:   profileVersion,
		Decision:         decision,
		RankedCandidates: ranked,
		Selected:         selected,
		ReasonCodes:      reasonCodes,
		LimitPerAttempt:  limitPerAttempt,
	})
	if err != nil {
		t.Fatalf("試験用 LegalQueryPlan を作成できません: %v", err)
	}
	return plan
}

func mustArgumentError(
	t *testing.T,
	field string,
	reason string,
) legalquery.ArgumentError {
	t.Helper()

	argumentError, err := legalquery.NewArgumentError(field, reason)
	if err != nil {
		t.Fatalf("試験用 ArgumentError を作成できません: %v", err)
	}
	return argumentError
}
