package mcp

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestQueryLegalInformationOutputSchemaAcceptsModelSerialization(
	t *testing.T,
) {
	t.Parallel()

	fixture := newQuerySchemaModelFixture(t)
	completed, err := legalquery.NewLegalQueryCompletedResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{fixture.nonemptyAttempt},
	)
	if err != nil {
		t.Fatalf("completed result を作成できません: %v", err)
	}
	empty, err := legalquery.NewLegalQueryEmptyResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{fixture.emptyAttempt},
	)
	if err != nil {
		t.Fatalf("empty result を作成できません: %v", err)
	}
	partial, err := legalquery.NewLegalQueryPartialResult(
		fixture.availablePlan,
		[]legalquery.LegalQueryAttempt{
			fixture.nonemptyAttempt,
			fixture.failedAttempt,
		},
	)
	if err != nil {
		t.Fatalf("partial result を作成できません: %v", err)
	}
	clarification, err := legalquery.NewLegalQueryNeedsClarificationResult(
		fixture.clarificationPlan,
		[]legalquery.LegalQueryQuestion{
			legalquery.LegalQueryQuestionTask,
		},
	)
	if err != nil {
		t.Fatalf("needs clarification result を作成できません: %v", err)
	}
	unavailable, err := legalquery.NewLegalQueryCapabilityUnavailableResult(
		fixture.unavailablePlan,
	)
	if err != nil {
		t.Fatalf("capability unavailable result を作成できません: %v", err)
	}
	unsupported, err := legalquery.NewLegalQueryUnsupportedResult(
		fixture.unsupportedPlan,
	)
	if err != nil {
		t.Fatalf("unsupported result を作成できません: %v", err)
	}

	schema := newQueryLegalInformationOutputSchema()
	for name, result := range map[string]legalquery.LegalQueryResult{
		"completed":              completed,
		"empty":                  empty,
		"partial":                partial,
		"needs_clarification":    clarification,
		"capability_unavailable": unavailable,
		"unsupported":            unsupported,
	} {
		t.Run(name, func(t *testing.T) {
			encoded, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				t.Fatalf("result を JSON に変換できません: %v", marshalErr)
			}
			var instance any
			if unmarshalErr := json.Unmarshal(encoded, &instance); unmarshalErr != nil {
				t.Fatalf("result JSON を復元できません: %v", unmarshalErr)
			}
			assertQuerySchemaAccepts(t, schema, instance)
		})
	}
}

func TestQueryLegalInformationInputSchemaExtensionsMatchRequestBoundary(
	t *testing.T,
) {
	t.Parallel()

	inputSchema := newQueryLegalInformationInputSchema()
	if got := inputSchema.Extra["x-maxJsonBytes"]; got !=
		queryLegalInformationMaxArgumentsBytes {
		t.Fatalf("x-maxJsonBytes = %v", got)
	}
	querySchema := inputSchema.Properties["query"]
	if got := querySchema.Extra["x-maxUtf8Bytes"]; got != legalquery.MaxQueryBytes {
		t.Fatalf("x-maxUtf8Bytes = %v", got)
	}
	if got := querySchema.Extra["x-trimUnicodeWhitespace"]; got != true {
		t.Fatalf("x-trimUnicodeWhitespace = %v", got)
	}
	for name, query := range map[string]string{
		"trimmed empty": "\u3000\u00a0",
		"byte overflow": strings.Repeat("あ", 683),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := legalquery.NewRequest(
				legalquery.RequestValues{Query: query},
			); err == nil {
				t.Fatal("Request が schema 拡張で示した境界違反を受理しました")
			}
		})
	}
}

type querySchemaModelFixture struct {
	availablePlan     legalquery.LegalQueryPlan
	clarificationPlan legalquery.LegalQueryPlan
	unavailablePlan   legalquery.LegalQueryPlan
	unsupportedPlan   legalquery.LegalQueryPlan
	nonemptyAttempt   legalquery.LegalQueryLawSearchAttempt
	emptyAttempt      legalquery.LegalQueryLawSearchAttempt
	failedAttempt     legalquery.LegalQueryFailedAttempt
}

