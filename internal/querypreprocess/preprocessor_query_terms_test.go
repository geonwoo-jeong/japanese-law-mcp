package querypreprocess_test

import (
	"context"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func TestPreprocessExtractsQuotedPhrasesWithoutLosingOriginalSurface(t *testing.T) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	const query = "裁判例で「著作権 演奏権」と「検索」を検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	terms := result.QueryTermMentions()
	want := []string{"著作権 演奏権", "検索"}
	if len(terms) != len(want) {
		t.Fatalf("SOT-MODEL-025: queryTermMentions = %#v", terms)
	}
	for index, mention := range terms {
		if mention.Surface() != want[index] ||
			mention.Kind() != legalquery.QueryTermMentionQuotedPhrase {
			t.Fatalf(
				"SOT-MODEL-025: queryTermMentions[%d] = %#v",
				index,
				mention,
			)
		}
		assertSpan(
			t,
			query,
			mention.Span(),
			mention.Surface(),
			strings.Index(query, mention.Surface()),
		)
	}
	if len(result.CueMentions()) < 3 {
		t.Fatalf(
			"SOT-MODEL-025: 引用した cue と外側の cue が失われました: %#v",
			result.CueMentions(),
		)
	}
}

func TestPreprocessExtractsOnlyGrammaticallyAnchoredMorphologicalPhrases(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	for _, test := range []struct {
		name  string
		query string
		want  []string
	}{
		{
			name:  "関係節",
			query: "営業秘密に関する条文を検索",
			want:  []string{"営業秘密"},
		},
		{
			name:  "task cue の直接目的語",
			query: "供託制度を教えて",
			want:  []string{"供託制度"},
		},
		{
			name:  "task cue の手段を表す名詞句",
			query: "民法を最大件数で検索",
			want:  []string{"最大件数"},
		},
		{
			name:  "上限語を含む手段句",
			query: "上限内で検索してください。",
			want:  []string{"上限内"},
		},
		{
			name:  "法概念の後続文脈",
			query: "育休の給付について調べて",
			want:  []string{"給付"},
		},
		{
			name:  "制御語を検索語にしない",
			query: "実行検証用に供託を法令名と条文の二候補で検索",
			want:  []string{"供託"},
		},
		{
			name:  "並列された対象",
			query: "量子相続登記と月面抵当権を含む条文を検索",
			want:  []string{"量子相続登記", "月面抵当権"},
		},
		{
			name:  "resource cue の属格",
			query: "医療過誤の裁判例を検索",
			want:  []string{"医療過誤"},
		},
		{
			name:  "単独の弱い一般語",
			query: "供託",
			want:  []string{"供託"},
		},
		{
			name:  "数詞修飾を含む免許",
			query: "第2種免許を検索",
			want:  []string{"第2種免許"},
		},
		{
			name:  "数詞修飾を含む被保険者",
			query: "第1号被保険者を検索",
			want:  []string{"第1号被保険者"},
		},
		{
			name:  "一文字の一般語を自動採用しない",
			query: "税",
			want:  nil,
		},
		{
			name:  "非日本語の名詞句を自動採用しない",
			query: "deleteを検索",
			want:  nil,
		},
		{
			name:  "述語を越えた cue で目的語を採用しない",
			query: "営業秘密を説明して判例を検索",
			want:  nil,
		},
		{
			name:  "従属節を越えた cue で目的語を採用しない",
			query: "不法行為を確認してから裁判例を検索",
			want:  nil,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			terms := result.QueryTermMentions()
			if len(terms) != len(test.want) {
				t.Fatalf(
					"SOT-MODEL-025: %q の queryTermMentions = %#v, want %#v",
					test.query,
					terms,
					test.want,
				)
			}
			for index, mention := range terms {
				if mention.Surface() != test.want[index] ||
					mention.Kind() !=
						legalquery.QueryTermMentionMorphologicalPhrase {
					t.Fatalf(
						"SOT-MODEL-025: queryTermMentions[%d] = %#v, want %q",
						index,
						mention,
						test.want[index],
					)
				}
				assertSpan(
					t,
					test.query,
					mention.Span(),
					mention.Surface(),
					strings.Index(test.query, mention.Surface()),
				)
			}
		})
	}
}

