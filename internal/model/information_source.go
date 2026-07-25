package model

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// Authority は、情報提供主体の位置付けを表す。
type Authority string

const (
	// AuthorityOfficial は、政府機関または公的機関の公式情報を表す。
	AuthorityOfficial Authority = "official"
	// AuthoritySupplementary は、公式情報を補助する情報を表す。
	AuthoritySupplementary Authority = "supplementary"
)

// InformationSourceValues は、InformationSource の作成に必要な値を保持する。
type InformationSourceValues struct {
	ID         string
	Name       string
	Publisher  string
	Authority  Authority
	ServiceURL string
}

// InformationSource は、一つの情報提供サービスと提供主体を表す。
type InformationSource struct {
	id         string
	name       string
	publisher  string
	authority  Authority
	serviceURL string
}

// NewInformationSource は、検証済みで変更不能な InformationSource を返す。
func NewInformationSource(values InformationSourceValues) (InformationSource, error) {
	source := InformationSource{
		id:         values.ID,
		name:       values.Name,
		publisher:  values.Publisher,
		authority:  values.Authority,
		serviceURL: values.ServiceURL,
	}
	if err := source.Validate(); err != nil {
		return InformationSource{}, err
	}
	return source, nil
}

// ID は、プロジェクト内で不変の情報源識別子を返す。
func (s InformationSource) ID() string {
	return s.id
}

// Name は、情報提供サービスの名称を返す。
func (s InformationSource) Name() string {
	return s.name
}

// Publisher は、情報を公開する機関または組織を返す。
func (s InformationSource) Publisher() string {
	return s.publisher
}

// Authority は、情報提供主体の位置付けを返す。
func (s InformationSource) Authority() Authority {
	return s.authority
}

// ServiceURL は、情報提供サービスを確認できる公式 HTTPS URL を返す。
func (s InformationSource) ServiceURL() string {
	return s.serviceURL
}

// Validate は、InformationSource の必須項目と列挙値を確認する。
func (s InformationSource) Validate() error {
	switch {
	case s.id == "":
		return fmt.Errorf("情報源の id は必須です")
	case s.name == "":
		return fmt.Errorf("情報源の name は必須です")
	case s.publisher == "":
		return fmt.Errorf("情報源の publisher は必須です")
	case s.authority != AuthorityOfficial && s.authority != AuthoritySupplementary:
		return fmt.Errorf("情報源の authority は %q または %q でなければなりません", AuthorityOfficial, AuthoritySupplementary)
	case !isHTTPSURL(s.serviceURL):
		return fmt.Errorf("情報源の serviceUrl は認証情報を含まない HTTPS URL でなければなりません")
	default:
		return nil
	}
}

// MarshalJSON は、SOT-MODEL-010 の項目名で情報源を表す。
func (s InformationSource) MarshalJSON() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ID         string    `json:"id"`
		Name       string    `json:"name"`
		Publisher  string    `json:"publisher"`
		Authority  Authority `json:"authority"`
		ServiceURL string    `json:"serviceUrl"`
	}{
		ID:         s.id,
		Name:       s.name,
		Publisher:  s.publisher,
		Authority:  s.authority,
		ServiceURL: s.serviceURL,
	})
}

func isHTTPSURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil &&
		strings.EqualFold(parsed.Scheme, "https") &&
		parsed.Host != "" &&
		parsed.Hostname() != "" &&
		parsed.User == nil &&
		parsed.Opaque == ""
}
