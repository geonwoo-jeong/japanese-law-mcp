package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestSourceResourceKey(t *testing.T) {
	t.Parallel()

	got, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "official-law-source",
		ResourceType: "law",
		ResourceID:   "001-AbC/%2F:章",
		VersionID:    "Rev-001/Ａ",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-011: NewSourceResourceKey() のエラー = %v", err)
	}

	if got.SourceID() != "official-law-source" ||
		got.ResourceType() != "law" ||
		got.ResourceID() != "001-AbC/%2F:章" {
		t.Fatalf("SOT-MODEL-011: SourceResourceKey = %#v", got)
	}
	versionID, ok := got.VersionID()
	if !ok || versionID != "Rev-001/Ａ" {
		t.Fatalf("SOT-MODEL-011: VersionID() = %q, %t", versionID, ok)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-011: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/011: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/011: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"sourceId":     "official-law-source",
		"resourceType": "law",
		"resourceId":   "001-AbC/%2F:章",
		"versionId":    "Rev-001/Ａ",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/011: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestSourceResourceKeyOmitsAbsentVersionID(t *testing.T) {
	t.Parallel()

	got := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "official-law-source",
		ResourceType: "law",
		ResourceID:   "001",
	})
	if versionID, ok := got.VersionID(); ok || versionID != "" {
		t.Fatalf("SOT-MODEL-011: VersionID() = %q, %t", versionID, ok)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/011: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/011: JSON を再解析できない: %v", err)
	}
	if _, exists := object["versionId"]; exists {
		t.Fatalf("SOT-MODEL-009/011: 省略値 versionId が JSON に含まれた: %#v", object)
	}
	want := map[string]any{
		"sourceId":     "official-law-source",
		"resourceType": "law",
		"resourceId":   "001",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/011: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestSourceResourceKeyRejectsMissingRequiredValues(t *testing.T) {
	t.Parallel()

	valid := model.SourceResourceKeyValues{
		SourceID:     "official-law-source",
		ResourceType: "law",
		ResourceID:   "001",
		VersionID:    "revision-1",
	}
	tests := map[string]model.SourceResourceKeyValues{
		"sourceId の欠落": {
			ResourceType: valid.ResourceType,
			ResourceID:   valid.ResourceID,
			VersionID:    valid.VersionID,
		},
		"resourceType の欠落": {
			SourceID:   valid.SourceID,
			ResourceID: valid.ResourceID,
			VersionID:  valid.VersionID,
		},
		"resourceId の欠落": {
			SourceID:     valid.SourceID,
			ResourceType: valid.ResourceType,
			VersionID:    valid.VersionID,
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewSourceResourceKey(values); err == nil {
				t.Fatalf("SOT-MODEL-011: NewSourceResourceKey(%#v) が成功した", values)
			}
		})
	}
}

func TestZeroSourceResourceKeyCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.SourceResourceKey{}); err == nil {
		t.Fatal("SOT-MODEL-009/011: SourceResourceKey のゼロ値を JSON に変換できた")
	}
}

func TestSourceResourceKeyRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.SourceResourceKey
	if err := json.Unmarshal([]byte(
		`{"sourceId":"official-law-source","resourceType":"law","resourceId":"001"}`,
	), &got); err == nil {
		t.Fatal("SOT-ENG-002: 境界型を介さず SourceResourceKey を JSON から直接復元できた")
	}
}

func newSourceResourceKey(
	t *testing.T,
	values model.SourceResourceKeyValues,
) model.SourceResourceKey {
	t.Helper()

	got, err := model.NewSourceResourceKey(values)
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	return got
}
