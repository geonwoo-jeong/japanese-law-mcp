package compactipagenerate

import (
	"encoding/binary"
	"fmt"
	"math"
	"unicode/utf8"

	"github.com/ikawaha/kagome-dict/dict"
)

const compactPOSMagic = "JLMIPOS1"

func encodePOS(table dict.POSTable) ([]byte, int, error) {
	words, names := len(table.POSs), len(table.NameList)
	if words == 0 || uint64(words) > math.MaxUint32 || names == 0 || names > math.MaxUint16 {
		return nil, 0, fmt.Errorf("IPA の品詞表の件数が範囲外です")
	}
	classes, ids, err := classifyPOS(table)
	if err != nil {
		return nil, 0, err
	}
	count := len(classes)
	if count == 0 || count > math.MaxUint16 {
		return nil, 0, fmt.Errorf("IPA の品詞分類数が範囲外です")
	}
	data := []byte(compactPOSMagic)
	data = binary.LittleEndian.AppendUint32(data, uint32(words))
	data = binary.LittleEndian.AppendUint16(data, uint16(count))
	data = binary.LittleEndian.AppendUint16(data, uint16(names))
	for _, name := range table.NameList {
		length := len(name)
		if length > math.MaxUint16 || !utf8.ValidString(name) {
			return nil, 0, fmt.Errorf("IPA の品詞名が不正です")
		}
		data = binary.LittleEndian.AppendUint16(data, uint16(length))
		data = append(data, name...)
	}
	for _, row := range classes {
		length := len(row)
		if length > math.MaxUint16 {
			return nil, 0, fmt.Errorf("IPA の品詞階層が上限を超えています")
		}
		data = binary.LittleEndian.AppendUint16(data, uint16(length))
		for _, id := range row {
			data = binary.LittleEndian.AppendUint16(data, uint16(id))
		}
	}
	for _, id := range ids {
		data = binary.LittleEndian.AppendUint16(data, id)
	}
	return data, count, nil
}

func classifyPOS(table dict.POSTable) ([]dict.POS, []uint16, error) {
	ids := make([]uint16, 0, len(table.POSs))
	classes := make([]dict.POS, 0)
	seen := make(map[string]uint16)
	for _, row := range table.POSs {
		keyBytes := make([]byte, 0, 2*len(row))
		for _, id := range row {
			if int(id) >= len(table.NameList) {
				return nil, nil, fmt.Errorf("IPA の品詞名参照が範囲外です")
			}
			keyBytes = binary.LittleEndian.AppendUint16(keyBytes, uint16(id))
		}
		key := string(keyBytes)
		class, exists := seen[key]
		if !exists {
			count := len(classes)
			if count >= math.MaxUint16 {
				return nil, nil, fmt.Errorf("IPA の品詞分類が上限を超えています")
			}
			class = uint16(count)
			classes = append(classes, row)
			seen[key] = class
		}
		ids = append(ids, class)
	}
	return classes, ids, nil
}
