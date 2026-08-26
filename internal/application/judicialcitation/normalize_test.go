package judicialcitation_test

import (
	"context"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitation"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNormalizeLowerCourtDecision(t *testing.T) {
	t.Parallel()

	details := newDetails(t, "東京高等裁判所", "令和5年（ネ）第100号", "2025-01-31", "民法第177条")
	got, ok, err := judicialcitation.NormalizeLowerCourtDecision(details)
	if err != nil || !ok {
		t.Fatal("SOT-MODEL-035/SOT-ARCH-043: 原審情報を正規化できませんでした")
	}
	if got.CourtName() != "東京高等裁判所" ||
		got.CaseNumber() != "令和5年（ネ）第100号" ||
		got.CaseNumberSearch() == "" {
		t.Fatalf("正規化結果 = %#v", got)
	}
}

func TestNormalizeLowerCourtDecisionは欠落と不正値を区別する(t *testing.T) {
	t.Parallel()

	missing := newDetails(t, "", "", "", "")
	if _, ok, err := judicialcitation.NormalizeLowerCourtDecision(missing); err != nil || ok {
		t.Fatalf("欠落 metadata = (ok=%t, err=%v)", ok, err)
	}

	malformed := newDetails(t, "東京高等裁判所", "令和5年（ネ）第100号 補足", "", "")
	if _, ok, err := judicialcitation.NormalizeLowerCourtDecision(malformed); err == nil || ok {
		t.Fatalf("不正 metadata = (ok=%t, err=%v)", ok, err)
	}
}

func TestNormalizeReferencedProvisionsは正確一致と登録別名だけを解決する(t *testing.T) {
	t.Parallel()

	details := newDetails(
		t,
		"東京高等裁判所",
		"令和5年（ネ）第100号",
		"2025-01-31",
		"民法第177条、個情法附則第2条第1項、同法第709条",
	)
	provenance := newProvenance(t)
	result, err := judicialcitation.NormalizeReferencedProvisions(
		context.Background(),
		mustExactLawAliasResolver(t),
		details,
		provenance,
	)
	if err != nil {
		t.Fatalf("NormalizeReferencedProvisions() のエラー = %v", err)
	}
	if len(result.References()) != 2 || len(result.Unresolved()) != 1 {
		t.Fatalf("正規化結果 = %#v / %#v", result.References(), result.Unresolved())
	}
	if result.References()[1].Location().ArticleNumber() != "2" ||
		result.References()[1].Location().Provision() != model.LawArticleProvisionSupplementary {
		t.Fatalf("法条位置 = %#v", result.References())
	}
	if result.Unresolved()[0].Reason() != model.JudicialCitationUnresolvedReasonUnsupportedReference {
		t.Fatalf("unresolved reason = %q", result.Unresolved()[0].Reason())
	}
}

func TestNormalizeReferencedProvisionsは誤記と曖昧別名を未解決のまま保持する(t *testing.T) {
	t.Parallel()

	details := newDetails(
		t,
		"",
		"",
		"",
		"個人情報保護砲第177条、開示法第1条、未登録架空特別法名第3条、民法第1条第1項第1号",
	)
	result, err := judicialcitation.NormalizeReferencedProvisions(
		context.Background(),
		mustExactLawAliasResolver(t),
		details,
		newProvenance(t),
	)
	if err != nil {
		t.Fatalf("NormalizeReferencedProvisions() のエラー = %v", err)
	}
	if len(result.References()) != 0 || len(result.Unresolved()) != 4 {
		t.Fatalf("正規化結果 = %#v / %#v", result.References(), result.Unresolved())
	}
	if result.Unresolved()[0].Reason() != model.JudicialCitationUnresolvedReasonFuzzyMatchOnly {
		t.Fatalf("unresolved[0] reason = %q", result.Unresolved()[0].Reason())
	}
	if result.Unresolved()[1].Reason() != model.JudicialCitationUnresolvedReasonAmbiguousTarget {
		t.Fatalf("unresolved[1] reason = %q", result.Unresolved()[1].Reason())
	}
	if result.Unresolved()[2].Reason() != model.JudicialCitationUnresolvedReasonUnregisteredLawName {
		t.Fatalf("unresolved[2] reason = %q", result.Unresolved()[2].Reason())
	}
	if result.Unresolved()[3].Reason() != model.JudicialCitationUnresolvedReasonAmbiguousLawLocation {
		t.Fatalf("unresolved[3] reason = %q", result.Unresolved()[3].Reason())
	}
}

func mustExactLawAliasResolver(t *testing.T) judicialcitationnormalize.ExactLawAliasResolver {
	t.Helper()

	resolver, err := judicialcitationnormalize.NewExactLawAliasResolver([]lawnamelexicon.Entry{
		{
			ResourceID: "129AC0000000089",
			Canonical:  "民法",
		},
		{
			ResourceID: "416AC0000000057",
			Canonical:  "個人情報の保護に関する法律",
			Terms:      []string{"個人情報保護法", "個情法"},
		},
		{
			ResourceID: "disclosure-a",
			Canonical:  "独立行政法人等の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
		{
			ResourceID: "disclosure-b",
			Canonical:  "行政機関の保有する情報の公開に関する法律",
			Terms:      []string{"開示法"},
		},
	})
	if err != nil {
		t.Fatalf("resolver を構築できません: %v", err)
	}
	return resolver
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
