package judicialcitation_test

import (
	"context"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawtarget"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNormalizeLowerCourtDecision(t *testing.T) {
	t.Parallel()

	details := newDetails(t, "東京高等裁判所", "令和5年（ネ）第100号", "2025-01-31", "民法第177条")
	got, ok := judicialcitation.NormalizeLowerCourtDecision(details)
	if !ok {
		t.Fatal("SOT-MODEL-035/SOT-ARCH-043: 原審情報を正規化できませんでした")
	}
	if got.CourtName() != "東京高等裁判所" ||
		got.CaseNumber() != "令和5年（ネ）第100号" ||
		got.CaseNumberSearch() == "" {
		t.Fatalf("正規化結果 = %#v", got)
	}
}

func TestNormalizeReferencedProvisionsResolvesAndKeepsCurrentLaw(t *testing.T) {
	t.Parallel()

	details := newDetails(
		t,
		"東京高等裁判所",
		"令和5年（ネ）第100号",
		"2025-01-31",
		"民法第177条、同法第709条、附則第1条",
	)
	provenance := newProvenance(t)
	result, err := judicialcitation.NormalizeReferencedProvisions(
		context.Background(),
		fakeResolver{
			results: map[string]lawtarget.ResolvedLawTarget{
				"民法": mustResolvedLawTarget(t, "129AC0000000089", "民法", lawtarget.MatchKindExact),
				"同法": mustResolvedLawTarget(t, "129AC0000000089", "民法", lawtarget.MatchKindRegisteredTerm),
			},
		},
		details,
		provenance,
	)
	if err != nil {
		t.Fatalf("NormalizeReferencedProvisions() のエラー = %v", err)
	}
	if len(result.References()) != 3 || len(result.Unresolved()) != 0 {
		t.Fatalf("正規化結果 = %#v / %#v", result.References(), result.Unresolved())
	}
	if result.References()[1].Location().ArticleNumber() != "709" ||
		result.References()[2].Location().Provision() != model.LawArticleProvisionSupplementary {
		t.Fatalf("法条位置 = %#v", result.References())
	}
}

func TestNormalizeReferencedProvisionsRejectsFuzzyOrUnknownLaw(t *testing.T) {
	t.Parallel()

	details := newDetails(t, "", "", "", "みんぽう第177条")
	result, err := judicialcitation.NormalizeReferencedProvisions(
		context.Background(),
		fakeResolver{
			results: map[string]lawtarget.ResolvedLawTarget{
				"みんぽう": mustResolvedLawTarget(
					t,
					"129AC0000000089",
					"民法",
					lawtarget.MatchKindUniqueTypoCorrection,
				),
			},
		},
		details,
		newProvenance(t),
	)
	if err != nil {
		t.Fatalf("NormalizeReferencedProvisions() のエラー = %v", err)
	}
	if len(result.References()) != 0 || len(result.Unresolved()) != 1 {
		t.Fatalf("正規化結果 = %#v / %#v", result.References(), result.Unresolved())
	}
	if result.Unresolved()[0].Reason() != model.JudicialCitationUnresolvedReasonFuzzyMatchOnly {
		t.Fatalf("unresolved reason = %q", result.Unresolved()[0].Reason())
	}
}

type fakeResolver struct {
	results map[string]lawtarget.ResolvedLawTarget
}

func (f fakeResolver) ResolveLogicalInput(
	_ context.Context,
	query string,
) (lawtarget.ResolvedLawTarget, bool, error) {
	value, exists := f.results[query]
	return value, exists, nil
}

func mustResolvedLawTarget(
	t *testing.T,
	lawID string,
	title string,
	matchKind lawtarget.MatchKind,
) lawtarget.ResolvedLawTarget {
	t.Helper()
	target, err := lawtarget.NewResolvedLawTarget(lawID, title, matchKind)
	if err != nil {
		t.Fatalf("resolved law target を構築できません: %v", err)
	}
	return target
}

func newDetails(
	t *testing.T,
	lowerCourtName string,
	lowerCourtCaseNumber string,
	lowerCourtDate string,
	referencedProvisions string,
) model.JudicialDecisionDetails {
	t.Helper()
	var date *model.Date
	if lowerCourtDate != "" {
		value := newDate(t, lowerCourtDate)
		date = &value
	}
	var courtName *string
	if lowerCourtName != "" {
		courtName = &lowerCourtName
	}
	var caseNumber *string
	if lowerCourtCaseNumber != "" {
		caseNumber = &lowerCourtCaseNumber
	}
	var provisions *string
	if referencedProvisions != "" {
		provisions = &referencedProvisions
	}
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:                  newSummary(t),
		LowerCourtName:           courtName,
		LowerCourtCaseNumber:     caseNumber,
		LowerCourtDecisionDate:   date,
		ReferencedProvisionsText: provisions,
	})
	if err != nil {
		t.Fatalf("details を構築できません: %v", err)
	}
	return details
}

func newSummary(t *testing.T) model.JudicialDecisionSummary {
	t.Helper()
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          "12345",
		PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
		SourceCategoryLabel: "最高裁判例",
		CaseNumber:          "令和6年（受）第1号",
		DecisionDate:        newDate(t, "2026-08-26"),
		CourtName:           "最高裁判所",
		DetailURL:           "https://www.courts.go.jp/hanrei/12345/detail2/index.html",
		Documents:           []model.JudicialDocumentLink{},
		Source:              newInformationSource(t),
	})
	if err != nil {
		t.Fatalf("summary を構築できません: %v", err)
	}
	return summary
}

func newInformationSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "courts-hanrei",
		Name:       "裁判所 裁判例検索",
		Publisher:  "最高裁判所",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://www.courts.go.jp/hanrei/search1/index.html",
	})
	if err != nil {
		t.Fatalf("information source を構築できません: %v", err)
	}
	return source
}

func newProvenance(t *testing.T) model.Provenance {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "12345:detail2",
	})
	if err != nil {
		t.Fatalf("resource key を構築できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         newInformationSource(t),
		ResourceKey:    key,
		URL:            "https://www.courts.go.jp/hanrei/12345/detail2/index.html",
		RetrievedAt:    time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC),
		MediaType:      "text/html",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-045",
	})
	if err != nil {
		t.Fatalf("provenance を構築できません: %v", err)
	}
	return provenance
}

func newDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date を構築できません: %v", err)
	}
	return date
}
