package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestSourceResourceRef(t *testing.T) {
	t.Parallel()

	key := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "official-law-source",
		ResourceType: "law",
		ResourceID:   "001-AbC",
		VersionID:    "revision-1",
	})
	got, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "law-api-adapter-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-016: NewSourceResourceRef() のエラー = %v", err)
	}

	if got.ProviderID() != "law-api-adapter-v2" || got.Key() != key {
		t.Fatalf("SOT-MODEL-016: SourceResourceRef = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-016: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/016: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/016: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"providerId": "law-api-adapter-v2",
		"key": map[string]any{
			"sourceId":     "official-law-source",
			"resourceType": "law",
			"resourceId":   "001-AbC",
			"versionId":    "revision-1",
		},
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/016: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestSourceResourceRefRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	key := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "official-law-source",
		ResourceType: "law",
		ResourceID:   "001",
	})
	for name, values := range map[string]model.SourceResourceRefValues{
		"providerId の欠落": {
			Key: key,
		},
		"providerId の大文字": {
			ProviderID: "Law-API",
			Key:        key,
		},
		"providerId の先頭ハイフン": {
			ProviderID: "-law-api",
			Key:        key,
		},
		"providerId の末尾ハイフン": {
			ProviderID: "law-api-",
			Key:        key,
		},
		"providerId の underscore": {
			ProviderID: "law_api",
			Key:        key,
		},
		"key の欠落": {
			ProviderID: "law-api",
		},
	} {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewSourceResourceRef(values); err == nil {
				t.Fatalf("SOT-MODEL-016: NewSourceResourceRef(%#v) が成功した", values)
			}
		})
	}
}

func TestZeroSourceResourceRefCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.SourceResourceRef{}); err == nil {
		t.Fatal("SOT-MODEL-009/016: SourceResourceRef のゼロ値を JSON に変換できた")
	}
}

func TestSourceResourceRefRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.SourceResourceRef
	if err := json.Unmarshal([]byte(
		`{"providerId":"law-api","key":{"sourceId":"official-law-source","resourceType":"law","resourceId":"001"}}`,
	), &got); err == nil {
		t.Fatal("SOT-ENG-002: 境界型を介さず SourceResourceRef を JSON から直接復元できた")
	}
}
