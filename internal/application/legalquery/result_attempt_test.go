package legalquery

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const resultTestJudicialCoverageNotice = "裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。掲載情報だけから先例性、拘束力、確定性または現在の有効性を判断できません。"

func TestLegalQueryAttemptAcceptsSevenTypedSuccessVariants(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	lawSearchItems := []model.SourcedResource[model.LawSummary]{fixtures.lawSummary}
	lawSearch, err := NewLegalQueryLawSearchAttempt(
		LegalQueryLawSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "法令検索", "step-law-search"),
			Page:             resultTestPage(t, 1, false, 1, model.TotalRelationExact),
			Items:            lawSearchItems,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: law search attempt を作成できません: %v", err)
	}
	lawContent, err := NewLegalQueryLawContentSearchAttempt(
		LegalQueryLawContentSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "法令本文検索", "step-law-content"),
			Page:             resultTestPage(t, 1, false, 1, model.TotalRelationExact),
			Items: []model.SourcedResource[model.LawContentMatch]{
				fixtures.lawContent,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: law content attempt を作成できません: %v", err)
	}
	lawDocument, err := NewLegalQueryLawDocumentAttempt(
		LegalQueryLawDocumentAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "法令読取り", "step-law-document"),
			Item:             fixtures.lawDocument,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: law document attempt を作成できません: %v", err)
	}
	lawArticle, err := NewLegalQueryLawArticleAttempt(
		LegalQueryLawArticleAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "条文読取り", "step-law-article"),
			Item:             fixtures.lawArticle,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: law article attempt を作成できません: %v", err)
	}
	lawUpdates, err := NewLegalQueryLawUpdatesAttempt(
		LegalQueryLawUpdatesAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "更新一覧", "step-law-updates"),
			Page:             resultTestPage(t, 1, false, 1, model.TotalRelationExact),
			Items: []model.SourcedResource[model.LawUpdate]{
				fixtures.lawUpdate,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: law updates attempt を作成できません: %v", err)
	}
	judicialSearch, err := NewLegalQueryJudicialSearchAttempt(
		LegalQueryJudicialSearchAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "裁判例検索", "step-judicial-search"),
			Page:             resultTestPage(t, 1, false, 1, model.TotalRelationExact),
			Items: []model.SourcedResource[model.JudicialDecisionSummary]{
				fixtures.judicialSummary,
			},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: judicial search attempt を作成できません: %v", err)
	}
	judicialDecision, err := NewLegalQueryJudicialDecisionAttempt(
		LegalQueryJudicialDecisionAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             planTestStep(t, "裁判例読取り", "step-judicial-decision"),
			Item:             fixtures.judicialDecision,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: judicial decision attempt を作成できません: %v", err)
	}

	tests := []struct {
		name           string
		attempt        LegalQueryAttempt
		resultKind     string
		outcome        string
		resultFields   []string
		coverageNotice bool
	}{
		{"法令検索", lawSearch, "law_search", "completed", []string{"page", "items"}, false},
		{"法令本文検索", lawContent, "law_content_search", "completed", []string{"page", "items"}, false},
		{"法令読取り", lawDocument, "law_document", "completed", []string{"item"}, false},
		{"条文読取り", lawArticle, "law_article", "completed", []string{"item"}, false},
		{"更新一覧", lawUpdates, "law_updates", "completed", []string{"page", "items"}, false},
		{"裁判例検索", judicialSearch, "judicial_decision_search", "completed", []string{"coverageNotice", "page", "items"}, true},
		{"裁判例読取り", judicialDecision, "judicial_decision", "completed", []string{"coverageNotice", "item"}, true},
	}
	for _, test := range tests {
		if test.attempt.InterpretationID() != "interpretation-1" ||
			test.attempt.CapabilityMajorVersion() != 1 ||
			string(test.attempt.Outcome()) != test.outcome {
			t.Fatalf("SOT-MODEL-024: %s attempt = %#v", test.name, test.attempt)
		}
		object := attemptTestJSONObject(t, test.attempt)
		if object["resultKind"] != test.resultKind || object["outcome"] != test.outcome {
			t.Fatalf("SOT-MODEL-024: %s JSON = %#v", test.name, object)
		}
		result, ok := object["result"].(map[string]any)
		if !ok || len(result) != len(test.resultFields) {
			t.Fatalf("SOT-MODEL-024: %s result = %#v", test.name, object["result"])
		}
		for _, field := range test.resultFields {
			if _, exists := result[field]; !exists {
				t.Fatalf("SOT-MODEL-024: %s result.%s がありません", test.name, field)
			}
		}
		if test.coverageNotice && result["coverageNotice"] != resultTestJudicialCoverageNotice {
			t.Fatalf("SOT-MODEL-024: coverageNotice = %#v", result["coverageNotice"])
		}
	}

	lawSearchItems[0] = model.SourcedResource[model.LawSummary]{}
	gotItems := lawSearch.Items()
	gotItems[0] = model.SourcedResource[model.LawSummary]{}
	if len(lawSearch.Items()) != 1 ||
		lawSearch.Items()[0].Data().LawID() != "law-1" {
		t.Fatal("SOT-MODEL-024: attempt の items を入力または getter から変更できました")
	}
}

