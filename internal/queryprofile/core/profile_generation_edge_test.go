package core

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestProfileは一般検索語の論理演算子を型付き条件へ変換する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		query       string
		allTerms    []string
		anyTerms    []string
		excludeTerm []string
	}{
		{
			name:     "すべて",
			query:    "法令本文から「営業秘密」と「個人情報」をすべて含む条文を検索してください。",
			allTerms: []string{"営業秘密", "個人情報"},
		},
		{
			name:     "二件の全件指定",
			query:    "法令本文から「営業秘密」と「個人情報」について二つとも含む条文を検索してください。",
			allTerms: []string{"営業秘密", "個人情報"},
		},
		{
			name:  "数によらない全件指定",
			query: "法令本文から「営業秘密」と「個人情報」と「匿名加工情報」について三つとも含む条文を検索してください。",
			allTerms: []string{
				"営業秘密",
				"個人情報",
				"匿名加工情報",
			},
		},
		{
			name:     "いずれか",
			query:    "法令本文から「営業秘密」または「個人情報」を含む条文を検索してください。",
			anyTerms: []string{"営業秘密", "個人情報"},
		},
		{
			name:        "除外",
			query:       "法令本文から「営業秘密」と「個人情報」を含み「営業上の秘密」を除く条文を検索してください。",
			allTerms:    []string{"営業秘密", "個人情報"},
			excludeTerm: []string{"営業上の秘密"},
		},
		{
			name:        "除外対象の位置",
			query:       "法令本文から「匿名加工情報」を除いて「委託契約」を検索してください。",
			allTerms:    []string{"委託契約"},
			excludeTerm: []string{"匿名加工情報"},
		},
		{
			name:        "いずれかと除外",
			query:       "法令本文から「営業秘密」または「個人情報」を含み「公開情報」を除く条文を検索してください。",
			anyTerms:    []string{"営業秘密", "個人情報"},
			excludeTerm: []string{"公開情報"},
		},
		{
			name:        "検索結果単位",
			query:       "法令本文から「個人データ」と「第三者提供」の両方を含み「匿名加工情報」を除く箇所を検索する",
			allTerms:    []string{"個人データ", "第三者提供"},
			excludeTerm: []string{"匿名加工情報"},
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
			input, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
			if !ok {
				t.Fatalf(
					"logical input = %T",
					candidates[0].Steps()[0].LogicalInput(),
				)
			}
			if !slices.Equal(input.AllTerms(), test.allTerms) ||
				!slices.Equal(input.AnyTerms(), test.anyTerms) ||
				!slices.Equal(input.ExcludeTerms(), test.excludeTerm) {
				t.Fatalf(
					"terms = all:%#v any:%#v exclude:%#v",
					input.AllTerms(),
					input.AnyTerms(),
					input.ExcludeTerms(),
				)
			}
		})
	}
}

func TestProfileは本文検索の完全日付を構造化根拠にする(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"2023年12月1日時点の法令本文で「行政指導」を含む箇所を探す",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf("candidates = %#v", candidates)
	}
	if !slices.Equal(
		candidates[0].EvidenceCodes(),
		[]legalquery.EvidenceCode{
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceMorphologicalContext,
		},
	) {
		t.Fatalf(
			"SOT-MODEL-022: evidence = %#v",
			candidates[0].EvidenceCodes(),
		)
	}
}

func TestProfileは確認したいを法令本文の読取り意図として扱う(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"「情報公開法」の本文を確認したいです。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 2 {
		t.Fatalf("candidates = %#v", candidates)
	}
	if generation.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf("selection mode = %q", generation.SelectionMode())
	}

	lawIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		if !slices.Equal(
			candidate.EvidenceCodes(),
			[]legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceOfficialAlias,
			},
		) {
			t.Fatalf("evidence = %#v", candidate.EvidenceCodes())
		}
		steps := candidate.Steps()
		if len(steps) != 1 {
			t.Fatalf("steps = %#v", steps)
		}
		input, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
		if !ok {
			t.Fatalf("logical input = %T", steps[0].LogicalInput())
		}
		lawID, exists := input.LawID()
		if !exists {
			t.Fatalf("law_read に lawId がありません: %#v", input)
		}
		lawIDs = append(lawIDs, lawID)
	}
	slices.Sort(lawIDs)
	if !slices.Equal(
		lawIDs,
		[]string{"411AC0000000042", "413AC0000000140"},
	) {
		t.Fatalf("lawIds = %#v", lawIDs)
	}
}

