package legalquerycorpus

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExecutionOutcomeは四variantを保持する(t *testing.T) {
	t.Parallel()

	collection, err := NewCollectionSuccessOutcome(1000)
	if err != nil {
		t.Fatalf("SOT-ENG-026: collection_success error = %v", err)
	}
	read, err := NewReadSuccessOutcome()
	if err != nil {
		t.Fatalf("SOT-ENG-026: read_success error = %v", err)
	}
	failure, err := NewFailureOutcome(model.ErrorCodeSourceUnavailable)
	if err != nil {
		t.Fatalf("SOT-ENG-026: failure error = %v", err)
	}
	timeout, err := NewTimeoutOutcome()
	if err != nil {
		t.Fatalf("SOT-ENG-026: timeout error = %v", err)
	}

	if collection.Kind() != ExecutionOutcomeKindCollectionSuccess ||
		collection.SourceItemCount() != 1000 ||
		read.Kind() != ExecutionOutcomeKindReadSuccess ||
		failure.Kind() != ExecutionOutcomeKindFailure ||
		failure.ErrorCode() != model.ErrorCodeSourceUnavailable ||
		timeout.Kind() != ExecutionOutcomeKindTimeout {
		t.Fatal("SOT-ENG-026: execution outcome の値が保持されていない")
	}
}

