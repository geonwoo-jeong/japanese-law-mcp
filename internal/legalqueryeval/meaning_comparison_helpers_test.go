package legalqueryeval

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type comparisonStepFixture struct {
	name                   string
	task                   legalquery.Task
	resource               legalquery.Resource
	inputKind              legalquery.LogicalInputKind
	logicalInput           legalquery.LogicalInput
	capabilityID           string
	capabilityMajorVersion int
	requiredPacks          []string
}

func comparisonStepFixtures(t *testing.T) []comparisonStepFixture {
	t.Helper()

	asOf := mustComparisonDate(t, "2025-04-01")
	lawRef := mustComparisonRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"503AC0000000037",
	)
	judicialRef := mustComparisonRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)

	return []comparisonStepFixture{
		{
			name:                   "法令検索",
			task:                   legalquery.TaskSearch,
			resource:               legalquery.ResourceLaw,
			inputKind:              legalquery.InputKindLawSearch,
			logicalInput:           mustLawSearchInput(t, "行政手続法", asOf),
			capabilityID:           "law.search",
			capabilityMajorVersion: 1,
		},
		{
			name:      "法令本文検索",
			task:      legalquery.TaskSearch,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawContentSearch,
			logicalInput: mustLawContentSearchInput(
				t,
				[]string{"永住許可"},
				[]string{"在留資格"},
				[]string{"経過措置"},
				asOf,
			),
			capabilityID:           "law.content.search",
			capabilityMajorVersion: 1,
		},
		{
			name:                   "法令読取り",
			task:                   legalquery.TaskRead,
			resource:               legalquery.ResourceLaw,
			inputKind:              legalquery.InputKindLawRead,
			logicalInput:           mustLawReadInput(t, lawRef),
			capabilityID:           "law.document.read",
			capabilityMajorVersion: 1,
		},
		{
			name:      "条文読取り",
			task:      legalquery.TaskRead,
			resource:  legalquery.ResourceLawProvision,
			inputKind: legalquery.InputKindLawArticleRead,
			logicalInput: mustLawArticleReadInput(
				t,
				"503AC0000000037",
				mustComparisonLocation(t, "22", 2),
				asOf,
			),
			capabilityID:           "law.article.read",
			capabilityMajorVersion: 1,
		},
		{
			name:                   "更新一覧",
			task:                   legalquery.TaskListUpdates,
			resource:               legalquery.ResourceLaw,
			inputKind:              legalquery.InputKindLawUpdates,
			logicalInput:           mustLawUpdatesInput(t, "2026-07-27"),
			capabilityID:           "law.update.list",
			capabilityMajorVersion: 1,
		},
		{
			name:                   "裁判例検索",
			task:                   legalquery.TaskSearch,
			resource:               legalquery.ResourceJudicialDecision,
			inputKind:              legalquery.InputKindJudicialDecisionSearch,
			logicalInput:           mustJudicialSearchInput(t, "永住許可"),
			capabilityID:           "judicial-decision.search",
			capabilityMajorVersion: 1,
			requiredPacks:          []string{"judicial-cases"},
		},
		{
			name:                   "裁判例読取り",
			task:                   legalquery.TaskRead,
			resource:               legalquery.ResourceJudicialDecision,
			inputKind:              legalquery.InputKindJudicialDecisionRead,
			logicalInput:           mustJudicialReadInput(t, judicialRef),
			capabilityID:           "judicial-decision.read",
			capabilityMajorVersion: 1,
			requiredPacks:          []string{"judicial-cases"},
		},
	}
}

func changedLogicalInputFixture(
	t *testing.T,
	base comparisonStepFixture,
) comparisonStepFixture {
	t.Helper()

	changed := base
	switch base.inputKind {
	case legalquery.InputKindLawSearch:
		changed.logicalInput = mustLawSearchInput(
			t,
			"行政事件訴訟法",
			mustComparisonDate(t, "2025-04-01"),
		)
	case legalquery.InputKindLawContentSearch:
		changed.logicalInput = mustLawContentSearchInput(
			t,
			[]string{"永住許可"},
			[]string{"退去強制"},
			[]string{"経過措置"},
			mustComparisonDate(t, "2025-04-01"),
		)
	case legalquery.InputKindLawRead:
		changed.logicalInput = mustLawReadInput(
			t,
			mustComparisonRef(
				t,
				"e-gov-law-api-v2",
				"e-gov-law-api-v2",
				"law",
				"323AC0000000131",
			),
		)
	case legalquery.InputKindLawArticleRead:
		changed.logicalInput = mustLawArticleReadInput(
			t,
			"503AC0000000037",
			mustComparisonLocation(t, "22", 3),
			mustComparisonDate(t, "2025-04-01"),
		)
	case legalquery.InputKindLawUpdates:
		changed.logicalInput = mustLawUpdatesInput(t, "2026-07-28")
	case legalquery.InputKindJudicialDecisionSearch:
		changed.logicalInput = mustJudicialSearchInput(t, "在留資格取消し")
	case legalquery.InputKindJudicialDecisionRead:
		changed.logicalInput = mustJudicialReadInput(
			t,
			mustComparisonRef(
				t,
				"courts-hanrei-html",
				"courts-hanrei",
				"judicial-decision",
				"95571/detail2",
			),
		)
	default:
		t.Fatalf("試験用 logical input の種類が定義されていません: %q", base.inputKind)
	}
	return changed
}

