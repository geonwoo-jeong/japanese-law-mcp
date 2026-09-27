package kagome

import (
	"context"
	"encoding/binary"
	"reflect"
	"testing"

	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome-dict/ipa"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

func TestCompactIPAReconstructsEveryPOSAndPreservesOtherParts(t *testing.T) {
	compact, err := loadCompactIPADictionary()
	if err != nil {
		t.Fatal(err)
	}
	reference := ipa.DictShrink()
	if !reflect.DeepEqual(compact.pos.names, reference.POSTable.NameList) {
		t.Fatal("SOT-ARCH-021: 品詞名表が固定版IPAと一致しません")
	}
	if len(compact.pos.wordClasses)/2 != len(reference.POSTable.POSs) {
		t.Fatal("SOT-ARCH-021: 品詞語数が固定版IPAと一致しません")
	}
	for index, want := range reference.POSTable.POSs {
		offset := index * 2
		class := uint16(compact.pos.wordClasses[offset]) | uint16(compact.pos.wordClasses[offset+1])<<8
		if !reflect.DeepEqual(dict.POS(compact.pos.classes[class]), want) {
			t.Fatalf("SOT-ARCH-021: 語 %d の品詞IDが固定版IPAと一致しません", index)
		}
		got, err := compact.pos.tokenPOS(tokenizer.Token{Class: tokenizer.KNOWN, ID: index})
		if err != nil {
			t.Fatal(err)
		}
		wantNames := make([]string, len(want))
		for item, id := range want {
			wantNames[item] = reference.POSTable.NameList[id]
		}
		if !reflect.DeepEqual(got, wantNames) {
			t.Fatalf("SOT-ARCH-021: 語 %d の品詞が固定版IPAと一致しません", index)
		}
	}
	expected := *reference
	expected.POSTable = dict.POSTable{}
	expected.Contents = nil
	if !reflect.DeepEqual(compact.dictionary, &expected) {
		t.Fatal("SOT-ARCH-021: 品詞以外の辞書構成が固定版IPAと一致しません")
	}
	t.Logf("全%d語、%d品詞分類、%d品詞名とその他の全辞書部分が一致しました", len(reference.POSTable.POSs), len(compact.pos.classes), len(compact.pos.names))
}

func TestCompactIPADoesNotExposeSharedPOS(t *testing.T) {
	analyzer, err := NewAnalyzer([]string{"試験登録語"})
	if err != nil {
		t.Fatal(err)
	}
	input := "働いていた人が申請できる条件"
	before, err := analyzer.AnalyzeTokenOccurrences(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	for _, token := range before {
		pos := token.PartOfSpeech()
		for index := range pos {
			pos[index] = "変更"
		}
	}
	after, err := analyzer.AnalyzeTokenOccurrences(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) {
		t.Fatal("SOT-ARCH-021: accessorの返却値から共有品詞が変更されました")
	}
}

func compactPOSSynthetic() string {
	data := append([]byte(compactPOSMagic), make([]byte, 8)...)
	binary.LittleEndian.PutUint32(data[8:12], 2)
	binary.LittleEndian.PutUint16(data[12:14], 1)
	binary.LittleEndian.PutUint16(data[14:16], 1)
	name := []byte("名詞")
	data = binary.LittleEndian.AppendUint16(data, 6)
	data = append(data, name...)
	data = binary.LittleEndian.AppendUint16(data, 1)
	data = binary.LittleEndian.AppendUint16(data, 0)
	data = binary.LittleEndian.AppendUint16(data, 0)
	data = binary.LittleEndian.AppendUint16(data, 0)
	return string(data)
}
func TestCompactPOSRejectsTruncationAndInvalidReferences(t *testing.T) {
	input := compactPOSSynthetic()
	if _, err := decodeCompactPOS(input); err != nil {
		t.Fatal(err)
	}
	for length := 0; length < len(input); length++ {
		if _, err := decodeCompactPOS(input[:length]); err == nil {
			t.Fatalf("SOT-ARCH-021: %d byteで切れた品詞表を受理しました", length)
		}
	}
	variants := map[string]string{
		"形式":     "X" + input[1:],
		"余剰":     input + "\x00\x00",
		"分類参照":   input[:len(input)-2] + "\xff\xff",
		"UTF-8":  input[:18] + "\xff" + input[19:],
		"空の語数":   input[:8] + "\x00\x00\x00\x00" + input[12:],
		"空の分類数":  input[:12] + "\x00\x00" + input[14:],
		"空の品詞名数": input[:14] + "\x00\x00" + input[16:],
		"語数":     input[:8] + "\xff\xff\xff\xff" + input[12:],
		"品詞名参照":  input[:26] + "\xff\xff" + input[28:],
	}
	for name, value := range variants {
		if _, err := decodeCompactPOS(value); err == nil {
			t.Fatalf("SOT-ARCH-021: 不正な%sを受理しました", name)
		}
	}
	table, err := decodeCompactPOS(input)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range []int{-1, 2} {
		if _, err := table.tokenPOS(tokenizer.Token{Class: tokenizer.KNOWN, ID: id}); err == nil {
			t.Fatal("SOT-ARCH-021: 範囲外の語IDを受理しました")
		}
	}
}
