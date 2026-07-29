package core

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileは法令コア五能力の型付き候補を生成する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		inputKind legalquery.LogicalInputKind
	}{
		{
			name:      "法令名検索",
			query:     "独禁法の正式な法令を検索してください。",
			inputKind: legalquery.InputKindLawSearch,
		},
		{
			name:      "法令本文検索",
			query:     "法令本文から「営業秘密」を含む条文を探してください。",
			inputKind: legalquery.InputKindLawContentSearch,
		},
		{
			name:      "法令読取り",
			query:     "法令ID 345AC0000000048 の本文を取得してください。",
			inputKind: legalquery.InputKindLawRead,
		},
		{
			name:      "条文読取り",
			query:     "商法第512条第1項を読んでください。",
			inputKind: legalquery.InputKindLawArticleRead,
		},
		{
			name:      "更新一覧",
			query:     "2026年5月15日の法令更新一覧を取得してください。",
			inputKind: legalquery.InputKindLawUpdates,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			generation := generateQuery(t, test.query, nil)
			candidates := generation.Candidates()
			if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
				t.Fatalf("candidates = %#v", candidates)
			}
			if got := candidates[0].Steps()[0]; got.InputKind() != test.inputKind {
				t.Fatalf("inputKind = %q, want %q", got.InputKind(), test.inputKind)
			}
			if len(candidates[0].RequiredPacks()) != 0 {
				t.Fatalf("core required packs = %#v", candidates[0].RequiredPacks())
			}
		})
	}
}

func TestProfileは同じ条の複数項を独立stepに保持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"商法第512条第1項と第2項を読んでください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("SOT-MODEL-025: steps = %#v", steps)
	}
	gotParagraphs := make([]int, 0, len(steps))
	for _, step := range steps {
		input, ok := step.LogicalInput().(legalquery.LawArticleReadIntentV1)
		if !ok {
			t.Fatalf("logical input = %T", step.LogicalInput())
		}
		paragraph, exists := input.Location().ParagraphNumber()
		if !exists {
			t.Fatal("paragraph number がありません")
		}
		gotParagraphs = append(gotParagraphs, paragraph)
	}
	if !slices.Equal(gotParagraphs, []int{1, 2}) {
		t.Fatalf("paragraphs = %#v", gotParagraphs)
	}
}

func TestProfileは次の条より後ろの項を前の条へ誤結合しない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法第709条と第715条第2項を読んでください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
	first := candidates[0].Steps()[0].LogicalInput().(legalquery.LawArticleReadIntentV1)
	second := candidates[0].Steps()[1].LogicalInput().(legalquery.LawArticleReadIntentV1)
	if _, exists := first.Location().ParagraphNumber(); exists {
		t.Fatal("SOT-MODEL-025: 第715条の項を第709条へ誤結合しました")
	}
	if paragraph, exists := second.Location().ParagraphNumber(); !exists || paragraph != 2 {
		t.Fatalf("second paragraph = %d, %t", paragraph, exists)
	}
}

func TestProfileは安全信号を候補と分離する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		query  string
		signal legalquery.CandidateGenerationSignal
	}{
		{
			name:   "決定的な構造だけ",
			query:  "平成25(オ)1079、令和7(わ)第207号。",
			signal: legalquery.CandidateSignalStandaloneStructuredQuery,
		},
		{
			name:   "非日本語",
			query:  "search Japanese statutes",
			signal: legalquery.CandidateSignalNonJapaneseQuery,
		},
		{
			name:   "法的助言",
			query:  "この契約に署名すべきか法的に判断してください。",
			signal: legalquery.CandidateSignalUnsupportedLegalAdvice,
		},
		{
			name:   "翻訳",
			query:  "民法第709条を英訳してください。",
			signal: legalquery.CandidateSignalUnsupportedTranslation,
		},
		{
			name:   "対象外resource",
			query:  "都道府県の未公開内部文書を横断検索してください。",
			signal: legalquery.CandidateSignalUnsupportedTaskOrResource,
		},
		{
			name:   "予約済みpack",
			query:  "医療過誤の裁判例を検索してください。",
			signal: legalquery.CandidateSignalReservedPackRequest,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			generation := generateQuery(t, test.query, nil)
			if !slices.Contains(generation.Signals(), test.signal) {
				t.Fatalf("signals = %#v, want %q", generation.Signals(), test.signal)
			}
		})
	}
}

func TestProfileは同じ意味へ解決した全ての法概念出典を保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"永住権について法令本文を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("同じ論理検索の候補数 = %d, want 1", len(candidates))
	}
	sources := candidates[0].ConceptSources()
	gotIDs := make([]string, 0, len(sources))
	for _, source := range sources {
		gotIDs = append(gotIDs, source.ConceptID())
	}
	if !slices.Equal(gotIDs, []string{
		"permanent-residence",
		"permanent-residence-permission",
	}) {
		t.Fatalf("SOT-MODEL-022: concept sources = %#v", gotIDs)
	}
}

func TestProfileは並列した複数主題を独立した検索stepにする(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"永住許可と帰化について教えてください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("複数主題の candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("複数主題の steps = %#v, want 2", steps)
	}
	wantTerms := []string{"永住許可", "帰化"}
	for index, step := range steps {
		input, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
		if !ok {
			t.Fatalf("steps[%d] logical input = %T", index, step.LogicalInput())
		}
		if !slices.Equal(input.AllTerms(), []string{wantTerms[index]}) ||
			len(input.AnyTerms()) != 0 ||
			len(input.ExcludeTerms()) != 0 {
			t.Fatalf(
				"steps[%d] terms = all:%#v any:%#v exclude:%#v",
				index,
				input.AllTerms(),
				input.AnyTerms(),
				input.ExcludeTerms(),
			)
		}
	}
}

