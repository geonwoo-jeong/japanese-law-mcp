package model

import (
	"encoding/json"
	"fmt"
)

// SourceResourceRefValues は、SourceResourceRef の作成に必要な値を保持する。
type SourceResourceRefValues struct {
	ProviderID string
	Key        SourceResourceKey
}

// SourceResourceRef は、取得に使用したプロバイダーと資源キーを表す不変な参照である。
type SourceResourceRef struct {
	providerID string
	key        SourceResourceKey
}

// NewSourceResourceRef は、単体で確認できる構造を検証した SourceResourceRef を返す。
func NewSourceResourceRef(values SourceResourceRefValues) (SourceResourceRef, error) {
	ref := SourceResourceRef{
		providerID: values.ProviderID,
		key:        values.Key,
	}
	if err := ref.Validate(); err != nil {
		return SourceResourceRef{}, err
	}
	return ref, nil
}

// ProviderID は、資源の取得に使用したプロバイダー識別子を返す。
func (r SourceResourceRef) ProviderID() string {
	return r.providerID
}

// Key は、情報源上の資源と版を表す共通キーを返す。
func (r SourceResourceRef) Key() SourceResourceKey {
	return r.key
}

// Validate は、SourceResourceRef 単体で確認できる構造を検証する。
func (r SourceResourceRef) Validate() error {
	if !providerIDPattern.MatchString(r.providerID) {
		return fmt.Errorf("providerId は小文字の ASCII 英数字と内部のハイフンで構成しなければなりません")
	}
	if err := r.key.Validate(); err != nil {
		return fmt.Errorf("key が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、SOT-MODEL-016 の項目名で参照を表す。
func (r SourceResourceRef) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ProviderID string            `json:"providerId"`
		Key        SourceResourceKey `json:"key"`
	}{
		ProviderID: r.providerID,
		Key:        r.key,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*SourceResourceRef) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SourceResourceRef は JSON から直接復元できません。境界専用の入力型から NewSourceResourceRef を使用してください",
	)
}
