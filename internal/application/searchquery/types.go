// Package searchquery は、プロバイダー非依存の検索語解決を提供する。
package searchquery

import "context"

const (
	maxEntryCount        = 50000
	maxTermsPerEntry     = 128
	maxTermBytes         = 2048
	maxQueryBytes        = 4096
	maxAnalyzedTermCount = 64
)

// Analyzer は、自然文から能力別辞書に登録された語を抽出する。
type Analyzer interface {
	RegisteredTerms(context.Context, string) ([]string, error)
}

// EntryValues は、一つの資源に対応する正式検索語と登録語を保持する。
type EntryValues struct {
	ResourceID string
	Canonical  string
	Terms      []string
}

type target struct {
	resourceID string
	canonical  string
}

type fuzzyTerm struct {
	value   string
	targets []target
}
