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

// ConditionalTieBreakName は、条件付き tie-break の識別子である。
type ConditionalTieBreakName string

const (
	ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit = ConditionalTieBreakName(
		"lawAliasCollisionGroupsOverCandidateLimit",
	)
)

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
	ConditionalTieBreaks       map[ConditionalTieBreakName][]QueryTieBreak
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
	conditionalTieBreaks       map[ConditionalTieBreakName][]QueryTieBreak
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
		conditionalTieBreaks:       cloneConditionalTieBreaks(values.ConditionalTieBreaks),
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

// ConditionalTieBreaks は、条件名ごとの完全順を深い複製として返す。
func (m QueryProfileMetadata) ConditionalTieBreaks() map[ConditionalTieBreakName][]QueryTieBreak {
	return cloneConditionalTieBreaks(m.conditionalTieBreaks)
}

// Validate は、profile ID、版、対象、score、選択および tie-break を確認する。
func (m QueryProfileMetadata) Validate() error {
	switch m.schemaVersion {
	case 1:
		if _, present := m.selection.BranchRetentionMargin(); present {
			return fmt.Errorf("schemaVersion 1 に branchRetentionMargin は指定できません")
		}
	case 2:
		if _, present := m.selection.BranchRetentionMargin(); !present {
			return fmt.Errorf("schemaVersion 2 には branchRetentionMargin が必須です")
		}
	default:
		return fmt.Errorf("profile schemaVersion は 1 または 2 でなければなりません")
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
	for name, values := range m.conditionalTieBreaks {
		switch name {
		case ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit:
		default:
			return fmt.Errorf("未知の conditional tie-break %q は指定できません", name)
		}
		if err := validateCompleteTieBreakOrder(values); err != nil {
			return fmt.Errorf("conditionalTieBreaks.%s: %w", name, err)
		}
	}
	return nil
}

func validateCompleteTieBreakOrder(values []QueryTieBreak) error {
	if len(values) != len(requiredQueryTieBreak) {
		return fmt.Errorf("完全な tie-break 順が必要です")
	}
	seen := make(map[QueryTieBreak]struct{}, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			return fmt.Errorf("tie-break を重複させることはできません")
		}
		seen[value] = struct{}{}
	}
	for _, required := range requiredQueryTieBreak {
		if _, exists := seen[required]; !exists {
			return fmt.Errorf("必須 tie-break %q がありません", required)
		}
	}
	return nil
}

func cloneConditionalTieBreaks(
	values map[ConditionalTieBreakName][]QueryTieBreak,
) map[ConditionalTieBreakName][]QueryTieBreak {
	if len(values) == 0 {
		return nil
	}
	cloned := make(
		map[ConditionalTieBreakName][]QueryTieBreak,
		len(values),
	)
	for name, order := range values {
		cloned[name] = append([]QueryTieBreak(nil), order...)
	}
	return cloned
}
