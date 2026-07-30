package legalquery_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestNewPreprocessResultKeepsAllFactsAndCopiesInput(t *testing.T) {
	t.Parallel()

	query := "民法第709条第1項を2026年7月1日に検索（永住権、129AC0000000089）"
	ref := newSourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"129AC0000000089",
	)
	originalRef := ref

	lawNames := []legalquery.LawNameMention{
		mustLawNameMention(
			t,
			spanForSurface(t, query, "民法"),
			"民法",
			"129AC0000000089",
		),
	}
	concepts := []legalquery.LegalConceptMention{
		mustConceptMention(
			t,
			spanForSurface(t, query, "永住権"),
			"永住権",
			"permanent-residence",
		),
	}
	cues := []legalquery.CueMention{
		mustCueMention(
			t,
			spanForSurface(t, query, "検索"),
			"検索",
			"core-law-v1",
			"task-search",
		),
	}
	identifiers := []legalquery.IdentifierMention{
		mustLawIDMention(
			t,
			spanForSurface(t, query, "129AC0000000089"),
			"129AC0000000089",
		),
	}
	dates := []legalquery.DateMention{
		mustDateMention(
			t,
			spanForSurface(t, query, "2026年7月1日"),
			"2026年7月1日",
			"2026-07-01",
		),
	}
	articles := []legalquery.ArticleMention{
		mustArticleMention(
			t,
			spanForSurface(t, query, "第709条"),
			"第709条",
			"709",
		),
	}
	paragraphs := []legalquery.ParagraphMention{
		mustParagraphMention(
			t,
			spanForSurface(t, query, "第1項"),
			"第1項",
			1,
		),
	}

	result, err := legalquery.NewPreprocessResult(legalquery.PreprocessResultValues{
		Query:                query,
		ComparisonKey:        "民法第709条第1項を2026年7月1日に検索永住権129ac0000000089",
		Ref:                  &ref,
		LawNameMentions:      lawNames,
		LegalConceptMentions: concepts,
		CueMentions:          cues,
		IdentifierMentions:   identifiers,
		DateMentions:         dates,
		ArticleMentions:      articles,
		ParagraphMentions:    paragraphs,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: NewPreprocessResult() のエラー = %v", err)
	}

	ref = newSourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	lawNames[0] = legalquery.LawNameMention{}
	concepts[0] = legalquery.LegalConceptMention{}
	cues[0] = legalquery.CueMention{}
	identifiers[0] = legalquery.IdentifierMention{}
	dates[0] = legalquery.DateMention{}
	articles[0] = legalquery.ArticleMention{}
	paragraphs[0] = legalquery.ParagraphMention{}

	if result.Query() != query ||
		result.ComparisonKey() !=
			"民法第709条第1項を2026年7月1日に検索永住権129ac0000000089" {
		t.Fatalf(
			"SOT-MODEL-025: query/comparison key = %q / %q",
			result.Query(),
			result.ComparisonKey(),
		)
	}
	gotRef, hasRef := result.Ref()
	if !hasRef || gotRef != originalRef {
		t.Fatalf("SOT-MODEL-025: Ref() = %#v, %t", gotRef, hasRef)
	}
	if got := result.LawNameMentions(); len(got) != 1 ||
		got[0].LawID() != "129AC0000000089" {
		t.Fatalf("SOT-MODEL-025: LawNameMentions() = %#v", got)
	}
	if got := result.LegalConceptMentions(); len(got) != 1 ||
		got[0].ConceptID() != "permanent-residence" {
		t.Fatalf("SOT-MODEL-025: LegalConceptMentions() = %#v", got)
	}
	if got := result.CueMentions(); len(got) != 1 ||
		got[0].CueID() != "task-search" {
		t.Fatalf("SOT-MODEL-025: CueMentions() = %#v", got)
	}
	if got := result.IdentifierMentions(); len(got) != 1 ||
		got[0].LawID() != "129AC0000000089" {
		t.Fatalf("SOT-MODEL-025: IdentifierMentions() = %#v", got)
	}
	if got := result.DateMentions(); len(got) != 1 ||
		got[0].Date().String() != "2026-07-01" {
		t.Fatalf("SOT-MODEL-025: DateMentions() = %#v", got)
	}
	if got := result.ArticleMentions(); len(got) != 1 ||
		got[0].ArticleNumber() != "709" {
		t.Fatalf("SOT-MODEL-025: ArticleMentions() = %#v", got)
	}
	if got := result.ParagraphMentions(); len(got) != 1 ||
		got[0].ParagraphNumber() != 1 {
		t.Fatalf("SOT-MODEL-025: ParagraphMentions() = %#v", got)
	}

	gotLawNames := result.LawNameMentions()
	gotConcepts := result.LegalConceptMentions()
	gotCues := result.CueMentions()
	gotIdentifiers := result.IdentifierMentions()
	gotDates := result.DateMentions()
	gotArticles := result.ArticleMentions()
	gotParagraphs := result.ParagraphMentions()
	gotLawNames[0] = legalquery.LawNameMention{}
	gotConcepts[0] = legalquery.LegalConceptMention{}
	gotCues[0] = legalquery.CueMention{}
	gotIdentifiers[0] = legalquery.IdentifierMention{}
	gotDates[0] = legalquery.DateMention{}
	gotArticles[0] = legalquery.ArticleMention{}
	gotParagraphs[0] = legalquery.ParagraphMention{}

	if result.LawNameMentions()[0].LawID() != "129AC0000000089" ||
		result.LegalConceptMentions()[0].ConceptID() != "permanent-residence" ||
		result.CueMentions()[0].CueID() != "task-search" ||
		result.IdentifierMentions()[0].LawID() != "129AC0000000089" ||
		result.DateMentions()[0].Date().String() != "2026-07-01" ||
		result.ArticleMentions()[0].ArticleNumber() != "709" ||
		result.ParagraphMentions()[0].ParagraphNumber() != 1 {
		t.Fatal("SOT-MODEL-025: getter の配列から result を変更できました")
	}
	if err := result.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-025: Validate() のエラー = %v", err)
	}
}

