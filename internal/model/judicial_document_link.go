package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// JudicialDocumentKind は、裁判例に付随する公式文書の種類を表す。
type JudicialDocumentKind string

const (
	// JudicialDocumentKindFullText は、裁判例の全文を表す。
	JudicialDocumentKindFullText JudicialDocumentKind = "full_text"
	// JudicialDocumentKindSummary は、裁判例の要旨を表す。
	JudicialDocumentKindSummary JudicialDocumentKind = "summary"
	// JudicialDocumentKindAttachment は、裁判例の別紙その他の添付文書を表す。
	JudicialDocumentKindAttachment JudicialDocumentKind = "attachment"

	// JudicialDocumentMediaTypePDF は、PDF 形式の公式文書を表す。
	JudicialDocumentMediaTypePDF = "application/pdf"
)

// JudicialDocumentLinkValues は、JudicialDocumentLink の作成に必要な値を保持する。
type JudicialDocumentLinkValues struct {
	Kind      JudicialDocumentKind
	Label     string
	MediaType string
	URL       string
}

// JudicialDocumentLink は、裁判所が直接示した一つの公式文書を表す。
type JudicialDocumentLink struct {
	kind      JudicialDocumentKind
	label     string
	mediaType string
	url       string
}

// NewJudicialDocumentLink は、検証済みで変更不能な公式文書リンクを返す。
func NewJudicialDocumentLink(
	values JudicialDocumentLinkValues,
) (JudicialDocumentLink, error) {
	link := JudicialDocumentLink{
		kind:      values.Kind,
		label:     values.Label,
		mediaType: values.MediaType,
		url:       values.URL,
	}
	if err := link.Validate(); err != nil {
		return JudicialDocumentLink{}, err
	}
	return link, nil
}

// Kind は、公式文書の種類を返す。
func (l JudicialDocumentLink) Kind() JudicialDocumentKind {
	return l.kind
}

// Label は、情報源が表示した文書名を返す。
func (l JudicialDocumentLink) Label() string {
	return l.label
}

// MediaType は、公式文書のメディア型を返す。
func (l JudicialDocumentLink) MediaType() string {
	return l.mediaType
}

// URL は、裁判所が示した公式 HTTPS URL を返す。
func (l JudicialDocumentLink) URL() string {
	return l.url
}

// Validate は、文書種別、表示名、メディア型および URL を確認する。
func (l JudicialDocumentLink) Validate() error {
	switch l.kind {
	case JudicialDocumentKindFullText,
		JudicialDocumentKindSummary,
		JudicialDocumentKindAttachment:
	default:
		return fmt.Errorf(
			"kind は full_text、summary または attachment でなければなりません",
		)
	}
	switch {
	case l.label == "":
		return fmt.Errorf("label は必須です")
	case l.mediaType != JudicialDocumentMediaTypePDF:
		return fmt.Errorf("mediaType は application/pdf でなければなりません")
	case !isCourtsOfficialHTTPSURL(l.url):
		return fmt.Errorf(
			"url は認証情報を含まない https://www.courts.go.jp/ の URL でなければなりません",
		)
	default:
		return nil
	}
}

// MarshalJSON は、SOT-MODEL-020 の項目名で公式文書リンクを表す。
func (l JudicialDocumentLink) MarshalJSON() ([]byte, error) {
	if err := l.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Kind      JudicialDocumentKind `json:"kind"`
		Label     string               `json:"label"`
		MediaType string               `json:"mediaType"`
		URL       string               `json:"url"`
	}{
		Kind:      l.kind,
		Label:     l.label,
		MediaType: l.mediaType,
		URL:       l.url,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*JudicialDocumentLink) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialDocumentLink は JSON から直接復元できません。" +
			"境界専用の入力型から NewJudicialDocumentLink を使用してください",
	)
}

func isCourtsOfficialHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil &&
		strings.EqualFold(parsed.Scheme, "https") &&
		strings.EqualFold(parsed.Hostname(), "www.courts.go.jp") &&
		parsed.Port() == "" &&
		parsed.User == nil &&
		parsed.Opaque == "" &&
		parsed.Path != ""
}