func TestLegalQueryAttemptRejectsStepVariantAndPageCountMismatch(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	if _, err := NewLegalQueryLawSearchAttempt(LegalQueryLawSearchAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             planTestStep(t, "法令本文検索", "step-wrong-kind"),
		Page:             resultTestUnknownPage(t, 0),
		Items:            []model.SourcedResource[model.LawSummary]{},
	}); err == nil {
		t.Fatal("SOT-MODEL-024: resultKind と一致しない step を受理しました")
	}
	if _, err := NewLegalQueryLawSearchAttempt(LegalQueryLawSearchAttemptValues{
		InterpretationID: "interpretation-1",
		Step:             planTestStep(t, "法令検索", "step-wrong-count"),
		Page:             resultTestUnknownPage(t, 0),
		Items:            []model.SourcedResource[model.LawSummary]{fixtures.lawSummary},
	}); err == nil {
		t.Fatal("SOT-MODEL-024: returnedCount と items 件数の不一致を受理しました")
	}
}

func TestLegalQueryAttemptRejectsCapabilitySpecificResourceMismatch(t *testing.T) {
	t.Parallel()

	fixtures := newAttemptPayloadFixtures(t)
	badLawSummary := attemptTestSourced(
		t,
		fixtures.lawInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "e-gov-law-api-v2",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.lawSummary.Data(),
	)
	badLawContent := attemptTestSourced(
		t,
		fixtures.lawInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "e-gov-law-api-v2",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
			Location:     "main:article=1",
		},
		fixtures.lawContent.Data(),
	)
	badLawDocument := attemptTestSourced(
		t,
		fixtures.lawInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "e-gov-law-api-v2",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.lawDocument.Data(),
	)
	badLawArticle := attemptTestSourced(
		t,
		fixtures.lawInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "e-gov-law-api-v2",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.lawArticle.Data(),
	)
	badLawUpdate := attemptTestSourced(
		t,
		fixtures.lawInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "e-gov-law-api-v1",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.lawUpdate.Data(),
	)
	badJudicialSummary := attemptTestSourced(
		t,
		fixtures.judicialInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "courts-hanrei-html",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.judicialSummary.Data(),
	)
	badJudicialDecision := attemptTestSourced(
		t,
		fixtures.judicialInformationSource,
		attemptTestSourcedValues{
			ProviderID:   "courts-hanrei-html",
			ResourceType: "fixture",
			ResourceID:   "fixture-1",
		},
		fixtures.judicialDecision.Data(),
	)

	tests := []struct {
		name      string
		construct func() error
	}{
		{
			name: "法令検索",
			construct: func() error {
				_, err := NewLegalQueryLawSearchAttempt(
					LegalQueryLawSearchAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"法令検索",
							"step-bad-law-search",
						),
						Page: resultTestPage(
							t,
							1,
							false,
							1,
							model.TotalRelationExact,
						),
						Items: []model.SourcedResource[model.LawSummary]{
							badLawSummary,
						},
					},
				)
				return err
			},
		},
		{
			name: "法令本文検索",
			construct: func() error {
				_, err := NewLegalQueryLawContentSearchAttempt(
					LegalQueryLawContentSearchAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"法令本文検索",
							"step-bad-law-content",
						),
						Page: resultTestPage(
							t,
							1,
							false,
							1,
							model.TotalRelationExact,
						),
						Items: []model.SourcedResource[model.LawContentMatch]{
							badLawContent,
						},
					},
				)
				return err
			},
		},
		{
			name: "法令読取り",
			construct: func() error {
				_, err := NewLegalQueryLawDocumentAttempt(
					LegalQueryLawDocumentAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"法令読取り",
							"step-bad-law-document",
						),
						Item: badLawDocument,
					},
				)
				return err
			},
		},
		{
			name: "条文読取り",
			construct: func() error {
				_, err := NewLegalQueryLawArticleAttempt(
					LegalQueryLawArticleAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"条文読取り",
							"step-bad-law-article",
						),
						Item: badLawArticle,
					},
				)
				return err
			},
		},
		{
			name: "更新一覧",
			construct: func() error {
				_, err := NewLegalQueryLawUpdatesAttempt(
					LegalQueryLawUpdatesAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"更新一覧",
							"step-bad-law-updates",
						),
						Page: resultTestPage(
							t,
							1,
							false,
							1,
							model.TotalRelationExact,
						),
						Items: []model.SourcedResource[model.LawUpdate]{
							badLawUpdate,
						},
					},
				)
				return err
			},
		},
		{
			name: "裁判例検索",
			construct: func() error {
				_, err := NewLegalQueryJudicialSearchAttempt(
					LegalQueryJudicialSearchAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"裁判例検索",
							"step-bad-judicial-search",
						),
						Page: resultTestPage(
							t,
							1,
							false,
							1,
							model.TotalRelationExact,
						),
						Items: []model.SourcedResource[model.JudicialDecisionSummary]{
							badJudicialSummary,
						},
					},
				)
				return err
			},
		},
		{
			name: "裁判例読取り",
			construct: func() error {
				_, err := NewLegalQueryJudicialDecisionAttempt(
					LegalQueryJudicialDecisionAttemptValues{
						InterpretationID: "interpretation-1",
						Step: planTestStep(
							t,
							"裁判例読取り",
							"step-bad-judicial-read",
						),
						Item: badJudicialDecision,
					},
				)
				return err
			},
		},
	}
	for _, test := range tests {
		if err := test.construct(); err == nil {
			t.Fatalf("SOT-MODEL-024: %s の capability 固有制約違反を受理しました", test.name)
		}
	}
}

