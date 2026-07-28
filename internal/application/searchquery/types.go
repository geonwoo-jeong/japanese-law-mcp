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

// MatchKind は、辞書 entry へ到達したプロバイダー非依存の照合方法を表す。
type MatchKind string

const (
	// MatchKindExact は、登録語との完全一致を表す。
	MatchKindExact MatchKind = "exact"
	// MatchKindComparisonNormalized は、比較用正規化後の一致を表す。
	MatchKindComparisonNormalized MatchKind = "comparison_normalized"
	// MatchKindRegisteredTerm は、Kagome が抽出した登録語との一致を表す。
	MatchKindRegisteredTerm MatchKind = "registered_term"
	// MatchKindUniqueTypoCorrection は、一意な軽微誤記補正を表す。
	MatchKindUniqueTypoCorrection MatchKind = "unique_typo_correction"
)

// Match は、一つの辞書 entry と照合方法を表す不変な値である。
type Match struct {
	resourceID string
	canonical  string
	kind       MatchKind
}

// ResourceID は、辞書 entry の不透明な資源識別子を返す。
func (m Match) ResourceID() string {
	return m.resourceID
}

// Canonical は、辞書で検証済みの正式検索語を返す。
func (m Match) Canonical() string {
	return m.canonical
}

// Kind は、辞書 entry へ到達した照合方法を返す。
func (m Match) Kind() MatchKind {
	return m.kind
}

type target struct {
	resourceID string
	canonical  string
}

type fuzzyTerm struct {
	value   string
	targets []target
}
