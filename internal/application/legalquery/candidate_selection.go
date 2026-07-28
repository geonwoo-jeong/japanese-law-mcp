package legalquery

import "fmt"

// QuerySelectionMode は、profile が候補の自動選択を許すかを表す。
type QuerySelectionMode string

const (
	// QuerySelectionModeAutomatic は、閾値と明示された hedge pair による選択を許す。
	QuerySelectionModeAutomatic QuerySelectionMode = "automatic"
	// QuerySelectionModeClarificationRequired は、score にかかわらず明確化を必要とする。
	QuerySelectionModeClarificationRequired QuerySelectionMode = "clarification_required"
)

// CandidateHedgePairValues は、独立実行できる候補対を構築する値である。
type CandidateHedgePairValues struct {
	FirstCandidateID  string
	SecondCandidateID string
}

// CandidateHedgePair は、profile が明示的に hedge を許可した候補対である。
type CandidateHedgePair struct {
	firstCandidateID  string
	secondCandidateID string
}

// NewCandidateHedgePair は、相異なる二つの候補 ID を持つ不変な対を返す。
func NewCandidateHedgePair(
	values CandidateHedgePairValues,
) (CandidateHedgePair, error) {
	pair := CandidateHedgePair{
		firstCandidateID:  values.FirstCandidateID,
		secondCandidateID: values.SecondCandidateID,
	}
	if err := pair.Validate(); err != nil {
		return CandidateHedgePair{}, err
	}
	return pair, nil
}

// FirstCandidateID は、profile 順で先の候補 ID を返す。
func (p CandidateHedgePair) FirstCandidateID() string {
	return p.firstCandidateID
}

// SecondCandidateID は、profile 順で後の候補 ID を返す。
func (p CandidateHedgePair) SecondCandidateID() string {
	return p.secondCandidateID
}

// Validate は、候補 ID の構造と自己参照の禁止を確認する。
func (p CandidateHedgePair) Validate() error {
	if err := validateQueryPlanID(
		"firstCandidateId",
		p.firstCandidateID,
	); err != nil {
		return err
	}
	if err := validateQueryPlanID(
		"secondCandidateId",
		p.secondCandidateID,
	); err != nil {
		return err
	}
	if p.firstCandidateID == p.secondCandidateID {
		return fmt.Errorf("hedge pair は相異なる候補を必要とします")
	}
	return nil
}

// UnmarshalJSON は、profile を介さない直接復元を拒否する。
func (*CandidateHedgePair) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"CandidateHedgePair は JSON から直接復元できません",
	)
}

func isQuerySelectionMode(value QuerySelectionMode) bool {
	return value == QuerySelectionModeAutomatic ||
		value == QuerySelectionModeClarificationRequired
}
