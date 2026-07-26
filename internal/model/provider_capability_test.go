package model_test

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestProviderCapability(t *testing.T) {
	t.Parallel()

	got, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           "law.search",
		MajorVersion: 1,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-013: NewProviderCapability() のエラー = %v", err)
	}

	if got.ID() != "law.search" ||
		got.MajorVersion() != 1 ||
		got.Level() != model.CapabilityLevelCore ||
		got.Stability() != model.CapabilityStabilityStable {
		t.Fatalf("SOT-MODEL-013: ProviderCapability = %#v", got)
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-MODEL-013: Validate() のエラー = %v", err)
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009/013: json.Marshal() のエラー = %v", err)
	}
	var object map[string]any
	if err := json.Unmarshal(encoded, &object); err != nil {
		t.Fatalf("SOT-MODEL-009/013: JSON を再解析できない: %v", err)
	}
	want := map[string]any{
		"id":           "law.search",
		"majorVersion": float64(1),
		"level":        "core",
		"stability":    "stable",
	}
	if !reflect.DeepEqual(object, want) {
		t.Fatalf("SOT-MODEL-009/013: JSON = %#v、期待値 = %#v", object, want)
	}
}

func TestProviderCapabilityAcceptsDefinedIDForms(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id    string
		level model.CapabilityLevel
	}{
		{id: "law.search", level: model.CapabilityLevelCore},
		{id: "law.content-search", level: model.CapabilityLevelExtended},
		{
			id:    "provider.e-gov-law-api-v2.search",
			level: model.CapabilityLevelProviderSpecific,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.id, func(t *testing.T) {
			t.Parallel()

			_, err := model.NewProviderCapability(model.ProviderCapabilityValues{
				ID:           test.id,
				MajorVersion: 1,
				Level:        test.level,
				Stability:    model.CapabilityStabilityExperimental,
			})
			if err != nil {
				t.Fatalf("SOT-MODEL-013: %q を拒否した: %v", test.id, err)
			}
		})
	}
}

func TestProviderCapabilityRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	valid := model.ProviderCapabilityValues{
		ID:           "law.search",
		MajorVersion: 1,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	}

	tests := map[string]model.ProviderCapabilityValues{
		"segment が一つ": {
			ID:           "law",
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"大文字": {
			ID:           "Law.search",
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"空の segment": {
			ID:           "law..search",
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"不正なハイフン": {
			ID:           "law.-search",
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"underscore": {
			ID:           "law.content_search",
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"majorVersion がゼロ": {
			ID:           valid.ID,
			MajorVersion: 0,
			Level:        valid.Level,
			Stability:    valid.Stability,
		},
		"未知の level": {
			ID:           valid.ID,
			MajorVersion: valid.MajorVersion,
			Level:        model.CapabilityLevel("optional"),
			Stability:    valid.Stability,
		},
		"未知の stability": {
			ID:           valid.ID,
			MajorVersion: valid.MajorVersion,
			Level:        valid.Level,
			Stability:    model.CapabilityStability("preview"),
		},
		"provider 固有 level で共通 namespace": {
			ID:           "law.search",
			MajorVersion: valid.MajorVersion,
			Level:        model.CapabilityLevelProviderSpecific,
			Stability:    valid.Stability,
		},
		"共通 level で provider namespace": {
			ID:           "provider.e-gov-law-api-v2.search",
			MajorVersion: valid.MajorVersion,
			Level:        model.CapabilityLevelCore,
			Stability:    valid.Stability,
		},
	}

	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewProviderCapability(values); err == nil {
				t.Fatalf("SOT-MODEL-013: NewProviderCapability(%#v) が成功した", values)
			}
		})
	}
}

func TestZeroProviderCapabilityCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.ProviderCapability{}); err == nil {
		t.Fatal("SOT-MODEL-009/013: ProviderCapability のゼロ値を JSON に変換できた")
	}
}