func TestPreprocessResultKeepsCueTaskRelationsAndCopiesInput(t *testing.T) {
	t.Parallel()

	query := "影響グラフを作成してください"
	subject := mustCueMention(
		t,
		spanForSurface(t, query, "影響グラフ"),
		"影響グラフ",
		"core",
		"task-graph",
	)
	predicate := mustCueMention(
		t,
		spanForSurface(t, query, "作成してください"),
		"作成してください",
		"core",
		"predicate-create",
	)
	relation := mustCueTaskRelation(
		t,
		query,
		subject,
		predicate,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate,
		mustQuerySpan(t, 0, len(query)),
		legalquery.CueTaskRelationObjectPredicate,
	)
	relations := []legalquery.CueTaskRelation{relation}

	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueMentions:      []legalquery.CueMention{subject, predicate},
			CueTaskRelations: relations,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: relation を持つ result のエラー = %v", err)
	}

	relations[0] = legalquery.CueTaskRelation{}
	got := result.CueTaskRelations()
	if len(got) != 1 ||
		got[0].Subject().CueID() != "task-graph" ||
		got[0].Predicate().CueID() != "predicate-create" {
		t.Fatalf("SOT-MODEL-025: CueTaskRelations() = %#v", got)
	}
	got[0] = legalquery.CueTaskRelation{}
	if result.CueTaskRelations()[0].Subject().CueID() != "task-graph" {
		t.Fatal("SOT-MODEL-025: getter から relation を変更できました")
	}

	withoutRelations, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         query,
			ComparisonKey: querynormalization.ComparisonKey(query),
			CueMentions:   []legalquery.CueMention{subject, predicate},
		},
	)
	if err != nil || withoutRelations.CueTaskRelations() != nil {
		t.Fatalf(
			"SOT-MODEL-025: relation 省略時の互換性 = %#v, %v",
			withoutRelations.CueTaskRelations(),
			err,
		)
	}
}

