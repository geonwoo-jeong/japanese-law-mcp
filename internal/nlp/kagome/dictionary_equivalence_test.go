package kagome

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

func TestAnalyzerPreservesFullDictionaryTokenization(t *testing.T) {
	terms := []string{"試験登録語", "個情法", "民法", "検索"}
	analyzer, err := NewAnalyzer(terms)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 解析器を構築できません: %v", err)
	}
	loaded, err := loadCompactIPADictionary()
	if err != nil {
		t.Fatalf("SOT-ENG-051: 生成辞書を読み込めません: %v", err)
	}
	compact := loaded.dictionary
	if compact.Contents != nil {
		t.Fatal("SOT-ARCH-021: 使用しない辞書付加情報を保持しています")
	}
	full := ipa.Dict()
	assertTokenizationDictionaryEqual(t, compact, full)
	reference := fullDictionaryTokenizer(t, full, terms)
	inputs := []string{
		"", "民法第709条を検索してください。", "個情法と個情法を確認する",
		"試験登録語を含む裁判例を検索", "働いていた人が申請できる条件",
		"ひらがなとカタカナ、半角ｶﾅ", "１２３と123、ＡＢＣとabc",
		"甲と乙について、それぞれ調べる。", "未登録語XYZ🧪の前後",
		"令和4年（ネ）第10039号", "a\tb\nc", strings.Repeat("あ", 1024),
	}
	for index, input := range inputs {
		got, analyzeErr := analyzer.AnalyzeTokenOccurrences(context.Background(), input)
		if analyzeErr != nil {
			t.Fatalf("SOT-ARCH-021: 入力 %d の解析に失敗しました: %v", index, analyzeErr)
		}
		want := reference.Analyze(input, tokenizer.Search)
		assertTokenOccurrencesEqual(t, index, got, want)
	}
}

func assertTokenizationDictionaryEqual(t *testing.T, compact, full *dict.Dict) {
	t.Helper()
	if len(full.Contents) == 0 {
		t.Fatal("SOT-ARCH-021: 比較用の完全辞書に付加情報がありません")
	}
	parts := []struct {
		name  string
		left  any
		right any
	}{
		{"形態素", compact.Morphs, full.Morphs},
		{"付加情報の定義", compact.ContentsMeta, full.ContentsMeta},
		{"索引", compact.Index, full.Index},
		{"接続コスト", compact.Connection, full.Connection},
		{"文字種", compact.CharClass, full.CharClass},
		{"文字分類", compact.CharCategory, full.CharCategory},
		{"未知語起動", compact.InvokeList, full.InvokeList},
		{"未知語連結", compact.GroupList, full.GroupList},
		{"未知語辞書", compact.UnkDict, full.UnkDict},
	}
	for _, part := range parts {
		if !reflect.DeepEqual(part.left, part.right) {
			t.Errorf("SOT-ARCH-021: %s が完全辞書と一致しません", part.name)
		}
	}
}

func fullDictionaryTokenizer(t *testing.T, full *dict.Dict, terms []string) *tokenizer.Tokenizer {
	t.Helper()
	records := make(dict.UserDictRecords, 0, len(terms))
	for _, term := range terms {
		records = append(records, dict.UserDicRecord{
			Text: term, Tokens: []string{term}, Yomi: []string{"*"}, Pos: "検索登録語",
		})
	}
	userDictionary, err := records.NewUserDict()
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 比較用の登録語辞書を構築できません: %v", err)
	}
	reference, err := tokenizer.New(full, tokenizer.UserDict(userDictionary), tokenizer.OmitBosEos())
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 比較用の解析器を構築できません: %v", err)
	}
	return reference
}

func assertTokenOccurrencesEqual(t *testing.T, inputIndex int, got []TokenOccurrence, want []tokenizer.Token) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("SOT-ARCH-021: 入力 %d の token 数が完全辞書と一致しません", inputIndex)
	}
	for index, token := range want {
		occurrence := got[index]
		if occurrence.Surface() != token.Surface ||
			occurrence.StartByte() != token.Position ||
			occurrence.EndByte() != token.Position+len(token.Surface) ||
			occurrence.UserDictionary() != (token.Class == tokenizer.USER) ||
			!reflect.DeepEqual(occurrence.PartOfSpeech(), token.POS()) {
			t.Errorf("SOT-ARCH-021: 入力 %d の token %d が完全辞書と一致しません", inputIndex, index)
		}
	}
}