func mustExpectedMeaning(
	t *testing.T,
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
			MeaningID:     "expected-meaning",
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

func mustCandidate(
	t *testing.T,
	candidateID string,
	semanticScore int,
	confidence legalquery.Confidence,
	evidenceCodes []legalquery.EvidenceCode,
	conceptSources []legalquery.LegalConceptSource,
	requiredPacks []string,
	steps ...comparisonStepFixture,
) legalquery.LegalQueryCandidate {
	t.Helper()

	candidateSteps := make([]legalquery.LegalQueryCandidateStep, 0, len(steps))
	for index, step := range steps {
		candidateStep, err := legalquery.NewLegalQueryCandidateStep(
			legalquery.LegalQueryCandidateStepValues{
				StepID:                 candidateID + "-step-" + string(rune('a'+index)),
				Task:                   step.task,
				Resource:               step.resource,
				CapabilityID:           step.capabilityID,
				CapabilityMajorVersion: step.capabilityMajorVersion,
				InputKind:              step.inputKind,
				LogicalInput:           step.logicalInput,
			},
		)
		if err != nil {
			t.Fatalf("試験用 LegalQueryCandidateStep を作成できません: %v", err)
		}
		candidateSteps = append(candidateSteps, candidateStep)
	}

	candidate, err := legalquery.NewLegalQueryCandidate(
		legalquery.LegalQueryCandidateValues{
			CandidateID:    candidateID,
			SemanticScore:  semanticScore,
			Confidence:     confidence,
			EvidenceCodes:  evidenceCodes,
			ConceptSources: conceptSources,
			RequiredPacks:  requiredPacks,
			Steps:          candidateSteps,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalQueryCandidate を作成できません: %v", err)
	}
	return candidate
}

func mustLawSearchInput(
	t *testing.T,
	query string,
	asOf model.Date,
) legalquery.LawSearchIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: query, AsOf: &asOf},
	)
	if err != nil {
		t.Fatalf("試験用 LawSearchIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustLawContentSearchInput(
	t *testing.T,
	allTerms []string,
	anyTerms []string,
	excludeTerms []string,
	asOf model.Date,
) legalquery.LawContentSearchIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     allTerms,
			AnyTerms:     anyTerms,
			ExcludeTerms: excludeTerms,
			AsOf:         &asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawContentSearchIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustLawReadInput(
	t *testing.T,
	ref model.SourceResourceRef,
) legalquery.LawReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{Ref: &ref},
	)
	if err != nil {
		t.Fatalf("試験用 LawReadIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustLawReadInputByID(
	t *testing.T,
	lawID string,
	revisionID string,
	asOf *model.Date,
) legalquery.LawReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{
			LawID:      lawID,
			RevisionID: revisionID,
			AsOf:       asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawReadIntentV1 を ID 形で作成できません: %v", err)
	}
	return input
}

func mustLawArticleReadInput(
	t *testing.T,
	lawID string,
	location model.LawArticleLocation,
	asOf model.Date,
) legalquery.LawArticleReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    lawID,
			Location: location,
			AsOf:     &asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleReadIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustLawArticleReadInputByRef(
	t *testing.T,
	ref model.SourceResourceRef,
	location model.LawArticleLocation,
	asOf *model.Date,
) legalquery.LawArticleReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			Ref:      &ref,
			Location: location,
			AsOf:     asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleReadIntentV1 を ref 形で作成できません: %v", err)
	}
	return input
}

func mustLawUpdatesInput(
	t *testing.T,
	value string,
) legalquery.LawUpdateListIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawUpdateListIntentV1(
		legalquery.LawUpdateListIntentV1Values{
			Date: mustComparisonDate(t, value),
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawUpdateListIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustJudicialSearchInput(
	t *testing.T,
	query string,
) legalquery.JudicialDecisionSearchIntentV1 {
	t.Helper()
	input, err := legalquery.NewJudicialDecisionSearchIntentV1(
		legalquery.JudicialDecisionSearchIntentV1Values{Query: query},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionSearchIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustJudicialReadInput(
	t *testing.T,
	ref model.SourceResourceRef,
) legalquery.JudicialDecisionReadIntentV1 {
	t.Helper()
	input, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: ref},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionReadIntentV1 を作成できません: %v", err)
	}
	return input
}

func mustComparisonConceptSource(
	t *testing.T,
	conceptID string,
	title string,
	url string,
	confirmedOn string,
) legalquery.LegalConceptSource {
	t.Helper()
	source, err := legalquery.NewLegalConceptSource(
		legalquery.LegalConceptSourceValues{
			ConceptID:   conceptID,
			Title:       title,
			URL:         url,
			ConfirmedOn: mustComparisonDate(t, confirmedOn),
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalConceptSource を作成できません: %v", err)
	}
	return source
}

func mustComparisonDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用 Date を作成できません: %v", err)
	}
	return date
}

func mustComparisonLocation(
	t *testing.T,
	article string,
	paragraph int,
) model.LawArticleLocation {
	t.Helper()
	location, err := model.NewLawArticleLocation(
		model.LawArticleLocationValues{
			Provision:       model.LawArticleProvisionMain,
			ArticleNumber:   article,
			ParagraphNumber: &paragraph,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleLocation を作成できません: %v", err)
	}
	return location
}

func mustComparisonLocationWithoutParagraph(
	t *testing.T,
	provision model.LawArticleProvision,
	article string,
) model.LawArticleLocation {
	t.Helper()
	location, err := model.NewLawArticleLocation(
		model.LawArticleLocationValues{
			Provision:     provision,
			ArticleNumber: article,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawArticleLocation を項なしで作成できません: %v", err)
	}
	return location
}

func mustComparisonRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}

func mustComparisonRefWithVersion(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を版付きで作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を版付きで作成できません: %v", err)
	}
	return ref
}
