package querypreprocess_test

import (
	"context"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestPreprocessKeepsLawAndArticleSpansForProfilePairing(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	const query = "民法第709条と刑法第199条"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	laws := result.LawNameMentions()
	if len(laws) != 2 {
		t.Fatalf("SOT-MODEL-025: lawNameMentions = %#v", laws)
	}
	if laws[0].LawID() != civilLawID || laws[0].Canonical() != "民法" {
		t.Fatalf("SOT-MODEL-025: 最初の法令名 = %#v", laws[0])
	}
	assertSpan(t, query, laws[0].Span(), laws[0].Surface(), 0)
	if laws[1].LawID() != penalLawID || laws[1].Canonical() != "刑法" {
		t.Fatalf("SOT-MODEL-025: 二番目の法令名 = %#v", laws[1])
	}
	assertSpan(
		t,
		query,
		laws[1].Span(),
		laws[1].Surface(),
		strings.Index(query, "刑法"),
	)

	articles := result.ArticleMentions()
	if len(articles) != 2 {
		t.Fatalf("SOT-MODEL-025: articleMentions = %#v", articles)
	}
	if articles[0].ArticleNumber() != "709" ||
		articles[0].Provision() != model.LawArticleProvisionMain {
		t.Fatalf("SOT-MODEL-025: 最初の条番号 = %#v", articles[0])
	}
	assertSpan(
		t,
		query,
		articles[0].Span(),
		articles[0].Surface(),
		strings.Index(query, "第709条"),
	)
	if articles[1].ArticleNumber() != "199" ||
		articles[1].Provision() != model.LawArticleProvisionMain {
		t.Fatalf("SOT-MODEL-025: 二番目の条番号 = %#v", articles[1])
	}
	assertSpan(
		t,
		query,
		articles[1].Span(),
		articles[1].Surface(),
		strings.Index(query, "第199条"),
	)
	if laws[0].Span().StartByte() >= articles[0].Span().StartByte() ||
		articles[0].Span().StartByte() >= laws[1].Span().StartByte() ||
		laws[1].Span().StartByte() >= articles[1].Span().StartByte() {
		t.Fatalf(
			"SOT-MODEL-025: profile が対応付けに使う順序を失いました: %#v",
			snapshotResult(result),
		)
	}
}

func TestPreprocessKeepsMultipleParagraphsAndNormalizesBranchArticles(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	const query = "民法第709条第1項及び第2項、附則第３９８条の２"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	paragraphs := result.ParagraphMentions()
	if len(paragraphs) != 2 ||
		paragraphs[0].ParagraphNumber() != 1 ||
		paragraphs[1].ParagraphNumber() != 2 {
		t.Fatalf("SOT-MODEL-025: paragraphMentions = %#v", paragraphs)
	}
	assertSpan(
		t,
		query,
		paragraphs[0].Span(),
		paragraphs[0].Surface(),
		strings.Index(query, "第1項"),
	)
	assertSpan(
		t,
		query,
		paragraphs[1].Span(),
		paragraphs[1].Surface(),
		strings.Index(query, "第2項"),
	)

	articles := result.ArticleMentions()
	if len(articles) != 2 {
		t.Fatalf("SOT-MODEL-025: articleMentions = %#v", articles)
	}
	if articles[0].ArticleNumber() != "709" ||
		articles[0].Provision() != model.LawArticleProvisionMain {
		t.Fatalf("SOT-MODEL-025: 本則の条番号 = %#v", articles[0])
	}
	if articles[1].ArticleNumber() != "398_2" ||
		articles[1].Provision() != model.LawArticleProvisionSupplementary {
		t.Fatalf("SOT-MODEL-025: 附則の枝番号 = %#v", articles[1])
	}
	const supplementarySurface = "附則第３９８条の２"
	assertSpan(
		t,
		query,
		articles[1].Span(),
		articles[1].Surface(),
		strings.Index(query, supplementarySurface),
	)
}

func TestPreprocessAcceptsOnlyCompleteRealExplicitDates(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	const query = "2026年7月1日、2026-07-02、2026年2月30日、2026年7月、" +
		civilRevisionID
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	dates := result.DateMentions()
	if len(dates) != 2 {
		t.Fatalf("SOT-MODEL-025: dateMentions = %#v", dates)
	}
	if dates[0].Surface() != "2026年7月1日" ||
		dates[0].Date().String() != "2026-07-01" {
		t.Fatalf("SOT-MODEL-025: 和文日付 = %#v", dates[0])
	}
	if dates[1].Surface() != "2026-07-02" ||
		dates[1].Date().String() != "2026-07-02" {
		t.Fatalf("SOT-MODEL-025: ISO 日付 = %#v", dates[1])
	}
	for _, mention := range dates {
		assertSpan(
			t,
			query,
			mention.Span(),
			mention.Surface(),
			strings.Index(query, mention.Surface()),
		)
	}
}

func TestPreprocessRecognizesOnlyIdentifiersKnownByFixedLawSnapshot(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	const unknownLawID = "999AC0000009999"
	query := strings.Join([]string{
		civilLawID,
		civilRevisionID,
		civilLawNumber,
		unknownLawID,
	}, "、")
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	identifiers := result.IdentifierMentions()
	if len(identifiers) != 3 {
		t.Fatalf("SOT-MODEL-025: identifierMentions = %#v", identifiers)
	}
	assertIdentifierMention(
		t,
		query,
		identifiers,
		legalquery.IdentifierMentionLawID,
		civilLawID,
		civilLawID,
	)
	assertIdentifierMention(
		t,
		query,
		identifiers,
		legalquery.IdentifierMentionLawRevisionID,
		civilRevisionID,
		civilLawID,
	)
	assertIdentifierMention(
		t,
		query,
		identifiers,
		legalquery.IdentifierMentionLawNumber,
		civilLawNumber,
		civilLawID,
	)
	for _, mention := range identifiers {
		if mention.Surface() == unknownLawID {
			t.Fatalf(
				"SOT-MODEL-025: 未知の識別子を公式識別子として採用しました: %#v",
				mention,
			)
		}
	}
}

func TestPreprocessRecognizesHistoricalRevisionForKnownLawID(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	tests := map[string]string{
		"過去の改正法令": civilLawID + "_20240401_504AC0000000102",
		"制定時の零ID": civilLawID + "_19000101_000000000000000",
	}
	for name, revisionID := range tests {
		name, revisionID := name, revisionID
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			query := "法令履歴ID " + revisionID + " の法令本文を読む"

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}

			identifiers := result.IdentifierMentions()
			if len(identifiers) != 1 {
				t.Fatalf("SOT-MODEL-025: identifierMentions = %#v", identifiers)
			}
			assertIdentifierMention(
				t,
				query,
				identifiers,
				legalquery.IdentifierMentionLawRevisionID,
				revisionID,
				civilLawID,
			)
		})
	}
}