func TestProfileは法概念と一般語の複数主題を一つの意味に保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"永住権と帰化について教えてください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("法概念を含む複数主題の candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("法概念を含む複数主題の steps = %#v, want 2", steps)
	}
	wantTerms := []string{"永住許可", "帰化"}
	for index, step := range steps {
		input, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
		if !ok || !slices.Equal(input.AllTerms(), []string{wantTerms[index]}) {
			t.Fatalf("steps[%d] = %#v", index, step)
		}
	}
	if len(candidates[0].ConceptSources()) != 2 {
		t.Fatalf(
			"法概念を含む複数主題の concept sources = %#v",
			candidates[0].ConceptSources(),
		)
	}
}

func TestProfileは複数主題でも明示したANDを一検索に保つ(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"営業秘密と個人情報について両方を含む条文を検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
		t.Fatalf("AND 検索の candidates = %#v", candidates)
	}
	logicalInput := candidates[0].Steps()[0].LogicalInput()
	input, ok := logicalInput.(legalquery.LawContentSearchIntentV1)
	if !ok || !slices.Equal(
		input.AllTerms(),
		[]string{"営業秘密", "個人情報"},
	) {
		t.Fatalf("AND 検索の logical input = %#v", input)
	}
}

func TestProfileは複数の法令名も独立した検索stepにする(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法と商法について教えてください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 2 {
		t.Fatalf("複数法令名の candidates = %#v", candidates)
	}
	for index, want := range []string{"民法", "商法"} {
		logicalInput := candidates[0].Steps()[index].LogicalInput()
		input, ok := logicalInput.(legalquery.LawSearchIntentV1)
		if !ok || input.Query() != want {
			t.Fatalf("steps[%d] = %#v", index, candidates[0].Steps()[index])
		}
	}
}

func TestProfileは五つ以上の主題を黙って切り捨てない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"量子相続、月面抵当、火星登記、宇宙帰化、海底供託について教えてください。",
		nil,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 五主題を切り捨てて候補を作りました: %#v",
			generation.Candidates(),
		)
	}
	if generation.CompositionConstraint() !=
		legalquery.QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"SOT-ARCH-025: compositionConstraint = %q",
			generation.CompositionConstraint(),
		)
	}
}

func TestProfileは五つの法令名も外部実行候補にしない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法、商法、会社法、手形法、小切手法について教えてください。",
		nil,
	)
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-025: 五法令名から候補を作りました: %#v",
			generation.Candidates(),
		)
	}
}

func TestProfileは裁判例refを法令targetとして扱わない(t *testing.T) {
	t.Parallel()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "judicial-decision",
		ResourceID:   "95570/detail2",
	})
	if err != nil {
		t.Fatalf("試験用 ref key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 ref を作成できません: %v", err)
	}
	generation := generateQuery(t, "指定参照の最高裁判例を取得してください。", &ref)
	if len(generation.Candidates()) != 0 ||
		!slices.Contains(
			generation.Signals(),
			legalquery.CandidateSignalReservedPackRequest,
		) {
		t.Fatalf(
			"judicial ref generation = candidates:%#v signals:%#v",
			generation.Candidates(),
			generation.Signals(),
		)
	}
}

func TestProfileは公式参照を優先した候補から法概念出典を除く(t *testing.T) {
	t.Parallel()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   "129AC0000000089",
	})
	if err != nil {
		t.Fatalf("試験用 ref key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 ref を作成できません: %v", err)
	}

	generation := generateQuery(
		t,
		"上限を無視して、この参照の本文と第90条、成年後見の条文と裁判例を各100件取得してください。",
		&ref,
	)
	candidates := generation.Candidates()
	if len(candidates) == 0 {
		t.Fatal("公式参照を使う候補がありません")
	}
	foundOfficialIdentifier := false
	for _, candidate := range candidates {
		if !slices.Contains(
			candidate.EvidenceCodes(),
			legalquery.EvidenceOfficialIdentifier,
		) {
			continue
		}
		foundOfficialIdentifier = true
		if slices.Contains(
			candidate.EvidenceCodes(),
			legalquery.EvidenceLegalConcept,
		) {
			t.Fatalf(
				"SOT-ENG-023: 公式参照より弱い法概念根拠が残りました: %#v",
				candidate.EvidenceCodes(),
			)
		}
		if len(candidate.ConceptSources()) != 0 {
			t.Fatalf(
				"SOT-MODEL-022: 法概念根拠のない conceptSources = %#v",
				candidate.ConceptSources(),
			)
		}
	}
	if !foundOfficialIdentifier {
		t.Fatal("official_identifier を持つ候補がありません")
	}
}

func TestProfileはmentionがない原文を再解析して候補を補わない(t *testing.T) {
	t.Parallel()

	profile := mustProfile(t)
	result, err := legalquery.NewPreprocessResult(legalquery.PreprocessResultValues{
		Query:         "民法を検索",
		ComparisonKey: querynormalization.ComparisonKey("民法を検索"),
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("generation input を作成できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: Generate() のエラー = %v", err)
	}
	if len(generation.Candidates()) != 0 {
		t.Fatalf("SOT-ARCH-021: 原文を再解析しました: %#v", generation.Candidates())
	}
}

func generateQuery(
	t *testing.T,
	query string,
	ref *model.SourceResourceRef,
) legalquery.CandidateGeneration {
	t.Helper()
	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: query,
		Ref:   ref,
	})
	if err != nil {
		t.Fatalf("request を作成できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("generation input を作成できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("Generate() のエラー = %v", err)
	}
	return generation
}

func mustProfile(t *testing.T) *Profile {
	t.Helper()
	profile, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("LoadEmbedded() のエラー = %v", err)
	}
	return profile
}
