package legalquery

import "fmt"

// QuerySelectionPolicyValues は、selector が使う閾値と margin である。
type QuerySelectionPolicyValues struct {
	SingleThreshold           int
	MinimumExecutionThreshold int
	SingleMargin              int
	HedgeMargin               int
	BranchRetentionMargin     int
	BranchRetentionPresent    bool
	ScoreMinimum              int
	ScoreMaximum              int
}

// QuerySelectionPolicy は、profile の選択境界を不変に保持する。
type QuerySelectionPolicy struct {
	singleThreshold           int
	minimumExecutionThreshold int
	singleMargin              int
	hedgeMargin               int
	branchRetentionMargin     int
	branchRetentionPresent    bool
	scoreMinimum              int
	scoreMaximum              int
}

// NewQuerySelectionPolicy は、score 範囲内の選択境界を返す。
func NewQuerySelectionPolicy(
	values QuerySelectionPolicyValues,
) (QuerySelectionPolicy, error) {
	policy := QuerySelectionPolicy{
		singleThreshold:           values.SingleThreshold,
		minimumExecutionThreshold: values.MinimumExecutionThreshold,
		singleMargin:              values.SingleMargin,
		hedgeMargin:               values.HedgeMargin,
		branchRetentionMargin:     values.BranchRetentionMargin,
		branchRetentionPresent:    values.BranchRetentionPresent,
		scoreMinimum:              values.ScoreMinimum,
		scoreMaximum:              values.ScoreMaximum,
	}
	if err := policy.Validate(); err != nil {
		return QuerySelectionPolicy{}, err
	}
	return policy, nil
}

// SingleThreshold は、単独選択 score の下限を返す。
func (p QuerySelectionPolicy) SingleThreshold() int {
	return p.singleThreshold
}

// MinimumExecutionThreshold は、外部実行できる score の下限を返す。
func (p QuerySelectionPolicy) MinimumExecutionThreshold() int {
	return p.minimumExecutionThreshold
}

// SingleMargin は、単独選択に必要な一位と二位の差を返す。
func (p QuerySelectionPolicy) SingleMargin() int {
	return p.singleMargin
}

// HedgeMargin は、二候補を限定実行できる最大差を返す。
func (p QuerySelectionPolicy) HedgeMargin() int {
	return p.hedgeMargin
}

// BranchRetentionMargin は、限定分岐保持 margin と存在有無を返す。
func (p QuerySelectionPolicy) BranchRetentionMargin() (int, bool) {
	return p.branchRetentionMargin, p.branchRetentionPresent
}

// Validate は、閾値と margin の整合を確認する。
func (p QuerySelectionPolicy) Validate() error {
	if p.scoreMinimum < 0 ||
		p.scoreMaximum < p.scoreMinimum ||
		p.minimumExecutionThreshold < p.scoreMinimum ||
		p.singleThreshold < p.minimumExecutionThreshold ||
		p.singleThreshold > p.scoreMaximum {
		return fmt.Errorf("selection threshold が score 範囲内の昇順ではありません")
	}
	if p.singleMargin < 0 ||
		p.hedgeMargin < 0 ||
		p.hedgeMargin > p.singleMargin ||
		p.singleMargin > p.scoreMaximum-p.scoreMinimum {
		return fmt.Errorf("selection margin が有効ではありません")
	}
	if p.branchRetentionPresent &&
		(p.branchRetentionMargin < 0 ||
			p.branchRetentionMargin > p.scoreMaximum-p.scoreMinimum) {
		return fmt.Errorf("branchRetentionMargin が score 範囲内ではありません")
	}
	if !p.branchRetentionPresent && p.branchRetentionMargin != 0 {
		return fmt.Errorf(
			"存在しない branchRetentionMargin に値を保持することはできません",
		)
	}
	return nil
}

func (p QuerySelectionPolicy) matchesScore(score QueryScorePolicy) bool {
	return p.scoreMinimum == score.Minimum() &&
		p.scoreMaximum == score.Maximum()
}
