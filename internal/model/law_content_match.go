package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// LawContentMatchValues は、LawContentMatch の作成に必要な値を保持する。
type LawContentMatchValues struct {
	Law      LawSummary
	Location string
	Text     string
	Citation Citation
}

// LawContentMatch は、法令本文検索で一致した一つの箇所を表す。
type LawContentMatch struct {
	law      LawSummary
	location string
	text     string
	citation Citation
}

// NewLawContentMatch は、検証済みの LawContentMatch を返す。
func NewLawContentMatch(values LawContentMatchValues) (LawContentMatch, error) {
	match := LawContentMatch{
		law:      values.Law,
		location: values.Location,
		text:     values.Text,
		citation: values.Citation,
	}
	if err := match.Validate(); err != nil {
		return LawContentMatch{}, err
	}
	return match, nil
}

// Law は、一致箇所を含む法令リビジョンを返す。
func (m LawContentMatch) Law() LawSummary {
	return m.law
}

// Location は、情報源が返した法令内の位置を返す。
func (m LawContentMatch) Location() string {
	return m.location
}

// Text は、一致箇所の本文を返す。
func (m LawContentMatch) Text() string {
	return m.text
}

// Citation は、一致箇所の確認可能な出典を返す。
func (m LawContentMatch) Citation() Citation {
	return m.citation
}

// Validate は、必須項目と法令・出典の同一性を確認する。
func (m LawContentMatch) Validate() error {
	if err := m.law.Validate(); err != nil {
		return fmt.Errorf("law が有効ではありません: %w", err)
	}
	switch {
	case m.location == "":
		return fmt.Errorf("location は必須です")
	case !utf8.ValidString(m.location):
		return fmt.Errorf("location は有効な UTF-8 でなければなりません")
	case m.text == "":
		return fmt.Errorf("text は必須です")
	case !utf8.ValidString(m.text):
		return fmt.Errorf("text は有効な UTF-8 でなければなりません")
	}
	if err := m.citation.Validate(); err != nil {
		return fmt.Errorf("citation が有効ではありません: %w", err)
	}
	citationLocation, exists := m.citation.Location()
	if !exists {
		return fmt.Errorf("citation.location は必須です")
	}
	if citationLocation != m.location {
		return fmt.Errorf("citation.location と location が一致しません")
	}
	if m.citation.Source() != m.law.Source() {
		return fmt.Errorf("citation.source と law.source が一致しません")
	}
	if m.citation.LawID() != m.law.LawID() {
		return fmt.Errorf("citation.lawId と law.lawId が一致しません")
	}
	if m.citation.RevisionID() != m.law.RevisionID() {
		return fmt.Errorf("citation.revisionId と law.revisionId が一致しません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-007 の項目名で一致箇所を表す。
func (m LawContentMatch) MarshalJSON() ([]byte, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Law      LawSummary `json:"law"`
		Location string     `json:"location"`
		Text     string     `json:"text"`
		Citation Citation   `json:"citation"`
	}{
		Law:      m.law,
		Location: m.location,
		Text:     m.text,
		Citation: m.citation,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*LawContentMatch) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LawContentMatch は JSON から直接復元できません。境界専用の入力型から NewLawContentMatch を使用してください",
	)
}
