package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
)

type sourcedResourceData interface {
	Validate() error
}

// SourcedResourceValues は、SourcedResource の作成に必要な値を保持する。
type SourcedResourceValues[T sourcedResourceData] struct {
	Ref        SourceResourceRef
	Provenance []Provenance
	Data       T
}

// SourcedResource は、型付き情報モデルと、その主資源および出典経路を結び付ける。
type SourcedResource[T sourcedResourceData] struct {
	ref        SourceResourceRef
	provenance []Provenance
	data       T
}

// NewSourcedResource は、入力を複製して検証済みの SourcedResource を返す。
func NewSourcedResource[T sourcedResourceData](
	values SourcedResourceValues[T],
) (SourcedResource[T], error) {
	resource := SourcedResource[T]{
		ref:        values.Ref,
		provenance: cloneProvenance(values.Provenance),
		data:       values.Data,
	}
	if err := resource.Validate(); err != nil {
		return SourcedResource[T]{}, err
	}
	return resource, nil
}

// Ref は、取得に使用したプロバイダーと主資源の参照を返す。
func (r SourcedResource[T]) Ref() SourceResourceRef {
	return r.ref
}

// Provenance は、取得、抽出、正規化または加工の経路の複製を返す。
func (r SourcedResource[T]) Provenance() []Provenance {
	return cloneProvenance(r.provenance)
}

// Data は、能力 SOT が定義する型付き情報モデルを返す。
func (r SourcedResource[T]) Data() T {
	return r.data
}

// Validate は、主資源、出典経路および型付き情報モデルの従属制約を確認する。
func (r SourcedResource[T]) Validate() error {
	_, err := r.validatedDataJSON()
	return err
}

func (r SourcedResource[T]) validatedDataJSON() ([]byte, error) {
	if reflect.TypeFor[T]().Kind() != reflect.Struct {
		return nil, fmt.Errorf("data は具体的な struct 値型でなければなりません")
	}
	if err := r.ref.Validate(); err != nil {
		return nil, fmt.Errorf("ref が有効ではありません: %w", err)
	}
	if len(r.provenance) == 0 {
		return nil, fmt.Errorf("provenance は一件以上でなければなりません")
	}

	refKey := r.ref.Key()
	for index, provenance := range r.provenance {
		if err := provenance.Validate(); err != nil {
			return nil, fmt.Errorf("provenance[%d] が有効ではありません: %w", index, err)
		}
		if provenance.ResourceKey().SourceID() != refKey.SourceID() {
			return nil, fmt.Errorf(
				"provenance[%d].resourceKey.sourceId と ref.key.sourceId が一致しません",
				index,
			)
		}
		if err := validateProvenanceInputSources(provenance, refKey.SourceID()); err != nil {
			return nil, fmt.Errorf(
				"provenance[%d] の inputKeys が有効ではありません: %w",
				index,
				err,
			)
		}
	}
	if r.provenance[len(r.provenance)-1].ResourceKey() != refKey {
		return nil, fmt.Errorf("最後の provenance.resourceKey と ref.key が一致しません")
	}
	if err := r.data.Validate(); err != nil {
		return nil, fmt.Errorf("data が有効ではありません: %w", err)
	}

	dataJSON, err := json.Marshal(r.data)
	if err != nil {
		return nil, fmt.Errorf("data を JSON に変換できません: %w", err)
	}
	if bytes.Equal(bytes.TrimSpace(dataJSON), []byte("null")) {
		return nil, fmt.Errorf("data を JSON の null として表すことはできません")
	}
	return dataJSON, nil
}

// MarshalJSON は、SOT-IF-015 の項目名で型付き情報源結果を表す。
func (r SourcedResource[T]) MarshalJSON() ([]byte, error) {
	dataJSON, err := r.validatedDataJSON()
	if err != nil {
		return nil, err
	}

	return json.Marshal(struct {
		Ref        SourceResourceRef `json:"ref"`
		Provenance []Provenance      `json:"provenance"`
		Data       json.RawMessage   `json:"data"`
	}{
		Ref:        r.ref,
		Provenance: cloneProvenance(r.provenance),
		Data:       dataJSON,
	})
}

// UnmarshalJSON は、境界専用の入力型を介さない直接復元を拒否する。
func (*SourcedResource[T]) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"SourcedResource は JSON から直接復元できません。境界専用の入力型から NewSourcedResource を使用してください",
	)
}

func cloneProvenance(values []Provenance) []Provenance {
	if values == nil {
		return nil
	}
	cloned := make([]Provenance, len(values))
	copy(cloned, values)
	return cloned
}

func validateProvenanceInputSources(
	provenance Provenance,
	refSourceID string,
) error {
	if provenance.Transformation() == ProvenanceTransformationDerived {
		return nil
	}
	inputKeys, exists := provenance.InputKeys()
	if !exists {
		return nil
	}
	for index, inputKey := range inputKeys {
		if inputKey.SourceID() != refSourceID {
			return fmt.Errorf(
				"derived 以外の inputKeys[%d].sourceId は ref.key.sourceId と一致しなければなりません",
				index,
			)
		}
	}
	return nil
}
