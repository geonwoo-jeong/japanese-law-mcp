package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestInformationSource(t *testing.T) {
	t.Parallel()

	got, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-010: NewInformationSource() のエラー = %v", err)
	}

	if got.ID() != "e-gov-law-api-v2" ||
		got.Name() != "e-Gov 法令 API" ||
		got.Publisher() != "デジタル庁" ||
		got.Authority() != model.AuthorityOfficial ||
		got.ServiceURL() != "https://laws.e-gov.go.jp/api/2/" {
		t.Fatalf("SOT-MODEL-010: InformationSource = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-010: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/010: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/010: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"id":         "e-gov-law-api-v2",
		"name":       "e-Gov 法令 API",
		"publisher":  "デジタル庁",
		"authority":  "official",
		"serviceUrl": "https://laws.e-gov.go.jp/api/2/",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/010: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestInformationSourceRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	}

	tests := map[string]model.InformationSourceValues{
		"id の欠落": {
			Name:       valid.Name,
			Publisher:  valid.Publisher,
			Authority:  valid.Authority,
			ServiceURL: valid.ServiceURL,
		},
		"name の欠落": {
			ID:         valid.ID,
			Publisher:  valid.Publisher,
			Authority:  valid.Authority,
			ServiceURL: valid.ServiceURL,
		},
		"publisher の欠落": {
			ID:         valid.ID,
			Name:       valid.Name,
			Authority:  valid.Authority,
			ServiceURL: valid.ServiceURL,
		},
		"未知の authority": {
			ID:         valid.ID,
			Name:       valid.Name,
			Publisher:  valid.Publisher,
			Authority:  model.Authority("trusted"),
			ServiceURL: valid.ServiceURL,
		},
		"HTTP URL": {
			ID:         valid.ID,
			Name:       valid.Name,
			Publisher:  valid.Publisher,
			Authority:  valid.Authority,
			ServiceURL: "http://laws.e-gov.go.jp/api/2/",
		},
		"相対 URL": {
			ID:         valid.ID,
			Name:       valid.Name,
			Publisher:  valid.Publisher,
			Authority:  valid.Authority,
			ServiceURL: "/api/2/",
		},
		"認証情報を含む URL": { //nolint:gosec // SOT-MODEL-010: 認証情報を含む URL の拒否を確認する固定テスト値である。
			ID:         valid.ID,
			Name:       valid.Name,
			Publisher:  valid.Publisher,
			Authority:  valid.Authority,
			ServiceURL: "https://user:password@laws.e-gov.go.jp/api/2/",
		},
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewInformationSource(values); err == nil {
				t.Fatalf("SOT-MODEL-010: NewInformationSource(%#v) が成功した", values)
			}
		})
	}
}

func TestZeroInformationSourceCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.InformationSource{}); err == nil {
		t.Fatal("SOT-MODEL-009/010: InformationSource のゼロ値を JSON に変換できた")
	}
}