func TestPreprocessResultRejectsInvalidCueTaskRelationSequence(t *testing.T) {
	t.Parallel()

	query := "検索してください。比較"
	direct := mustCueMention(
		t,
		spanForSurface(t, query, "検索してください"),
		"検索してください",
		"core",
		"task-search",
	)
	standalone := mustCueMention(
		t,
		spanForSurface(t, query, "比較"),
		"比較",
		"core",
		"task-compare",
	)
	directRelation := mustCueTaskRelation(
		t,
		query,
		direct,
		direct,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskExpression,
		direct.Span(),
		legalquery.CueTaskRelationDirectTask,
	)
	standaloneRelation := mustCueTaskRelation(
		t,
		query,
		standalone,
		standalone,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskObject,
		standalone.Span(),
		legalquery.CueTaskRelationStandaloneTask,
	)

	valid, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueMentions:      []legalquery.CueMention{direct, standalone},
			CueTaskRelations: []legalquery.CueTaskRelation{directRelation, standaloneRelation},
		},
	)
	if err != nil || len(valid.CueTaskRelations()) != 2 {
		t.Fatalf("SOT-MODEL-029: 正規順の relation を拒否しました: %v", err)
	}

	tests := map[string]legalquery.PreprocessResultValues{
		"参照先 cue 欠落": {
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueMentions:      []legalquery.CueMention{direct},
			CueTaskRelations: []legalquery.CueTaskRelation{standaloneRelation},
		},
		"逆順": {
			Query:         query,
			ComparisonKey: querynormalization.ComparisonKey(query),
			CueMentions:   []legalquery.CueMention{direct, standalone},
			CueTaskRelations: []legalquery.CueTaskRelation{
				standaloneRelation,
				directRelation,
			},
		},
		"重複": {
			Query:         query,
			ComparisonKey: querynormalization.ComparisonKey(query),
			CueMentions:   []legalquery.CueMention{direct, standalone},
			CueTaskRelations: []legalquery.CueTaskRelation{
				directRelation,
				directRelation,
			},
		},
		"relation だけ": {
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueTaskRelations: []legalquery.CueTaskRelation{directRelation},
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewPreprocessResult(values); err == nil {
				t.Fatal("SOT-MODEL-029: 不正な relation sequence を受理しました")
			}
		})
	}

	tooMany := make([]legalquery.CueTaskRelation, 129)
	for index := range tooMany {
		tooMany[index] = directRelation
	}
	if _, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueMentions:      []legalquery.CueMention{direct, standalone},
			CueTaskRelations: tooMany,
		},
	); err == nil {
		t.Fatal("SOT-MODEL-029: 百二十九件の relation を受理しました")
	}

	longerQuery := "検索してください "
	longerDirect := mustCueMention(
		t,
		spanForSurface(t, longerQuery, "検索してください"),
		"検索してください",
		"core",
		"task-search",
	)
	outOfRangeRelation := mustCueTaskRelation(
		t,
		longerQuery,
		longerDirect,
		longerDirect,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskExpression,
		mustQuerySpan(t, 0, len(longerQuery)),
		legalquery.CueTaskRelationDirectTask,
	)
	if _, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:            direct.Surface(),
			ComparisonKey:    querynormalization.ComparisonKey(direct.Surface()),
			CueMentions:      []legalquery.CueMention{direct},
			CueTaskRelations: []legalquery.CueTaskRelation{outOfRangeRelation},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-029: query の範囲外にある clause span を受理しました")
	}
}

func TestPreprocessResultは同じ開始位置の異なるRelationを保持する(t *testing.T) {
	t.Parallel()

	query := "検索してください  。"
	cue := mustCueMention(
		t,
		spanForSurface(t, query, "検索してください"),
		"検索してください",
		"core",
		"task-search",
	)
	shortClause := mustCueTaskRelation(
		t,
		query,
		cue,
		cue,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskExpression,
		cue.Span(),
		legalquery.CueTaskRelationDirectTask,
	)
	longClause := mustCueTaskRelation(
		t,
		query,
		cue,
		cue,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskExpression,
		mustQuerySpan(t, 0, len(query)-len("。")),
		legalquery.CueTaskRelationDirectTask,
	)

	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:            query,
			ComparisonKey:    querynormalization.ComparisonKey(query),
			CueMentions:      []legalquery.CueMention{cue},
			CueTaskRelations: []legalquery.CueTaskRelation{shortClause, longClause},
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-029: 異なる clause span の relation を拒否しました: %v", err)
	}
	got := result.CueTaskRelations()
	if len(got) != 2 ||
		got[0].ClauseSpan() == got[1].ClauseSpan() {
		t.Fatalf("SOT-MODEL-029: cueTaskRelations = %#v", got)
	}
}

