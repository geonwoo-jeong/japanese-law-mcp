package legalquery_test

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestPreprocessMentionConstructorsKeepEveryProviderIndependentFact(t *testing.T) {
	t.Parallel()

	lawSpan := mustQuerySpan(t, 0, len("民法"))
	lawName, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
		Span:       lawSpan,
		Surface:    "民法",
		LawID:      "129AC0000000089",
		RevisionID: "129AC0000000089_20250601_504AC0000000068",
		LawNumber:  "明治二十九年法律第八十九号",
		Canonical:  "民法",
		MatchKind:  legalquery.PreprocessMatchExact,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: law name mention を作成できません: %v", err)
	}
	if lawName.Span() != lawSpan ||
		lawName.Surface() != "民法" ||
		lawName.LawID() != "129AC0000000089" ||
		lawName.RevisionID() != "129AC0000000089_20250601_504AC0000000068" ||
		lawName.LawNumber() != "明治二十九年法律第八十九号" ||
		lawName.Canonical() != "民法" ||
		lawName.MatchKind() != legalquery.PreprocessMatchExact {
		t.Fatalf("SOT-MODEL-025: law name mention = %#v", lawName)
	}

	concept, err := legalquery.NewLegalConceptMention(
		legalquery.LegalConceptMentionValues{
			Span:      mustQuerySpan(t, 0, len("永住権")),
			Surface:   "永住権",
			ConceptID: "permanent-residence",
			Canonical: "永住許可",
			MatchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: legal concept mention を作成できません: %v", err)
	}
	if concept.Surface() != "永住権" ||
		concept.ConceptID() != "permanent-residence" ||
		concept.Canonical() != "永住許可" ||
		concept.MatchKind() != legalquery.PreprocessMatchRegisteredTerm {
		t.Fatalf("SOT-MODEL-025: legal concept mention = %#v", concept)
	}

	cue, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      mustQuerySpan(t, 0, len("検索")),
		Surface:   "検索",
		ProfileID: "core-law-v1",
		CueID:     "task-search",
		MatchKind: legalquery.PreprocessMatchComparisonNormalized,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: cue mention を作成できません: %v", err)
	}
	if cue.Surface() != "検索" ||
		cue.ProfileID() != "core-law-v1" ||
		cue.CueID() != "task-search" ||
		cue.MatchKind() != legalquery.PreprocessMatchComparisonNormalized {
		t.Fatalf("SOT-MODEL-025: cue mention = %#v", cue)
	}

	lawID, err := legalquery.NewIdentifierMention(
		legalquery.IdentifierMentionValues{
			Span:    mustQuerySpan(t, 0, len("129AC0000000089")),
			Surface: "129AC0000000089",
			Kind:    legalquery.IdentifierMentionLawID,
			LawID:   "129AC0000000089",
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: law ID mention を作成できません: %v", err)
	}
	if lawID.Kind() != legalquery.IdentifierMentionLawID ||
		lawID.LawID() != "129AC0000000089" {
		t.Fatalf("SOT-MODEL-025: law ID mention = %#v", lawID)
	}
	if revisionID, exists := lawID.RevisionID(); exists || revisionID != "" {
		t.Fatalf("SOT-MODEL-025: law ID の RevisionID() = %q, %t", revisionID, exists)
	}
	if lawNumber, exists := lawID.LawNumber(); exists || lawNumber != "" {
		t.Fatalf("SOT-MODEL-025: law ID の LawNumber() = %q, %t", lawNumber, exists)
	}

	date := mustPreprocessDate(t, "2026-07-01")
	dateMention, err := legalquery.NewDateMention(legalquery.DateMentionValues{
		Span:    mustQuerySpan(t, 0, len("2026年7月1日")),
		Surface: "2026年7月1日",
		Date:    date,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: date mention を作成できません: %v", err)
	}
	if dateMention.Date() != date || dateMention.Surface() != "2026年7月1日" {
		t.Fatalf("SOT-MODEL-025: date mention = %#v", dateMention)
	}

	article, err := legalquery.NewArticleMention(legalquery.ArticleMentionValues{
		Span:          mustQuerySpan(t, 0, len("附則第398条の2")),
		Surface:       "附則第398条の2",
		Provision:     model.LawArticleProvisionSupplementary,
		ArticleNumber: "398_2",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: article mention を作成できません: %v", err)
	}
	if article.Provision() != model.LawArticleProvisionSupplementary ||
		article.ArticleNumber() != "398_2" ||
		article.Surface() != "附則第398条の2" {
		t.Fatalf("SOT-MODEL-025: article mention = %#v", article)
	}

	paragraph, err := legalquery.NewParagraphMention(
		legalquery.ParagraphMentionValues{
			Span:            mustQuerySpan(t, 0, len("第2項")),
			Surface:         "第2項",
			ParagraphNumber: 2,
		},
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: paragraph mention を作成できません: %v", err)
	}
	if paragraph.ParagraphNumber() != 2 || paragraph.Surface() != "第2項" {
		t.Fatalf("SOT-MODEL-025: paragraph mention = %#v", paragraph)
	}

	mentions := []interface{ Validate() error }{
		lawName,
		concept,
		cue,
		lawID,
		dateMention,
		article,
		paragraph,
	}
	for index, mention := range mentions {
		if err := mention.Validate(); err != nil {
			t.Fatalf("SOT-MODEL-025: mentions[%d].Validate() = %v", index, err)
		}
	}
}

func TestIdentifierMentionAcceptsOnlyThreeClosedShapes(t *testing.T) {
	t.Parallel()

	revisionID := "129AC0000000089_20250601_504AC0000000068"
	lawNumber := "明治二十九年法律第八十九号"
	tests := []struct {
		name      string
		values    legalquery.IdentifierMentionValues
		revision  string
		lawNumber string
	}{
		{
			name: "law ID",
			values: legalquery.IdentifierMentionValues{
				Span:    mustQuerySpan(t, 0, len("129AC0000000089")),
				Surface: "129AC0000000089",
				Kind:    legalquery.IdentifierMentionLawID,
				LawID:   "129AC0000000089",
			},
		},
		{
			name: "revision ID",
			values: legalquery.IdentifierMentionValues{
				Span:       mustQuerySpan(t, 0, len(revisionID)),
				Surface:    revisionID,
				Kind:       legalquery.IdentifierMentionLawRevisionID,
				LawID:      "129AC0000000089",
				RevisionID: &revisionID,
			},
			revision: revisionID,
		},
		{
			name: "law number",
			values: legalquery.IdentifierMentionValues{
				Span:      mustQuerySpan(t, 0, len(lawNumber)),
				Surface:   lawNumber,
				Kind:      legalquery.IdentifierMentionLawNumber,
				LawID:     "129AC0000000089",
				LawNumber: &lawNumber,
			},
			lawNumber: lawNumber,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			mention, err := legalquery.NewIdentifierMention(test.values)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: identifier mention を作成できません: %v", err)
			}
			gotRevision, hasRevision := mention.RevisionID()
			gotLawNumber, hasLawNumber := mention.LawNumber()
			if gotRevision != test.revision ||
				hasRevision != (test.revision != "") ||
				gotLawNumber != test.lawNumber ||
				hasLawNumber != (test.lawNumber != "") {
				t.Fatalf(
					"SOT-MODEL-025: optional identifier = %q/%t, %q/%t",
					gotRevision,
					hasRevision,
					gotLawNumber,
					hasLawNumber,
				)
			}
		})
	}
}

