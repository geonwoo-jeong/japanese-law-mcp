package querynormalization

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func TestComparisonShareStage27DifferentialFixtures(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"空", ""},
		{"変更のない法令名", "個人情報の保護に関する法律"},
		{"変更のないASCII", "abc012xyz"},
		{"NFKCだけの変更", "ａｂｃ１２３"},
		{"互換文字", "㍿㌕①ⅣﬃÅ"},
		{"結合文字", "e\u0301か\u3099カ\u309a"},
		{"結合文字順序", "a\u0315\u0300\u0327"},
		{"非starter連続", strings.Repeat("\u0300", 32)},
		{"半角濁音", "ｶﾞﾊﾟｳﾞ"},
		{"ASCII境界", "@AZ[az{"},
		{"ASCII小文字化", "AaZz"},
		{"片仮名変換境界", "\u30a0\u30a1\u30f6\u30f7\u30fc\u30fd\u30fe\u30ff"},
		{"ひらがな反復", "\u309d\u309e"},
		{"先頭削除", "。、　民法"},
		{"先頭連続削除後の変換", " 。、Aカ"},
		{"途中削除", "民 法・刑法"},
		{"末尾削除", "民法 。"},
		{"全削除", "\t\r\n \u0085\u00a0\u2028\u2029\u3000。、（）"},
		{"多byte接頭辞後の変換", "民法Aカ"},
		{"長い接頭辞後の変換", strings.Repeat("民法", 256) + "A"},
		{"削除されない記号と制御文字", "\x00\x01\x7f+$=♥🙂\u200b\ufeff"},
		{"有効な置換文字", "\ufffd"},
		{"有効な置換文字を含む接頭辞", "民\ufffd法A"},
		{"不正UTF8単独", "\xff"},
		{"不正UTF8先頭", "\xff民法"},
		{"不正UTF8途中", "民\xff法"},
		{"不正UTF8末尾", "民法\xff"},
		{"不正UTF8と後続変換", "民\xff法A カ"},
		{"不正UTF8連続", "\x80\xbf\xc0\xaf\xff"},
		{"切断したUTF8", "\xe3\x81"},
		{"置換文字の切断", "\xef\xbf"},
		{"surrogateのUTF8", "\xed\xa0\x80"},
		{"Unicode上限超過", "\xf4\x90\x80\x80"},
		{"不正UTF8とNFKC", "Ａ\xffｶﾞ\xc0\xafｅ\u0301"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertComparisonShareStage27Equal(t, test.input)
		})
	}
}

func TestComparisonShareStage27ByteAndKanaBoundaries(t *testing.T) {
	// 一 byte の全値を含め、不正 UTF-8 の開始位置と変更済み経路を比較する。
	for value := 0; value <= 255; value++ {
		input := string([]byte{byte(value)})
		assertComparisonShareStage27Equal(t, input)
		assertComparisonShareStage27Equal(t, "民"+input+"法A")
		assertComparisonShareStage27Equal(t, " 。A"+input+"法")
	}
	for _, bounds := range [][2]rune{{'\u309c', '\u3100'}, {'\uff60', '\uffa0'}} {
		for current := bounds[0]; current <= bounds[1]; current++ {
			assertComparisonShareStage27Equal(t, "民"+string(current)+"法")
		}
	}
}

func TestComparisonShareStage27AllDictionaryStrings(t *testing.T) {
	// 固定 asset の全文字列値を比較し、field 選択による語の取りこぼしを避ける。
	assets := []struct {
		path    string
		sha256  string
		strings int
	}{
		{"lawnamelexicon/data/egov-current.json", "217b2d4053ff191dadfcde053a57710558d11a94770eeddb68332968129ff80e", 51048},
		{"lawnamelexicon/data/supplemental.json", "e196e736cd1e4e87e77e781183b9b3f9f9fbb9537a22d0ec36a2896c5f11b763", 33},
		{"legalconceptlexicon/data/current.json", "814fc5c5c59eb761879825d47f095b2e82267b0f60dca77eb765b30fdb03f2af", 177},
		{"queryprofile/core/data/cues.json", "fdee3edf0594808f901f02de8e3eee91601e44727ee9f2c684d4cf7d9f434152", 305},
		{"queryprofile/judicialcases/data/cues.json", "7386f92fa4efb71051d8b6174b9a766be2d923930803615692264028b244562a", 72},
	}
	for _, asset := range assets {
		t.Run(asset.path, func(t *testing.T) {
			raw, err := os.ReadFile(filepath.Join("..", filepath.FromSlash(asset.path)))
			if err != nil {
				t.Fatalf("固定辞書を読めません: %v", err)
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != asset.sha256 {
				t.Fatalf("固定辞書の SHA256 が一致しません: %s", got)
			}
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				t.Fatalf("固定辞書が JSON ではありません: %v", err)
			}
			if got := compareStage27JSONStrings(t, value); got != asset.strings {
				t.Fatalf("比較した全文字列値は %d 件、期待値は %d 件です", got, asset.strings)
			}
		})
	}
}

func compareStage27JSONStrings(t *testing.T, value any) int {
	t.Helper()
	count := 0
	switch typed := value.(type) {
	case string:
		assertComparisonShareStage27Equal(t, typed)
		return 1
	case []any:
		for _, item := range typed {
			count += compareStage27JSONStrings(t, item)
		}
	case map[string]any:
		for _, item := range typed {
			count += compareStage27JSONStrings(t, item)
		}
	}
	return count
}

func assertComparisonShareStage27Equal(t *testing.T, input string) {
	t.Helper()
	got, want := ComparisonKey(input), comparisonKeyStage27Original(input)
	if got != want {
		t.Fatalf("正規化の byte が従来と異なります: 入力=%x 候補=%x 従来=%x", input, got, want)
	}
}

// 元の関数本体をそのまま保持する独立 oracle。候補やその callback は呼ばない。
func comparisonKeyStage27Original(value string) string {
	normalized := norm.NFKC.String(value)
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, current := range normalized {
		if unicode.IsSpace(current) || unicode.IsPunct(current) {
			continue
		}
		switch {
		case current >= 'A' && current <= 'Z':
			current += 'a' - 'A'
		case current >= '\u30a1' && current <= '\u30f6':
			current -= '\u0060'
		case current == '\u30fd':
			current = '\u309d'
		case current == '\u30fe':
			current = '\u309e'
		}
		builder.WriteRune(current)
	}
	return builder.String()
}