func TestProfileは取得意図と対象外意図の混在で強い根拠候補を保持する(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法第709条を英訳して法的に判断してください。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 {
		t.Fatalf(
			"SOT-MODEL-026: mixed unsupported candidates = %#v, want 1 件",
			candidates,
		)
	}
	candidate := candidates[0]
	if !slices.Equal(
		candidate.EvidenceCodes(),
		[]legalquery.EvidenceCode{
			legalquery.EvidenceStructuredReference,
			legalquery.EvidenceExplicitResource,
			legalquery.EvidenceOfficialAlias,
		},
	) {
		t.Fatalf(
			"SOT-MODEL-026: mixed unsupported evidence = %#v",
			candidate.EvidenceCodes(),
		)
	}
	steps := candidate.Steps()
	if len(steps) != 1 {
		t.Fatalf(
			"SOT-MODEL-026: mixed unsupported steps = %#v, want 1 件",
			steps,
		)
	}
	input, ok := steps[0].LogicalInput().(legalquery.LawArticleReadIntentV1)
	if !ok {
		t.Fatalf(
			"SOT-MODEL-026: mixed unsupported logical input = %T",
			steps[0].LogicalInput(),
		)
	}
	lawID, exists := input.LawID()
	location := input.Location()
	if !exists ||
		lawID != "129AC0000000089" ||
		location.Provision() != "main" ||
		location.ArticleNumber() != "709" {
		t.Fatalf(
			"SOT-MODEL-026: mixed unsupported logical input = %#v",
			input,
		)
	}
	if !slices.Contains(
		generation.Signals(),
		legalquery.CandidateSignalUnsupportedTranslation,
	) ||
		!slices.Contains(
			generation.Signals(),
			legalquery.CandidateSignalUnsupportedLegalAdvice,
		) {
		t.Fatalf(
			"SOT-PROD-011: mixed unsupported generation = %#v",
			generation,
		)
	}
}

func TestProfileは未知の法概念事実をfailClosedで拒否する(t *testing.T) {
	t.Parallel()

	const query = "未知概念を検索"
	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: 0,
		EndByte:   len("未知概念"),
	})
	if err != nil {
		t.Fatalf("span を作成できません: %v", err)
	}
	mention, err := legalquery.NewLegalConceptMention(
		legalquery.LegalConceptMentionValues{
			Span:      span,
			Surface:   "未知概念",
			ConceptID: "unknown-concept",
			Canonical: "未知概念",
			MatchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
	)
	if err != nil {
		t.Fatalf("concept mention を作成できません: %v", err)
	}
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                query,
			ComparisonKey:        string(querynormalization.ComparisonKey(query)),
			LegalConceptMentions: []legalquery.LegalConceptMention{mention},
		},
	)
	if err != nil {
		t.Fatalf("preprocess result を作成できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("generation input を作成できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	if _, err := mustProfile(t).Generate(input, scope); err == nil {
		t.Fatal("未知の conceptId を空候補として受理しました")
	}
}

func TestProfileは候補上限を黙って切り捨てない(t *testing.T) {
	t.Parallel()

	const candidateCount = maximumGeneratedCandidates + 1
	var query strings.Builder
	mentions := make([]legalquery.LawNameMention, 0, candidateCount)
	for index := 0; index < candidateCount; index++ {
		if index > 0 {
			query.WriteRune('、')
		}
		start := query.Len()
		surface := fmt.Sprintf("試験法%02d", index)
		query.WriteString(surface)
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: start,
			EndByte:   query.Len(),
		})
		if err != nil {
			t.Fatalf("law span を作成できません: %v", err)
		}
		mention, err := legalquery.NewLawNameMention(
			legalquery.LawNameMentionValues{
				Span:       span,
				Surface:    surface,
				LawID:      fmt.Sprintf("test-law-%02d", index),
				RevisionID: fmt.Sprintf("test-revision-%02d", index),
				LawNumber:  fmt.Sprintf("試験法律第%d号", index+1),
				Canonical:  surface,
				MatchKind:  legalquery.PreprocessMatchRegisteredTerm,
			},
		)
		if err != nil {
			t.Fatalf("law mention を作成できません: %v", err)
		}
		mentions = append(mentions, mention)
	}
	query.WriteString("を検索")
	cueStart := query.Len() - len("検索")
	cueSpan, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: cueStart,
		EndByte:   query.Len(),
	})
	if err != nil {
		t.Fatalf("cue span を作成できません: %v", err)
	}
	cue, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      cueSpan,
		Surface:   "検索",
		ProfileID: "core",
		CueID:     "task-search",
		MatchKind: legalquery.PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("cue mention を作成できません: %v", err)
	}
	rawQuery := query.String()
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:           rawQuery,
			ComparisonKey:   string(querynormalization.ComparisonKey(rawQuery)),
			LawNameMentions: mentions,
			CueMentions:     []legalquery.CueMention{cue},
		},
	)
	if err != nil {
		t.Fatalf("preprocess result を作成できません: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("generation input を作成できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	if _, err := mustProfile(t).Generate(input, scope); err == nil ||
		!strings.Contains(err.Error(), "16 件以下") {
		t.Fatalf("candidate 上限エラー = %v", err)
	}
}

func TestProfileは衝突する公式略称を固定候補上限まで順位付けする(
	t *testing.T,
) {
	t.Parallel()

	generation := generateQuery(
		t,
		"中央省庁等改革関連法か財源確保法と呼ばれる法令の本文を一つ読みたいです。",
		nil,
	)
	candidates := generation.Candidates()
	if len(candidates) != maximumGeneratedCandidates {
		t.Fatalf(
			"公式略称衝突の candidates = %d, want %d",
			len(candidates),
			maximumGeneratedCandidates,
		)
	}
	if generation.SelectionMode() !=
		legalquery.QuerySelectionModeClarificationRequired {
		t.Fatalf(
			"公式略称衝突の selection mode = %q",
			generation.SelectionMode(),
		)
	}

	lawIDs := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		steps := candidate.Steps()
		if len(steps) != 1 {
			t.Fatalf("公式略称衝突の steps = %#v", steps)
		}
		input, ok := steps[0].LogicalInput().(legalquery.LawReadIntentV1)
		if !ok {
			t.Fatalf(
				"公式略称衝突の logical input = %T",
				steps[0].LogicalInput(),
			)
		}
		lawID, exists := input.LawID()
		if !exists {
			t.Fatal("公式略称衝突の lawId がありません")
		}
		lawIDs = append(lawIDs, lawID)
	}
	for _, expected := range []string{
		"356AC0000000039",
		"358AC0000000045",
		"411AC0000000089",
		"411AC0000000091",
	} {
		if !slices.Contains(lawIDs, expected) {
			t.Fatalf(
				"公式略称衝突の candidates に lawId %q がありません: %#v",
				expected,
				lawIDs,
			)
		}
	}
}

