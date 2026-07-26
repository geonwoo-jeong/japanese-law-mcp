package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

const (
	LawDocumentFormatXML  = "xml"
	LawDocumentFormatHTML = "html"
	LawDocumentFormatText = "text"
)

// LawDocumentRepresentationValues は、LawDocumentRepresentation の作成に必要な値を保持する。
type LawDocumentRepresentationValues struct {
	Law      LawSummary
	AsOf     *Date
	Format   string
	Content  string
	Citation Citation
}

// LawDocumentRepresentation は、内部 capability の法令本文表現を保持する。
type LawDocumentRepresentation struct {
	law      LawSummary
	asOf     *Date
	format   string
	content  string
	citation Citation
}

// NewLawDocumentRepresentation は、入力を複製して検証済みの LawDocumentRepresentation を返す。
func NewLawDocumentRepresentation(
	values LawDocumentRepresentationValues,
) (LawDocumentRepresentation, error) {
	representation := LawDocumentRepresentation{
		law:      values.Law,
		asOf:     cloneOptionalDate(values.AsOf),
		format:   values.Format,
		content:  values.Content,
		citation: values.Citation,
	}
	if err := representation.Validate(); err != nil {
		return LawDocumentRepresentation{}, err
	}
	return representation, nil
}

// Law は、対象法令と選択されたリビジョンを返す。
func (r LawDocumentRepresentation) Law() LawSummary {
	return r.law
}

// AsOf は、検索基準日と有無を返す。
func (r LawDocumentRepresentation) AsOf() (Date, bool) {
	if r.asOf == nil {
		return Date{}, false
	}
	return *r.asOf, true
}

// Format は、表現形式を返す。
func (r LawDocumentRepresentation) Format() string {
	return r.format
}

// Content は、法令本文の文字列表現を返す。
func (r LawDocumentRepresentation) Content() string {
	return r.content
}

// Citation は、確認可能な出典を返す。
func (r LawDocumentRepresentation) Citation() Citation {
	return r.citation
}

// Validate は、LawDocumentRepresentation の項目整合を確認する。
func (r LawDocumentRepresentation) Validate() error {
	if err := r.law.Validate(); err != nil {
		return fmt.Errorf("law が有効ではありません: %w", err)
	}
	if r.asOf != nil {
		if err := r.asOf.Validate(); err != nil {
			return fmt.Errorf("asOf が有効ではありません: %w", err)
		}
	}
	switch r.format {
	case LawDocumentFormatXML, LawDocumentFormatHTML, LawDocumentFormatText:
	default:
		return fmt.Errorf("format は xml、html または text でなければなりません")
	}
	switch {
	case r.content == "":
		return fmt.Errorf("content は必須です")
	case !utf8.ValidString(r.content):
		return fmt.Errorf("content は有効な UTF-8 でなければなりません")
	}
	if err := r.citation.Validate(); err != nil {
		return fmt.Errorf("citation が有効ではありません: %w", err)
	}
	if r.citation.Source() != r.law.Source() {
		return fmt.Errorf("citation.source と law.source が一致しません")
	}
	if r.citation.LawID() != r.law.LawID() {
		return fmt.Errorf("citation.lawId と law.lawId が一致しません")
	}
	if r.citation.RevisionID() != r.law.RevisionID() {
		return fmt.Errorf("citation.revisionId と law.revisionId が一致しません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-017 の項目名で LawDocumentRepresentation を表す。
func (r LawDocumentRepresentation) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Law      LawSummary `json:"law"`
		AsOf     *Date      `json:"asOf,omitempty"`
		Format   string     `json:"format"`
		Content  string     `json:"content"`
		Citation Citation   `json:"citation"`
	}{
		Law:      r.law,
		AsOf:     cloneOptionalDate(r.asOf),
		Format:   r.format,
		Content:  r.content,
		Citation: r.citation,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawDocumentRepresentation) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawDocumentRepresentation は JSON から直接復元できません。境界専用の入力型から NewLawDocumentRepresentation を使用してください",
	)
}
