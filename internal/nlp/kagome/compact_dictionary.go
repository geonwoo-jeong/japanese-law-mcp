package kagome

import (
	"archive/zip"
	_ "embed"
	"fmt"
	"strings"
	"sync"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/dict"
	"github.com/ikawaha/kagome/v2/tokenizer"
)

// SOT-ENG-051: 固定版 IPA の索引と分割コストを保持し、既知語の品詞行だけを共有する。
//
//go:embed data/ipa-tokenization.dict
var compactIPADictionary string

//go:embed data/ipa-pos.bin
var compactIPAPOS string

const compactPOSMagic = "JLMIPOS1"

type compactPOSTable struct {
	wordClasses string
	classes     [][]dict.POSID
	names       []string
	classNames  [][]string
}

type compactDictionary struct {
	dictionary *dict.Dict
	pos        compactPOSTable
}

var loadCompactIPADictionary = sync.OnceValues(func() (*compactDictionary, error) {
	reader := strings.NewReader(compactIPADictionary)
	archive, err := zip.NewReader(reader, reader.Size())
	if err != nil {
		return nil, fmt.Errorf("形態素解析辞書のzipを読み込めません: %w", err)
	}
	if err := validateCompactDictionaryParts(archive); err != nil {
		return nil, err
	}
	dictionary, err := dict.Load(archive, false)
	if err != nil {
		return nil, fmt.Errorf("形態素解析辞書を読み込めません: %w", err)
	}
	pos, err := decodeCompactPOS(compactIPAPOS)
	if err != nil {
		return nil, err
	}
	if len(pos.wordClasses)/2 != len(dictionary.Morphs) {
		return nil, fmt.Errorf("形態素辞書と品詞表の語数が一致しません")
	}
	return &compactDictionary{dictionary: dictionary, pos: pos}, nil
})

func validateCompactDictionaryParts(archive *zip.Reader) error {
	required := map[string]bool{
		dict.MorphDictFileName: false, dict.ContentMetaFileName: false, dict.IndexDictFileName: false,
		dict.ConnectionDictFileName: false, dict.CharDefDictFileName: false, dict.UnkDictFileName: false, dict.DictInfoFileName: false,
	}
	for _, part := range archive.File {
		seen, ok := required[part.Name]
		if !ok || seen {
			return fmt.Errorf("形態素解析辞書の部分 %q が未定義または重複しています", part.Name)
		}
		required[part.Name] = true
	}
	for name, seen := range required {
		if !seen {
			return fmt.Errorf("形態素解析辞書の部分 %q がありません", name)
		}
	}
	return nil
}

// SOT-ARCH-021: 共有する品詞列は内部だけで参照し、公開accessorで複製する。
func (p compactPOSTable) tokenPOS(token tokenizer.Token) ([]string, error) {
	if token.Class != tokenizer.KNOWN {
		return token.POS(), nil
	}
	if token.ID < 0 || token.ID >= len(p.wordClasses)/2 {
		return nil, fmt.Errorf("形態素解析結果の品詞参照が範囲外です")
	}
	offset := token.ID * 2
	class := uint16(p.wordClasses[offset]) | uint16(p.wordClasses[offset+1])<<8
	return p.classNames[class], nil
}

type compactPOSReader struct {
	data   string
	offset int
}

func (r *compactPOSReader) take(size int) (string, error) {
	if size < 0 || size > len(r.data)-r.offset {
		return "", fmt.Errorf("形態素解析の共有品詞表が途中で終了しています")
	}
	value := r.data[r.offset : r.offset+size]
	r.offset += size
	return value, nil
}
func (r *compactPOSReader) uint16() (uint16, error) {
	value, err := r.take(2)
	if err != nil {
		return 0, err
	}
	return uint16(value[0]) | uint16(value[1])<<8, nil
}
func (r *compactPOSReader) uint32() (uint32, error) {
	value, err := r.take(4)
	if err != nil {
		return 0, err
	}
	return uint32(value[0]) | uint32(value[1])<<8 | uint32(value[2])<<16 | uint32(value[3])<<24, nil
}
func decodeCompactPOS(data string) (compactPOSTable, error) {
	reader := compactPOSReader{data: data}
	magic, err := reader.take(len(compactPOSMagic))
	if err != nil {
		return compactPOSTable{}, err
	}
	if magic != compactPOSMagic {
		return compactPOSTable{}, fmt.Errorf("形態素解析の共有品詞表の形式が未対応です")
	}
	words, err := reader.uint32()
	if err != nil {
		return compactPOSTable{}, err
	}
	classCount, err := reader.uint16()
	if err != nil {
		return compactPOSTable{}, err
	}
	nameCount, err := reader.uint16()
	if err != nil {
		return compactPOSTable{}, err
	}
	if words == 0 || classCount == 0 || nameCount == 0 {
		return compactPOSTable{}, fmt.Errorf("形態素解析の共有品詞表に必要な項目がありません")
	}
	names, err := readCompactPOSNames(&reader, int(nameCount))
	if err != nil {
		return compactPOSTable{}, err
	}
	classes, classNames, err := readCompactPOSClasses(&reader, int(classCount), names)
	if err != nil {
		return compactPOSTable{}, err
	}
	wordClasses := data[reader.offset:]
	if len(wordClasses)%2 != 0 || uint64(words) != uint64(len(wordClasses)/2) {
		return compactPOSTable{}, fmt.Errorf("形態素解析の共有品詞表の語数が一致しません")
	}
	for offset := 0; offset < len(wordClasses); offset += 2 {
		class := uint16(wordClasses[offset]) | uint16(wordClasses[offset+1])<<8
		if int(class) >= len(classes) {
			return compactPOSTable{}, fmt.Errorf("形態素解析の品詞分類参照が範囲外です")
		}
	}
	return compactPOSTable{wordClasses: wordClasses, classes: classes, names: names, classNames: classNames}, nil
}
func readCompactPOSNames(reader *compactPOSReader, count int) ([]string, error) {
	names := make([]string, count)
	for index := range names {
		length, err := reader.uint16()
		if err != nil {
			return nil, err
		}
		name, err := reader.take(int(length))
		if err != nil {
			return nil, err
		}
		if !utf8.ValidString(name) {
			return nil, fmt.Errorf("形態素解析の品詞名がUTF-8ではありません")
		}
		names[index] = name
	}
	return names, nil
}
func readCompactPOSClasses(reader *compactPOSReader, count int, names []string) ([][]dict.POSID, [][]string, error) {
	classes := make([][]dict.POSID, count)
	classNames := make([][]string, count)
	for index := range classes {
		length, err := reader.uint16()
		if err != nil {
			return nil, nil, err
		}
		row := make([]dict.POSID, int(length))
		rowNames := make([]string, int(length))
		for offset := range row {
			id, err := reader.uint16()
			if err != nil {
				return nil, nil, err
			}
			if int(id) >= len(names) {
				return nil, nil, fmt.Errorf("形態素解析の品詞名参照が範囲外です")
			}
			row[offset] = dict.POSID(id)
			rowNames[offset] = names[id]
		}
		classes[index] = row
		classNames[index] = rowNames
	}
	return classes, classNames, nil
}
