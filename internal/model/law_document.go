package model

import (
	"encoding/json"
	"fmt"
)

// LawDocumentValues は、LawDocument の作成に必要な値を保持する。
type LawDocumentValues struct {
	Law      LawSummary
	AsOf     *Date
	Content  string
	Citation Citation
}

// LawDocument は、公開互換の XML 法令本文を表す。
type LawDocument struct {
	law      LawSummary
	asOf     *Date
	content  string
	citation Citation
}

// NewLawDocument は、XML 形式に限定した検証済みの LawDocument を返す。
func NewLawDocument(values LawDocumentValues) (LawDocument, error) {
	document := LawDocument{
		law:      values.Law,
		asOf:     cloneOptionalDate(values.AsOf),
		content:  values.Content,
		citation: values.Citation,
	}
	if err := document.Validate(); err != nil {
		return LawDocument{}, err
	}
	return document, nil
}

// NewLawDocumentFromRepresentation は、XML 表現だけを公開互換の LawDocument へ投影する。
func NewLawDocumentFromRepresentation(
	representation LawDocumentRepresentation,
) (LawDocument, error) {
	if err := representation.Validate(); err != nil {
		return LawDocument{}, fmt.Errorf("representation が有効ではありません: %w", err)
	}
	if representation.Format() != LawDocumentFormatXML {
		return LawDocument{}, fmt.Errorf("format が xml の表現だけを LawDocument へ投影できます")
	}
	asOf, exists := representation.AsOf()
	var pointer *Date
	if exists {
		pointer = &asOf
	}
	return NewLawDocument(LawDocumentValues{
		Law:      representation.Law(),
		AsOf:     pointer,
		Content:  representation.Content(),
		Citation: representation.Citation(),
	})
}

// Law は、対象法令と選択されたリビジョンを返す。
func (d LawDocument) Law() LawSummary {
	return d.law
}

// AsOf は、検索基準日と有無を返す。
func (d LawDocument) AsOf() (Date, bool) {
	if d.asOf == nil {
		return Date{}, false
	}
	return *d.asOf, true
}

// Format は、公開モデルの固定形式 xml を返す。
func (d LawDocument) Format() string {
	return LawDocumentFormatXML
}

// Content は、UTF-8 XML の本文を返す。
func (d LawDocument) Content() string {
	return d.content
}

// Citation は、確認可能な出典を返す。
func (d LawDocument) Citation() Citation {
	return d.citation
}

// Validate は、LawDocument が XML 互換表現であることを確認する。
func (d LawDocument) Validate() error {
	representation, err := NewLawDocumentRepresentation(
		LawDocumentRepresentationValues{
			Law:      d.law,
			AsOf:     cloneOptionalDate(d.asOf),
			Format:   LawDocumentFormatXML,
			Content:  d.content,
			Citation: d.citation,
		},
	)
	if err != nil {
		return err
	}
	if representation.Format() != LawDocumentFormatXML {
		return fmt.Errorf("format は xml でなければなりません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-002 の項目名で LawDocument を表す。
func (d LawDocument) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Law      LawSummary `json:"law"`
		AsOf     *Date      `json:"asOf,omitempty"`
		Format   string     `json:"format"`
		Content  string     `json:"content"`
		Citation Citation   `json:"citation"`
	}{
		Law:      d.law,
		AsOf:     cloneOptionalDate(d.asOf),
		Format:   LawDocumentFormatXML,
		Content:  d.content,
		Citation: d.citation,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawDocument) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawDocument は JSON から直接復元できません。境界専用の入力型から NewLawDocument を使用してください",
	)
}
