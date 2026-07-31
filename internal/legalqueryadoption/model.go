package legalqueryadoption

// Profile は、採用済み profile の固定順 identity を保持する。
type Profile struct {
	profileID      string
	profileVersion string
	cueSetVersion  string
}

// ProfileID は profile ID を返す。
func (p Profile) ProfileID() string { return p.profileID }

// ProfileVersion は profile の意味版を返す。
func (p Profile) ProfileVersion() string { return p.profileVersion }

// CueSetVersion は profile の cue 成果物版を返す。
func (p Profile) CueSetVersion() string { return p.cueSetVersion }

// Manifest は、全履歴検証後の current adoption tuple である。
type Manifest struct {
	adoptionID         string
	previousAdoptionID string
	profileSetID       string
	profileSetVersion  string
	rankingVersion     string
	compositionVersion string
	evaluatorVersion   string
	profiles           []Profile
	corpusVersion      string
	holdoutDigest      string
	baselineVersion    string
	baselineSHA256     string
	catalogVersion     string
	catalogSHA256      string
}

// AdoptionID は current adoption の content-bound ID を返す。
func (m Manifest) AdoptionID() string { return m.adoptionID }

// PreviousAdoptionID は直前の採用 ID を返し、初回では空文字を返す。
func (m Manifest) PreviousAdoptionID() string { return m.previousAdoptionID }

// ProfileSetID は採用済み set ID を返す。
func (m Manifest) ProfileSetID() string { return m.profileSetID }

// ProfileSetVersion は採用済み set の不透明な版を返す。
func (m Manifest) ProfileSetVersion() string { return m.profileSetVersion }

// RankingVersion は共通順位校正版を返す。
func (m Manifest) RankingVersion() string { return m.rankingVersion }

// CompositionVersion は候補合成規則版を返す。
func (m Manifest) CompositionVersion() string { return m.compositionVersion }

// EvaluatorVersion は exact evaluator 意味版を返す。
func (m Manifest) EvaluatorVersion() string { return m.evaluatorVersion }

// CorpusVersion は標準 corpus 版を返す。
func (m Manifest) CorpusVersion() string { return m.corpusVersion }

// HoldoutDigest は標準 holdout digest を返す。
func (m Manifest) HoldoutDigest() string { return m.holdoutDigest }

// BaselineVersion は採用済み baseline 版を返す。
func (m Manifest) BaselineVersion() string { return m.baselineVersion }

// BaselineSHA256 は version baseline の原 byte digest を返す。
func (m Manifest) BaselineSHA256() string { return m.baselineSHA256 }

// CatalogVersion は現行検索例カタログ版を返す。
func (m Manifest) CatalogVersion() string { return m.catalogVersion }

// CatalogSHA256 は現行検索例 Markdown 集合の digest を返す。
func (m Manifest) CatalogSHA256() string { return m.catalogSHA256 }

// Profiles は production composition root の固定順を複製して返す。
func (m Manifest) Profiles() []Profile {
	return append([]Profile(nil), m.profiles...)
}
