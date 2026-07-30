package querypreprocess_test

import (
	"context"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestPreprocessは検証済み構文からCueTaskRelationを生成する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustCueRelationPreprocessor(t, cueRelationVocabulary())
	tests := []struct {
		name        string
		query       string
		wantKinds   []legalquery.CueTaskRelationKind
		wantSubject []string
	}{
		{
			name:  "完結 task と対象外目的語を両方保持する",
			query: "EDINETを検索してください。",
			wantKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationObjectPredicate,
				legalquery.CueTaskRelationDirectTask,
			},
			wantSubject: []string{
				"unsupported-external-object",
				"task-search",
			},
		},
		{
			name:  "可変対象を cue にせず read task を作る",
			query: "民法第709条を見せてください。",
			wantKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantSubject: []string{"task-read"},
		},
		{
			name:  "対象外目的語と構文述語を結ぶ",
			query: "影響グラフを作成してください。",
			wantKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationObjectPredicate,
			},
			wantSubject: []string{"unsupported-graph-object"},
		},
		{
			name:  "節全体の短縮 task を作る",
			query: " 比較 ",
			wantKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationStandaloneTask,
			},
			wantSubject: []string{"unsupported-compare-object"},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := preprocessCueRelationQuery(t, preprocessor, test.query)
			relations := result.CueTaskRelations()
			kinds := make([]legalquery.CueTaskRelationKind, 0, len(relations))
			subjects := make([]string, 0, len(relations))
			for _, relation := range relations {
				kinds = append(kinds, relation.Kind())
				subjects = append(subjects, relation.Subject().CueID())
			}
			if !slices.Equal(kinds, test.wantKinds) ||
				!slices.Equal(subjects, test.wantSubject) {
				t.Fatalf(
					"SOT-MODEL-030: relations = kinds:%#v subjects:%#v",
					kinds,
					subjects,
				)
			}
		})
	}
}

func TestPreprocessは言及と節の境界からTaskRelationを作らない(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustCueRelationPreprocessor(t, cueRelationVocabulary())
	tests := []struct {
		name       string
		query      string
		wantSearch bool
	}{
		{
			name:       "引用句",
			query:      "「影響グラフ」を含む条文を検索してください。",
			wantSearch: true,
		},
		{
			name:       "という語",
			query:      "影響グラフという語を含む条文を検索してください。",
			wantSearch: true,
		},
		{
			name:       "topic 表現",
			query:      "翻訳に関する規定を検索してください。",
			wantSearch: true,
		},
		{
			name:       "という文言",
			query:      "英語に翻訳してくださいという文言を含む条文を検索してください。",
			wantSearch: true,
		},
		{
			name:       "文末でない対象外述語",
			query:      "差分を説明する規定を検索してください。",
			wantSearch: true,
		},
		{
			name:  "別の節",
			query: "規定の影響グラフ。作成してください。",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result := preprocessCueRelationQuery(t, preprocessor, test.query)
			relations := result.CueTaskRelations()
			if !test.wantSearch && len(relations) == 0 {
				return
			}
			if !test.wantSearch ||
				len(relations) != 1 ||
				relations[0].Kind() != legalquery.CueTaskRelationDirectTask ||
				relations[0].Subject().CueID() != "task-search" {
				t.Fatalf(
					"cue-relation-task-and-mention/cue-relation-clause-scope: relations = %#v",
					relations,
				)
			}
		})
	}
}

func TestPreprocessは格助詞でない登録語を目的語接続に使わない(
	t *testing.T,
) {
	t.Parallel()

	cues := append(cueRelationVocabulary(), legalquery.CueVocabularyEntry{
		ProfileID:  "core",
		CueID:      "syntax-particle-lookalike",
		SyntaxRole: legalquery.CueSyntaxRoleNone,
		Terms:      []string{"を"},
	})
	preprocessor := mustCueRelationPreprocessor(t, cues)
	result := preprocessCueRelationQuery(
		t,
		preprocessor,
		"影響グラフを作成してください。",
	)
	if len(result.CueTaskRelations()) != 0 {
		t.Fatalf(
			"SOT-MODEL-030: 格助詞でない token から relation = %#v",
			result.CueTaskRelations(),
		)
	}
}

