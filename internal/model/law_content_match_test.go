package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLawContentMatch(t *testing.T) {
	t.Parallel()

	values := validLawContentMatchValues(t)
	got, err := model.NewLawContentMatch(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-007: NewLawContentMatch() のエラー = %v", err)
	}
	if got.Law() != values.Law ||
		got.Location() != values.Location ||
		got.Text() != values.Text ||
		got.Citation() != values.Citation {
		t.Fatalf("SOT-MODEL-007: LawContentMatch = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-007: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-007/009: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-007/009: JSON を再解析できない: %v", err)
	}
	if !reflect.DeepEqual(object["location"], values.Location) ||
		!reflect.DeepEqual(object["text"], values.Text) {
		t.Fatalf("SOT-MODEL-007/009: JSON = %#v", object)
	}
	for _, key := range []string{"law", "location", "text", "citation"} {
		if _, exists := object[key]; !exists {
			t.Fatalf("SOT-MODEL-007/009: JSON に %s がない: %s", key, encoded)
		}
	}
}

func TestLawContentMatchRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := validLawContentMatchValues(t)
	otherLaw, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "other-law",
		RevisionID: "other-revision",
		Title:      "別の法令",
		Source:     valid.Law.Source(),
	})
	if err != nil {
		t.Fatalf("別の LawSummary を作成できない: %v", err)
	}
	withoutLocation, err := model.NewCitation(model.CitationValues{
		Source:     valid.Law.Source(),
		LawID:      valid.Law.LawID(),
		RevisionID: valid.Law.RevisionID(),
		URL:        "https://laws.e-gov.go.jp/law/" + valid.Law.LawID(),
	})
	if err != nil {
		t.Fatalf("位置なし Citation を作成できない: %v", err)
	}
	mismatchedLocation, err := model.NewCitation(model.CitationValues{
		Source:     valid.Law.Source(),
		LawID:      valid.Law.LawID(),
		RevisionID: valid.Law.RevisionID(),
		Location:   "main:article=9",
		URL:        "https://laws.e-gov.go.jp/law/" + valid.Law.LawID(),
	})
	if err != nil {
		t.Fatalf("異なる位置の Citation を作成できない: %v", err)
	}

	tests := map[string]model.LawContentMatchValues{
		"law のゼロ値": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Law = model.LawSummary{}
		}),
		"location の欠落": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Location = ""
		}),
		"location の UTF-8 不正": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Location = string([]byte{0xff})
		}),
		"text の欠落": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Text = ""
		}),
		"text の UTF-8 不正": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Text = string([]byte{0xff})
		}),
		"citation のゼロ値": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = model.Citation{}
		}),
		"citation.location の欠落": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = withoutLocation
		}),
		"citation.location の不一致": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = mismatchedLocation
		}),
		"citation.source の不一致": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = newContentCitation(
				t,
				newDocumentSupplementaryLegalSource(t),
				values.Law.LawID(),
				values.Law.RevisionID(),
			)
		}),
		"citation.lawId の不一致": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = newContentCitation(
				t,
				values.Law.Source(),
				otherLaw.LawID(),
				values.Law.RevisionID(),
			)
		}),
		"citation.revisionId の不一致": withLawContentMatchChange(valid, func(values *model.LawContentMatchValues) {
			values.Citation = newContentCitation(
				t,
				values.Law.Source(),
				values.Law.LawID(),
				otherLaw.RevisionID(),
			)
		}),
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewLawContentMatch(values); err == nil {
				t.Fatalf("SOT-MODEL-007: NewLawContentMatch(%#v) が成功した", values)
			}
		})
	}
}

func TestLawContentMatchRejectsInvalidZeroValueAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LawContentMatch{}); err == nil {
		t.Fatal("SOT-MODEL-007/009: ゼロ値を JSON に変換できた")
	}
	var got model.LawContentMatch
	if err := json.Unmarshal([]byte(`{}`), &got); err == nil {
		t.Fatal("SOT-MODEL-007/009: JSON から直接復元できた")
	}
}

func validLawContentMatchValues(t *testing.T) model.LawContentMatchValues {
	t.Helper()

	law := newLawSummary(t)
	return model.LawContentMatchValues{
		Law:      law,
		Location: "main:article=1;paragraph=2",
		Text:     "行政庁は、申請により求められた許認可等をするかどうかを審査する。",
		Citation: newContentCitation(t, law.Source(), law.LawID(), law.RevisionID()),
	}
}

func newContentCitation(
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
		Location:   "main:article=1;paragraph=2",
		URL:        "https://laws.e-gov.go.jp/law/" + lawID,
	})
	if err != nil {
		t.Fatalf("Citation を作成できない: %v", err)
	}
	return citation
}

func withLawContentMatchChange(
	values model.LawContentMatchValues,
	change func(*model.LawContentMatchValues),
) model.LawContentMatchValues {
	change(&values)
	return values
}
