package model_test

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProvenance(t *testing.T) {
	t.Parallel()

	values := validProvenanceValues(t)
	values.SourceUpdatedAt = "2026-07-25T10:00:00Z"
	values.Location = "LawBody/Article[1]"
	values.Transformation = model.ProvenanceTransformationDerived
	values.MethodID = "SOT-PROD-005"
	inputKey := newSourceResourceKey(t, model.SourceResourceKeyValues{
		SourceID:     "other-official-source",
		ResourceType: "reference",
		ResourceID:   "Ref-001",
	})
	values.InputKeys = []model.SourceResourceKey{inputKey}
	values.ContentDigest = "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	got, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}

	if got.Source() != values.Source ||
		got.ResourceKey() != values.ResourceKey ||
		got.URL() != values.URL ||
		!got.RetrievedAt().Equal(values.RetrievedAt) ||
		got.MediaType() != values.MediaType ||
		got.Transformation() != model.ProvenanceTransformationDerived {
		t.Fatalf("SOT-MODEL-012: Provenance = %#v", got)
	}
	if value, ok := got.SourceUpdatedAt(); !ok || value != values.SourceUpdatedAt {
		t.Fatalf("SOT-MODEL-012: SourceUpdatedAt() = %q, %t", value, ok)
	}
	if value, ok := got.Location(); !ok || value != values.Location {
		t.Fatalf("SOT-MODEL-012: Location() = %q, %t", value, ok)
	}
	if value, ok := got.MethodID(); !ok || value != values.MethodID {
		t.Fatalf("SOT-MODEL-012: MethodID() = %q, %t", value, ok)
	}
	if value, ok := got.ContentDigest(); !ok || value != values.ContentDigest {
		t.Fatalf("SOT-MODEL-012: ContentDigest() = %q, %t", value, ok)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-012: Validate() のエラー = %v", err)
	}

	values.InputKeys[0] = values.ResourceKey
	inputKeys, ok := got.InputKeys()
	if !ok || len(inputKeys) != 1 || inputKeys[0] != inputKey {
		t.Fatalf("SOT-MODEL-012: InputKeys() = %#v, %t", inputKeys, ok)
	}
	inputKeys[0] = values.ResourceKey
	if preserved, _ := got.InputKeys(); preserved[0] != inputKey {
		t.Fatalf("SOT-MODEL-012: inputKeys が外部から変更された: %#v", preserved)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/012: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/012: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"source": map[string]any{
			"id":         "e-gov-law-api-v2",
			"name":       "e-Gov 法令 API",
			"publisher":  "デジタル庁",
			"authority":  "official",
			"serviceUrl": "https://laws.e-gov.go.jp/api/2/",
		},
		"resourceKey": map[string]any{
			"sourceId":     "e-gov-law-api-v2",
			"resourceType": "law",
			"resourceId":   "001-AbC",
			"versionId":    "revision-1",
		},
		"url":             "https://laws.e-gov.go.jp/law/001-AbC",
		"retrievedAt":     "2026-07-26T11:00:00.123+09:00",
		"sourceUpdatedAt": "2026-07-25T10:00:00Z",
		"mediaType":       "application/json; charset=utf-8",
		"location":        "LawBody/Article[1]",
		"transformation":  "derived",
		"methodId":        "SOT-PROD-005",
		"inputKeys": []any{
			map[string]any{
				"sourceId":     "other-official-source",
				"resourceType": "reference",
				"resourceId":   "Ref-001",
			},
		},
		"contentDigest": "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/012: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestProvenanceOmitsAbsentOptionalValues(t *testing.T) {
	t.Parallel()

	values := validProvenanceValues(t)
	values.Transformation = model.ProvenanceTransformationUnchanged
	values.MethodID = ""

	got, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
	}
	if value, ok := got.SourceUpdatedAt(); ok || value != "" {
		t.Fatalf("SOT-MODEL-012: SourceUpdatedAt() = %q, %t", value, ok)
	}
	if value, ok := got.Location(); ok || value != "" {
		t.Fatalf("SOT-MODEL-012: Location() = %q, %t", value, ok)
	}
	if value, ok := got.MethodID(); ok || value != "" {
		t.Fatalf("SOT-MODEL-012: MethodID() = %q, %t", value, ok)
	}
	if values, ok := got.InputKeys(); ok || values != nil {
		t.Fatalf("SOT-MODEL-012: InputKeys() = %#v, %t", values, ok)
	}
	if value, ok := got.ContentDigest(); ok || value != "" {
		t.Fatalf("SOT-MODEL-012: ContentDigest() = %q, %t", value, ok)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/012: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/012: JSON を再解析できない: %v", err)
	}
	for _, key := range []string{
		"sourceUpdatedAt",
		"location",
		"methodId",
		"inputKeys",
		"contentDigest",
	} {
		if _, exists := object[key]; exists {
			t.Fatalf("SOT-MODEL-009/012: 省略値 %s が JSON に含まれた: %#v", key, object)
		}
	}
}

