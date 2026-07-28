package legalquery

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type executorTestPayloads struct {
	lawSearch        lawsearch.Page
	lawContent       lawcontentsearch.Page
	lawDocument      model.SourcedResource[model.LawDocumentRepresentation]
	lawArticle       model.SourcedResource[model.LawArticleFragment]
	lawUpdates       lawupdatelist.Page
	judicialSearch   judicialdecisionsearch.Page
	judicialDecision model.SourcedResource[model.JudicialDecisionDetails]
}

func newExecutorTestPayloads(t *testing.T) executorTestPayloads {
	t.Helper()
	fixtures := newAttemptPayloadFixtures(t)
	lawPage := mustExecutorTestSourcePage(t, 1)
	updatePage := mustExecutorTestSourcePage(t, 1)
	judicialPage := mustExecutorTestSourcePage(t, 1)
	lawSearchPage, err := lawsearch.NewPage(lawsearch.PageValues{
		Items: []model.SourcedResource[model.LawSummary]{fixtures.lawSummary},
		Page:  lawPage,
	})
	if err != nil {
		t.Fatalf("試験用 law search page を作成できません: %v", err)
	}
	lawContentPage, err := lawcontentsearch.NewPage(
		lawcontentsearch.PageValues{
			Items: []model.SourcedResource[model.LawContentMatch]{
				fixtures.lawContent,
			},
			Page: lawPage,
		},
	)
	if err != nil {
		t.Fatalf("試験用 law content page を作成できません: %v", err)
	}
	lawUpdatesPage, err := lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: []model.SourcedResource[model.LawUpdate]{fixtures.lawUpdate},
		Page:  updatePage,
		Date:  mustDate(t, "2026-07-27"),
	})
	if err != nil {
		t.Fatalf("試験用 law updates page を作成できません: %v", err)
	}
	judicialSearchPage, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{
				fixtures.judicialSummary,
			},
			Page: judicialPage,
		},
	)
	if err != nil {
		t.Fatalf("試験用 judicial search page を作成できません: %v", err)
	}
	return executorTestPayloads{
		lawSearch:        lawSearchPage,
		lawContent:       lawContentPage,
		lawDocument:      fixtures.lawDocument,
		lawArticle:       fixtures.lawArticle,
		lawUpdates:       lawUpdatesPage,
		judicialSearch:   judicialSearchPage,
		judicialDecision: fixtures.judicialDecision,
	}
}

func mustExecutorTestSourcePage(
	t *testing.T,
	count int,
) model.SourcePage {
	t.Helper()
	total := count
	page, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &total,
		TotalRelation: model.TotalRelationExact,
	})
	if err != nil {
		t.Fatalf("試験用 SourcePage を作成できません: %v", err)
	}
	return page
}

func newExecutorTestFacades(
	t *testing.T,
) (
	*executorTestRecorder,
	*executorCoreFacadeFake,
	*executorJudicialFacadeFake,
) {
	t.Helper()
	recorder := newExecutorTestRecorder()
	payloads := newExecutorTestPayloads(t)
	return recorder,
		&executorCoreFacadeFake{recorder: recorder, payloads: payloads},
		&executorJudicialFacadeFake{recorder: recorder, payloads: payloads}
}

func mustExecutorStep(
	t *testing.T,
	name string,
	stepID string,
) LegalQueryCandidateStep {
	t.Helper()
	values := validStepFixtures(t)[name].values
	values.StepID = stepID
	step, err := NewLegalQueryCandidateStep(values)
	if err != nil {
		t.Fatalf("試験用 executor step を作成できません: %v", err)
	}
	return step
}

func mustExecutorCandidate(
	t *testing.T,
	candidateID string,
	score int,
	steps ...LegalQueryCandidateStep,
) LegalQueryCandidate {
	t.Helper()
	requiredPacks := []string{}
	for _, step := range steps {
		if isJudicialInputKind(step.InputKind()) {
			requiredPacks = []string{"judicial-cases"}
			break
		}
	}
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    candidateID,
		SemanticScore:  score,
		Confidence:     ConfidenceHigh,
		EvidenceCodes:  []EvidenceCode{EvidenceOfficialAlias},
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  requiredPacks,
		Steps:          steps,
	})
	if err != nil {
		t.Fatalf("試験用 executor candidate を作成できません: %v", err)
	}
	return candidate
}

func mustExecutorSinglePlan(
	t *testing.T,
	steps ...LegalQueryCandidateStep,
) LegalQueryPlan {
	t.Helper()
	candidate := mustExecutorCandidate(t, "candidate-1", 600, steps...)
	return mustExecutorPlan(
		t,
		PlanDecisionSingle,
		[]LegalQueryCandidate{candidate},
		10,
	)
}

func mustExecutorHedgedPlan(
	t *testing.T,
	first []LegalQueryCandidateStep,
	second []LegalQueryCandidateStep,
) LegalQueryPlan {
	t.Helper()
	return mustExecutorPlan(
		t,
		PlanDecisionHedged,
		[]LegalQueryCandidate{
			mustExecutorCandidate(t, "candidate-1", 600, first...),
			mustExecutorCandidate(t, "candidate-2", 590, second...),
		},
		10,
	)
}

func mustExecutorPlan(
	t *testing.T,
	decision PlanDecision,
	candidates []LegalQueryCandidate,
	limit int,
) LegalQueryPlan {
	t.Helper()
	selections := make([]LegalQueryPlanSelection, 0, len(candidates))
	for _, candidate := range candidates {
		selection, err := NewLegalQueryPlanSelection(
			LegalQueryPlanSelectionValues{
				CandidateID:   candidate.CandidateID(),
				Availability:  SelectionAvailabilityAvailable,
				RequiredPacks: candidate.RequiredPacks(),
			},
		)
		if err != nil {
			t.Fatalf("試験用 executor selection を作成できません: %v", err)
		}
		selections = append(selections, selection)
	}
	reason := ReasonCodeSingleClearCandidate
	if decision == PlanDecisionHedged {
		reason = ReasonCodeHedgedCloseCandidates
	}
	plan, err := NewLegalQueryPlan(LegalQueryPlanValues{
		ProfileVersion:   "executor-test-v1",
		Decision:         decision,
		RankedCandidates: candidates,
		Selected:         selections,
		ReasonCodes:      []ReasonCode{reason},
		LimitPerAttempt:  limit,
	})
	if err != nil {
		t.Fatalf("試験用 executor plan を作成できません: %v", err)
	}
	return plan
}

func isJudicialInputKind(kind LogicalInputKind) bool {
	return kind == InputKindJudicialDecisionSearch ||
		kind == InputKindJudicialDecisionRead
}
