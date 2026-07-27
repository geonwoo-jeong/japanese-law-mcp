package legalquerycorpus

import "fmt"

const maximumExecutionActionOrdinal = 4

// ExecutionActionValues は、fake capability action の作成値を保持する。
type ExecutionActionValues struct {
	MeaningID    string
	StepOrdinal  int
	ReleaseOrder int
	Outcome      ExecutionOutcome
}

// ExecutionAction は、一つの選択 step に対する宣言的な fake 結果を保持する。
type ExecutionAction struct {
	meaningID    string
	stepOrdinal  int
	releaseOrder int
	outcome      ExecutionOutcome
	initialized  bool
}

// NewExecutionAction は、参照、局所順序および outcome を検証して返す。
func NewExecutionAction(values ExecutionActionValues) (ExecutionAction, error) {
	outcome, err := cloneExecutionOutcome(values.Outcome)
	if err != nil {
		return ExecutionAction{}, err
	}
	action := ExecutionAction{
		meaningID:    values.MeaningID,
		stepOrdinal:  values.StepOrdinal,
		releaseOrder: values.ReleaseOrder,
		outcome:      outcome,
		initialized:  true,
	}
	if err := action.Validate(); err != nil {
		return ExecutionAction{}, err
	}
	return action, nil
}

// MeaningID は、semantic case 内の期待意味参照を返す。
func (a ExecutionAction) MeaningID() string {
	return a.meaningID
}

// StepOrdinal は、意味内で一から始まる step 順を返す。
func (a ExecutionAction) StepOrdinal() int {
	return a.stepOrdinal
}

// ReleaseOrder は、fake clock が一から解放する終端 event 順を返す。
func (a ExecutionAction) ReleaseOrder() int {
	return a.releaseOrder
}

// Outcome は、宣言的結果 variant の複製を返す。
func (a ExecutionAction) Outcome() ExecutionOutcome {
	outcome, err := cloneExecutionOutcome(a.outcome)
	if err != nil {
		panic(fmt.Sprintf("検証済み ExecutionAction の outcome 複製に失敗しました: %v", err))
	}
	return outcome
}

// Validate は、action 単体で確認できる識別子、範囲および outcome を確認する。
func (a ExecutionAction) Validate() error {
	if !a.initialized {
		return fmt.Errorf("ExecutionAction は NewExecutionAction で作成しなければなりません")
	}
	if err := validateExpectedIdentifier("meaningId", a.meaningID); err != nil {
		return err
	}
	if a.stepOrdinal < 1 || a.stepOrdinal > maximumExecutionActionOrdinal {
		return fmt.Errorf("stepOrdinal は 1 以上 4 以下でなければなりません")
	}
	if a.releaseOrder < 1 || a.releaseOrder > maximumExecutionActionOrdinal {
		return fmt.Errorf("releaseOrder は 1 以上 4 以下でなければなりません")
	}
	if _, err := cloneExecutionOutcome(a.outcome); err != nil {
		return fmt.Errorf("execution action の outcome が有効ではありません: %w", err)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExecutionAction) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExecutionAction は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}