func TestPreprocessは異なるProfileのCueを関係付けない(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustCueRelationPreprocessor(
		t,
		[]legalquery.CueVocabularyEntry{
			{
				ProfileID:  "core",
				CueID:      "unsupported-graph-object",
				SyntaxRole: legalquery.CueSyntaxRoleTaskObject,
				Terms:      []string{"影響グラフ"},
			},
			{
				ProfileID:  "judicial-cases",
				CueID:      "task-search",
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索してください"},
			},
		},
	)
	result := preprocessCueRelationQuery(
		t,
		preprocessor,
		"影響グラフを検索してください。",
	)
	relations := result.CueTaskRelations()
	if len(relations) != 1 ||
		relations[0].Kind() != legalquery.CueTaskRelationDirectTask ||
		relations[0].Subject().ProfileID() != "judicial-cases" {
		t.Fatalf(
			"positive-cue-profile-isolation: relations = %#v",
			relations,
		)
	}
}

func cueRelationVocabulary() []legalquery.CueVocabularyEntry {
	return []legalquery.CueVocabularyEntry{
		{
			ProfileID:  "core",
			CueID:      "syntax-create",
			SyntaxRole: legalquery.CueSyntaxRoleTaskPredicate,
			Terms:      []string{"作成してください"},
		},
		{
			ProfileID:  "core",
			CueID:      "syntax-explain",
			SyntaxRole: legalquery.CueSyntaxRoleTaskPredicate,
			Terms:      []string{"説明する"},
		},
		{
			ProfileID:  "core",
			CueID:      "task-read",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"見せてください"},
		},
		{
			ProfileID:  "core",
			CueID:      "task-search",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"検索してください"},
		},
		{
			ProfileID:  "core",
			CueID:      "unsupported-compare-object",
			SyntaxRole: legalquery.CueSyntaxRoleTaskObject,
			Terms:      []string{"差分", "比較"},
		},
		{
			ProfileID:  "core",
			CueID:      "unsupported-external-object",
			SyntaxRole: legalquery.CueSyntaxRoleTaskObject,
			Terms:      []string{"EDINET"},
		},
		{
			ProfileID:  "core",
			CueID:      "unsupported-graph-object",
			SyntaxRole: legalquery.CueSyntaxRoleTaskObject,
			Terms:      []string{"影響グラフ"},
		},
		{
			ProfileID:  "core",
			CueID:      "unsupported-translation-expression",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"英語に翻訳してください"},
		},
		{
			ProfileID:  "core",
			CueID:      "unsupported-translation-object",
			SyntaxRole: legalquery.CueSyntaxRoleTaskObject,
			Terms:      []string{"翻訳"},
		},
	}
}

func mustCueRelationPreprocessor(
	t *testing.T,
	cues []legalquery.CueVocabularyEntry,
) legalquery.QueryPreprocessor {
	t.Helper()

	terms := make([]string, 0)
	for _, cue := range cues {
		terms = append(terms, cue.Terms...)
	}
	analyzer, err := kagome.NewAnalyzer(terms)
	if err != nil {
		t.Fatalf("形態素解析器を構築できません: %v", err)
	}
	preprocessor, err := querypreprocess.New(querypreprocess.Values{
		Analyzer: analyzer,
		Cues:     cues,
	})
	if err != nil {
		t.Fatalf("前処理器を構築できません: %v", err)
	}
	return preprocessor
}

func preprocessCueRelationQuery(
	t *testing.T,
	preprocessor legalquery.QueryPreprocessor,
	query string,
) legalquery.PreprocessResult {
	t.Helper()

	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	return result
}