func TestPreprocessDoesNotDuplicateTypedFactsAsMorphologicalPhrases(t *testing.T) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	const query = "2026年7月1日時点の民法第709条第1項を検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	if terms := result.QueryTermMentions(); len(terms) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: typed fact を一般検索語に重複させました: %#v",
			terms,
		)
	}
	if len(result.LawNameMentions()) != 1 ||
		len(result.DateMentions()) != 1 ||
		len(result.ArticleMentions()) != 1 ||
		len(result.ParagraphMentions()) != 1 {
		t.Fatalf(
			"SOT-MODEL-025: typed fact が失われました: %#v",
			snapshotResult(result),
		)
	}
}

func TestPreprocessDoesNotUseQuotedCueOrCaseNumberAsMorphologicalPhrase(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	for _, test := range []struct {
		name       string
		query      string
		wantQuoted []string
	}{
		{
			name:       "引用内の cue は別の句を起動しない",
			query:      "説明を「検索」",
			wantQuoted: []string{"検索"},
		},
		{
			name:  "事件番号は一般検索語にしない",
			query: "平成19(行ツ)164 の裁判例を検索",
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			quoted := make([]string, 0)
			for _, mention := range result.QueryTermMentions() {
				if mention.Kind() ==
					legalquery.QueryTermMentionMorphologicalPhrase {
					t.Fatalf(
						"SOT-MODEL-025: 一般名詞句を誤って採用しました: %#v",
						result.QueryTermMentions(),
					)
				}
				quoted = append(quoted, mention.Surface())
			}
			if len(quoted) != len(test.wantQuoted) {
				t.Fatalf(
					"SOT-MODEL-025: quoted phrases = %#v, want %#v",
					quoted,
					test.wantQuoted,
				)
			}
			for index := range quoted {
				if quoted[index] != test.wantQuoted[index] {
					t.Fatalf(
						"SOT-MODEL-025: quoted phrases = %#v, want %#v",
						quoted,
						test.wantQuoted,
					)
				}
			}
		})
	}
}

func TestPreprocessHandlesRepeatedTrimmedAndMalformedQuotesDeterministically(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	for _, test := range []struct {
		name       string
		query      string
		wantQuoted []string
	}{
		{
			name:       "同じ語の別 span",
			query:      "「営業秘密」と「営業秘密」を検索",
			wantQuoted: []string{"営業秘密", "営業秘密"},
		},
		{
			name:       "引用内部の Unicode 空白",
			query:      "『　営業秘密　』を検索",
			wantQuoted: []string{"営業秘密"},
		},
		{
			name:       "ASCII 引用の内部空白と英語",
			query:      "\"delete all\"を検索",
			wantQuoted: []string{"delete all"},
		},
		{
			name:       "曲線引用の演算子記号",
			query:      "“A|B”を検索",
			wantQuoted: []string{"A|B"},
		},
		{
			name:       "一文字の明示引用",
			query:      "「税」を検索",
			wantQuoted: []string{"税"},
		},
		{
			name:  "不完全な引用",
			query: "「営業秘密を検索",
		},
		{
			name:  "入れ子の引用",
			query: "「『営業秘密』」を検索",
		},
		{
			name:  "空白だけの引用",
			query: "“　 ”を検索",
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			quoted := make([]string, 0)
			for _, mention := range result.QueryTermMentions() {
				if mention.Kind() == legalquery.QueryTermMentionQuotedPhrase {
					quoted = append(quoted, mention.Surface())
				}
			}
			if len(quoted) != len(test.wantQuoted) {
				t.Fatalf(
					"SOT-MODEL-025: quoted phrases = %#v, want %#v",
					quoted,
					test.wantQuoted,
				)
			}
			for index := range quoted {
				if quoted[index] != test.wantQuoted[index] {
					t.Fatalf(
						"SOT-MODEL-025: quoted phrases = %#v, want %#v",
						quoted,
						test.wantQuoted,
					)
				}
			}
		})
	}
}

