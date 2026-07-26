package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestLegalSourceProjectsInformationSource(t *testing.T) {
	t.Parallel()

	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: NewInformationSource() のエラー = %v", err)
	}

	got, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("SOT-MODEL-003: NewLegalSource() のエラー = %v", err)
	}
	second, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("SOT-MODEL-003: 2 回目の NewLegalSource() のエラー = %v", err)
	}

	if got != second {
		t.Fatalf("SOT-MODEL-003: 同じ InformationSource の投影結果が一致しない: %#v、%#v", got, second)
	}
	if got.ID() != informationSource.ID() ||
		got.Name() != informationSource.Name() ||
		got.Authority() != informationSource.Authority() ||
		got.ServiceURL() != informationSource.ServiceURL() {
		t.Fatalf("SOT-MODEL-003: LegalSource = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-003: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-003/009: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-003/009: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"id":         informationSource.ID(),
		"name":       informationSource.Name(),
		"authority":  string(informationSource.Authority()),
		"serviceUrl": informationSource.ServiceURL(),
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-003/009: JSON = %#v、期待値 = %#v", object, want)
	}
	if _, exists := object["publisher"]; exists {
		t.Fatalf("SOT-MODEL-003: publisher が LegalSource に混入した: %s", encoded)
	}
}

func TestLegalSourceRejectsInvalidInformationSource(t *testing.T) {
	t.Parallel()

	if _, err := model.NewLegalSource(model.InformationSource{}); err == nil {
		t.Fatal("SOT-MODEL-003: 無効な InformationSource の投影が成功した")
	}
}

func TestZeroLegalSourceCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.LegalSource{}); err == nil {
		t.Fatal("SOT-MODEL-003/009: LegalSource のゼロ値を JSON に変換できた")
	}
}

func TestLegalSourceRejectsDirectJSONDecoding(t *testing.T) {
	t.Parallel()

	var got model.LegalSource
	if err := json.Unmarshal([]byte(
		`{"id":"e-gov-law-api-v2","name":"e-Gov 法令 API","authority":"official","serviceUrl":"https://laws.e-gov.go.jp/api/2/"}`,
	), &got); err == nil {
		t.Fatal("SOT-MODEL-003: LegalSource を JSON から直接復元できた")
	}
}