func TestCollectionSuccessOutcomeはsourceItemCount境界だけを受理する(t *testing.T) {
	t.Parallel()

	for _, value := range []int{0, 1000} {
		if _, err := NewCollectionSuccessOutcome(value); err != nil {
			t.Fatalf("SOT-ENG-026: sourceItemCount=%d error = %v", value, err)
		}
	}
	for _, value := range []int{-1, 1001} {
		if _, err := NewCollectionSuccessOutcome(value); err == nil {
			t.Fatalf("SOT-ENG-026: sourceItemCount=%d を受理した", value)
		}
	}
	if err := (CollectionSuccessOutcome{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: CollectionSuccessOutcome の zero value を受理した")
	}
}

func TestFailureOutcomeはfailedAttemptのerrorCodeだけを受理する(t *testing.T) {
	t.Parallel()

	for _, code := range allowedExecutionFailureCodesForTest() {
		if _, err := NewFailureOutcome(code); err != nil {
			t.Fatalf("SOT-ENG-026: errorCode=%q error = %v", code, err)
		}
	}
	for _, code := range []model.ErrorCode{
		model.ErrorCodeInvalidArgument,
		model.ErrorCodeUnsupportedCapability,
		model.ErrorCodeConfigurationRequired,
		"unknown",
	} {
		if _, err := NewFailureOutcome(code); err == nil {
			t.Fatalf("SOT-ENG-026: errorCode=%q を受理した", code)
		}
	}
	if err := (FailureOutcome{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: FailureOutcome の zero value を受理した")
	}
	if err := (ReadSuccessOutcome{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ReadSuccessOutcome の zero value を受理した")
	}
	if err := (TimeoutOutcome{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: TimeoutOutcome の zero value を受理した")
	}
}

func TestExecutionActionは参照と順序値を保持する(t *testing.T) {
	t.Parallel()

	outcome, _ := NewCollectionSuccessOutcome(40)
	action, err := NewExecutionAction(ExecutionActionValues{
		MeaningID:    "law-search",
		StepOrdinal:  4,
		ReleaseOrder: 1,
		Outcome:      outcome,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionAction() error = %v", err)
	}
	if action.MeaningID() != "law-search" ||
		action.StepOrdinal() != 4 ||
		action.ReleaseOrder() != 1 ||
		action.Outcome().(CollectionSuccessOutcome).SourceItemCount() != 40 {
		t.Fatalf("SOT-ENG-026: execution action = %#v", action)
	}
}

func TestExecutionActionは単体境界違反を拒否する(t *testing.T) {
	t.Parallel()

	outcome, _ := NewReadSuccessOutcome()
	valid := ExecutionActionValues{
		MeaningID:    "law-read",
		StepOrdinal:  1,
		ReleaseOrder: 1,
		Outcome:      outcome,
	}
	tests := map[string]func(ExecutionActionValues) ExecutionActionValues{
		"meaningId空": func(values ExecutionActionValues) ExecutionActionValues {
			values.MeaningID = ""
			return values
		},
		"meaningId形式": func(values ExecutionActionValues) ExecutionActionValues {
			values.MeaningID = "Law Read"
			return values
		},
		"stepOrdinal下限": func(values ExecutionActionValues) ExecutionActionValues {
			values.StepOrdinal = 0
			return values
		},
		"stepOrdinal上限": func(values ExecutionActionValues) ExecutionActionValues {
			values.StepOrdinal = 5
			return values
		},
		"releaseOrder下限": func(values ExecutionActionValues) ExecutionActionValues {
			values.ReleaseOrder = 0
			return values
		},
		"releaseOrder上限": func(values ExecutionActionValues) ExecutionActionValues {
			values.ReleaseOrder = 5
			return values
		},
		"outcomeなし": func(values ExecutionActionValues) ExecutionActionValues {
			values.Outcome = nil
			return values
		},
		"outcome pointer": func(values ExecutionActionValues) ExecutionActionValues {
			read := values.Outcome.(ReadSuccessOutcome)
			values.Outcome = &read
			return values
		},
	}
	for name, mutate := range tests {
		name, mutate := name, mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := NewExecutionAction(mutate(valid)); err == nil {
				t.Fatal("SOT-ENG-026: 不正な ExecutionAction を受理した")
			}
		})
	}
	if err := (ExecutionAction{}).Validate(); err == nil {
		t.Fatal("SOT-ENG-026: ExecutionAction の zero value を受理した")
	}
}

func TestExecutionActionはzeroValueOutcomeを拒否する(t *testing.T) {
	t.Parallel()

	for _, outcome := range []ExecutionOutcome{
		CollectionSuccessOutcome{},
		ReadSuccessOutcome{},
		FailureOutcome{},
		TimeoutOutcome{},
	} {
		if _, err := NewExecutionAction(ExecutionActionValues{
			MeaningID:    "law-search",
			StepOrdinal:  1,
			ReleaseOrder: 1,
			Outcome:      outcome,
		}); err == nil {
			t.Fatalf("SOT-ENG-026: zero value outcome %T を受理した", outcome)
		}
	}
}

func TestExecutionActionはoutcomeを複製する(t *testing.T) {
	t.Parallel()

	failure, _ := NewFailureOutcome(model.ErrorCodeSourceBusy)
	action, err := NewExecutionAction(ExecutionActionValues{
		MeaningID:    "law-search",
		StepOrdinal:  1,
		ReleaseOrder: 1,
		Outcome:      failure,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionAction() error = %v", err)
	}
	failure.errorCode = model.ErrorCodeInternalError
	got := action.Outcome().(FailureOutcome)
	got.errorCode = model.ErrorCodeInternalError
	if action.Outcome().(FailureOutcome).ErrorCode() != model.ErrorCodeSourceBusy {
		t.Fatal("SOT-ENG-026: ExecutionAction の outcome が外部から変更された")
	}
}

func TestExecutionActionGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	outcome, _ := NewFailureOutcome(model.ErrorCodeSourceBusy)
	action, err := NewExecutionAction(ExecutionActionValues{
		MeaningID:    "law-search",
		StepOrdinal:  1,
		ReleaseOrder: 1,
		Outcome:      outcome,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionAction() error = %v", err)
	}

	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				got := action.Outcome().(FailureOutcome)
				got.errorCode = model.ErrorCodeInternalError
			}
		}()
	}
	wait.Wait()
	if action.Outcome().(FailureOutcome).ErrorCode() != model.ErrorCodeSourceBusy {
		t.Fatal("SOT-ENG-026: 並行 getter が ExecutionAction を変更した")
	}
}

func TestExecutionActionとOutcomeはJSONから直接復元できない(t *testing.T) {
	t.Parallel()

	targets := []any{
		&ExecutionAction{},
		&CollectionSuccessOutcome{},
		&ReadSuccessOutcome{},
		&FailureOutcome{},
		&TimeoutOutcome{},
	}
	for _, target := range targets {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-ENG-026: %T を直接 JSON 復元できた", target)
		}
	}
}

func allowedExecutionFailureCodesForTest() []model.ErrorCode {
	return []model.ErrorCode{
		model.ErrorCodeNotFound,
		model.ErrorCodeAmbiguousLocation,
		model.ErrorCodeUnsupportedQuery,
		model.ErrorCodeSourceAuthFailed,
		model.ErrorCodeRateLimited,
		model.ErrorCodeSourceTimeout,
		model.ErrorCodeSourceUnavailable,
		model.ErrorCodeSourceBusy,
		model.ErrorCodeSourceContractChanged,
		model.ErrorCodeInvalidSourceResponse,
		model.ErrorCodeSourceResponseTooLarge,
		model.ErrorCodeSourceProcessingLimit,
		model.ErrorCodeUnsafeSourceContent,
		model.ErrorCodeInternalError,
	}
}
