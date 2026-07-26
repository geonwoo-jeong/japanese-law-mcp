package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

const (
	LawArticleFormatXML  = LawDocumentFormatXML
	LawArticleFormatHTML = LawDocumentFormatHTML
	LawArticleFormatText = LawDocumentFormatText
)

// LawArticleFragmentValues は、LawArticleFragment の作成に必要な値を保持する。
type LawArticleFragmentValues struct {
	Law      LawSummary
	Location LawArticleLocation
	Format   string
	Content  string
	Citation Citation
}

// LawArticleFragment は、一つの法令リビジョンから選択した条または項を保持する。
type LawArticleFragment struct {
	law      LawSummary
	location LawArticleLocation
	format   string
	content  string
	citation Citation
}

// NewLawArticleFragment は、検証済みの LawArticleFragment を返す。
func NewLawArticleFragment(values LawArticleFragmentValues) (LawArticleFragment, error) {
	fragment := LawArticleFragment{
		law:      values.Law,
		location: values.Location,
		format:   values.Format,
		content:  values.Content,
		citation: values.Citation,
	}
	if err := fragment.Validate(); err != nil {
		return LawArticleFragment{}, err
	}
	return fragment, nil
}

// Law は、対象法令と選択されたリビジョンを返す。
func (f LawArticleFragment) Law() LawSummary {
	return f.law
}

// Location は、選択した条または項の位置を返す。
func (f LawArticleFragment) Location() LawArticleLocation {
	return f.location
}

// Format は、表現形式を返す。
func (f LawArticleFragment) Format() string {
	return f.format
}

// Content は、条または項の文字列表現を返す。
func (f LawArticleFragment) Content() string {
	return f.content
}

// Citation は、確認可能な出典を返す。
func (f LawArticleFragment) Citation() Citation {
	return f.citation
}

// Validate は、条文 fragment の必須項目と法令・出典の同一性を確認する。
func (f LawArticleFragment) Validate() error {
	if err := f.law.Validate(); err != nil {
		return fmt.Errorf("law が有効ではありません: %w", err)
	}
	if err := f.location.Validate(); err != nil {
		return fmt.Errorf("location が有効ではありません: %w", err)
	}
	switch f.format {
	case LawArticleFormatXML, LawArticleFormatHTML, LawArticleFormatText:
	default:
		return fmt.Errorf("format は xml、html または text でなければなりません")
	}
	switch {
	case f.content == "":
		return fmt.Errorf("content は必須です")
	case !utf8.ValidString(f.content):
		return fmt.Errorf("content は有効な UTF-8 でなければなりません")
	}
	if err := f.citation.Validate(); err != nil {
		return fmt.Errorf("citation が有効ではありません: %w", err)
	}
	if _, exists := f.citation.Location(); !exists {
		return fmt.Errorf("citation.location は必須です")
	}
	if f.citation.Source() != f.law.Source() {
		return fmt.Errorf("citation.source と law.source が一致しません")
	}
	if f.citation.LawID() != f.law.LawID() {
		return fmt.Errorf("citation.lawId と law.lawId が一致しません")
	}
	if f.citation.RevisionID() != f.law.RevisionID() {
		return fmt.Errorf("citation.revisionId と law.revisionId が一致しません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-015 の項目名で条文 fragment を表す。
func (f LawArticleFragment) MarshalJSON() ([]byte, error) {
	if err := f.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Law      LawSummary         `json:"law"`
		Location LawArticleLocation `json:"location"`
		Format   string             `json:"format"`
		Content  string             `json:"content"`
		Citation Citation           `json:"citation"`
	}{
		Law:      f.law,
		Location: f.location,
		Format:   f.format,
		Content:  f.content,
		Citation: f.citation,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawArticleFragment) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawArticleFragment は JSON から直接復元できません。境界専用の入力型から NewLawArticleFragment を使用してください",
	)
}
