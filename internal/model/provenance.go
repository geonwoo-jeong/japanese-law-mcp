package model

import (
	"encoding/json"
	"fmt"
	"mime"
	"regexp"
	"strings"
	"time"
)

var contentDigestPattern = regexp.MustCompile(`^sha256:[0-9A-Fa-f]{64}$`)

// ProvenanceTransformation は、取得した情報へ適用した変換の種類を表す。
type ProvenanceTransformation string

const (
	// ProvenanceTransformationUnchanged は、取得した表現を変更していないことを表す。
	ProvenanceTransformationUnchanged ProvenanceTransformation = "unchanged"
	// ProvenanceTransformationExtracted は、取得した表現から一部を抽出したことを表す。
	ProvenanceTransformationExtracted ProvenanceTransformation = "extracted"
	// ProvenanceTransformationNormalized は、意味を変えずに共通モデルへ対応させたことを表す。
	ProvenanceTransformationNormalized ProvenanceTransformation = "normalized"
	// ProvenanceTransformationDerived は、計算、比較または推論を加えたことを表す。
	ProvenanceTransformationDerived ProvenanceTransformation = "derived"
)

// ProvenanceValues は、Provenance の作成に必要な値を保持する。
type ProvenanceValues struct {
	Source          InformationSource
	ResourceKey     SourceResourceKey
	URL             string
	RetrievedAt     time.Time
	SourceUpdatedAt string
	MediaType       string
	Location        string
	Transformation  ProvenanceTransformation
	MethodID        string
	InputKeys       []SourceResourceKey
	ContentDigest   string
}

// Provenance は、情報の取得元、取得時点および変換の種類を表す不変な出典である。
type Provenance struct {
	source          InformationSource
	resourceKey     SourceResourceKey
	url             string
	retrievedAt     time.Time
	sourceUpdatedAt string
	mediaType       string
	location        string
	transformation  ProvenanceTransformation
	methodID        string
	inputKeys       []SourceResourceKey
	contentDigest   string
}

// NewProvenance は、入力を複製して検証済みの Provenance を返す。
func NewProvenance(values ProvenanceValues) (Provenance, error) {
	provenance := Provenance{
		source:          values.Source,
		resourceKey:     values.ResourceKey,
		url:             values.URL,
		retrievedAt:     values.RetrievedAt.Round(0),
		sourceUpdatedAt: values.SourceUpdatedAt,
		mediaType:       values.MediaType,
		location:        values.Location,
		transformation:  values.Transformation,
		methodID:        values.MethodID,
		inputKeys:       cloneSourceResourceKeys(values.InputKeys),
		contentDigest:   values.ContentDigest,
	}
	if err := provenance.Validate(); err != nil {
		return Provenance{}, err
	}
	return provenance, nil
}

// Source は、情報を取得した情報源を返す。
func (p Provenance) Source() InformationSource {
	return p.source
}

// ResourceKey は、情報源上の資源と版を返す。
func (p Provenance) ResourceKey() SourceResourceKey {
	return p.resourceKey
}

// URL は、原文または公式掲載情報を確認できる HTTPS URL を返す。
func (p Provenance) URL() string {
	return p.url
}

// RetrievedAt は、このリクエストで情報を取得した時刻を返す。
func (p Provenance) RetrievedAt() time.Time {
	return p.retrievedAt
}

// SourceUpdatedAt は、情報源が明示した更新時点と有無を返す。
func (p Provenance) SourceUpdatedAt() (string, bool) {
	return p.sourceUpdatedAt, p.sourceUpdatedAt != ""
}

// MediaType は、取得した表現の MIME type を返す。
func (p Provenance) MediaType() string {
	return p.mediaType
}

// Location は、原文内で確認できる位置と有無を返す。
func (p Provenance) Location() (string, bool) {
	return p.location, p.location != ""
}

// Transformation は、取得した情報へ適用した変換の種類を返す。
func (p Provenance) Transformation() ProvenanceTransformation {
	return p.transformation
}

// MethodID は、変換方法を定義する SOT または契約の識別子と有無を返す。
func (p Provenance) MethodID() (string, bool) {
	return p.methodID, p.methodID != ""
}

// InputKeys は、加工に使用した入力資源の複製と有無を返す。
func (p Provenance) InputKeys() ([]SourceResourceKey, bool) {
	return cloneSourceResourceKeys(p.inputKeys), p.inputKeys != nil
}

// ContentDigest は、取得したバイト列の SHA-256 ダイジェストと有無を返す。
func (p Provenance) ContentDigest() (string, bool) {
	return p.contentDigest, p.contentDigest != ""
}