func TestPreprocessMentionConstructorsRejectInvalidValues(t *testing.T) {
	t.Parallel()

	validSpan := mustQuerySpan(t, 0, len("民法"))
	invalidSpan := legalquery.QuerySpan{}
	validRevision := "129AC0000000089_20250601_504AC0000000068"
	validLawNumber := "明治二十九年法律第八十九号"

	tests := map[string]func() error{
		"law name の span 欠落": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: invalidSpan, Surface: "民法", LawID: "law-1",
				RevisionID: "revision-1", LawNumber: "法律第一号",
				Canonical: "民法", MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の surface と span 幅の不一致": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民", LawID: "law-1",
				RevisionID: "revision-1", LawNumber: "法律第一号",
				Canonical: "民法", MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の law ID 欠落": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民法", RevisionID: "revision-1",
				LawNumber: "法律第一号", Canonical: "民法",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の revision ID 欠落": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民法", LawID: "law-1",
				LawNumber: "法律第一号", Canonical: "民法",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の法令番号欠落": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民法", LawID: "law-1",
				RevisionID: "revision-1", Canonical: "民法",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の正式名称欠落": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民法", LawID: "law-1",
				RevisionID: "revision-1", LawNumber: "法律第一号",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"law name の未知 match kind": func() error {
			_, err := legalquery.NewLawNameMention(legalquery.LawNameMentionValues{
				Span: validSpan, Surface: "民法", LawID: "law-1",
				RevisionID: "revision-1", LawNumber: "法律第一号",
				Canonical: "民法", MatchKind: legalquery.PreprocessMatchKind("unknown"),
			})
			return err
		},
		"concept ID 欠落": func() error {
			_, err := legalquery.NewLegalConceptMention(
				legalquery.LegalConceptMentionValues{
					Span: validSpan, Surface: "民法", Canonical: "民法",
					MatchKind: legalquery.PreprocessMatchExact,
				},
			)
			return err
		},
		"concept 正式表記欠落": func() error {
			_, err := legalquery.NewLegalConceptMention(
				legalquery.LegalConceptMentionValues{
					Span: validSpan, Surface: "民法", ConceptID: "civil-law",
					MatchKind: legalquery.PreprocessMatchExact,
				},
			)
			return err
		},
		"cue profile ID 欠落": func() error {
			_, err := legalquery.NewCueMention(legalquery.CueMentionValues{
				Span: validSpan, Surface: "民法", CueID: "law",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"cue ID 欠落": func() error {
			_, err := legalquery.NewCueMention(legalquery.CueMentionValues{
				Span: validSpan, Surface: "民法", ProfileID: "core-law-v1",
				MatchKind: legalquery.PreprocessMatchExact,
			})
			return err
		},
		"cue の誤記補正": func() error {
			_, err := legalquery.NewCueMention(legalquery.CueMentionValues{
				Span: validSpan, Surface: "民法", ProfileID: "core-law-v1",
				CueID: "law", MatchKind: legalquery.PreprocessMatchUniqueTypoCorrection,
			})
			return err
		},
		"identifier の未知 kind": func() error {
			_, err := legalquery.NewIdentifierMention(
				legalquery.IdentifierMentionValues{
					Span: validSpan, Surface: "民法",
					Kind: legalquery.IdentifierMentionKind("unknown"), LawID: "law-1",
				},
			)
			return err
		},
		"law ID kind に revision ID": func() error {
			_, err := legalquery.NewIdentifierMention(
				legalquery.IdentifierMentionValues{
					Span: validSpan, Surface: "民法",
					Kind: legalquery.IdentifierMentionLawID, LawID: "law-1",
					RevisionID: &validRevision,
				},
			)
			return err
		},
		"revision kind の revision ID 欠落": func() error {
			_, err := legalquery.NewIdentifierMention(
				legalquery.IdentifierMentionValues{
					Span: validSpan, Surface: "民法",
					Kind: legalquery.IdentifierMentionLawRevisionID, LawID: "law-1",
				},
			)
			return err
		},
		"revision kind に法令番号": func() error {
			_, err := legalquery.NewIdentifierMention(
				legalquery.IdentifierMentionValues{
					Span: validSpan, Surface: "民法",
					Kind: legalquery.IdentifierMentionLawRevisionID, LawID: "law-1",
					RevisionID: &validRevision, LawNumber: &validLawNumber,
				},
			)
			return err
		},
		"law number kind の法令番号欠落": func() error {
			_, err := legalquery.NewIdentifierMention(
				legalquery.IdentifierMentionValues{
					Span: validSpan, Surface: "民法",
					Kind: legalquery.IdentifierMentionLawNumber, LawID: "law-1",
				},
			)
			return err
		},
		"date のゼロ値": func() error {
			_, err := legalquery.NewDateMention(legalquery.DateMentionValues{
				Span: validSpan, Surface: "民法", Date: model.Date{},
			})
			return err
		},
		"article の未知 provision": func() error {
			_, err := legalquery.NewArticleMention(legalquery.ArticleMentionValues{
				Span: validSpan, Surface: "民法",
				Provision: model.LawArticleProvision("unknown"), ArticleNumber: "1",
			})
			return err
		},
		"article の不正な正規形": func() error {
			_, err := legalquery.NewArticleMention(legalquery.ArticleMentionValues{
				Span: validSpan, Surface: "民法",
				Provision: model.LawArticleProvisionMain, ArticleNumber: "398の2",
			})
			return err
		},
		"paragraph の零": func() error {
			_, err := legalquery.NewParagraphMention(
				legalquery.ParagraphMentionValues{
					Span: validSpan, Surface: "民法", ParagraphNumber: 0,
				},
			)
			return err
		},
	}
	for name, execute := range tests {
		name := name
		execute := execute
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if err := execute(); err == nil {
				t.Fatal("SOT-MODEL-025: 不正な mention を受理しました")
			}
		})
	}
}

func TestQuerySpanRejectsInvalidHalfOpenRange(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]legalquery.QuerySpanValues{
		"負の start": {StartByte: -1, EndByte: 1},
		"同じ境界":     {StartByte: 1, EndByte: 1},
		"逆転した境界":   {StartByte: 2, EndByte: 1},
		"ゼロ値":      {},
	} {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := legalquery.NewQuerySpan(values); err == nil {
				t.Fatal("SOT-MODEL-025: 不正な half-open span を受理しました")
			}
		})
	}

	span := mustQuerySpan(t, 3, 9)
	if span.StartByte() != 3 || span.EndByte() != 9 {
		t.Fatalf(
			"SOT-MODEL-025: QuerySpan = [%d,%d)",
			span.StartByte(),
			span.EndByte(),
		)
	}
	if err := span.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-025: QuerySpan.Validate() = %v", err)
	}
}

func mustQuerySpan(t *testing.T, startByte int, endByte int) legalquery.QuerySpan {
	t.Helper()

	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf("試験用 QuerySpan を作成できません: %v", err)
	}
	return span
}

func mustPreprocessDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用 Date を作成できません: %v", err)
	}
	return date
}
