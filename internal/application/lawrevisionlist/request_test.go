package lawrevisionlist

import (
	"strings"
	"testing"
)

func TestRequestNormalizesAndValidatesLawIDOrNumber(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{LawIDOrNumber: " 503AC0000000036 "})
	if err != nil {
		t.Fatalf("有効な Request を拒否しました: %v", err)
	}
	if request.LawIDOrNumber() != "503AC0000000036" {
		t.Fatalf("lawIdOrNumber = %q", request.LawIDOrNumber())
	}

	tests := map[string]string{
		"空":          " ",
		"制御文字":       "503AC\n0000000036",
		"256 byte 超": strings.Repeat("x", 257),
		"不正 UTF-8":   string([]byte{0xff}),
	}
	for name, value := range tests {
		name, value := name, value
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewRequest(RequestValues{LawIDOrNumber: value}); err == nil {
				t.Fatal("不正な Request を拒否しませんでした")
			}
		})
	}
}
