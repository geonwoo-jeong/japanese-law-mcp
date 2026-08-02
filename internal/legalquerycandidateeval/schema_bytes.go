package legalquerycandidateeval

import _ "embed"

var (
	//go:embed schema-v2.json
	canonicalSchemaV2 []byte

	//go:embed schema-v3.json
	canonicalSchemaV3 []byte
)

// CanonicalSchemaV2 は内部参照だけを持つ schema v2 の複製を返す。
func CanonicalSchemaV2() []byte {
	return append([]byte(nil), canonicalSchemaV2...)
}

// CanonicalSchemaV3 は内部参照だけを持つ schema v3 の複製を返す。
func CanonicalSchemaV3() []byte {
	return append([]byte(nil), canonicalSchemaV3...)
}