func TestPreprocessRejectsInvalidHistoricalRevisionWithoutLawIDFallback(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	valid := civilLawID + "_20240401_504AC0000000102"
	tests := map[string]string{
		"未知の法令ID":    "999AC0000009999_20240401_504AC0000000102",
		"存在しない日付":    civilLawID + "_20240230_504AC0000000102",
		"小文字の改正ID":   civilLawID + "_20240401_504ac0000000102",
		"前方のASCII連結": "X" + valid,
		"後方のASCII連結": valid + "X",
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			query := "法令履歴ID " + value + " の法令本文を読む"

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			if identifiers := result.IdentifierMentions(); len(identifiers) != 0 {
				t.Fatalf(
					"SOT-MODEL-025: 不正な履歴IDを法令IDへ縮退しました: %#v",
					identifiers,
				)
			}
		})
	}
}

func assertIdentifierMention(
	t *testing.T,
	query string,
	mentions []legalquery.IdentifierMention,
	kind legalquery.IdentifierMentionKind,
	surface string,
	lawID string,
) {
	t.Helper()

	startByte := strings.Index(query, surface)
	for _, mention := range mentions {
		if mention.Kind() != kind ||
			mention.Surface() != surface ||
			mention.LawID() != lawID ||
			mention.Span().StartByte() != startByte {
			continue
		}
		assertSpan(t, query, mention.Span(), surface, startByte)
		switch kind {
		case legalquery.IdentifierMentionLawRevisionID:
			revisionID, exists := mention.RevisionID()
			if !exists || revisionID != surface {
				t.Fatalf(
					"SOT-MODEL-025: revision ID = %q, %t",
					revisionID,
					exists,
				)
			}
		case legalquery.IdentifierMentionLawNumber:
			lawNumber, exists := mention.LawNumber()
			if !exists || lawNumber != surface {
				t.Fatalf(
					"SOT-MODEL-025: law number = %q, %t",
					lawNumber,
					exists,
				)
			}
		}
		return
	}
	t.Fatalf(
		"SOT-MODEL-025: kind=%s surface=%q lawID=%q の出現がありません: %#v",
		kind,
		surface,
		lawID,
		mentions,
	)
}