func TestLegalQueryFailedAttemptPublishesOnlyAllowedExecutionError(t *testing.T) {
	t.Parallel()

	step := planTestStep(t, "法令検索", "step-failed")
	for _, code := range []model.ErrorCode{
		model.ErrorCodeNotFound,
		model.ErrorCodeAmbiguousLocation,
		model.ErrorCodeUnsupportedQuery,
		model.ErrorCodeSourceAuthFailed,
		model.ErrorCodeRateLimited,
		model.ErrorCodeSourceTimeout,
		model.ErrorCodeSourceUnavailable,
		model.ErrorCodeSourceBusy,
		model.ErrorCodeSourceContractChanged,
		model.ErrorCodeInvalidSourceResponse,
		model.ErrorCodeSourceResponseTooLarge,
		model.ErrorCodeSourceProcessingLimit,
		model.ErrorCodeUnsafeSourceContent,
		model.ErrorCodeInternalError,
	} {
		publicError := resultTestError(t, code)
		attempt, err := NewLegalQueryFailedAttempt(LegalQueryFailedAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             step,
			Error:            publicError,
		})
		if err != nil {
			t.Fatalf("SOT-MODEL-024: error.code=%s を拒否しました: %v", code, err)
		}
		if string(attempt.Outcome()) != "failed" ||
			attempt.Error().Code() != code {
			t.Fatalf("SOT-MODEL-024: failed attempt = %#v", attempt)
		}
		object := attemptTestJSONObject(t, attempt)
		if object["outcome"] != "failed" || object["error"] == nil {
			t.Fatalf("SOT-MODEL-024: failed JSON = %#v", object)
		}
		if _, exists := object["resultKind"]; exists {
			t.Fatalf("SOT-MODEL-024: failed に resultKind があります: %#v", object)
		}
		if _, exists := object["result"]; exists {
			t.Fatalf("SOT-MODEL-024: failed に result があります: %#v", object)
		}
	}

	for _, code := range []model.ErrorCode{
		model.ErrorCodeInvalidArgument,
		model.ErrorCodeUnsupportedCapability,
		model.ErrorCodeConfigurationRequired,
	} {
		if _, err := NewLegalQueryFailedAttempt(LegalQueryFailedAttemptValues{
			InterpretationID: "interpretation-1",
			Step:             step,
			Error:            resultTestError(t, code),
		}); err == nil {
			t.Fatalf("SOT-MODEL-024: partial に入らない error.code=%s を受理しました", code)
		}
	}
}

