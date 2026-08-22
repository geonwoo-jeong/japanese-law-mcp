// Package lawtarget は、プロバイダー非依存の解決済み法令対象を提供する。
package lawtarget

import "fmt"

// MatchKind は、解決済み法令対象へ到達した照合方法を表す。
type MatchKind string

const (
	// MatchKindExact は、正式名称または登録別名との完全一致を表す。
	MatchKindExact MatchKind = "exact"
	// MatchKindComparisonNormalized は、比較用正規化後の一致を表す。
	MatchKindComparisonNormalized MatchKind = "comparison_normalized"
	// MatchKindRegisteredTerm は、Kagome 登録語との一致を表す。
	MatchKindRegisteredTerm MatchKind = "registered_term"
	// MatchKindUniqueTypoCorrection は、一意な軽微誤記補正を表す。
	MatchKindUniqueTypoCorrection MatchKind = "unique_typo_correction"
)

// ResolvedLawTarget は、一意に解決した法令対象の不変な内部値である。
type ResolvedLawTarget struct {
	lawID         string
	officialTitle string
	matchKind     MatchKind
}

// NewResolvedLawTarget は、法令 ID、正式名称および照合方法を検証して返す。
func NewResolvedLawTarget(
	lawID string,
	officialTitle string,
	matchKind MatchKind,
) (ResolvedLawTarget, error) {
	target := ResolvedLawTarget{
		lawID:         lawID,
		officialTitle: officialTitle,
		matchKind:     matchKind,
	}
	if err := target.Validate(); err != nil {
		return ResolvedLawTarget{}, err
	}
	return target, nil
}

// LawID は、辞書で確認した公式法令 ID を返す。
func (t ResolvedLawTarget) LawID() string {
	return t.lawID
}

// OfficialTitle は、確認検索に使う正式名称を返す。
func (t ResolvedLawTarget) OfficialTitle() string {
	return t.officialTitle
}

// MatchKind は、辞書 entry へ到達した照合方法を返す。
func (t ResolvedLawTarget) MatchKind() MatchKind {
	return t.matchKind
}

// Validate は、内部対象の必須項目を確認する。
func (t ResolvedLawTarget) Validate() error {
	if t.lawID == "" {
		return fmt.Errorf("resolved law target の lawId は必須です")
	}
	if t.officialTitle == "" {
		return fmt.Errorf("resolved law target の officialTitle は必須です")
	}
	switch t.matchKind {
	case MatchKindExact,
		MatchKindComparisonNormalized,
		MatchKindRegisteredTerm,
		MatchKindUniqueTypoCorrection:
		return nil
	default:
		return fmt.Errorf("resolved law target の matchKind が不正です")
	}
}
