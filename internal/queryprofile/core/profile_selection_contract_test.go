package core

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileContributionは弱い一般語を明確化必須にする(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"制度について法情報を探したいです。",
	)
	if contribution.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"SOT-MODEL-026: selection mode = %q, want %q",
			contribution.SelectionMode(),
			legalquery.QuerySelectionModeClarificationRequired,
		)
	}
	candidates := contribution.Candidates()
	if len(candidates) != 2 {
		t.Fatalf(
			"SOT-MODEL-026: 弱い一般語の candidates = %#v, want 2 件",
			candidates,
		)
	}
	assertLawSearchCandidate(
		t,
		candidates[0],
		"candidate-1-1",
		"制度",
	)
	assertLawContentSearchCandidate(
		t,
		candidates[1],
		"candidate-1-2",
		[]string{"制度"},
	)
	if len(contribution.HedgePairs()) != 0 {
		t.Fatalf(
			"SOT-ARCH-023: 弱い一般語の hedge pairs = %#v, want 0 件",
			contribution.HedgePairs(),
		)
	}
}

func TestProfileContributionは衝突する暫定法略称を明確化必須にする(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"暫定法の候補となる法令を示してください。",
	)
	if contribution.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"SOT-MODEL-026: selection mode = %q, want %q",
			contribution.SelectionMode(),
			legalquery.QuerySelectionModeClarificationRequired,
		)
	}
	candidates := contribution.Candidates()
	if len(candidates) != 2 {
		t.Fatalf(
			"SOT-MODEL-026: 暫定法の candidates = %#v, want 2 件",
			candidates,
		)
	}
	assertLawSearchCandidate(
		t,
		candidates[0],
		"candidate-1-1",
		"農林水産業施設災害復旧事業費国庫補助の暫定措置に関する法律",
	)
	assertLawSearchCandidate(
		t,
		candidates[1],
		"candidate-1-2",
		"関税暫定措置法",
	)
	if len(contribution.HedgePairs()) != 0 {
		t.Fatalf(
			"SOT-MODEL-026: 略称衝突の hedge pairs = %#v, want 0 件",
			contribution.HedgePairs(),
		)
	}
}

func TestProfileContributionは明示した二候補だけをhedgePairにする(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"実行検証用に保証を法令名と条文の二候補で検索してください。",
	)
	if contribution.SelectionMode() != legalquery.QuerySelectionModeAutomatic {
		t.Fatalf(
			"SOT-MODEL-026: selection mode = %q, want %q",
			contribution.SelectionMode(),
			legalquery.QuerySelectionModeAutomatic,
		)
	}
	candidates := contribution.Candidates()
	if len(candidates) != 2 {
		t.Fatalf(
			"SOT-MODEL-026: 明示二候補の candidates = %#v, want 2 件",
			candidates,
		)
	}
	assertLawAndContentSearchCandidates(t, candidates, "保証")
	assertSingleHedgePair(
		t,
		contribution,
		"candidate-1-1",
		"candidate-1-2",
	)
}

func TestProfileContributionは異なる単独候補法概念をhedgePairにする(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"育休について法情報を調べてください。",
	)
	if contribution.SelectionMode() != legalquery.QuerySelectionModeAutomatic {
		t.Fatalf(
			"SOT-MODEL-026: selection mode = %q, want %q",
			contribution.SelectionMode(),
			legalquery.QuerySelectionModeAutomatic,
		)
	}
	candidates := contribution.Candidates()
	if len(candidates) != 2 {
		t.Fatalf(
			"SOT-MODEL-026: 育休の candidates = %#v, want 2 件",
			candidates,
		)
	}
	assertLawContentSearchCandidate(
		t,
		candidates[0],
		"candidate-1-1",
		[]string{"育児休業"},
	)
	assertLawContentSearchCandidate(
		t,
		candidates[1],
		"candidate-1-2",
		[]string{"育児休業給付"},
	)
	if got := candidates[0].ConceptSources(); len(got) != 1 ||
		got[0].ConceptID() != "childcare-leave" {
		t.Fatalf(
			"SOT-MODEL-026: 第一候補の concept sources = %#v",
			got,
		)
	}
	if got := candidates[1].ConceptSources(); len(got) != 1 ||
		got[0].ConceptID() != "childcare-leave-benefit" {
		t.Fatalf(
			"SOT-MODEL-026: 第二候補の concept sources = %#v",
			got,
		)
	}
	assertSingleHedgePair(
		t,
		contribution,
		"candidate-1-1",
		"candidate-1-2",
	)
}

