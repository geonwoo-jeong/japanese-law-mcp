// Copyright 2025 The Go MCP SDK Authors. All rights reserved.
// Use of this source code is governed by the license
// that can be found in the LICENSE file.

// Package json provides internal JSON utilities.

package json

import (
	"bytes"
	"io"

	"github.com/segmentio/encoding/json"
)

type Decoder struct {
	dec *json.Decoder
}

func NewDecoder(r io.Reader) *Decoder {
	dec := json.NewDecoder(r)
	dec.DontMatchCaseInsensitiveStructFields()
	return &Decoder{dec: dec}
}

func (d *Decoder) Decode(v any) error {
	return d.dec.Decode(v)
}

func Unmarshal(data []byte, v any) error {
	// 小さい完全な一値だけ、固定版 Decoder の32KiB bufferを実長copyへ置き換える。
	// 旧 Decoder と同じ raw token を複製し、error と入力の所有権を維持する。
	if len(data) < 32768 && json.Valid(data) {
		_, err := json.Parse(bytes.Clone(bytes.Trim(data, " \t\r\n")), v, json.DontMatchCaseInsensitiveStructFields)
		return err
	}
	return NewDecoder(bytes.NewReader(data)).Decode(v)
}
