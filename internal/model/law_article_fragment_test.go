package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLawArticleFragment(t *testing.T) {
	t.Parallel()

	law := newLawSummary(t)
	location := newArticleLocation(t, model.LawArticleProvisionMain, "38_3", 2)
	citation := newArticleCitation(t, law, "main:article=38_3;paragraph=2")
	got, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      law,
		Location: location,
		Format:   model.LawArticleFormatXML,
		Content:  `<Paragraph Num="2"><ParagraphSentence>手続</ParagraphSentence></Paragraph>`,
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-015: NewLawArticleFragment() のエラー = %v", err)
	}

	if got.Law() != law ||
		got.Location() != location ||
		got.Format() != model.LawArticleFormatXML ||
		got.Content() == "" ||
		got.Citation() != citation {
		t.Fatalf("SOT-MODEL-015: LawArticleFragment = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-015: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/015: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/015: JSON を再解析できない: %v", err)
	}
	if !reflect.DeepEqual(object["location"], map[string]any{
		"provision":       "main",
		"articleNumber":   "38_3",
		"paragraphNumber": float64(2),
	}) {
		t.Fatalf("SOT-MODEL-009/015: location = %#v", object["location"])
	}
	for _, key := range []string{"law", "location", "format", "content", "citation"} {
		if _, exists := object[key]; !exists {
			t.Fatalf("SOT-MODEL-009/015: JSON に %s がない: %s", key, encoded)
		}
	}
}

func TestLawArticleFragmentAcceptsSupportedFormats(t *testing.T) {
	t.Parallel()

	for _, format := range []string{
		model.LawArticleFormatXML,
		model.LawArticleFormatHTML,
		model.LawArticleFormatText,
	} {
		format := format
		t.Run(format, func(t *testing.T) {
			t.Parallel()

			values := validLawArticleFragmentValues(t)
			values.Format = format
			if _, err := model.NewLawArticleFragment(values); err != nil {
				t.Fatalf("SOT-MODEL-015/017: format %q を拒否した: %v", format, err)
			}
		})
	}
}

func TestLawArticleFragmentRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := validLawArticleFragmentValues(t)
	otherLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "other-law",
		RevisionID: "other-revision",
		Title:      "別の法令",
		Source:     valid.Law.Source(),
	})
	if err != nil {
		t.Fatalf("別の LawSummary を作成できない: %v", err)
	}
	otherSource := newDocumentSupplementaryLegalSource(t)
	citationWithoutLocation, err := model.NewCitation(model.CitationValues{
		Source:     valid.Law.Source(),
		LawID:      valid.Law.LawID(),
		RevisionID: valid.Law.RevisionID(),
		URL:        "https://example.com/law",
	})
	if err != nil {
		t.Fatalf("location のない Citation を作成できない: %v", err)
	}

	tests := map[string]model.LawArticleFragmentValues{
		"law のゼロ値": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Law = model.LawSummary{}
		}),
		"location のゼロ値": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Location = model.LawArticleLocation{}
		}),
		"format の欠落": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Format = ""
		}),
		"未知の format": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Format = "pdf"
		}),
		"content の欠落": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Content = ""
		}),
		"content の UTF-8 不正": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Content = string([]byte{0xff})
		}),
		"citation のゼロ値": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Citation = model.Citation{}
		}),
		"citation.location の欠落": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Citation = citationWithoutLocation
		}),
		"citation.source の不一致": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Citation = newArticleCitationWithValues(
				t,
				otherSource,
				values.Law.LawID(),
				values.Law.RevisionID(),
			)
		}),
		"citation.lawId の不一致": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Citation = newArticleCitation(t, otherLaw, "main:article=1")
		}),
		"citation.revisionId の不一致": withLawArticleFragmentChange(valid, func(values *model.LawArticleFragmentValues) {
			values.Citation = newArticleCitationWithValues(
				t,
				values.Law.Source(),
				values.Law.LawID(),
				"other-revision",
			)
		}),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewLawArticleFragment(values); err == nil {
				t.Fatalf("SOT-MODEL-015: NewLawArticleFragment(%#v) が成功した", values)
			}
		})
	}
}

func TestLawArticleFragmentZeroValueCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LawArticleFragment{}); err == nil {
		t.Fatal("SOT-MODEL-009/015: LawArticleFragment のゼロ値を JSON に変換できた")
	}
}

func TestLawArticleFragmentRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.LawArticleFragment
	if err := json.Unmarshal([]byte(`{}`), &got); err == nil {
		t.Fatal("SOT-MODEL-009/015: LawArticleFragment を JSON から直接復元できた")
	}
}

func validLawArticleFragmentValues(t *testing.T) model.LawArticleFragmentValues {
	t.Helper()

	law := newLawSummary(t)
	return model.LawArticleFragmentValues{
		Law:      law,
		Location: newArticleLocation(t, model.LawArticleProvisionMain, "1", 0),
		Format:   model.LawArticleFormatXML,
		Content:  `<Article Num="1"></Article>`,
		Citation: newArticleCitation(t, law, "main:article=1"),
	}
}

func newArticleLocation(
	t *testing.T,
	provision model.LawArticleProvision,
	articleNumber string,
	paragraphNumber int,
) model.LawArticleLocation {
	t.Helper()

	values := model.LawArticleLocationValues{
		Provision:     provision,
		ArticleNumber: articleNumber,
	}
	if paragraphNumber != 0 {
		values.ParagraphNumber = &paragraphNumber
	}
	location, err := model.NewLawArticleLocation(values)
	if err != nil {
		t.Fatalf("LawArticleLocation を作成できない: %v", err)
	}
	return location
}

func newArticleCitation(
	t *testing.T,
	law model.LawSummary,
	location string,
) model.Citation {
	t.Helper()

	citation, err := model.NewCitation(model.CitationValues{
		Source:     law.Source(),
		LawID:      law.LawID(),
		RevisionID: law.RevisionID(),
		Location:   location,
		URL:        "https://laws.e-gov.go.jp/law/" + law.LawID(),
	})
	if err != nil {
		t.Fatalf("Citation を作成できない: %v", err)
	}
	return citation
}

func newArticleCitationWithValues(
	t *testing.T,
	source model.LegalSource,
	lawID string,
	revisionID string,
) model.Citation {
	t.Helper()

	citation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      lawID,
		RevisionID: revisionID,
		Location:   "main:article=1",
		URL:        "https://example.com/law",
	})
	if err != nil {
		t.Fatalf("Citation を作成できない: %v", err)
	}
	return citation
}

func withLawArticleFragmentChange(
	values model.LawArticleFragmentValues,
	change func(*model.LawArticleFragmentValues),
) model.LawArticleFragmentValues {
	change(&values)
	return values
}
