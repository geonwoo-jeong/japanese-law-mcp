package judicialcases

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func Test候補materializeは同点規則と重複意味の根拠統合を固定する(
	t *testing.T,
) {
	t.Parallel()

	profile := mustProfile(t)
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を構築できません: %v", err)
	}
	inputA := mustSearchInput(t, "あ検索")
	inputZ := mustSearchInput(t, "わ検索")
	drafts := []candidateDraft{
		{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
			steps: []stepDraft{{startByte: 30, input: inputZ}},
		},
		{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
			steps: []stepDraft{{startByte: 10, input: inputA}},
		},
		{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
			},
			steps: []stepDraft{{startByte: 5, input: inputA}},
		},
	}
	candidates, err := profile.materializeCandidates(drafts, scope)
	if err != nil {
		t.Fatalf("materializeCandidates() のエラー = %v", err)
	}
	if len(candidates) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
	firstInput := candidates[0].Steps()[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
	if firstInput.Query() != "あ検索" {
		t.Fatalf("同点規則後の先頭 query = %q", firstInput.Query())
	}
	if !slices.Equal(candidates[0].EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceMorphologicalContext,
	}) {
		t.Fatalf("統合 evidence = %#v", candidates[0].EvidenceCodes())
	}
	for _, candidate := range candidates {
		if !slices.Equal(
			candidate.RequiredPacks(),
			[]string{"judicial-cases"},
		) {
			t.Fatalf("required packs = %#v", candidate.RequiredPacks())
		}
	}
}

func Test候補materializeは裁判例以外のlogicalInputを拒否する(t *testing.T) {
	t.Parallel()

	input, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("law search input を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を構築できません: %v", err)
	}
	_, err = mustProfile(t).materializeCandidates(
		[]candidateDraft{{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
			},
			steps: []stepDraft{{input: input}},
		}},
		scope,
	)
	if err == nil {
		t.Fatal("法令 logical input を受理しました")
	}
}

func TestNilProfileはgetterとGenerateで安全に失敗する(t *testing.T) {
	t.Parallel()

	var profile *Profile
	if profile.Metadata().ProfileID() != "" ||
		profile.CueVocabulary() != nil {
		t.Fatal("nil profile getter が値を返しました")
	}
	if _, err := profile.Generate(
		legalquery.CandidateGenerationInput{},
		legalquery.CandidateIDScope{},
	); err == nil {
		t.Fatal("nil profile の Generate() が成功しました")
	}
}

func mustSearchInput(
	t *testing.T,
	query string,
) legalquery.JudicialDecisionSearchIntentV1 {
	t.Helper()

	input, err := legalquery.NewJudicialDecisionSearchIntentV1(
		legalquery.JudicialDecisionSearchIntentV1Values{Query: query},
	)
	if err != nil {
		t.Fatalf("search input を構築できません: %v", err)
	}
	return input
}
