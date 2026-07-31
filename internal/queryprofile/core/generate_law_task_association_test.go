package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestProfileは後続確認語を先行法令検索へ適用しない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"会社法を検索し、取締役の条文と2026年6月30日の更新一覧も確認してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 3 {
		t.Fatalf("step 数 = %d, want 3", len(steps))
	}
	if steps[0].InputKind() != legalquery.InputKindLawSearch ||
		steps[1].InputKind() != legalquery.InputKindLawContentSearch ||
		steps[2].InputKind() != legalquery.InputKindLawUpdates {
		t.Fatalf("SOT-MODEL-025: step 順 = %#v", steps)
	}
	search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || search.Query() != "会社法" {
		t.Fatalf("法令検索入力 = %#v", steps[0].LogicalInput())
	}
	content, ok := steps[1].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		len(content.AllTerms()) != 1 ||
		content.AllTerms()[0] != "取締役" {
		t.Fatalf("条文検索入力 = %#v", steps[1].LogicalInput())
	}
	update, ok := steps[2].LogicalInput().(legalquery.LawUpdateListIntentV1)
	if !ok || update.Date().String() != "2026-06-30" {
		t.Fatalf("更新一覧入力 = %#v", steps[2].LogicalInput())
	}
}

func TestProfileは法令名に直結した読取り語を維持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"2024年4月1日時点の民法を読んで",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawRead {
		t.Fatalf("SOT-MODEL-025: steps = %#v", steps)
	}
	read, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
	if !ok {
		t.Fatalf("法令読取り入力 = %T", steps[0].LogicalInput())
	}
	asOf, exists := read.AsOf()
	if !exists || asOf.String() != "2024-04-01" {
		t.Fatalf("法令読取り基準日 = %q, exists=%t", asOf.String(), exists)
	}
}

func TestProfileは明示検索と識別子読取りを同一候補に保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法を検索し、法令ID 129AC0000000089 の民法の全文も読む"
	generation := generateQuery(
		t,
		query,
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch ||
		steps[1].InputKind() != legalquery.InputKindLawRead {
		t.Fatalf("SOT-ENG-024: 明示検索と読取りの step = %#v", steps)
	}
	search, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || search.Query() != "民法" {
		t.Fatalf("法令検索入力 = %#v", steps[0].LogicalInput())
	}
	read, ok := steps[1].LogicalInput().(legalquery.LawReadIntentV1)
	lawID, hasLawID := read.LawID()
	if !ok || !hasLawID || lawID != "129AC0000000089" {
		t.Fatalf("法令読取り入力 = %#v", steps[1].LogicalInput())
	}
	if got := candidates[0].EvidenceCodes(); !slices.Equal(
		got,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
		},
	) {
		t.Fatalf("SOT-ARCH-036: 複数 step の根拠 = %#v", got)
	}
}

func TestProfileは同一読取りStepの識別子と別名を重複保持しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"法令ID 129AC0000000089 の民法を読む",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf("SOT-ARCH-036: 同一 step の候補 = %#v", candidates)
	}
	if got := candidates[0].EvidenceCodes(); !slices.Equal(
		got,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
	) {
		t.Fatalf("SOT-ARCH-036: 同一 step の根拠 = %#v", got)
	}
}

func TestProfileは法令名検索と識別子付き条文読取りの根拠を併存させる(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"刑法を検索し、法令ID 140AC0000000045 の刑法第199条も読む",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch ||
		steps[1].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf("SOT-ARCH-036: steps = %#v", steps)
	}
	if got := candidates[0].EvidenceCodes(); !slices.Equal(
		got,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
		},
	) {
		t.Fatalf("SOT-ARCH-036: 識別子付き条文読取りの根拠 = %#v", got)
	}
}

func TestProfileは同一条文読取りStepの識別子と別名を重複保持しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"法令ID 140AC0000000045 の刑法第199条を読む",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf("SOT-ARCH-036: steps = %#v", steps)
	}
	if got := candidates[0].EvidenceCodes(); !slices.Equal(
		got,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
	) {
		t.Fatalf("SOT-ARCH-036: 同一条文読取り step の根拠 = %#v", got)
	}
}

func TestProfileは法令履歴IDを版なし条文読取りの根拠に流用しない(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"法令履歴ID 140AC0000000045_20260521_507AC0000000039 の刑法第199条を読む",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("候補数 = %d, want 1", len(candidates))
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawArticleRead {
		t.Fatalf("SOT-ARCH-036: steps = %#v", steps)
	}
	if got := candidates[0].EvidenceCodes(); !slices.Equal(
		got,
		[]legalquery.EvidenceCode{
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
		},
	) {
		t.Fatalf("SOT-ARCH-036: 版なし条文読取りの根拠 = %#v", got)
	}
}

func TestProfileは更新資源の後続確認語を法令読取りへ適用しない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"会社法の更新一覧を確認してください。",
		nil,
	)
	for _, candidate := range generation.Candidates() {
		for _, step := range candidate.Steps() {
			if step.InputKind() == legalquery.InputKindLawRead {
				t.Fatalf(
					"SOT-MODEL-025: 更新資源の確認を法令読取りへ適用した step = %#v",
					step,
				)
			}
		}
	}
}

func TestProfileは一般検索語がなくても更新資源を法令読取りへ適用しない(
	t *testing.T,
) {
	t.Parallel()

	const query = "会社法の更新一覧を確認してください。"
	profile := mustProfile(t)
	preprocessed := prepareGenerationInput(t, profile, query)
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                query,
			ComparisonKey:        string(querynormalization.ComparisonKey(query)),
			LawNameMentions:      preprocessed.LawNameMentions(),
			LegalConceptMentions: preprocessed.LegalConceptMentions(),
			CueMentions:          preprocessed.CueMentions(),
			IdentifierMentions:   preprocessed.IdentifierMentions(),
			DateMentions:         preprocessed.DateMentions(),
			ArticleMentions:      preprocessed.ArticleMentions(),
			ParagraphMentions:    preprocessed.ParagraphMentions(),
			CaseNumberMentions:   preprocessed.CaseNumberMentions(),
		},
	)
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("試験用 generation input を作成できません: %v", err)
	}
	cues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf("試験用 cue を解決できません: %v", err)
	}
	if explicitDocumentReadRequested(input, cues) {
		t.Fatal("SOT-MODEL-025: 更新資源の確認を法令本文読取りへ関連付けました")
	}
}
