package legalquery

import "fmt"

// QueryCompositionConstraint は、profile または候補合成が検出した非実行制約を表す。
type QueryCompositionConstraint string

const (
	// QueryCompositionConstraintNone は、候補合成による非実行制約がないことを表す。
	QueryCompositionConstraintNone QueryCompositionConstraint = "none"
	// QueryCompositionConstraintIneligible は、必須構成員を安全に合成できないことを表す。
	QueryCompositionConstraintIneligible QueryCompositionConstraint = "composition_ineligible"
	// QueryCompositionConstraintStepLimitExceeded は、必須 step が四件上限を超えたことを表す。
	QueryCompositionConstraintStepLimitExceeded QueryCompositionConstraint = "step_limit_exceeded"
)

// Validate は、候補合成制約が定義済みの値であることを確認する。
func (c QueryCompositionConstraint) Validate() error {
	switch c {
	case QueryCompositionConstraintNone,
		QueryCompositionConstraintIneligible,
		QueryCompositionConstraintStepLimitExceeded:
		return nil
	default:
		return fmt.Errorf("compositionConstraint が定義されていません")
	}
}
