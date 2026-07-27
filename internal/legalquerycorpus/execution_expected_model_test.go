package legalquerycorpus

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestExpectedAttemptは四つの形を保持する(t *testing.T) {
	t.Parallel()

	read := mustCompletedReadAttempt(t, "law-read", 1)
	collection := mustCompletedCollectionAttempt(t, "law-search", 2, 20, true)
	empty := mustEmptyAttempt(t, "law-updates", 3)
	failed := mustFailedAttempt(t, "law-search", 4, model.ErrorCodeSourceBusy)

	if read.Outcome() != legalquery.LegalQueryAttemptOutcomeCompleted ||
		read.PublishedItemCount() != 1 ||
		collection.Outcome() != legalquery.LegalQueryAttemptOutcomeCompleted ||
		collection.PublishedItemCount() != 20 ||
		!collection.HasMore() ||
		empty.Outcome() != legalquery.LegalQueryAttemptOutcomeEmpty ||
		empty.PublishedItemCount() != 0 ||
		empty.HasMore() ||
		failed.Outcome() != legalquery.LegalQueryAttemptOutcomeFailed ||
		failed.ErrorCode() != model.ErrorCodeSourceBusy {
		t.Fatal("SOT-ENG-026: expected attempt の値が保持されていない")
	}
}

func TestExpectedAttemptはheaderとvariant境界を検証する(t *testing.T) {
	t.Parallel()

	for _, values := range []ExpectedCompletedReadAttemptValues{
		{MeaningID: "", StepOrdinal: 1, PublishedItemCount: 1},
		{MeaningID: "Law Read", StepOrdinal: 1, PublishedItemCount: 1},
		{MeaningID: "law-read", StepOrdinal: 0, PublishedItemCount: 1},
		{MeaningID: "law-read", StepOrdinal: 5, PublishedItemCount: 1},
		{MeaningID: "law-read", StepOrdinal: 1, PublishedItemCount: 0},
		{MeaningID: "law-read", StepOrdinal: 1, PublishedItemCount: 2},
	} {
		if _, err := NewExpectedCompletedReadAttempt(values); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な read attempt を受理した: %#v", values)
		}
	}

	for _, count := range []int{1, 20} {
		if _, err := NewExpectedCompletedCollectionAttempt(
			ExpectedCompletedCollectionAttemptValues{
				MeaningID:          "law-search",
				StepOrdinal:        1,
				PublishedItemCount: count,
				HasMore:            true,
			},
		); err != nil {
			t.Fatalf("SOT-ENG-026: collection count=%d error = %v", count, err)
		}
	}
	for _, count := range []int{0, 21} {
		if _, err := NewExpectedCompletedCollectionAttempt(
			ExpectedCompletedCollectionAttemptValues{
				MeaningID:          "law-search",
				StepOrdinal:        1,
				PublishedItemCount: count,
				HasMore:            true,
			},
		); err == nil {
			t.Fatalf("SOT-ENG-026: collection count=%d を受理した", count)
		}
	}

	for _, values := range []ExpectedEmptyAttemptValues{
		{
			MeaningID:          "law-search",
			StepOrdinal:        1,
			PublishedItemCount: 1,
			HasMore:            false,
		},
		{
			MeaningID:          "law-search",
			StepOrdinal:        1,
			PublishedItemCount: 0,
			HasMore:            true,
		},
	} {
		if _, err := NewExpectedEmptyAttempt(values); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な empty attempt を受理した: %#v", values)
		}
	}
}

func TestExpectedFailedAttemptは公開可能なcodeだけを受理する(t *testing.T) {
	t.Parallel()

	for _, code := range allowedExecutionFailureCodesForTest() {
		if _, err := NewExpectedFailedAttempt(ExpectedFailedAttemptValues{
			MeaningID:   "law-search",
			StepOrdinal: 1,
			ErrorCode:   code,
		}); err != nil {
			t.Fatalf("SOT-ENG-026: errorCode=%q error = %v", code, err)
		}
	}
	for _, code := range []model.ErrorCode{
		model.ErrorCodeInvalidArgument,
		model.ErrorCodeUnsupportedCapability,
		model.ErrorCodeConfigurationRequired,
		"unknown",
	} {
		if _, err := NewExpectedFailedAttempt(ExpectedFailedAttemptValues{
			MeaningID:   "law-search",
			StepOrdinal: 1,
			ErrorCode:   code,
		}); err == nil {
			t.Fatalf("SOT-ENG-026: errorCode=%q を受理した", code)
		}
	}
}

