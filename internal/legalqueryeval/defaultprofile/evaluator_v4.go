package defaultprofile

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"

// NewV4 は、SOT-ENG-047 の source 世代で v3 と同じ評価写像を構築する。
func NewV4() (*Evaluator, error) {
	planning, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	return NewWithPlanningV4(planning)
}

// NewWithPlanningV4 は、明示した候補依存を v3 と同じ評価写像へ接続する。
func NewWithPlanningV4(planning Planning) (*Evaluator, error) {
	return newWithPlanning(planning, planningFailureScoredMismatch)
}
