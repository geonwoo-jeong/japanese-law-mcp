package model_test

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewDate(t *testing.T) {
	t.Parallel()

	got, err := model.NewDate("2024-02-29")
	if err != nil {
		t.Fatalf("SOT-MODEL-009: NewDate() のエラー = %v", err)
	}
	if got.String() != "2024-02-29" {
		t.Fatalf("SOT-MODEL-009: Date.String() = %q", got.String())
	}
	if got.IsZero() {
		t.Fatal("SOT-MODEL-009: 有効な日付がゼロ値と判定された")
	}

	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatalf("SOT-MODEL-009: json.Marshal() のエラー = %v", err)
	}
	if string(encoded) != `"2024-02-29"` {
		t.Fatalf("SOT-MODEL-009: JSON = %s", encoded)
	}
}

func TestNewDateRejectsInvalidDate(t *testing.T) {
	t.Parallel()

	for _, value := range []string{
		"",
		"2023-02-29",
		"2024-2-29",
		"2024-02-29T00:00:00Z",
		"２０２４-０２-２９",
	} {
		value := value
		t.Run(value, func(t *testing.T) {
			t.Parallel()

			if _, err := model.NewDate(value); err == nil {
				t.Fatalf("SOT-MODEL-009: NewDate(%q) が成功した", value)
			}
		})
	}
}

func TestZeroDateCannotBeSerialized(t *testing.T) {
	t.Parallel()

	if _, err := json.Marshal(model.Date{}); err == nil {
		t.Fatal("SOT-MODEL-009: Date のゼロ値を JSON に変換できた")
	}
}
