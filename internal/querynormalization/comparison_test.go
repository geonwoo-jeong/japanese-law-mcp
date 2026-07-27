package querynormalization

import "testing"

func TestComparisonKeyは既存の比較用正規化を一意に提供する(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		input string
		want  string
	}{
		"NFKC と ASCII 小文字化": {
			input: " ＡＢＣ ",
			want:  "abc",
		},
		"Unicode 空白と句読点": {
			input: "民 法（第一条）。",
			want:  "民法第一条",
		},
		"全角と半角の片仮名": {
			input: "カタカナ・ｶﾀｶﾅ",
			want:  "かたかなかたかな",
		},
		"片仮名の繰返し記号": {
			input: "カヽヾ",
			want:  "かゝゞ",
		},
		"比較文字がない": {
			input: "　。、",
			want:  "",
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for iteration := 0; iteration < 10; iteration++ {
				if got := ComparisonKey(test.input); got != test.want {
					t.Fatalf(
						"ComparisonKey(%q) = %q、期待値は %q です",
						test.input,
						got,
						test.want,
					)
				}
			}
		})
	}
}