func TestPreprocessResultRequiresCanonicalMentionOrderWithoutDeduplication(t *testing.T) {
	t.Parallel()

	query := "民法と刑法"
	civilSpan := spanForSurface(t, query, "民法")
	civilFirstRune := mustQuerySpan(t, 0, len("民"))
	criminalSpan := spanForSurface(t, query, "刑法")
	civilA := mustLawNameMention(t, civilSpan, "民法", "law-a")
	civilB := mustLawNameMention(t, civilSpan, "民法", "law-b")
	civilShort := mustLawNameMention(t, civilFirstRune, "民", "law-c")
	criminal := mustLawNameMention(t, criminalSpan, "刑法", "law-d")

	valid, err := legalquery.NewPreprocessResult(legalquery.PreprocessResultValues{
		Query:           query,
		ComparisonKey:   "民法と刑法",
		LawNameMentions: []legalquery.LawNameMention{civilA, civilB, civilShort, criminal},
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 正規順の曖昧な複数対象を拒否しました: %v", err)
	}
	got := valid.LawNameMentions()
	if len(got) != 4 ||
		got[0].LawID() != "law-a" ||
		got[1].LawID() != "law-b" ||
		got[2].LawID() != "law-c" ||
		got[3].LawID() != "law-d" {
		t.Fatalf("SOT-MODEL-025: 複数対象が縮約されました: %#v", got)
	}

	tests := map[string][]legalquery.LawNameMention{
		"startByte の逆順": {criminal, civilA},
		"同じ startByte で短い span が先": {
			civilShort,
			civilA,
		},
		"同じ span で識別子が逆順": {civilB, civilA},
		"同じ対象の重複":         {civilA, civilA},
	}
	for name, mentions := range tests {
		name := name
		mentions := mentions
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewPreprocessResult(
				legalquery.PreprocessResultValues{
					Query:           query,
					ComparisonKey:   "民法と刑法",
					LawNameMentions: mentions,
				},
			); err == nil {
				t.Fatal("SOT-MODEL-025: 非正規順または重複を受理しました")
			}
		})
	}
}

func TestPreprocessResultRejectsSpanOutsideOriginalUTF8Surface(t *testing.T) {
	t.Parallel()

	query := "民法"
	tests := map[string]legalquery.LawNameMention{
		"rune の途中": mustLawNameMention(
			t,
			mustQuerySpan(t, 1, 4),
			"民",
			"law-1",
		),
		"query の範囲外": mustLawNameMention(
			t,
			mustQuerySpan(t, len(query), len(query)+len("法")),
			"法",
			"law-1",
		),
		"surface の不一致": mustLawNameMention(
			t,
			mustQuerySpan(t, 0, len(query)),
			"刑法",
			"law-1",
		),
	}
	for name, mention := range tests {
		name := name
		mention := mention
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewPreprocessResult(
				legalquery.PreprocessResultValues{
					Query:           query,
					ComparisonKey:   "民法",
					LawNameMentions: []legalquery.LawNameMention{mention},
				},
			); err == nil {
				t.Fatal("SOT-MODEL-025: 原文に対応しない span/surface を受理しました")
			}
		})
	}
}

func TestPreprocessResultRejectsInvalidQueryComparisonKeyAndRef(t *testing.T) {
	t.Parallel()

	validRef := newSourceResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		"129AC0000000089",
	)
	tests := map[string]legalquery.PreprocessResultValues{
		"空の query": {
			Query: "",
		},
		"ASCII 制御文字を含む query": {
			Query:         "民\n法",
			ComparisonKey: "民法",
		},
		"2048 byte を超える query": {
			Query:         strings.Repeat("法", 683),
			ComparisonKey: "法",
		},
		"不正な UTF-8 の query": {
			Query:         string([]byte{'a', 0xff}),
			ComparisonKey: "a",
		},
		"4096 byte を超える comparison key": {
			Query:         "民法",
			ComparisonKey: strings.Repeat("a", 4097),
		},
		"不正な UTF-8 の comparison key": {
			Query:         "民法",
			ComparisonKey: string([]byte{'a', 0xff}),
		},
		"query から導出されていない comparison key": {
			Query:         "民法",
			ComparisonKey: "刑法",
		},
		"不正な ref": {
			Query:         "民法",
			ComparisonKey: "民法",
			Ref:           &model.SourceResourceRef{},
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewPreprocessResult(values); err == nil {
				t.Fatal("SOT-MODEL-025: 不正な前処理結果を受理しました")
			}
		})
	}

	withoutRef, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         "。",
			ComparisonKey: "",
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 空になり得る comparison key を拒否しました: %v", err)
	}
	if got, exists := withoutRef.Ref(); exists || got != (model.SourceResourceRef{}) {
		t.Fatalf("SOT-MODEL-025: 省略した Ref() = %#v, %t", got, exists)
	}

	withRef, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         "民法",
			ComparisonKey: "民法",
			Ref:           &validRef,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 有効な ref を拒否しました: %v", err)
	}
	if got, exists := withRef.Ref(); !exists || got != validRef {
		t.Fatalf("SOT-MODEL-025: Ref() = %#v, %t", got, exists)
	}
}

