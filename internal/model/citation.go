package model

import (
	"encoding/json"
	"fmt"
	"unicode/utf8"
)

// CitationValues は、Citation の作成に必要な値を保持する。
type CitationValues struct {
	Source     LegalSource
	LawID      string
	RevisionID string
	Location   string
	URL        string
}

// Citation は、返された法情報の確認可能な出典を表す。
type Citation struct {
	source     LegalSource
	lawID      string
	revisionID string
	location   string
	url        string
}

// NewCitation は、入力を複製して検証済みの Citation を返す。
func NewCitation(values CitationValues) (Citation, error) {
	citation := Citation{
		source:     values.Source,
		lawID:      values.LawID,
		revisionID: values.RevisionID,
		location:   values.Location,
		url:        values.URL,
	}
	if err := citation.Validate(); err != nil {
		return Citation{}, err
	}
	return citation, nil
}

// Source は、引用元の法令情報源を返す。
func (c Citation) Source() LegalSource {
	return c.source
}

// LawID は、引用元の法令識別子を返す。
func (c Citation) LawID() string {
	return c.lawID
}

// RevisionID は、引用元の法令リビジョン識別子を返す。
func (c Citation) RevisionID() string {
	return c.revisionID
}

// Location は、法令内位置と有無を返す。
func (c Citation) Location() (string, bool) {
	return c.location, c.location != ""
}

// URL は、原文を確認できる HTTPS URL を返す。
func (c Citation) URL() string {
	return c.url
}

// Validate は、Citation の必須項目と URL を確認する。
func (c Citation) Validate() error {
	if err := c.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	switch {
	case c.lawID == "":
		return fmt.Errorf("lawId は必須です")
	case c.revisionID == "":
		return fmt.Errorf("revisionId は必須です")
	case !utf8.ValidString(c.lawID):
		return fmt.Errorf("lawId は有効な UTF-8 でなければなりません")
	case !utf8.ValidString(c.revisionID):
		return fmt.Errorf("revisionId は有効な UTF-8 でなければなりません")
	case !utf8.ValidString(c.location):
		return fmt.Errorf("location は有効な UTF-8 でなければなりません")
	case !isHTTPSURL(c.url):
		return fmt.Errorf("url は認証情報を含まない HTTPS URL でなければなりません")
	default:
		return nil
	}
}

// MarshalJSON は、SOT-MODEL-004 の項目名で Citation を表す。
func (c Citation) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Source     LegalSource `json:"source"`
		LawID      string      `json:"lawId"`
		RevisionID string      `json:"revisionId"`
		Location   string      `json:"location,omitempty"`
		URL        string      `json:"url"`
	}{
		Source:     c.source,
		LawID:      c.lawID,
		RevisionID: c.revisionID,
		Location:   c.location,
		URL:        c.url,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Citation) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Citation は JSON から直接復元できません。境界専用の入力型から NewCitation を使用してください",
	)
}