func TestExecutionExpectedResultは三statusの内部整合を検証する(t *testing.T) {
	t.Parallel()

	valid := []ExecutionExpectedResultValues{
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 2,
			Attempts: []ExpectedAttempt{
				mustCompletedReadAttempt(t, "law-read", 1),
				mustCompletedCollectionAttempt(t, "law-search", 2, 1, false),
				mustEmptyAttempt(t, "law-updates", 3),
			},
		},
		{
			Status:            legalquery.LegalQueryResultStatusEmpty,
			ReturnedItemCount: 0,
			Attempts: []ExpectedAttempt{
				mustEmptyAttempt(t, "law-search", 1),
			},
		},
		{
			Status:            legalquery.LegalQueryResultStatusPartial,
			ReturnedItemCount: 1,
			Attempts: []ExpectedAttempt{
				mustCompletedReadAttempt(t, "law-read", 1),
				mustFailedAttempt(t, "law-search", 2, model.ErrorCodeSourceBusy),
			},
		},
		{
			Status:            legalquery.LegalQueryResultStatusPartial,
			ReturnedItemCount: 0,
			Attempts: []ExpectedAttempt{
				mustEmptyAttempt(t, "law-search", 1),
				mustFailedAttempt(t, "law-read", 2, model.ErrorCodeSourceBusy),
			},
		},
	}
	for _, values := range valid {
		result, err := NewExecutionExpectedResult(values)
		if err != nil {
			t.Fatalf("SOT-ENG-026: expected result error = %v", err)
		}
		if result.Terminal() != ExecutionExpectedTerminalResult ||
			result.Status() != values.Status ||
			result.ReturnedItemCount() != values.ReturnedItemCount {
			t.Fatalf("SOT-ENG-026: expected result = %#v", result)
		}
	}
}