func TestPreprocessResultRejectsMentionCountLimitsWithoutTruncation(t *testing.T) {
	t.Parallel()

	query := "法第1項"
	lawSpan := spanForSurface(t, query, "法")
	paragraphSpan := spanForSurface(t, query, "第1項")

	lawNames := make([]legalquery.LawNameMention, 0, 65)
	concepts := make([]legalquery.LegalConceptMention, 0, 64)
	cues := make([]legalquery.CueMention, 0, 129)
	for index := 0; index < 65; index++ {
		lawNames = append(
			lawNames,
			mustLawNameMention(
				t,
				lawSpan,
				"法",
				fmt.Sprintf("law-%03d", index),
			),
		)
	}
	for index := 0; index < 64; index++ {
		concepts = append(
			concepts,
			mustConceptMention(
				t,
				lawSpan,
				"法",
				fmt.Sprintf("concept-%03d", index),
			),
		)
	}
	for index := 0; index < 129; index++ {
		cues = append(
			cues,
			mustCueMention(
				t,
				lawSpan,
				"法",
				"profile-1",
				fmt.Sprintf("cue-%03d", index),
			),
		)
	}

	tests := map[string]legalquery.PreprocessResultValues{
		"通常配列の六十四件超過": {
			Query:           query,
			ComparisonKey:   query,
			LawNameMentions: lawNames,
		},
		"cue 配列の百二十八件超過": {
			Query:         query,
			ComparisonKey: query,
			CueMentions:   cues,
		},
		"全出現の二百五十六件超過": {
			Query:                query,
			ComparisonKey:        query,
			LawNameMentions:      lawNames[:64],
			LegalConceptMentions: concepts,
			CueMentions:          cues[:128],
			ParagraphMentions: []legalquery.ParagraphMention{
				mustParagraphMention(t, paragraphSpan, "第1項", 1),
			},
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewPreprocessResult(values); err == nil {
				t.Fatal("SOT-MODEL-025: 上限超過を切り捨てて受理しました")
			}
		})
	}
}

func TestPreprocessResultDoesNotContainPlannerOrProviderDecision(t *testing.T) {
	t.Parallel()

	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:         "民法",
			ComparisonKey: "民法",
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: result を作成できません: %v", err)
	}

	forbiddenNames := map[string]struct{}{
		"score":         {},
		"semanticscore": {},
		"confidence":    {},
		"task":          {},
		"resource":      {},
		"capability":    {},
		"requiredpacks": {},
		"candidates":    {},
		"selection":     {},
		"providerid":    {},
		"route":         {},
	}
	resultType := reflect.TypeOf(result)
	for index := 0; index < resultType.NumField(); index++ {
		fieldName := strings.ToLower(resultType.Field(index).Name)
		if _, forbidden := forbiddenNames[fieldName]; forbidden {
			t.Fatalf("SOT-MODEL-025: result が禁止フィールド %q を持ちます", fieldName)
		}
	}
	for index := 0; index < resultType.NumMethod(); index++ {
		methodName := strings.ToLower(resultType.Method(index).Name)
		if _, forbidden := forbiddenNames[methodName]; forbidden {
			t.Fatalf("SOT-MODEL-025: result が禁止 getter %q を持ちます", methodName)
		}
	}
}