func TestProvenanceAcceptsDefinedTransformations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		transformation model.ProvenanceTransformation
		methodID       string
		inputKeys      []model.SourceResourceKey
	}{
		{
			name:           "unchanged",
			transformation: model.ProvenanceTransformationUnchanged,
		},
		{
			name:           "extracted",
			transformation: model.ProvenanceTransformationExtracted,
			methodID:       "SOT-IF-011",
		},
		{
			name:           "normalized",
			transformation: model.ProvenanceTransformationNormalized,
			methodID:       "SOT-IF-009",
		},
		{
			name:           "derived の空 inputKeys",
			transformation: model.ProvenanceTransformationDerived,
			methodID:       "SOT-PROD-005",
			inputKeys:      []model.SourceResourceKey{},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			values := validProvenanceValues(t)
			values.Transformation = test.transformation
			values.MethodID = test.methodID
			values.InputKeys = test.inputKeys
			got, err := model.NewProvenance(values)
			if err != nil {
				t.Fatalf("SOT-MODEL-012: NewProvenance() のエラー = %v", err)
			}

			if test.transformation != model.ProvenanceTransformationDerived {
				return
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatalf("SOT-MODEL-009/012: json.Marshal() のエラー = %v", err)
			}
			var object map[string]any
			if err := json.Unmarshal(encoded, &object); err != nil {
				t.Fatalf("SOT-MODEL-009/012: JSON を再解析できない: %v", err)
			}
			inputKeys, exists := object["inputKeys"]
			if !exists || !reflect.DeepEqual(inputKeys, []any{}) {
				t.Fatalf("SOT-MODEL-009/012: inputKeys = %#v, %t", inputKeys, exists)
			}
		})
	}
}

func TestProvenanceAcceptsSourceUpdatedDateOrDateTime(t *testing.T) {
	t.Parallel()

	for _, sourceUpdatedAt := range []string{
		"2026-07-25",
		"2026-07-25T10:00:00Z",
		"2026-07-25T19:00:00.123+09:00",
	} {
		sourceUpdatedAt := sourceUpdatedAt
		t.Run(sourceUpdatedAt, func(t *testing.T) {
			t.Parallel()

			values := validProvenanceValues(t)
			values.SourceUpdatedAt = sourceUpdatedAt
			got, err := model.NewProvenance(values)
			if err != nil {
				t.Fatalf("SOT-MODEL-009/012: %q を拒否した: %v", sourceUpdatedAt, err)
			}
			if value, ok := got.SourceUpdatedAt(); !ok || value != sourceUpdatedAt {
				t.Fatalf("SOT-MODEL-012: SourceUpdatedAt() = %q, %t", value, ok)
			}
		})
	}
}

func TestProvenanceRemovesMonotonicClockReading(t *testing.T) {
	t.Parallel()

	retrievedAt := time.Now()
	withoutMonotonic := retrievedAt.Round(0)
	// time.Time.Equal は単調時計成分を比較しないため、内部状態を確認するこの試験では使用しない。
	if reflect.DeepEqual(retrievedAt, withoutMonotonic) {
		t.Skip("この環境の time.Now() は単調時計成分を含みません")
	}

	values := validProvenanceValues(t)
	values.RetrievedAt = retrievedAt
	got, err := model.NewProvenance(values)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/012: NewProvenance() のエラー = %v", err)
	}
	if !reflect.DeepEqual(got.RetrievedAt(), withoutMonotonic) {
		t.Fatalf(
			"SOT-MODEL-009/012: RetrievedAt() = %#v、期待値 = %#v",
			got.RetrievedAt(),
			withoutMonotonic,
		)
	}
}

func TestProvenanceRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	tests := map[string]func(*model.ProvenanceValues){
		"source の欠落": func(values *model.ProvenanceValues) {
			values.Source = model.InformationSource{}
		},
		"resourceKey の欠落": func(values *model.ProvenanceValues) {
			values.ResourceKey = model.SourceResourceKey{}
		},
		"source と resourceKey の不一致": func(values *model.ProvenanceValues) {
			values.ResourceKey = newSourceResourceKey(t, model.SourceResourceKeyValues{
				SourceID:     "other-official-source",
				ResourceType: "law",
				ResourceID:   "001",
			})
		},
		"HTTP URL": func(values *model.ProvenanceValues) {
			values.URL = "http://laws.e-gov.go.jp/law/001"
		},
		"retrievedAt の欠落": func(values *model.ProvenanceValues) {
			values.RetrievedAt = time.Time{}
		},
		"retrievedAt の JSON 範囲外": func(values *model.ProvenanceValues) {
			values.RetrievedAt = time.Date(
				10000,
				time.January,
				1,
				0,
				0,
				0,
				0,
				time.UTC,
			)
		},
		"sourceUpdatedAt の日付不正": func(values *model.ProvenanceValues) {
			values.SourceUpdatedAt = "2023-02-29"
		},
		"sourceUpdatedAt のタイムゾーン欠落": func(values *model.ProvenanceValues) {
			values.SourceUpdatedAt = "2026-07-25T10:00:00"
		},
		"mediaType の欠落": func(values *model.ProvenanceValues) {
			values.MediaType = ""
		},
		"mediaType の形式不正": func(values *model.ProvenanceValues) {
			values.MediaType = "application"
		},
		"mediaType の subtype wildcard": func(values *model.ProvenanceValues) {
			values.MediaType = "application/*"
		},
		"mediaType の全 wildcard": func(values *model.ProvenanceValues) {
			values.MediaType = "*/*"
		},
		"未知の transformation": func(values *model.ProvenanceValues) {
			values.Transformation = model.ProvenanceTransformation("converted")
		},
		"extracted の methodId 欠落": func(values *model.ProvenanceValues) {
			values.Transformation = model.ProvenanceTransformationExtracted
			values.MethodID = ""
		},
		"normalized の methodId 欠落": func(values *model.ProvenanceValues) {
			values.Transformation = model.ProvenanceTransformationNormalized
			values.MethodID = ""
		},
		"derived の methodId 欠落": func(values *model.ProvenanceValues) {
			values.Transformation = model.ProvenanceTransformationDerived
			values.MethodID = ""
			values.InputKeys = []model.SourceResourceKey{}
		},
		"derived の inputKeys 欠落": func(values *model.ProvenanceValues) {
			values.Transformation = model.ProvenanceTransformationDerived
			values.MethodID = "SOT-PROD-005"
			values.InputKeys = nil
		},
		"inputKeys の無効な要素": func(values *model.ProvenanceValues) {
			values.InputKeys = []model.SourceResourceKey{{}}
		},
		"contentDigest の prefix 不正": func(values *model.ProvenanceValues) {
			values.ContentDigest = "sha512:0123456789abcdef"
		},
		"contentDigest の長さ不正": func(values *model.ProvenanceValues) {
			values.ContentDigest = "sha256:0123456789abcdef"
		},
		"contentDigest の hex 不正": func(values *model.ProvenanceValues) {
			values.ContentDigest = "sha256:zzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzzz"
		},
	}
	for name, change := range tests {
		name, change := name, change
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			values := validProvenanceValues(t)
			change(&values)
			if _, err := model.NewProvenance(values); err == nil {
				t.Fatalf("SOT-MODEL-009/012: 無効な ProvenanceValues %#v を受理した", values)
			}
		})
	}
}

func TestZeroProvenanceCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.Provenance{}); err == nil {
		t.Fatal("SOT-MODEL-009/012: Provenance のゼロ値を JSON に変換できた")
	}
}

func TestProvenanceRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.Provenance
	err := json.Unmarshal([]byte(`{}`), &got)
	if err == nil || !strings.Contains(err.Error(), "NewProvenance") {
		t.Fatalf("SOT-ENG-002: Provenance の直接 JSON 復元エラー = %v", err)
	}
}

func validProvenanceValues(t *testing.T) model.ProvenanceValues {
	t.Helper()

	return model.ProvenanceValues{
		Source: newInformationSource(t),
		ResourceKey: newSourceResourceKey(t, model.SourceResourceKeyValues{
			SourceID:     "e-gov-law-api-v2",
			ResourceType: "law",
			ResourceID:   "001-AbC",
			VersionID:    "revision-1",
		}),
		URL: "https://laws.e-gov.go.jp/law/001-AbC",
		RetrievedAt: time.Date(
			2026,
			time.July,
			26,
			11,
			0,
			0,
			123_000_000,
			time.FixedZone("JST", 9*60*60),
		),
		MediaType:      "application/json; charset=utf-8",
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       "SOT-IF-009",
	}
}
