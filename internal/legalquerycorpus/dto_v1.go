package legalquerycorpus

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// decodeJSONV1 は、後続の厳格 loader が schema と byte 安全性を確認した成果物を
// private v1 DTO へ変換し、DTO 固有の未知項目と末尾値も防御的に拒否する。
func decodeJSONV1(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("JSON 成果物の型付き構造が有効ではありません")
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("JSON 成果物の末尾には別の値を置けません")
	}
	return nil
}
