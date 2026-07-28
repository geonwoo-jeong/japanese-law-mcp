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

func TestProfileは取得意図と対象外意図の混在から候補を作らない(t *testing.T) {
	t.Parallel()

	generation := generateQuery(
		t,
		"民法第709条を英訳して法的に判断してください。",
		nil,
	)
	if len(generation.Candidates()) != 0 ||
		!slices.Contains(
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