func TestPreprocessDoesNotReinterpretMalformedQuotesAsMorphologicalPhrases(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	for _, query := range []string{
		"「営業秘密を検索",
		"「『営業秘密』」を検索",
		"『営業秘密」を検索",
		"説明を「検索",
		"説明を「『検索』」",
		"\"判例 \"民法\"\"を検索",
		"\"営業秘密 \"検索\"\"を検索",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			if terms := result.QueryTermMentions(); len(terms) != 0 {
				t.Fatalf(
					"SOT-MODEL-025: 不正な引用内を一般検索語に再解釈しました: %#v",
					terms,
				)
			}
		})
	}
}

func TestPreprocessKeepsQuotedPhraseThatAlsoMatchesLawName(t *testing.T) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	const query = "裁判例で「民法」を検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	terms := result.QueryTermMentions()
	laws := result.LawNameMentions()
	if len(terms) != 1 ||
		terms[0].Surface() != "民法" ||
		terms[0].Kind() != legalquery.QueryTermMentionQuotedPhrase ||
		len(laws) != 1 ||
		laws[0].Surface() != "民法" {
		t.Fatalf(
			"SOT-MODEL-025: 引用と法令名の併存 = terms:%#v laws:%#v",
			terms,
			laws,
		)
	}
}

func TestPreprocessFailsInsteadOfTruncatingQuotedPhraseOverflow(t *testing.T) {
	t.Parallel()

	preprocessor := mustQueryTermPreprocessor(t)
	query := strings.Repeat("「語」", 65)
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err == nil {
		t.Fatalf(
			"SOT-MODEL-025: 六十四件を超える引用句を受理しました: %#v",
			snapshotResult(result),
		)
	}
	if len(result.QueryTermMentions()) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: エラー時に途中の query terms を返しました: %#v",
			result.QueryTermMentions(),
		)
	}
}

func mustQueryTermPreprocessor(t *testing.T) legalquery.QueryPreprocessor {
	t.Helper()

	concepts := append(
		testConceptEntries(),
		legalconceptlexicon.Entry{
			ConceptID:       "childcare-leave",
			Canonical:       "育休",
			Terms:           []string{"育休"},
			ComparisonTerms: []string{"育休"},
			SourceName:      "試験用の公的情報源",
			SourceURL:       "https://example.go.jp/childcare-leave",
			ConfirmedAt:     "2026-07-28",
			MappingNote:     "試験用の法概念対応",
			SelectionPolicy: legalconceptlexicon.SelectionPolicySingleCandidate,
			Candidates: []legalconceptlexicon.Candidate{
				{
					Task:         legalquery.TaskSearch,
					Resource:     legalquery.ResourceLawProvision,
					InputKind:    legalquery.InputKindLawContentSearch,
					OfficialTerm: "育児休業",
				},
			},
		},
	)
	return mustNewPreprocessor(
		t,
		testLawEntries(),
		concepts,
		[]legalquery.CueVocabularyEntry{
			{
				ProfileID:  "query-term-test",
				CueID:      "operator-dual-candidate",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				Terms:      []string{"二候補"},
			},
			{
				ProfileID:  "query-term-test",
				CueID:      "task-search",
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索", "探す", "探して", "調べて", "教えて"},
			},
			{
				ProfileID:  "query-term-test",
				CueID:      "resource-law",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				Terms:      []string{"法令名", "法令本文", "法情報"},
			},
			{
				ProfileID:  "query-term-test",
				CueID:      "resource-provision",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				Terms:      []string{"条文"},
			},
			{
				ProfileID:  "query-term-test",
				CueID:      "resource-judicial",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				Terms:      []string{"裁判例", "判例"},
			},
		},
	)
}