func TestProfileContributionは五主題を候補ゼロでも明確化必須にする(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"量子相続、月面抵当、火星登記、宇宙帰化、海底供託について教えてください。",
	)
	if len(contribution.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 五主題の candidates = %#v, want 0 件",
			contribution.Candidates(),
		)
	}
	if contribution.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"SOT-MODEL-026: selection mode = %q, want %q",
			contribution.SelectionMode(),
			legalquery.QuerySelectionModeClarificationRequired,
		)
	}
	if len(contribution.HedgePairs()) != 0 {
		t.Fatalf(
			"SOT-MODEL-026: 五主題の hedge pairs = %#v, want 0 件",
			contribution.HedgePairs(),
		)
	}
}

func TestProfileContributionは翻訳信号と法令検索候補を併存させる(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"民法を検索し英訳して",
	)
	candidates := contribution.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"SOT-MODEL-026: 混在要求の candidates = %#v, want 1 件",
			candidates,
		)
	}
	steps := candidates[0].Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch {
		t.Fatalf(
			"SOT-MODEL-026: 混在要求の law search step = %#v",
			steps,
		)
	}
	input, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || input.Query() != "民法" {
		t.Fatalf(
			"SOT-MODEL-026: 混在要求の logical input = %#v",
			steps[0].LogicalInput(),
		)
	}
	if !hasProfileSignal(
		contribution.Signals(),
		"unsupported_translation",
	) {
		t.Fatalf(
			"SOT-MODEL-026: translation signal がありません: %#v",
			contribution.Signals(),
		)
	}
}

func TestProfileContributionは対象外資源だけの弱い候補を除去する(
	t *testing.T,
) {
	t.Parallel()

	contribution := generateSelectionContribution(
		t,
		"都道府県の未公開内部文書を横断検索してください。",
	)
	if len(contribution.Candidates()) != 0 {
		t.Fatalf(
			"SOT-MODEL-026: 対象外資源だけの candidates = %#v, want 0 件",
			contribution.Candidates(),
		)
	}
	if !hasProfileSignal(
		contribution.Signals(),
		"unsupported_task_or_resource",
	) {
		t.Fatalf(
			"SOT-MODEL-026: unsupported resource signal がありません: %#v",
			contribution.Signals(),
		)
	}
}

func generateSelectionContribution(
	t *testing.T,
	query string,
) legalquery.QueryProfileContribution {
	t.Helper()

	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("製品前処理器を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
	})
	if err != nil {
		t.Fatalf("統合法情報照会 request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("製品前処理のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("profile 入力を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("candidate ID scope を構築できません: %v", err)
	}
	contribution, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("core profile contribution のエラー = %v", err)
	}
	return contribution
}

func assertSingleHedgePair(
	t *testing.T,
	contribution legalquery.QueryProfileContribution,
	firstCandidateID string,
	secondCandidateID string,
) {
	t.Helper()

	pairs := contribution.HedgePairs()
	if len(pairs) != 1 {
		t.Fatalf(
			"SOT-MODEL-026: hedge pairs = %#v, want 1 件",
			pairs,
		)
	}
	if pairs[0].FirstCandidateID() != firstCandidateID ||
		pairs[0].SecondCandidateID() != secondCandidateID {
		t.Fatalf(
			"SOT-MODEL-026: hedge pair = (%q, %q), want (%q, %q)",
			pairs[0].FirstCandidateID(),
			pairs[0].SecondCandidateID(),
			firstCandidateID,
			secondCandidateID,
		)
	}
}

