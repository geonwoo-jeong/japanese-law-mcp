package model_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type sourcedResourceTestData struct {
	value string
}

func (d sourcedResourceTestData) Validate() error {
	if d.value == "" {
		return fmt.Errorf("value は必須です")
	}
	return nil
}

func (d sourcedResourceTestData) MarshalJSON() ([]byte, error) {
	if err := d.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Value string `json:"value"`
	}{
		Value: d.value,
	})
}

type zeroSourcedResourceTestData struct{}

func (zeroSourcedResourceTestData) Validate() error {
	return nil
}

type nullSourcedResourceTestData struct{}

func (nullSourcedResourceTestData) Validate() error {
	return nil
}

func (nullSourcedResourceTestData) MarshalJSON() ([]byte, error) {
	return []byte("null"), nil
}

type failingMarshalSourcedResourceTestData struct{}

func (failingMarshalSourcedResourceTestData) Validate() error {
	return nil
}

func (failingMarshalSourcedResourceTestData) MarshalJSON() ([]byte, error) {
	return nil, errors.New("試験用の JSON エラー")
}

type invalidJSONSourcedResourceTestData struct{}

func (invalidJSONSourcedResourceTestData) Validate() error {
	return nil
}

func (invalidJSONSourcedResourceTestData) MarshalJSON() ([]byte, error) {
	return []byte("{"), nil
}

type mapSourcedResourceTestData map[string]string

func (mapSourcedResourceTestData) Validate() error {
	return nil
}

type sliceSourcedResourceTestData []string

func (sliceSourcedResourceTestData) Validate() error {
	return nil
}

func TestSourcedResource(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	firstKey := provenance[0].ResourceKey()
	finalKey := provenance[1].ResourceKey()
	data := sourcedResourceTestData{value: "法令概要"}
	got, err := model.NewSourcedResource(model.SourcedResourceValues[sourcedResourceTestData]{
		Ref:        ref,
		Provenance: provenance,
		Data:       data,
	})
	if err != nil {
		t.Fatalf("SOT-IF-015: NewSourcedResource() のエラー = %v", err)
	}

	provenance[0] = provenance[1]
	if got.Ref() != ref {
		t.Fatalf("SOT-IF-015: Ref() = %#v", got.Ref())
	}
	if got.Data() != data {
		t.Fatalf("SOT-IF-015: Data() = %#v", got.Data())
	}
	gotProvenance := got.Provenance()
	if len(gotProvenance) != 2 ||
		gotProvenance[0].ResourceKey() != firstKey ||
		gotProvenance[1].ResourceKey() != finalKey {
		t.Fatalf("SOT-IF-015: Provenance() = %#v", gotProvenance)
	}
	gotProvenance[0] = gotProvenance[1]
	preserved := got.Provenance()
	if preserved[0].ResourceKey() != firstKey {
		t.Fatalf("SOT-IF-015: provenance が外部から変更された: %#v", preserved)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-IF-015: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: JSON を再解析できない: %v", err)
	}
	if len(object) != 3 {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: JSON の項目 = %#v", object)
	}
	if !reflect.DeepEqual(object["data"], map[string]any{"value": "法令概要"}) {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: data = %#v", object["data"])
	}
	if values, ok := object["provenance"].([]any); !ok || len(values) != 2 {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: provenance = %#v", object["provenance"])
	}
	if _, ok := object["ref"].(map[string]any); !ok {
		t.Fatalf("SOT-MODEL-009/SOT-IF-015: ref = %#v", object["ref"])
	}
}

func TestSourcedResourceAcceptsForeignDerivedInput(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	values := validProvenanceValues(t)
	values.Transformation = model.ProvenanceTransformationDerived
	values.MethodID = "SOT-PROD-005"
	values.InputKeys = []model.SourceResourceKey{
		newSourceResourceKey(t, model.SourceResourceKeyValues{
			SourceID:     "other-official-source",
			ResourceType: "reference",
			ResourceID:   "Other-001",
		}),
	}
	derived, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}

	got, err := model.NewSourcedResource(model.SourcedResourceValues[sourcedResourceTestData]{
		Ref:        ref,
		Provenance: []model.Provenance{derived},
		Data:       sourcedResourceTestData{value: "加工結果"},
	})
	if err != nil {
		t.Fatalf("SOT-IF-015: derived の別情報源 inputKeys を拒否した: %v", err)
	}
	if got.Provenance()[0].ResourceKey() != provenance[1].ResourceKey() {
		t.Fatalf("SOT-IF-015: derived の resourceKey = %#v", got.Provenance()[0].ResourceKey())
	}
}