func TestLegalQueryAttemptsRejectDirectJSONRestore(t *testing.T) {
	t.Parallel()

	for _, target := range []any{
		&LegalQueryLawSearchAttempt{},
		&LegalQueryLawContentSearchAttempt{},
		&LegalQueryLawDocumentAttempt{},
		&LegalQueryLawArticleAttempt{},
		&LegalQueryLawUpdatesAttempt{},
		&LegalQueryJudicialSearchAttempt{},
		&LegalQueryJudicialDecisionAttempt{},
		&LegalQueryFailedAttempt{},
	} {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-MODEL-024: %T を JSON から直接復元できました", target)
		}
	}
}

type attemptPayloadFixtures struct {
	lawInformationSource      model.InformationSource
	judicialInformationSource model.InformationSource
	lawSummary                model.SourcedResource[model.LawSummary]
	lawContent                model.SourcedResource[model.LawContentMatch]
	lawDocument               model.SourcedResource[model.LawDocumentRepresentation]
	lawArticle                model.SourcedResource[model.LawArticleFragment]
	lawUpdate                 model.SourcedResource[model.LawUpdate]
	judicialSummary           model.SourcedResource[model.JudicialDecisionSummary]
	judicialDecision          model.SourcedResource[model.JudicialDecisionDetails]
}

func newAttemptPayloadFixtures(t *testing.T) attemptPayloadFixtures {
	t.Helper()

	lawInformationSource := resultTestInformationSource(
		t,
		"e-gov-law-api-v2",
		"https://laws.e-gov.go.jp/",
	)
	lawSource, err := model.NewLegalSource(lawInformationSource)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できません: %v", err)
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID: "law-1", RevisionID: "revision-1", Title: "行政手続法", Source: lawSource,
	})
	if err != nil {
		t.Fatalf("試験用 LawSummary を作成できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source: lawSource, LawID: "law-1", RevisionID: "revision-1",
		Location: "main:article=1", URL: "https://laws.e-gov.go.jp/law/law-1",
	})
	if err != nil {
		t.Fatalf("試験用 Citation を作成できません: %v", err)
	}
	document, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law: law, Format: model.LawDocumentFormatText, Content: "本文", Citation: citation,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LawDocumentRepresentation を作成できません: %v", err)
	}
	location := mustLawArticleLocation(t)
	article, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law: law, Location: location, Format: model.LawArticleFormatText,
		Content: "第一条", Citation: citation,
	})
	if err != nil {
		t.Fatalf("試験用 LawArticleFragment を作成できません: %v", err)
	}
	content, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      law,
		Location: "main:article=1",
		Text:     "本文",
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("試験用 LawContentMatch を作成できません: %v", err)
	}
	update, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn: mustDate(t, "2026-07-27"),
		LawID:     "law-1",
		Title:     "行政手続法",
		Source:    lawSource,
	})
	if err != nil {
		t.Fatalf("試験用 LawUpdate を作成できません: %v", err)
	}

	judicialSource := resultTestInformationSource(
		t,
		"courts-hanrei",
		"https://www.courts.go.jp/hanrei/",
	)
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID: "95570", PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例", CaseNumber: "令和6年（受）第1号",
			DecisionDate: mustDate(t, "2025-03-03"), CourtName: "最高裁判所",
			DetailURL: "https://www.courts.go.jp/hanrei/95570/detail2/index.html",
			Documents: []model.JudicialDocumentLink{}, Source: judicialSource,
		},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionSummary を作成できません: %v", err)
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: summary},
	)
	if err != nil {
		t.Fatalf("試験用 JudicialDecisionDetails を作成できません: %v", err)
	}

	return attemptPayloadFixtures{
		lawInformationSource:      lawInformationSource,
		judicialInformationSource: judicialSource,
		lawSummary: attemptTestSourced(
			t,
			lawInformationSource,
			attemptTestSourcedValues{
				ProviderID:   "e-gov-law-api-v2",
				ResourceType: "law",
				ResourceID:   "law-1",
				VersionID:    "revision-1",
			},
			law,
		),
		lawContent: attemptTestSourced(
			t,
			lawInformationSource,
			attemptTestSourcedValues{
				ProviderID:   "e-gov-law-api-v2",
				ResourceType: "law",
				ResourceID:   "law-1",
				VersionID:    "revision-1",
				Location:     "main:article=1",
			},
			content,
		),
		lawDocument: attemptTestSourced(
			t,
			lawInformationSource,
			attemptTestSourcedValues{
				ProviderID:   "e-gov-law-api-v2",
				ResourceType: "law",
				ResourceID:   "law-1",
				VersionID:    "revision-1",
			},
			document,
		),
		lawArticle: attemptTestSourced(
			t,
			lawInformationSource,
			attemptTestSourcedValues{
				ProviderID:   "e-gov-law-api-v2",
				ResourceType: "law",
				ResourceID:   "law-1",
				VersionID:    "revision-1",
			},
			article,
		),
		lawUpdate: attemptTestSourced(
			t,
			lawInformationSource,
			attemptTestSourcedValues{
				ProviderID:   "e-gov-law-api-v1",
				ResourceType: "law-update-list",
				ResourceID:   "2026-07-27",
			},
			update,
		),
		judicialSummary: attemptTestSourced(
			t,
			judicialSource,
			attemptTestSourcedValues{
				ProviderID:   "courts-hanrei-html",
				ResourceType: "judicial-decision",
				ResourceID:   "95570/detail2",
			},
			summary,
		),
		judicialDecision: attemptTestSourced(
			t,
			judicialSource,
			attemptTestSourcedValues{
				ProviderID:   "courts-hanrei-html",
				ResourceType: "judicial-decision",
				ResourceID:   "95570/detail2",
			},
			details,
		),
	}
}

