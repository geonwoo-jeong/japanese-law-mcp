package legalquery_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestQueryTermMentionKeepsOriginalSpanSurfaceAndKind(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name    string
		surface string
		kind    legalquery.QueryTermMentionKind
	}{
		{
			name:    "引用句",
			surface: "著作権 演奏権",
			kind:    legalquery.QueryTermMentionQuotedPhrase,
		},
		{
			name:    "形態素句",
			surface: "営業秘密",
			kind:    legalquery.QueryTermMentionMorphologicalPhrase,
		},
	} {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			span := mustQuerySpan(t, len("前"), len("前")+len(test.surface))
			mention, err := legalquery.NewQueryTermMention(
				legalquery.QueryTermMentionValues{
					Span:    span,
					Surface: test.surface,
					Kind:    test.kind,
				},
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: query term mention を作成できません: %v", err)
			}
			if mention.Span() != span ||
				mention.Surface() != test.surface ||
				mention.Kind() != test.kind {
				t.Fatalf("SOT-MODEL-025: query term mention = %#v", mention)
			}
			if err := mention.Validate(); err != nil {
				t.Fatalf("SOT-MODEL-025: Validate() のエラー = %v", err)
			}
		})
	}
}

func TestQueryTermMentionRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	validSpan := mustQuerySpan(t, 0, len("供託"))
	for name, values := range map[string]legalquery.QueryTermMentionValues{
		"span 欠落": {
			Surface: "供託",
			Kind:    legalquery.QueryTermMentionMorphologicalPhrase,
		},
		"surface 欠落": {
			Span: validSpan,
			Kind: legalquery.QueryTermMentionMorphologicalPhrase,
		},
		"surface と span 幅の不一致": {
			Span:    validSpan,
			Surface: "供",
			Kind:    legalquery.QueryTermMentionMorphologicalPhrase,
		},
		"未知の kind": {
			Span:    validSpan,
			Surface: "供託",
			Kind:    legalquery.QueryTermMentionKind("unknown"),
		},
		"引用句の先頭空白": {
			Span:    mustQuerySpan(t, 0, len(" 税")),
			Surface: " 税",
			Kind:    legalquery.QueryTermMentionQuotedPhrase,
		},
		"一文字の形態素句": {
			Span:    mustQuerySpan(t, 0, len("税")),
			Surface: "税",
			Kind:    legalquery.QueryTermMentionMorphologicalPhrase,
		},
		"非日本語の形態素句": {
			Span:    mustQuerySpan(t, 0, len("delete")),
			Surface: "delete",
			Kind:    legalquery.QueryTermMentionMorphologicalPhrase,
		},
	} {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewQueryTermMention(values); err == nil {
				t.Fatal("SOT-MODEL-025: 不正な query term mention を受理しました")
			}
		})
	}
}

func TestPreprocessResultCopiesOrdersAndLimitsQueryTermMentions(t *testing.T) {
	t.Parallel()

	const query = "営業秘密と供託"
	terms := []legalquery.QueryTermMention{
		mustQueryTermMention(
			t,
			spanForSurface(t, query, "営業秘密"),
			"営業秘密",
			legalquery.QueryTermMentionMorphologicalPhrase,
		),
		mustQueryTermMention(
			t,
			spanForSurface(t, query, "供託"),
			"供託",
			legalquery.QueryTermMentionMorphologicalPhrase,
		),
	}
	result, err := legalquery.NewPreprocessResult(legalquery.PreprocessResultValues{
		Query:             query,
		ComparisonKey:     query,
		QueryTermMentions: terms,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: query term 付き result を作成できません: %v", err)
	}

	terms[0] = legalquery.QueryTermMention{}
	got := result.QueryTermMentions()
	if len(got) != 2 ||
		got[0].Surface() != "営業秘密" ||
		got[1].Surface() != "供託" {
		t.Fatalf("SOT-MODEL-025: QueryTermMentions() = %#v", got)
	}
	got[0] = legalquery.QueryTermMention{}
	if result.QueryTermMentions()[0].Surface() != "営業秘密" {
		t.Fatal("SOT-MODEL-025: getter から query term mention を変更できました")
	}

	reversed := result.QueryTermMentions()
	reversed[0], reversed[1] = reversed[1], reversed[0]
	if _, resultErr := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:             query,
			ComparisonKey:     query,
			QueryTermMentions: reversed,
		},
	); resultErr == nil {
		t.Fatal("SOT-MODEL-025: 非正規順の query term mentions を受理しました")
	}

	repeatedQuery := strings.Repeat("用語", 65)
	overLimit := make([]legalquery.QueryTermMention, 0, 65)
	for index := 0; index < 65; index++ {
		startByte := index * len("用語")
		overLimit = append(overLimit, mustQueryTermMention(
			t,
			mustQuerySpan(t, startByte, startByte+len("用語")),
			"用語",
			legalquery.QueryTermMentionMorphologicalPhrase,
		))
	}
	if _, resultErr := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:             repeatedQuery,
			ComparisonKey:     repeatedQuery,
			QueryTermMentions: overLimit,
		},
	); resultErr == nil {
		t.Fatal("SOT-MODEL-025: 六十四件を超える query term mentions を受理しました")
	}
}

func TestPreprocessResultCountsQueryTermMentionsInTotalLimit(t *testing.T) {
	t.Parallel()

	const termCount = 64
	query := strings.Repeat("用語", termCount) + "第1項"
	firstTermSpan := mustQuerySpan(t, 0, len("用語"))
	lawNames := make([]legalquery.LawNameMention, 0, termCount)
	concepts := make([]legalquery.LegalConceptMention, 0, termCount)
	cues := make([]legalquery.CueMention, 0, termCount)
	terms := make([]legalquery.QueryTermMention, 0, termCount)
	for index := 0; index < termCount; index++ {
		lawNames = append(lawNames, mustLawNameMention(
			t,
			firstTermSpan,
			"用語",
			fmt.Sprintf("law-%03d", index),
		))
		concepts = append(concepts, mustConceptMention(
			t,
			firstTermSpan,
			"用語",
			fmt.Sprintf("concept-%03d", index),
		))
		cues = append(cues, mustCueMention(
			t,
			firstTermSpan,
			"用語",
			"profile-1",
			fmt.Sprintf("cue-%03d", index),
		))

		startByte := index * len("用語")
		terms = append(terms, mustQueryTermMention(
			t,
			mustQuerySpan(t, startByte, startByte+len("用語")),
			"用語",
			legalquery.QueryTermMentionMorphologicalPhrase,
		))
	}
	paragraph := mustParagraphMention(
		t,
		spanForSurface(t, query, "第1項"),
		"第1項",
		1,
	)

	if _, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                query,
			ComparisonKey:        query,
			LawNameMentions:      lawNames,
			LegalConceptMentions: concepts,
			CueMentions:          cues,
			QueryTermMentions:    terms,
			ParagraphMentions:    []legalquery.ParagraphMention{paragraph},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-025: query term を含む二百五十六件超過を受理しました")
	}
}

func mustQueryTermMention(
	t *testing.T,
	span legalquery.QuerySpan,
	surface string,
	kind legalquery.QueryTermMentionKind,
) legalquery.QueryTermMention {
	t.Helper()

	mention, err := legalquery.NewQueryTermMention(
		legalquery.QueryTermMentionValues{
			Span:    span,
			Surface: surface,
			Kind:    kind,
		},
	)
	if err != nil {
		t.Fatalf("試験用 QueryTermMention を作成できません: %v", err)
	}
	return mention
}
