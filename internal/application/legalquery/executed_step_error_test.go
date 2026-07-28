package legalquery

import (
	"errors"
	"testing"
)

func TestExecutedStepErrorPreservesExecutedPortFailure(t *testing.T) {
	t.Parallel()

	cause := errors.New("試験用の provider error")
	executed, err := NewExecutedStepError(cause)
	if err != nil {
		t.Fatalf("SOT-ARCH-023: ExecutedStepError を作成できません: %v", err)
	}
	if err := executed.Validate(); err != nil {
		t.Fatalf("SOT-ARCH-023: ExecutedStepError が有効ではありません: %v", err)
	}
	if !errors.Is(executed, cause) {
		t.Fatal("SOT-ARCH-023: provider error を Unwrap できません")
	}
	var classified ExecutedStepError
	if !errors.As(executed, &classified) {
		t.Fatal("SOT-ARCH-023: 実行済み step の失敗として分類できません")
	}
}

func TestExecutedStepErrorRejectsMissingCause(t *testing.T) {
	t.Parallel()

	var typedNil *executedStepTypedNilError
	for name, cause := range map[string]error{
		"nil":       nil,
		"typed nil": typedNil,
	} {
		name, cause := name, cause
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExecutedStepError(cause); err == nil {
				t.Fatal("SOT-ARCH-023: 原因のない実行済み step error を受理しました")
			}
		})
	}

	if err := (ExecutedStepError{}).Validate(); err == nil {
		t.Fatal("SOT-ARCH-023: zero-value ExecutedStepError を受理しました")
	}
}

type executedStepTypedNilError struct{}

func (*executedStepTypedNilError) Error() string {
	return "呼び出してはなりません"
}
