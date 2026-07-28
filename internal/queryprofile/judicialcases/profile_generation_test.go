package judicialcases

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileは引用語と形態素語を裁判例検索へ変換する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		query     string
		wantQuery string
	}{
		{
			name:      "形態素語",
			query:     "医療過誤の裁判例を検索してください。",
			wantQuery: "医療過誤",
		},
		{
			name:      "引用された事件番号",
			query:     "「令和5年（受）第123号」の判例を検索してください。",
			wantQuery: "令和5年（受）第123号",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, nil)
			candidate := findSearchCandidate(t, generation, test.wantQuery)
			if !slices.Equal(
				candidate.RequiredPacks(),
				[]string{"judicial-cases"},
			) {
				t.Fatalf("required packs = %#v", candidate.RequiredPacks())
			}
			if !slices.Equal(
				candidate.EvidenceCodes(),
				[]legalquery.EvidenceCode{
					legalquery.EvidenceExplicitTask,
					legalquery.EvidenceExplicitResource,
					legalquery.EvidenceMorphologicalContext,
				},
			) {
				t.Fatalf("evidence = %#v", candidate.EvidenceCodes())
			}
		})
	}
}

func TestProfileは日付surfaceを構造化参照検索へ保持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"2024年1月15日の裁判例を検索してください。",
		nil,
	)
	candidate := findSearchCandidate(t, generation, "2024年1月15日")
	if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
	}) {
		t.Fatalf("日付検索 evidence = %#v", candidate.EvidenceCodes())
	}
}

func TestProfileは裁判例向け法概念の正式語と公的出典を保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"ネット中傷の裁判例を検索してください。",
		nil,
	)
	candidate := findSearchCandidate(t, generation, "名誉毀損")
	sources := candidate.ConceptSources()
	if len(sources) != 1 ||
		sources[0].ConceptID() != "online-defamation" ||
		sources[0].Title() != "警察庁 インターネット上の誹謗中傷等への対応" ||
		sources[0].URL() !=
			"https://www.npa.go.jp/bureau/cyber/countermeasures/defamation.html" ||
		sources[0].ConfirmedOn().String() != "2026-07-28" {
		t.Fatalf("concept sources = %#v", sources)
	}
	if !slices.Contains(
		candidate.EvidenceCodes(),
		legalquery.EvidenceLegalConcept,
	) {
		t.Fatalf("concept evidence = %#v", candidate.EvidenceCodes())
	}
	if generation.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf("selection mode = %q", generation.SelectionMode())
	}
}

func TestProfileは個別検索を最大四stepまで原文順に保持する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「医療過誤」と「損害賠償」を個別に裁判例検索してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("個別検索 candidates = %#v", candidates)
	}
	steps := candidates[0].Steps()
	if len(steps) != 2 {
		t.Fatalf("個別検索 steps = %#v", steps)
	}
	got := make([]string, 0, len(steps))
	for _, step := range steps {
		input, ok := step.LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if !ok {
			t.Fatalf("logical input = %T", step.LogicalInput())
		}
		got = append(got, input.Query())
	}
	if !slices.Equal(got, []string{"医療過誤", "損害賠償"}) {
		t.Fatalf("検索 step 順 = %#v", got)
	}
	if !slices.Equal(
		candidates[0].RequiredPacks(),
		[]string{"judicial-cases"},
	) {
		t.Fatalf("required packs = %#v", candidates[0].RequiredPacks())
	}
}

func TestProfileは五件の個別検索を切り捨てず明確化する(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「医療過誤」、「損害賠償」、「養育費」、「成年後見」、「名誉毀損」を個別に裁判例検索してください。",
		nil,
	)
	if len(generation.Candidates()) != 0 ||
		generation.SelectionMode() !=
			legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"五件検索 generation = candidates:%#v mode:%q",
			generation.Candidates(),
			generation.SelectionMode(),
		)
	}
}

func TestProfileはrefなしでread候補を作らない(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
		ref   *model.SourceResourceRef
	}{
		{
			name:  "ref なし",
			query: "最高裁判例を取得してください。",
		},
		{
			name:  "題名から推測しない",
			query: "「最高裁令和5年判決」の裁判例本文を取得してください。",
		},
		{
			name:  "URL から推測しない",
			query: "「https://www.courts.go.jp/hanrei/123」の裁判例本文を取得してください。",
		},
		{
			name:  "法令 ref を転用しない",
			query: "指定参照の最高裁判例を取得してください。",
			ref: func() *model.SourceResourceRef {
				ref := newTestRef(t, "law", "129AC0000000089")
				return &ref
			}(),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			generation := generateQuery(t, test.query, test.ref)
			for _, candidate := range generation.Candidates() {
				for _, step := range candidate.Steps() {
					if step.InputKind() ==
						legalquery.InputKindJudicialDecisionRead {
						t.Fatalf("read 候補を推測しました: %#v", candidate)
					}
				}
			}
		})
	}
}

func TestProfileは裁判例resourceなしの一般検索を引き取らない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(t, "医療過誤を検索してください。", nil)
	if len(generation.Candidates()) != 0 {
		t.Fatalf("resource 根拠なしの candidates = %#v", generation.Candidates())
	}
}

func TestProfileは非日本語入力の意味候補を作らない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(t, "search judicial cases", nil)
	if len(generation.Candidates()) != 0 ||
		!slices.Equal(
			generation.Signals(),
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalNonJapaneseQuery,
			},
		) {
		t.Fatalf(
			"non-Japanese generation = candidates:%#v signals:%#v",
			generation.Candidates(),
			generation.Signals(),
		)
	}
}

func TestProfileは位置付きfactがない原文を再解析しない(t *testing.T) {
	t.Parallel()

	const query = "医療過誤の裁判例を検索してください。"
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         query,
			ComparisonKey: querynormalization.ComparisonKey(query),
		},
	)
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
	generation, err := mustProfile(t).Generate(input, scope)
	if err != nil {
		t.Fatalf("Generate() のエラー = %v", err)
	}
	if len(generation.Candidates()) != 0 {
		t.Fatalf(
			"SOT-ARCH-021: 原文を再解析しました: %#v",
			generation.Candidates(),
		)
	}
}

func findSearchCandidate(
	t *testing.T,
	generation legalquery.CandidateGeneration,
	query string,
) legalquery.LegalQueryCandidate {
	t.Helper()

	for _, candidate := range generation.Candidates() {
		steps := candidate.Steps()
		if len(steps) != 1 {
			continue
		}
		input, ok := steps[0].LogicalInput().(legalquery.JudicialDecisionSearchIntentV1)
		if ok && input.Query() == query {
			return candidate
		}
	}
	t.Fatalf(
		"query %q の裁判例検索候補がありません: candidates=%#v",
		query,
		generation.Candidates(),
	)
	return legalquery.LegalQueryCandidate{}
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
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("generation input を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を構築できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf(
			"Generate() のエラー = %v; cues=%#v terms=%#v concepts=%#v dates=%#v",
			err,
			preprocessed.CueMentions(),
			preprocessed.QueryTermMentions(),
			preprocessed.LegalConceptMentions(),
			preprocessed.DateMentions(),
		)
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