type attemptTestSourcedValues struct {
	ProviderID   string
	ResourceType string
	ResourceID   string
	VersionID    string
	Location     string
}

func attemptTestSourced[T interface{ Validate() error }](
	t *testing.T,
	source model.InformationSource,
	values attemptTestSourcedValues,
	data T,
) model.SourcedResource[T] {
	t.Helper()
	var ref model.SourceResourceRef
	if values.VersionID == "" {
		ref = newLegalQuerySourceResourceRef(
			t,
			values.ProviderID,
			source.ID(),
			values.ResourceType,
			values.ResourceID,
		)
	} else {
		ref = newVersionedSourceResourceRef(
			t,
			values.ProviderID,
			source.ID(),
			values.ResourceType,
			values.ResourceID,
			values.VersionID,
		)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source: source, ResourceKey: ref.Key(), URL: source.ServiceURL(),
		RetrievedAt:    time.Date(2026, time.July, 28, 0, 0, 0, 0, time.UTC),
		MediaType:      "application/json",
		Location:       values.Location,
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		t.Fatalf("試験用 Provenance を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[T]{
		Ref: ref, Provenance: []model.Provenance{provenance}, Data: data,
	})
	if err != nil {
		t.Fatalf("試験用 SourcedResource を作成できません: %v", err)
	}
	return resource
}

func resultTestInformationSource(
	t *testing.T,
	id string,
	serviceURL string,
) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID: id, Name: "公式情報源", Publisher: "公的機関",
		Authority: model.AuthorityOfficial, ServiceURL: serviceURL,
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できません: %v", err)
	}
	return source
}

func resultTestUnknownPage(t *testing.T, count int) LegalQueryPagePreview {
	t.Helper()
	page, err := NewLegalQueryPagePreview(LegalQueryPagePreviewValues{
		ReturnedCount: count,
	})
	if err != nil {
		t.Fatalf("試験用 page preview を作成できません: %v", err)
	}
	return page
}

func resultTestError(t *testing.T, code model.ErrorCode) model.ErrorResult {
	t.Helper()
	result, err := model.NewErrorResult(model.ErrorResultValues{Code: code})
	if err != nil {
		t.Fatalf("試験用 ErrorResult を作成できません: %v", err)
	}
	return result
}

func attemptTestJSONObject(t *testing.T, value any) map[string]any {
	t.Helper()
	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("SOT-MODEL-024: JSON に変換できません: %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("試験用 JSON を読み取れません: %v", err)
	}
	return object
}