func TestProfileは公式略称衝突の順位付け前処理量を制限する(
	t *testing.T,
) {
	t.Parallel()

	const (
		query          = "通称法を取得して"
		aliasEnd       = len("通称法")
		candidateCount = 64
	)
	aliasSpan, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: 0,
		EndByte:   aliasEnd,
	})
	if err != nil {
		t.Fatalf("alias span を作成できません: %v", err)
	}
	mentions := make([]legalquery.LawNameMention, 0, candidateCount)
	for index := 0; index < candidateCount; index++ {
		mention, mentionErr := legalquery.NewLawNameMention(
			legalquery.LawNameMentionValues{
				Span:       aliasSpan,
				Surface:    "通称法",
				LawID:      fmt.Sprintf("test-law-%02d", index),
				RevisionID: fmt.Sprintf("test-revision-%02d", index),
				LawNumber:  fmt.Sprintf("試験法律第%d号", index+1),
				Canonical:  fmt.Sprintf("試験法%02d", index),
				MatchKind:  legalquery.PreprocessMatchRegisteredTerm,
			},
		)
		if mentionErr != nil {
			t.Fatalf("law mention を作成できません: %v", mentionErr)
		}
		mentions = append(mentions, mention)
	}
	cueSpan, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: aliasEnd + len("を"),
		EndByte:   len(query),
	})
	if err != nil {
		t.Fatalf("cue span を作成できません: %v", err)
	}
	cue, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      cueSpan,
		Surface:   "取得して",
		ProfileID: "core",
		CueID:     "task-read",
		MatchKind: legalquery.PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("cue mention を作成できません: %v", err)
	}
	result, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:           query,
			ComparisonKey:   string(querynormalization.ComparisonKey(query)),
			LawNameMentions: mentions,
			CueMentions:     []legalquery.CueMention{cue},
		},
	)
	if err != nil {
		t.Fatalf("preprocess result を作成できません: %v", err)
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
		t.Fatalf("公式略称衝突の入力上限を順位付けできません: %v", err)
	}
	if len(generation.Candidates()) != maximumGeneratedCandidates {
		t.Fatalf(
			"公式略称衝突の入力上限 = %d, want %d",
			len(generation.Candidates()),
			maximumGeneratedCandidates,
		)
	}
}