func TestExecutionExpectedResultは不整合を拒否する(t *testing.T) {
	t.Parallel()

	completed := mustCompletedReadAttempt(t, "law-read", 1)
	empty := mustEmptyAttempt(t, "law-search", 1)
	failed := mustFailedAttempt(t, "law-read", 1, model.ErrorCodeSourceBusy)
	fiveAttempts := []ExpectedAttempt{empty, empty, empty, empty, empty}
	tests := []ExecutionExpectedResultValues{
		{
			Status:            legalquery.LegalQueryResultStatusNeedsClarification,
			ReturnedItemCount: 1,
			Attempts:          []ExpectedAttempt{completed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 0,
			Attempts:          []ExpectedAttempt{},
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 1,
			Attempts:          []ExpectedAttempt{completed, failed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusEmpty,
			ReturnedItemCount: 0,
			Attempts:          []ExpectedAttempt{completed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusPartial,
			ReturnedItemCount: 0,
			Attempts:          []ExpectedAttempt{failed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 2,
			Attempts:          []ExpectedAttempt{completed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 41,
			Attempts:          []ExpectedAttempt{completed},
		},
		{
			Status:            legalquery.LegalQueryResultStatusEmpty,
			ReturnedItemCount: 0,
			Attempts:          fiveAttempts,
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 1,
			Attempts:          []ExpectedAttempt{ExpectedCompletedReadAttempt{}},
		},
		{
			Status:            legalquery.LegalQueryResultStatusCompleted,
			ReturnedItemCount: 1,
			Attempts: func() []ExpectedAttempt {
				attempt := completed
				return []ExpectedAttempt{&attempt}
			}(),
		},
	}
	for _, values := range tests {
		if _, err := NewExecutionExpectedResult(values); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な expected result を受理した: %#v", values)
		}
	}
}

func TestExecutionExpectedErrorは全failedと先頭codeを検証する(t *testing.T) {
	t.Parallel()

	first := mustFailedAttempt(t, "law-search", 1, model.ErrorCodeSourceBusy)
	second := mustFailedAttempt(t, "law-read", 2, model.ErrorCodeInternalError)
	expected, err := NewExecutionExpectedError(ExecutionExpectedErrorValues{
		ErrorCode: model.ErrorCodeSourceBusy,
		Attempts:  []ExpectedAttempt{first, second},
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: expected error error = %v", err)
	}
	if expected.Terminal() != ExecutionExpectedTerminalError ||
		expected.ErrorCode() != model.ErrorCodeSourceBusy ||
		len(expected.Attempts()) != 2 {
		t.Fatalf("SOT-ENG-026: expected error = %#v", expected)
	}

	for _, values := range []ExecutionExpectedErrorValues{
		{ErrorCode: model.ErrorCodeSourceBusy, Attempts: []ExpectedAttempt{}},
		{
			ErrorCode: model.ErrorCodeInternalError,
			Attempts:  []ExpectedAttempt{first, second},
		},
		{
			ErrorCode: model.ErrorCodeSourceBusy,
			Attempts:  []ExpectedAttempt{mustCompletedReadAttempt(t, "law-read", 1)},
		},
		{
			ErrorCode: model.ErrorCodeInvalidArgument,
			Attempts:  []ExpectedAttempt{first},
		},
	} {
		if _, err := NewExecutionExpectedError(values); err == nil {
			t.Fatalf("SOT-ENG-026: 不正な expected error を受理した: %#v", values)
		}
	}
}

func TestExecutionExpectedはattemptを深く複製する(t *testing.T) {
	t.Parallel()

	attempt := mustFailedAttempt(t, "law-search", 1, model.ErrorCodeSourceBusy)
	attempts := []ExpectedAttempt{attempt}
	expected, err := NewExecutionExpectedError(ExecutionExpectedErrorValues{
		ErrorCode: model.ErrorCodeSourceBusy,
		Attempts:  attempts,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionExpectedError() error = %v", err)
	}
	attempt.errorCode = model.ErrorCodeInternalError
	attempts[0] = mustFailedAttempt(t, "changed", 1, model.ErrorCodeInternalError)
	got := expected.Attempts()
	gotAttempt := got[0].(ExpectedFailedAttempt)
	gotAttempt.errorCode = model.ErrorCodeInternalError
	got[0] = gotAttempt
	if expected.Attempts()[0].(ExpectedFailedAttempt).ErrorCode() !=
		model.ErrorCodeSourceBusy {
		t.Fatal("SOT-ENG-026: expected attempts が外部から変更された")
	}
}

func TestExecutionExpectedGetterは並行読取りで共有状態を変更しない(t *testing.T) {
	t.Parallel()

	expected, err := NewExecutionExpectedResult(ExecutionExpectedResultValues{
		Status:            legalquery.LegalQueryResultStatusCompleted,
		ReturnedItemCount: 1,
		Attempts: []ExpectedAttempt{
			mustCompletedReadAttempt(t, "law-read", 1),
		},
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: NewExecutionExpectedResult() error = %v", err)
	}
	var wait sync.WaitGroup
	for index := 0; index < 32; index++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			for repeat := 0; repeat < 100; repeat++ {
				attempt := expected.Attempts()[0].(ExpectedCompletedReadAttempt)
				attempt.publishedItemCount = 0
			}
		}()
	}
	wait.Wait()
	if expected.Attempts()[0].(ExpectedCompletedReadAttempt).PublishedItemCount() != 1 {
		t.Fatal("SOT-ENG-026: 並行 getter が expected result を変更した")
	}
}

func TestExecutionExpected型はzero値と直接JSON復元を拒否する(t *testing.T) {
	t.Parallel()

	validations := []interface{ Validate() error }{
		ExpectedCompletedReadAttempt{},
		ExpectedCompletedCollectionAttempt{},
		ExpectedEmptyAttempt{},
		ExpectedFailedAttempt{},
		ExecutionExpectedResult{},
		ExecutionExpectedError{},
	}
	for _, value := range validations {
		if err := value.Validate(); err == nil {
			t.Fatalf("SOT-ENG-026: zero value %T を受理した", value)
		}
	}
	targets := []any{
		&ExpectedCompletedReadAttempt{},
		&ExpectedCompletedCollectionAttempt{},
		&ExpectedEmptyAttempt{},
		&ExpectedFailedAttempt{},
		&ExecutionExpectedResult{},
		&ExecutionExpectedError{},
	}
	for _, target := range targets {
		if err := json.Unmarshal([]byte(`{}`), target); err == nil {
			t.Fatalf("SOT-ENG-026: %T を直接 JSON 復元できた", target)
		}
	}
}

func mustCompletedReadAttempt(
	t *testing.T,
	meaningID string,
	stepOrdinal int,
) ExpectedCompletedReadAttempt {
	t.Helper()
	attempt, err := NewExpectedCompletedReadAttempt(
		ExpectedCompletedReadAttemptValues{
			MeaningID:          meaningID,
			StepOrdinal:        stepOrdinal,
			PublishedItemCount: 1,
		},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: completed read attempt error = %v", err)
	}
	return attempt
}

func mustCompletedCollectionAttempt(
	t *testing.T,
	meaningID string,
	stepOrdinal int,
	publishedItemCount int,
	hasMore bool,
) ExpectedCompletedCollectionAttempt {
	t.Helper()
	attempt, err := NewExpectedCompletedCollectionAttempt(
		ExpectedCompletedCollectionAttemptValues{
			MeaningID:          meaningID,
			StepOrdinal:        stepOrdinal,
			PublishedItemCount: publishedItemCount,
			HasMore:            hasMore,
		},
	)
	if err != nil {
		t.Fatalf("SOT-ENG-026: completed collection attempt error = %v", err)
	}
	return attempt
}

func mustEmptyAttempt(
	t *testing.T,
	meaningID string,
	stepOrdinal int,
) ExpectedEmptyAttempt {
	t.Helper()
	attempt, err := NewExpectedEmptyAttempt(ExpectedEmptyAttemptValues{
		MeaningID:          meaningID,
		StepOrdinal:        stepOrdinal,
		PublishedItemCount: 0,
		HasMore:            false,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: empty attempt error = %v", err)
	}
	return attempt
}

func mustFailedAttempt(
	t *testing.T,
	meaningID string,
	stepOrdinal int,
	errorCode model.ErrorCode,
) ExpectedFailedAttempt {
	t.Helper()
	attempt, err := NewExpectedFailedAttempt(ExpectedFailedAttemptValues{
		MeaningID:   meaningID,
		StepOrdinal: stepOrdinal,
		ErrorCode:   errorCode,
	})
	if err != nil {
		t.Fatalf("SOT-ENG-026: failed attempt error = %v", err)
	}
	return attempt
}