func newQuerySchemaModelFixture(t *testing.T) querySchemaModelFixture {
	t.Helper()

	firstStep := querySchemaModelStep(t, "step-search")
	secondStep := querySchemaModelStep(t, "step-fallback")
	availableCandidate := querySchemaModelCandidate(
		t,
		"candidate-available",
		nil,
		firstStep,
		secondStep,
	)
	unavailableCandidate := querySchemaModelCandidate(
		t,
		"candidate-unavailable",
		[]string{"judicial-cases"},
		firstStep,
		secondStep,
	)

	item := querySchemaModelLawItem(t)
	nonemptyPage := querySchemaModelPage(t, 1, false, 1)
	emptyPage := querySchemaModelPage(t, 0, false, 0)
	nonemptyAttempt, err := legalquery.NewLegalQueryLawSearchAttempt(
		legalquery.LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             firstStep,
			Page:             nonemptyPage,
			Items:            []model.SourcedResource[model.LawSummary]{item},
		},
	)
	if err != nil {
		t.Fatalf("nonempty attempt を作成できません: %v", err)
	}
	emptyAttempt, err := legalquery.NewLegalQueryLawSearchAttempt(
		legalquery.LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             firstStep,
			Page:             emptyPage,
			Items:            []model.SourcedResource[model.LawSummary]{},
		},
	)
	if err != nil {
		t.Fatalf("empty attempt を作成できません: %v", err)
	}
	sourceError, err := model.NewErrorResult(model.ErrorResultValues{
		Code: model.ErrorCodeSourceTimeout,
	})
	if err != nil {
		t.Fatalf("ErrorResult を作成できません: %v", err)
	}
	failedAttempt, err := legalquery.NewLegalQueryFailedAttempt(
		legalquery.LegalQueryFailedAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             secondStep,
			Error:            sourceError,
		},
	)
	if err != nil {
		t.Fatalf("failed attempt を作成できません: %v", err)
	}

	return querySchemaModelFixture{
		availablePlan: querySchemaModelPlan(
			t,
			legalquery.PlanDecisionSingle,
			availableCandidate,
			legalquery.SelectionAvailabilityAvailable,
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeSingleClearCandidate,
			},
		),
		clarificationPlan: querySchemaModelPlan(
			t,
			legalquery.PlanDecisionNeedsClarification,
			availableCandidate,
			legalquery.SelectionAvailabilityAvailable,
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeBelowExecutionThreshold,
			},
		),
		unavailablePlan: querySchemaModelPlan(
			t,
			legalquery.PlanDecisionCapabilityUnavailable,
			unavailableCandidate,
			legalquery.SelectionAvailabilityPackDisabled,
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeRequiredPackDisabled,
			},
		),
		unsupportedPlan: querySchemaModelPlan(
			t,
			legalquery.PlanDecisionUnsupported,
			availableCandidate,
			"",
			[]legalquery.ReasonCode{
				legalquery.ReasonCodeNonJapaneseQuery,
			},
		),
		nonemptyAttempt: nonemptyAttempt,
		emptyAttempt:    emptyAttempt,
		failedAttempt:   failedAttempt,
	}
}

func querySchemaModelStep(
	t *testing.T,
	stepID string,
) legalquery.LegalQueryCandidateStep {
	t.Helper()
	intent, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("law search intent を作成できません: %v", err)
	}
	step, err := legalquery.NewLegalQueryCandidateStep(
		legalquery.LegalQueryCandidateStepValues{
			StepID:                 stepID,
			Task:                   legalquery.TaskSearch,
			Resource:               legalquery.ResourceLaw,
			CapabilityID:           lawsearch.CapabilityID,
			CapabilityMajorVersion: lawsearch.MajorVersion,
			InputKind:              legalquery.InputKindLawSearch,
			LogicalInput:           intent,
		},
	)
	if err != nil {
		t.Fatalf("candidate step を作成できません: %v", err)
	}
	return step
}

