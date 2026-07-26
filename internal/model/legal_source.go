package model

import (
	"encoding/json"
	"fmt"
)

// LegalSource は、法情報を提供するサービスと提供主体の位置付けを表す。
type LegalSource struct {
	id         string
	name       string
	authority  Authority
	serviceURL string
}

// NewLegalSource は、InformationSource を法令向けの公開モデルへ決定的に投影する。
func NewLegalSource(source InformationSource) (LegalSource, error) {
	if err := source.Validate(); err != nil {
		return LegalSource{}, fmt.Errorf("InformationSource が有効ではありません: %w", err)
	}

	projected := LegalSource{
		id:         source.ID(),
		name:       source.Name(),
		authority:  source.Authority(),
		serviceURL: source.ServiceURL(),
	}
	if err := projected.Validate(); err != nil {
		return LegalSource{}, err
	}
	return projected, nil
}

// ID は、プロジェクト内の情報源識別子を返す。
func (s LegalSource) ID() string {
	return s.id
}

// Name は、情報源サービスの名称を返す。
func (s LegalSource) Name() string {
	return s.name
}

// Authority は、提供主体の位置付けを返す。
func (s LegalSource) Authority() Authority {
	return s.authority
}

// ServiceURL は、情報源サービスの公式 HTTPS URL を返す。
func (s LegalSource) ServiceURL() string {
	return s.serviceURL
}

// Validate は、LegalSource の必須項目、列挙値および URL を確認する。
func (s LegalSource) Validate() error {
	switch {
	case s.id == "":
		return fmt.Errorf("法令情報源の id は必須です")
	case s.name == "":
		return fmt.Errorf("法令情報源の name は必須です")
	case s.authority != AuthorityOfficial && s.authority != AuthoritySupplementary:
		return fmt.Errorf(
			"法令情報源の authority は %q または %q でなければなりません",
			AuthorityOfficial,
			AuthoritySupplementary,
		)
	case !isHTTPSURL(s.serviceURL):
		return fmt.Errorf("法令情報源の serviceUrl は認証情報を含まない HTTPS URL でなければなりません")
	default:
		return nil
	}
}

// MarshalJSON は、SOT-MODEL-003 の項目名で法令情報源を表す。
func (s LegalSource) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ID         string    `json:"id"`
		Name       string    `json:"name"`
		Authority  Authority `json:"authority"`
		ServiceURL string    `json:"serviceUrl"`
	}{
		ID:         s.id,
		Name:       s.name,
		Authority:  s.authority,
		ServiceURL: s.serviceURL,
	})
}

// UnmarshalJSON は、InformationSource からの投影を介さない直接復元を拒否する。
func (*LegalSource) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalSource は JSON から直接復元できません。InformationSource から NewLegalSource を使用してください",
	)
}