func assertLawSearchCandidate(
	t *testing.T,
	candidate legalquery.LegalQueryCandidate,
	candidateID string,
	query string,
) {
	t.Helper()

	if candidate.CandidateID() != candidateID {
		t.Fatalf(
			"SOT-MODEL-026: candidateId = %q, want %q",
			candidate.CandidateID(),
			candidateID,
		)
	}
	steps := candidate.Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawSearch {
		t.Fatalf(
			"SOT-MODEL-026: %s の law search step = %#v",
			candidateID,
			steps,
		)
	}
	input, ok := steps[0].LogicalInput().(legalquery.LawSearchIntentV1)
	if !ok || input.Query() != query {
		t.Fatalf(
			"SOT-MODEL-026: %s の law search input = %#v, want %q",
			candidateID,
			steps[0].LogicalInput(),
			query,
		)
	}
}

func assertLawContentSearchCandidate(
	t *testing.T,
	candidate legalquery.LegalQueryCandidate,
	candidateID string,
	allTerms []string,
) {
	t.Helper()

	if candidate.CandidateID() != candidateID {
		t.Fatalf(
			"SOT-MODEL-026: candidateId = %q, want %q",
			candidate.CandidateID(),
			candidateID,
		)
	}
	steps := candidate.Steps()
	if len(steps) != 1 ||
		steps[0].InputKind() != legalquery.InputKindLawContentSearch {
		t.Fatalf(
			"SOT-MODEL-026: %s の law content search step = %#v",
			candidateID,
			steps,
		)
	}
	input, ok := steps[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
	if !ok ||
		!slices.Equal(input.AllTerms(), allTerms) ||
		len(input.AnyTerms()) != 0 ||
		len(input.ExcludeTerms()) != 0 {
		t.Fatalf(
			"SOT-MODEL-026: %s の law content input = %#v, want allTerms=%#v",
			candidateID,
			steps[0].LogicalInput(),
			allTerms,
		)
	}
}

func assertLawAndContentSearchCandidates(
	t *testing.T,
	candidates []legalquery.LegalQueryCandidate,
	query string,
) {
	t.Helper()

	if len(candidates) != 2 ||
		candidates[0].CandidateID() != "candidate-1-1" ||
		candidates[1].CandidateID() != "candidate-1-2" {
		t.Fatalf(
			"SOT-MODEL-026: 明示二候補の candidate IDs = %#v",
			candidates,
		)
	}
	lawSearchFound := false
	contentSearchFound := false
	for _, candidate := range candidates {
		steps := candidate.Steps()
		if len(steps) != 1 {
			t.Fatalf(
				"SOT-MODEL-026: %s の steps = %#v, want 1 件",
				candidate.CandidateID(),
				steps,
			)
		}
		switch steps[0].InputKind() {
		case legalquery.InputKindLawSearch:
			assertLawSearchCandidate(
				t,
				candidate,
				candidate.CandidateID(),
				query,
			)
			lawSearchFound = true
		case legalquery.InputKindLawContentSearch:
			assertLawContentSearchCandidate(
				t,
				candidate,
				candidate.CandidateID(),
				[]string{query},
			)
			contentSearchFound = true
		default:
			t.Fatalf(
				"SOT-MODEL-026: %s の inputKind = %q",
				candidate.CandidateID(),
				steps[0].InputKind(),
			)
		}
	}
	if !lawSearchFound || !contentSearchFound {
		t.Fatalf(
			"SOT-MODEL-026: 法令名検索と条文検索の二候補ではありません: %#v",
			candidates,
		)
	}
}

func hasProfileSignal[T ~string](signals []T, expected string) bool {
	for _, signal := range signals {
		if string(signal) == expected {
			return true
		}
	}
	return false
}