// Validate は、Provenance の必須項目、形式および変換ごとの制約を確認する。
func (p Provenance) Validate() error {
	if err := p.source.Validate(); err != nil {
		return fmt.Errorf("source が有効ではありません: %w", err)
	}
	if err := p.resourceKey.Validate(); err != nil {
		return fmt.Errorf("resourceKey が有効ではありません: %w", err)
	}
	if p.source.ID() != p.resourceKey.SourceID() {
		return fmt.Errorf("source.id と resourceKey.sourceId が一致しません")
	}
	if !isHTTPSURL(p.url) {
		return fmt.Errorf("url は認証情報を含まない HTTPS URL でなければなりません")
	}
	if err := validateRetrievedAt(p.retrievedAt); err != nil {
		return err
	}
	if p.sourceUpdatedAt != "" && !isDateOrDateTime(p.sourceUpdatedAt) {
		return fmt.Errorf("sourceUpdatedAt は実在する date またはタイムゾーンを含む RFC 3339 date-time でなければなりません")
	}
	if !isMIMEType(p.mediaType) {
		return fmt.Errorf("mediaType は MIME type でなければなりません")
	}
	if !isProvenanceTransformation(p.transformation) {
		return fmt.Errorf("transformation が定義されていません")
	}
	if p.transformation != ProvenanceTransformationUnchanged && p.methodID == "" {
		return fmt.Errorf("unchanged 以外の transformation では methodId が必須です")
	}
	if p.transformation == ProvenanceTransformationDerived && p.inputKeys == nil {
		return fmt.Errorf("derived の transformation では inputKeys が必須です")
	}
	for index, key := range p.inputKeys {
		if err := key.Validate(); err != nil {
			return fmt.Errorf("inputKeys[%d] が有効ではありません: %w", index, err)
		}
	}
	if p.contentDigest != "" && !contentDigestPattern.MatchString(p.contentDigest) {
		return fmt.Errorf("contentDigest は sha256: に続く 64 桁の hexadecimal でなければなりません")
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-012 の項目名で出典を表す。
func (p Provenance) MarshalJSON() ([]byte, error) {
	if err := p.Validate(); err != nil {
		return nil, err
	}

	var inputKeys *[]SourceResourceKey
	if p.inputKeys != nil {
		cloned := cloneSourceResourceKeys(p.inputKeys)
		inputKeys = &cloned
	}
	return json.Marshal(struct {
		Source          InformationSource        `json:"source"`
		ResourceKey     SourceResourceKey        `json:"resourceKey"`
		URL             string                   `json:"url"`
		RetrievedAt     time.Time                `json:"retrievedAt"`
		SourceUpdatedAt string                   `json:"sourceUpdatedAt,omitempty"`
		MediaType       string                   `json:"mediaType"`
		Location        string                   `json:"location,omitempty"`
		Transformation  ProvenanceTransformation `json:"transformation"`
		MethodID        string                   `json:"methodId,omitempty"`
		InputKeys       *[]SourceResourceKey     `json:"inputKeys,omitempty"`
		ContentDigest   string                   `json:"contentDigest,omitempty"`
	}{
		Source:          p.source,
		ResourceKey:     p.resourceKey,
		URL:             p.url,
		RetrievedAt:     p.retrievedAt,
		SourceUpdatedAt: p.sourceUpdatedAt,
		MediaType:       p.mediaType,
		Location:        p.location,
		Transformation:  p.transformation,
		MethodID:        p.methodID,
		InputKeys:       inputKeys,
		ContentDigest:   p.contentDigest,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*Provenance) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Provenance は JSON から直接復元できません。境界専用の入力型から NewProvenance を使用してください",
	)
}

func cloneSourceResourceKeys(values []SourceResourceKey) []SourceResourceKey {
	if values == nil {
		return nil
	}
	cloned := make([]SourceResourceKey, len(values))
	copy(cloned, values)
	return cloned
}

func validateRetrievedAt(value time.Time) error {
	if value.IsZero() {
		return fmt.Errorf("retrievedAt は必須です")
	}
	if _, err := value.MarshalJSON(); err != nil {
		return fmt.Errorf("retrievedAt はタイムゾーンを含む RFC 3339 date-time でなければなりません: %w", err)
	}
	return nil
}

func isDateOrDateTime(value string) bool {
	if _, err := NewDate(value); err == nil {
		return true
	}
	_, err := time.Parse(time.RFC3339, value)
	return err == nil
}

func isMIMEType(value string) bool {
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	mediaTypeName, mediaSubtype, found := strings.Cut(mediaType, "/")
	return found &&
		mediaTypeName != "" &&
		mediaSubtype != "" &&
		!strings.Contains(mediaType, "*")
}

func isProvenanceTransformation(value ProvenanceTransformation) bool {
	return value == ProvenanceTransformationUnchanged ||
		value == ProvenanceTransformationExtracted ||
		value == ProvenanceTransformationNormalized ||
		value == ProvenanceTransformationDerived
}