func spanForSurface(t *testing.T, query string, surface string) legalquery.QuerySpan {
	t.Helper()

	start := strings.Index(query, surface)
	if start < 0 {
		t.Fatalf("試験用 query %q に surface %q がありません", query, surface)
	}
	return mustQuerySpan(t, start, start+len(surface))
}

func mustLawNameMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	lawID string,
) legalquery.LawNameMention {
	t.Helper()

	mention, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
		Span:       span,
		Surface:    surface,
		LawID:      lawID,
		RevisionID: lawID + "-revision",
		LawNumber:  lawID + "-number",
		Canonical:  surface,
		MatchKind:  legalquery.PreprocessMatchExact,
	})
	if err != nil {
		t.Fatalf("試験用 LawNameMention を作成できません: %v", err)
	}
	return mention
}

func mustConceptMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	conceptID string,
) legalquery.LegalConceptMention {
	t.Helper()

	mention, err := legalquery.NewLegalConceptMention(
		legalquery.LegalConceptMentionValues{
			Span:      span,
			Surface:   surface,
			ConceptID: conceptID,
			Canonical: surface,
			MatchKind: legalquery.PreprocessMatchExact,
		},
	)
	if err != nil {
		t.Fatalf("試験用 LegalConceptMention を作成できません: %v", err)
	}
	return mention
}

func mustCueMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	profileID string,
	cueID string,
) legalquery.CueMention {
	t.Helper()

	mention, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      span,
		Surface:   surface,
		ProfileID: profileID,
		CueID:     cueID,
		MatchKind: legalquery.PreprocessMatchExact,
	})
	if err != nil {
		t.Fatalf("試験用 CueMention を作成できません: %v", err)
	}
	return mention
}

func mustCueTaskRelation(
	t *testing.T,
	query string,
	subject legalquery.CueMention,
	predicate legalquery.CueMention,
	subjectRole legalquery.CueSyntaxRole,
	predicateRole legalquery.CueSyntaxRole,
	clauseSpan legalquery.QuerySpan,
	kind legalquery.CueTaskRelationKind,
) legalquery.CueTaskRelation {
	t.Helper()

	relation, err := legalquery.NewCueTaskRelation(
		legalquery.CueTaskRelationValues{
			Query:         query,
			Subject:       subject,
			Predicate:     predicate,
			SubjectRole:   subjectRole,
			PredicateRole: predicateRole,
			ClauseSpan:    clauseSpan,
			Kind:          kind,
		},
	)
	if err != nil {
		t.Fatalf("試験用 CueTaskRelation を作成できません: %v", err)
	}
	return relation
}

func mustLawIDMention(
	t *testing.T,
	span legalquery.QuerySpan,
	lawID string,
) legalquery.IdentifierMention {
	t.Helper()

	mention, err := legalquery.NewIdentifierMention(
		legalquery.IdentifierMentionValues{
			Span: span, Surface: lawID,
			Kind: legalquery.IdentifierMentionLawID, LawID: lawID,
		},
	)
	if err != nil {
		t.Fatalf("試験用 IdentifierMention を作成できません: %v", err)
	}
	return mention
}

func mustDateMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	value string,
) legalquery.DateMention {
	t.Helper()

	mention, err := legalquery.NewDateMention(legalquery.DateMentionValues{
		Span: span, Surface: surface, Date: mustPreprocessDate(t, value),
	})
	if err != nil {
		t.Fatalf("試験用 DateMention を作成できません: %v", err)
	}
	return mention
}

func mustArticleMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	articleNumber string,
) legalquery.ArticleMention {
	t.Helper()

	mention, err := legalquery.NewArticleMention(legalquery.ArticleMentionValues{
		Span: span, Surface: surface,
		Provision: model.LawArticleProvisionMain, ArticleNumber: articleNumber,
	})
	if err != nil {
		t.Fatalf("試験用 ArticleMention を作成できません: %v", err)
	}
	return mention
}

func mustParagraphMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	paragraphNumber int,
) legalquery.ParagraphMention {
	t.Helper()

	mention, err := legalquery.NewParagraphMention(
		legalquery.ParagraphMentionValues{
			Span: span, Surface: surface, ParagraphNumber: paragraphNumber,
		},
	)
	if err != nil {
		t.Fatalf("試験用 ParagraphMention を作成できません: %v", err)
	}
	return mention
}