func querySchemaModelCandidate(
	t *testing.T,
	id string,
	packs []string,
	steps ...legalquery.LegalQueryCandidateStep,
) legalquery.LegalQueryCandidate {
	t.Helper()
	candidate, err := legalquery.NewLegalQueryCandidate(
		legalquery.LegalQueryCandidateValues{
			CandidateID:    id,
			SemanticScore:  900,
			Confidence:     legalquery.ConfidenceHigh,
			EvidenceCodes:  []legalquery.EvidenceCode{legalquery.EvidenceExplicitTask},
			ConceptSources: []legalquery.LegalConceptSource{},
			RequiredPacks:  packs,
			Steps:          steps,
		},
	)
	if err != nil {
		t.Fatalf("candidate を作成できません: %v", err)
	}
	return candidate
}

func querySchemaModelPlan(
	t *testing.T,
	decision legalquery.PlanDecision,
	candidate legalquery.LegalQueryCandidate,
	availability legalquery.SelectionAvailability,
	reasons []legalquery.ReasonCode,
) legalquery.LegalQueryPlan {
	t.Helper()
	selections := []legalquery.LegalQueryPlanSelection{}
	if availability != "" {
		selection, err := legalquery.NewLegalQueryPlanSelection(
			legalquery.LegalQueryPlanSelectionValues{
				CandidateID:   candidate.CandidateID(),
				Availability:  availability,
				RequiredPacks: candidate.RequiredPacks(),
			},
		)
		if err != nil {
			t.Fatalf("selection を作成できません: %v", err)
		}
		selections = append(selections, selection)
	}
	plan, err := legalquery.NewLegalQueryPlan(legalquery.LegalQueryPlanValues{
		ProfileVersion:   "schema-serializer-v1",
		Decision:         decision,
		RankedCandidates: []legalquery.LegalQueryCandidate{candidate},
		Selected:         selections,
		ReasonCodes:      reasons,
		LimitPerAttempt:  legalquery.DefaultLimitPerAttempt,
	})
	if err != nil {
		t.Fatalf("plan を作成できません: %v", err)
	}
	return plan
}

func querySchemaModelLawItem(
	t *testing.T,
) model.SourcedResource[model.LawSummary] {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api",
		Name:       "e-Gov 法令 API",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "HTTPS://laws.e-gov.go.jp/",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("LegalSource を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source.ID(),
		ResourceType: "law",
		ResourceID:   "law-1",
		VersionID:    "revision-1",
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: strings.Repeat("a", 65),
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            "HTTPS://laws.e-gov.go.jp/law/law-1",
		RetrievedAt:    time.Date(2026, 7, 28, 10, 0, 0, 0, time.FixedZone("JST", 9*60*60)),
		MediaType:      "application/json",
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		t.Fatalf("Provenance を作成できません: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "revision-1",
		Title:      "民法",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("LawSummary を作成できません: %v", err)
	}
	item, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawSummary]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       summary,
	})
	if err != nil {
		t.Fatalf("SourcedResource を作成できません: %v", err)
	}
	return item
}

func querySchemaModelPage(
	t *testing.T,
	returnedCount int,
	hasMore bool,
	totalCount int,
) legalquery.LegalQueryPagePreview {
	t.Helper()
	page, err := legalquery.NewLegalQueryPagePreview(
		legalquery.LegalQueryPagePreviewValues{
			ReturnedCount: returnedCount,
			HasMore:       &hasMore,
			TotalCount:    &totalCount,
			TotalRelation: model.TotalRelationExact,
		},
	)
	if err != nil {
		t.Fatalf("page preview を作成できません: %v", err)
	}
	return page
}