func TestSourcedResourceAcceptsSameSourceNonDerivedInput(t *testing.T) {
	t.Parallel()

	ref, _ := validSourcedResourceMetadata(t)
	values := validProvenanceValues(t)
	values.InputKeys = []model.SourceResourceKey{ref.Key()}
	normalized, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[sourcedResourceTestData]{
			Ref:        ref,
			Provenance: []model.Provenance{normalized},
			Data:       sourcedResourceTestData{value: "正規化結果"},
		},
	); err != nil {
		t.Fatalf("SOT-IF-015: 同じ情報源の inputKeys を拒否した: %v", err)
	}
}

func TestSourcedResourceAcceptsValidZeroData(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[zeroSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       zeroSourcedResourceTestData{},
		},
	); err != nil {
		t.Fatalf("SOT-IF-015: Validate() が成功する data のゼロ値を拒否した: %v", err)
	}
}

func TestSourcedResourceRejectsInvalidMetadata(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	otherSourceProvenance := provenanceForSource(t, "other-official-source")
	foreignInput := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "other-official-source",
		ResourceType: "reference",
		ResourceID:   "Other-001",
	})
	normalizedValues := validProvenanceValues(t)
	normalizedValues.InputKeys = []model.SourceResourceKey{foreignInput}
	normalizedWithForeignInput, err := model.NewProvenance(normalizedValues)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}

	tests := map[string]model.SourcedResourceValues[sourcedResourceTestData]{
		"ref の欠落": {
			Provenance: provenance,
			Data:       sourcedResourceTestData{value: "法令概要"},
		},
		"provenance が nil": {
			Ref:  ref,
			Data: sourcedResourceTestData{value: "法令概要"},
		},
		"provenance が空": {
			Ref:        ref,
			Provenance: []model.Provenance{},
			Data:       sourcedResourceTestData{value: "法令概要"},
		},
		"provenance が無効": {
			Ref:        ref,
			Provenance: []model.Provenance{{}},
			Data:       sourcedResourceTestData{value: "法令概要"},
		},
		"provenance の情報源が不一致": {
			Ref:        ref,
			Provenance: []model.Provenance{otherSourceProvenance},
			Data:       sourcedResourceTestData{value: "法令概要"},
		},
		"非 derived の inputKeys に別情報源": {
			Ref:        ref,
			Provenance: []model.Provenance{normalizedWithForeignInput},
			Data:       sourcedResourceTestData{value: "法令概要"},
		},
		"data が無効": {
			Ref:        ref,
			Provenance: provenance,
			Data:       sourcedResourceTestData{},
		},
	}
	for name, values := range tests {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewSourcedResource(values); err == nil {
				t.Fatal("SOT-IF-015: 不正な値を受理した")
			}
		})
	}
}

func TestSourcedResourceRejectsFinalResourceKeyMismatch(t *testing.T) {
	t.Parallel()

	ref, _ := validSourcedResourceMetadata(t)
	versionID, _ := ref.Key().VersionID()
	tests := map[string]model.SourceResourceKeyValues{
		"resourceType": {
			SourceID:     ref.Key().SourceID(),
			ResourceType: "law-document",
			ResourceID:   ref.Key().ResourceID(),
			VersionID:    versionID,
		},
		"resourceId": {
			SourceID:     ref.Key().SourceID(),
			ResourceType: ref.Key().ResourceType(),
			ResourceID:   "Other-Law",
			VersionID:    versionID,
		},
		"versionId": {
			SourceID:     ref.Key().SourceID(),
			ResourceType: ref.Key().ResourceType(),
			ResourceID:   ref.Key().ResourceID(),
			VersionID:    "other-revision",
		},
	}
	for name, keyValues := range tests {
		name := name
		keyValues := keyValues
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := validProvenanceValues(t)
			values.ResourceKey = newSourceResourceKey(t, keyValues)
			final, err := model.NewProvenance(values)
			if err != nil {
				t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
			}
			if _, err := model.NewSourcedResource(
				model.SourcedResourceValues[sourcedResourceTestData]{
					Ref:        ref,
					Provenance: []model.Provenance{final},
					Data:       sourcedResourceTestData{value: "法令概要"},
				},
			); err == nil {
				t.Fatalf("SOT-IF-015: 最後の resourceKey.%s の不一致を受理した", name)
			}
		})
	}
}

