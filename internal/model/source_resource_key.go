package model

import (
	"encoding/json"
	"fmt"
)

// SourceResourceKeyValues は、SourceResourceKey の作成に必要な値を保持する。
type SourceResourceKeyValues struct {
	SourceID     string
	ResourceType string
	ResourceID   string
	VersionID    string
}

// SourceResourceKey は、一つの情報源上の資源と版を表す不変な共通キーである。
type SourceResourceKey struct {
	sourceID     string
	resourceType string
	resourceID   string
	versionID    string
}

// NewSourceResourceKey は、識別子を変更せずに検証済みの SourceResourceKey を返す。
func NewSourceResourceKey(values SourceResourceKeyValues) (SourceResourceKey, error) {
	key := SourceResourceKey{
		sourceID:     values.SourceID,
		resourceType: values.ResourceType,
		resourceID:   values.ResourceID,
		versionID:    values.VersionID,
	}
	if err := key.Validate(); err != nil {
		return SourceResourceKey{}, err
	}
	return key, nil
}

// SourceID は、資源が属する情報源の識別子を返す。
func (k SourceResourceKey) SourceID() string {
	return k.sourceID
}

// ResourceType は、能力別 SOT が定義する資源種別を返す。
func (k SourceResourceKey) ResourceType() string {
	return k.resourceType
}

// ResourceID は、情報源が使用する不透明な資源識別子を返す。
func (k SourceResourceKey) ResourceID() string {
	return k.resourceID
}

// VersionID は、情報源が明示する版の識別子と有無を返す。
func (k SourceResourceKey) VersionID() (string, bool) {
	return k.versionID, k.versionID != ""
}

// Validate は、SourceResourceKey の必須項目を確認する。
func (k SourceResourceKey) Validate() error {
	switch {
	case k.sourceID == "":
		return fmt.Errorf("sourceId は必須です")
	case k.resourceType == "":
		return fmt.Errorf("resourceType は必須です")
	case k.resourceID == "":
		return fmt.Errorf("resourceId は必須です")
	default:
		return nil
	}
}

// MarshalJSON は、SOT-MODEL-011 の項目名で共通キーを表す。
func (k SourceResourceKey) MarshalJSON() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		SourceID     string `json:"sourceId"`
		ResourceType string `json:"resourceType"`
		ResourceID   string `json:"resourceId"`
		VersionID    string `json:"versionId,omitempty"`
	}{
		SourceID:     k.sourceID,
		ResourceType: k.resourceType,
		ResourceID:   k.resourceID,
		VersionID:    k.versionID,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*SourceResourceKey) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SourceResourceKey は JSON から直接復元できません。境界専用の入力型から NewSourceResourceKey を使用してください",
	)
}
