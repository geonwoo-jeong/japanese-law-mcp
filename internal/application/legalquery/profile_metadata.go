package legalquery

import "fmt"

// QueryTieBreak は、同点候補へ適用する決定的な比較順である。
type QueryTieBreak string

const (
	QueryTieBreakEvidenceSet      QueryTieBreak = "evidence_set"
	QueryTieBreakStepCount        QueryTieBreak = "step_count"
	QueryTieBreakMeaningSignature QueryTieBreak = "meaning_signature"
	QueryTieBreakSourcePosition   QueryTieBreak = "source_position"
)

var requiredQueryTieBreak = []QueryTieBreak{
	QueryTieBreakEvidenceSet,
	QueryTieBreakStepCount,
	QueryTieBreakMeaningSignature,
	QueryTieBreakSourcePosition,
}

// QueryProfileMetadataValues は、profile metadata を構築する値である。
type QueryProfileMetadataValues struct {
	SchemaVersion              int
	ProfileID                  string
	ProfileVersion             string
	RankingVersion             string
	CueSetVersion              string
	LawNameLexiconVersion      string
	LegalConceptLexiconVersion string
	Targets                    []QueryProfileTarget
	Score                      QueryScorePolicy
	Selection                  QuerySelectionPolicy
	TieBreak                   []QueryTieBreak
}

// QueryProfileMetadata は、profile の版付き規則を不変に保持する。
type QueryProfileMetadata struct {
	schemaVersion              int
	profileID                  string
	profileVersion             string
	rankingVersion             string
	cueSetVersion              string
	lawNameLexiconVersion      string
	legalConceptLexiconVersion string
	targets                    []QueryProfileTarget
	score                      QueryScorePolicy
	selection                  QuerySelectionPolicy
	tieBreak                   []QueryTieBreak
}

// NewQueryProfileMetadata は、起動時に完全検証した metadata を返す。
func NewQueryProfileMetadata(
	values QueryProfileMetadataValues,
) (QueryProfileMetadata, error) {
	metadata := QueryProfileMetadata{
		schemaVersion:              values.SchemaVersion,
		profileID:                  values.ProfileID,
		profileVersion:             values.ProfileVersion,
		rankingVersion:             values.RankingVersion,
		cueSetVersion:              values.CueSetVersion,
		lawNameLexiconVersion:      values.LawNameLexiconVersion,
		legalConceptLexiconVersion: values.LegalConceptLexiconVersion,
		targets:                    append([]QueryProfileTarget(nil), values.Targets...),
		score:                      values.Score,
		selection:                  values.Selection,
		tieBreak:                   append([]QueryTieBreak(nil), values.TieBreak...),
	}
	if err := metadata.Validate(); err != nil {
		return QueryProfileMetadata{}, err
	}
	return metadata, nil
}

func (m QueryProfileMetadata) SchemaVersion() int {
	return m.schemaVersion
}

func (m QueryProfileMetadata) ProfileID() string {
	return m.profileID
}

func (m QueryProfileMetadata) ProfileVersion() string {
	return m.profileVersion
}

func (m QueryProfileMetadata) RankingVersion() string {
	return m.rankingVersion
}

func (m QueryProfileMetadata) CueSetVersion() string {
	return m.cueSetVersion
}

func (m QueryProfileMetadata) LawNameLexiconVersion() string {
	return m.lawNameLexiconVersion
}

func (m QueryProfileMetadata) LegalConceptLexiconVersion() string {
	return m.legalConceptLexiconVersion
}

func (m QueryProfileMetadata) Targets() []QueryProfileTarget {
	return append([]QueryProfileTarget(nil), m.targets...)
}

func (m QueryProfileMetadata) Score() QueryScorePolicy {
	return m.score
}

func (m QueryProfileMetadata) Selection() QuerySelectionPolicy {
	return m.selection
}

func (m QueryProfileMetadata) TieBreak() []QueryTieBreak {
	return append([]QueryTieBreak(nil), m.tieBreak...)
}

// Validate は、profile ID、版、対象、score、選択および tie-break を確認する。
func (m QueryProfileMetadata) Validate() error {
	if m.schemaVersion != 1 {
		return fmt.Errorf("profile schemaVersion は 1 でなければなりません")
	}
	if err := validateQueryPlanID("profileId", m.profileID); err != nil {
		return err
	}
	for field, value := range map[string]string{
		"profileVersion":             m.profileVersion,
		"rankingVersion":             m.rankingVersion,
		"cueSetVersion":              m.cueSetVersion,
		"lawNameLexiconVersion":      m.lawNameLexiconVersion,
		"legalConceptLexiconVersion": m.legalConceptLexiconVersion,
	} {
		if err := validateProfileVersion(value); err != nil {
			return fmt.Errorf("%s が有効ではありません: %w", field, err)
		}
	}
	if len(m.targets) == 0 || len(m.targets) > 7 {
		return fmt.Errorf("profile targets は一件以上七件以下でなければなりません")
	}
	seenTargets := make(map[LogicalInputKind]struct{}, len(m.targets))
	for index, target := range m.targets {
		if err := target.Validate(); err != nil {
			return fmt.Errorf("targets[%d]: %w", index, err)
		}
		if _, exists := seenTargets[target.InputKind()]; exists {
			return fmt.Errorf("profile target を重複させることはできません")
		}
		seenTargets[target.InputKind()] = struct{}{}
	}
	if err := m.score.Validate(); err != nil {
		return fmt.Errorf("score policy が有効ではありません: %w", err)
	}
	if err := m.selection.Validate(); err != nil {
		return fmt.Errorf("selection policy が有効ではありません: %w", err)
	}
	if !m.selection.matchesScore(m.score) {
		return fmt.Errorf("selection と score の範囲が一致しません")
	}
	if len(m.tieBreak) != len(requiredQueryTieBreak) {
		return fmt.Errorf("tieBreak は完全な固定順を必要とします")
	}
	for index, value := range m.tieBreak {
		if value != requiredQueryTieBreak[index] {
			return fmt.Errorf("tieBreak が定義済みの完全順と一致しません")
		}
	}
	return nil
}
