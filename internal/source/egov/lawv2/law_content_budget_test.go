package lawv2

import (
	"testing"
	"time"
)

func TestLawContentBudgetsAreIndependentFromLawSearch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  int64
		want int64
	}{
		{
			name: "laws responseBytes",
			got:  int64(maximumResponseBytes),
			want: 8 * 1024 * 1024,
		},
		{
			name: "laws decompressedBytes",
			got:  int64(maximumDecompressedBytes),
			want: 16 * 1024 * 1024,
		},
		{
			name: "keyword responseBytes",
			got:  int64(lawContentResponseBytes),
			want: 16 * 1024 * 1024,
		},
		{
			name: "keyword decompressedBytes",
			got:  int64(lawContentDecompressedBytes),
			want: 32 * 1024 * 1024,
		},
		{
			name: "keyword parser inputBytes",
			got:  int64(lawContentParserInputBytes),
			want: 32 * 1024 * 1024,
		},
		{
			name: "keyword entriesOrObjects",
			got:  int64(lawContentJSONValues),
			want: 500000,
		},
		{
			name: "keyword depth",
			got:  int64(lawContentJSONDepth),
			want: 64,
		},
		{
			name: "keyword parseTimeout",
			got:  int64(lawContentParseTimeout),
			want: int64(5 * time.Second),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if test.got != test.want {
				t.Fatalf("SOT-IF-004: budget = %d、期待値 = %d", test.got, test.want)
			}
		})
	}
}