func TestSourcedResourceRejectsPointerData(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	data := &sourcedResourceTestData{value: "法令概要"}
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[*sourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       data,
		},
	); err == nil {
		t.Fatal("SOT-IF-015: pointer 型の data を受理した")
	}
}

func TestSourcedResourceRejectsInterfaceData(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	var data interface {
		Validate() error
	} = sourcedResourceTestData{value: "法令概要"}
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[interface{ Validate() error }]{
			Ref:        ref,
			Provenance: provenance,
			Data:       data,
		},
	); err == nil {
		t.Fatal("SOT-IF-015: interface 型の data を受理した")
	}
}

func TestSourcedResourceRejectsReferenceData(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[mapSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       mapSourcedResourceTestData{"value": "法令概要"},
		},
	); err == nil {
		t.Fatal("SOT-IF-015: map 型の data を受理した")
	}
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[sliceSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       sliceSourcedResourceTestData{"法令概要"},
		},
	); err == nil {
		t.Fatal("SOT-IF-015: slice 型の data を受理した")
	}
}

func TestSourcedResourceRejectsNullOrInvalidDataJSONAtConstruction(t *testing.T) {
	t.Parallel()

	ref, provenance := validSourcedResourceMetadata(t)
	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[nullSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       nullSourcedResourceTestData{},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-009/SOT-IF-015: JSON の null になる data を受理した")
	}

	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[failingMarshalSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       failingMarshalSourcedResourceTestData{},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-009/SOT-IF-015: JSON 変換に失敗する data を受理した")
	}

	if _, err := model.NewSourcedResource(
		model.SourcedResourceValues[invalidJSONSourcedResourceTestData]{
			Ref:        ref,
			Provenance: provenance,
			Data:       invalidJSONSourcedResourceTestData{},
		},
	); err == nil {
		t.Fatal("SOT-MODEL-009/SOT-IF-015: 不正な JSON を返す data を受理した")
	}
}

func TestSourcedResourceZeroValueCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.SourcedResource[sourcedResourceTestData]{}); err == nil {
		t.Fatal("SOT-MODEL-009/SOT-IF-015: SourcedResource のゼロ値を JSON に変換した")
	}
}

func TestSourcedResourceRejectsDirectJSONDecode(t *testing.T) {
	t.Parallel()

	var got model.SourcedResource[sourcedResourceTestData]
	err := json.Unmarshal([]byte(`{"ref":{},"provenance":[],"data":{}}`), &got)
	if err == nil || !strings.Contains(err.Error(), "NewSourcedResource") {
		t.Fatalf("SOT-ENG-002: SourcedResource の直接 JSON 復元エラー = %v", err)
	}
}

func validSourcedResourceMetadata(
	t *testing.T,
) (model.SourceResourceRef, []model.Provenance) {
	t.Helper()

	finalValues := validProvenanceValues(t)
	final, err := model.NewProvenance(finalValues)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        finalValues.ResourceKey,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-016: NewSourceResourceRef() のエラー = %v", err)
	}

	firstValues := finalValues
	firstValues.ResourceKey = newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     finalValues.ResourceKey.SourceID(),
		ResourceType: "search-response",
		ResourceID:   "Search-001",
	})
	firstValues.URL = "https://laws.e-gov.go.jp/api/2/laws"
	first, err := model.NewProvenance(firstValues)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}
	return ref, []model.Provenance{first, final}
}

func provenanceForSource(t *testing.T, sourceID string) model.Provenance {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "別の公式情報源",
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.go.jp/api/",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: NewInformationSource() のエラー = %v", err)
	}
	key := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   "001-AbC",
		VersionID:    "revision-1",
	})
	values := validProvenanceValues(t)
	values.Source = source
	values.ResourceKey = key
	values.URL = "https://example.go.jp/law/001-AbC"
	provenance, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}
	return provenance
}
