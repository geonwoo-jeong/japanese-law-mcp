package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestJudicialDocumentLink(t *testing.T) {
	t.Parallel()

	got, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "全文",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/app/files/hanrei_jp/570/095570_hanrei.pdf",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-020: NewJudicialDocumentLink() のエラー = %v", err)
	}

	if got.Kind() != model.JudicialDocumentKindFullText ||
		got.Label() != "全文" ||
		got.MediaType() != model.JudicialDocumentMediaTypePDF ||
		got.URL() != "https://www.courts.go.jp/app/files/hanrei_jp/570/095570_hanrei.pdf" {
		t.Fatalf("SOT-MODEL-020: JudicialDocumentLink = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-020: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/020: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/020: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"kind":      "full_text",
		"label":     "全文",
		"mediaType": "application/pdf",
		"url":       "https://www.courts.go.jp/app/files/hanrei_jp/570/095570_hanrei.pdf",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/020: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestJudicialDocumentLinkAcceptsKinds(t *testing.T) {
	t.Parallel()

	for _, kind := range []model.JudicialDocumentKind{
		model.JudicialDocumentKindFullText,
		model.JudicialDocumentKindSummary,
		model.JudicialDocumentKindAttachment,
	} {
		kind := kind
		t.Run(string(kind), func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewJudicialDocumentLink(model.JudicialDocumentLinkValues{
				Kind:      kind,
				Label:     "公式文書",
				MediaType: model.JudicialDocumentMediaTypePDF,
				URL:       "https://www.courts.go.jp/app/files/example.pdf",
			}); err != nil {
				t.Fatalf("SOT-MODEL-020: kind %q を拒否した: %v", kind, err)
			}
		})
	}
}

func TestJudicialDocumentLinkRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := model.JudicialDocumentLinkValues{
		Kind:      model.JudicialDocumentKindFullText,
		Label:     "全文",
		MediaType: model.JudicialDocumentMediaTypePDF,
		URL:       "https://www.courts.go.jp/app/files/example.pdf",
	}
	tests := map[string]model.JudicialDocumentLinkValues{
		"未知の kind": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.Kind = model.JudicialDocumentKind("opinion")
			},
		),
		"label の欠落": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.Label = ""
			},
		),
		"未知の mediaType": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.MediaType = "text/html"
			},
		),
		"HTTP URL": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.URL = "http://www.courts.go.jp/app/files/example.pdf"
			},
		),
		"別ホスト": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.URL = "https://courts.go.jp/app/files/example.pdf"
			},
		),
		"認証情報を含む URL": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.URL = "https://user:password@www.courts.go.jp/app/files/example.pdf"
			},
		),
		"ポートを含む URL": withJudicialDocumentLinkChange(
			valid,
			func(values *model.JudicialDocumentLinkValues) {
				values.URL = "https://www.courts.go.jp:443/app/files/example.pdf"
			},
		),
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewJudicialDocumentLink(values); err == nil {
				t.Fatalf(
					"SOT-MODEL-020: NewJudicialDocumentLink(%#v) が成功した",
					values,
				)
			}
		})
	}
}

func TestJudicialDocumentLinkRejectsZeroValueAndDirectJSONDecode(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.JudicialDocumentLink{}); err == nil {
		t.Fatal("SOT-MODEL-009/020: JudicialDocumentLink のゼロ値を JSON に変換できた")
	}

	var link model.JudicialDocumentLink
	if err := json.Unmarshal(
		[]byte(
			`{"kind":"full_text","label":"全文","mediaType":"application/pdf","url":"https://www.courts.go.jp/example.pdf"}`,
		),
		&link,
	); err == nil {
		t.Fatal("SOT-MODEL-020: JudicialDocumentLink を JSON から直接復元できた")
	}
}

func withJudicialDocumentLinkChange(
	values model.JudicialDocumentLinkValues,
	change func(*model.JudicialDocumentLinkValues),
) model.JudicialDocumentLinkValues {
	change(&values)
	return values
}
