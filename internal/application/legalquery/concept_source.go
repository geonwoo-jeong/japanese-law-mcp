package legalquery

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LegalConceptSourceValues は、LegalConceptSource の作成に必要な値を保持する。
type LegalConceptSourceValues struct {
	ConceptID   string
	Title       string
	URL         string
	ConfirmedOn model.Date
}

// LegalConceptSource は、法概念との対応を確認した公的資料を表す。
type LegalConceptSource struct {
	conceptID   string
	title       string
	url         string
	confirmedOn model.Date
}

// NewLegalConceptSource は、検証済みの法概念資料を返す。
func NewLegalConceptSource(
	values LegalConceptSourceValues,
) (LegalConceptSource, error) {
	source := LegalConceptSource{
		conceptID:   values.ConceptID,
		title:       values.Title,
		url:         values.URL,
		confirmedOn: values.ConfirmedOn,
	}
	if err := source.Validate(); err != nil {
		return LegalConceptSource{}, err
	}
	return source, nil
}

// ConceptID は、法概念辞書内の安定した識別子を返す。
func (s LegalConceptSource) ConceptID() string {
	return s.conceptID
}

// Title は、対応を確認した公的資料名を返す。
func (s LegalConceptSource) Title() string {
	return s.title
}

// URL は、公的資料の絶対 HTTPS URL を返す。
func (s LegalConceptSource) URL() string {
	return s.url
}

// ConfirmedOn は、対応を確認した暦日を返す。
func (s LegalConceptSource) ConfirmedOn() model.Date {
	return s.confirmedOn
}

// Validate は、法概念資料の必須値、URL および確認日を検証する。
func (s LegalConceptSource) Validate() error {
	if err := validateRequiredInternalText("conceptId", s.conceptID); err != nil {
		return err
	}
	if err := validateRequiredInternalText("title", s.title); err != nil {
		return err
	}
	if !isCredentialFreeHTTPSURL(s.url) {
		return fmt.Errorf("url は利用者情報を含まない絶対 HTTPS URL でなければなりません")
	}
	if err := s.confirmedOn.Validate(); err != nil {
		return fmt.Errorf("confirmedOn が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、統合照会で公開できる公的資料の項目だけを表す。
func (s LegalConceptSource) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ConceptID   string     `json:"conceptId"`
		Title       string     `json:"title"`
		URL         string     `json:"url"`
		ConfirmedOn model.Date `json:"confirmedOn"`
	}{
		ConceptID:   s.conceptID,
		Title:       s.title,
		URL:         s.url,
		ConfirmedOn: s.confirmedOn,
	})
}

// UnmarshalJSON は、辞書 loader を介さない直接復元を拒否する。
func (*LegalConceptSource) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalConceptSource は JSON から直接復元できません。NewLegalConceptSource を使用してください",
	)
}

func validateRequiredInternalText(field string, value string) error {
	switch {
	case value == "":
		return fmt.Errorf("%s は必須です", field)
	case !utf8.ValidString(value):
		return fmt.Errorf("%s は有効な UTF-8 でなければなりません", field)
	case containsASCIIControl(value):
		return fmt.Errorf("%s に ASCII 制御文字を含めることはできません", field)
	default:
		return nil
	}
}

func isCredentialFreeHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.User == nil &&
		parsed.Opaque == ""
}
