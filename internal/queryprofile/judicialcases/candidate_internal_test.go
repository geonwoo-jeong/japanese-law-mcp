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
	records, err := profile.materializeCandidateRecords(drafts, scope)
	if err != nil {
		t.Fatalf("materializeCandidateRecords() のエラー = %v", err)
	}
	candidates := make([]legalquery.LegalQueryCandidate, 0, len(records))
	for _, record := range records {
		candidates = append(candidates, record.candidate)
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
	if len(records[0].sourceStarts) != 1 ||
		records[0].sourceStarts[0] != 5 {
		t.Fatalf(
			"SOT-MODEL-028: 重複意味の最小原文位置 = %#v",
			records[0].sourceStarts,
		)
	}
	members, err := judicialCompositionMembers(
		records,
		legalquery.QuerySelectionModeAutomatic,
	)
	if err != nil {
		t.Fatalf("複数候補の composition members = %v", err)
	}
	if len(members) != len(records) {
		t.Fatalf(
			"SOT-ARCH-027: 複数候補の sidecar = %#v",
			members,
		)
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
