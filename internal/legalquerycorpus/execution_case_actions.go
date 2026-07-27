package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func validateExecutionActions(values []ExecutionAction) error {
	if len(values) < 1 || len(values) > maximumExecutionCaseActions {
		return fmt.Errorf("actions は一件以上四件以下でなければなりません")
	}

	releases := make([]bool, len(values)+1)
	closedMeanings := make(map[string]bool, 2)
	currentMeaning := ""
	meaningCount := 0
	previousStep := 0
	previousRelease := 0
	for _, action := range values {
		if err := action.Validate(); err != nil {
			return fmt.Errorf("execution case の action が有効ではありません: %w", err)
		}
		if action.ReleaseOrder() > len(values) || releases[action.ReleaseOrder()] {
			return fmt.Errorf("releaseOrder は一から action 件数までの順列でなければなりません")
		}
		releases[action.ReleaseOrder()] = true

		if action.MeaningID() != currentMeaning {
			if closedMeanings[action.MeaningID()] {
				return fmt.Errorf("同じ meaningId の actions は一つの連続 block にしてください")
			}
			if currentMeaning != "" {
				closedMeanings[currentMeaning] = true
			}
			meaningCount++
			if meaningCount > 2 {
				return fmt.Errorf("actions が参照する meaningId は二件以下でなければなりません")
			}
			if action.StepOrdinal() != 1 {
				return fmt.Errorf("各 meaning の stepOrdinal は 1 から始めなければなりません")
			}
			currentMeaning = action.MeaningID()
		} else {
			if action.StepOrdinal() != previousStep+1 {
				return fmt.Errorf("同じ meaning の stepOrdinal は一ずつ増やしてください")
			}
			if action.ReleaseOrder() <= previousRelease {
				return fmt.Errorf("同じ meaning 内で releaseOrder は step 順を逆転できません")
			}
		}
		previousStep = action.StepOrdinal()
		previousRelease = action.ReleaseOrder()
	}
	return nil
}

func validateExecutionActionProjection(
	actions []ExecutionAction,
	expected ExecutionExpected,
) error {
	attempts := expected.Attempts()
	if len(actions) != len(attempts) {
		return fmt.Errorf("actions と expected attempts の件数が一致しません")
	}
	allFailed := true
	for index, action := range actions {
		attempt := attempts[index]
		if action.MeaningID() != attempt.MeaningID() ||
			action.StepOrdinal() != attempt.StepOrdinal() {
			return fmt.Errorf("expected attempts は actions と同じ plan 順でなければなりません")
		}
		failed, err := validateExecutionOutcomeProjection(action.Outcome(), attempt)
		if err != nil {
			return err
		}
		allFailed = allFailed && failed
	}
	if allFailed != (expected.Terminal() == ExecutionExpectedTerminalError) {
		return fmt.Errorf("execution expected の terminal が action の成否と一致しません")
	}
	return nil
}

func validateExecutionOutcomeProjection(
	outcome ExecutionOutcome,
	attempt ExpectedAttempt,
) (bool, error) {
	switch typed := outcome.(type) {
	case CollectionSuccessOutcome:
		return false, validateCollectionSuccessProjection(typed, attempt)
	case ReadSuccessOutcome:
		if _, ok := attempt.(ExpectedCompletedReadAttempt); !ok {
			return false, fmt.Errorf("read_success は completed read attempt に投影してください")
		}
		return false, nil
	case FailureOutcome:
		failed, ok := attempt.(ExpectedFailedAttempt)
		if !ok || failed.ErrorCode() != typed.ErrorCode() {
			return true, fmt.Errorf("failure は同じ errorCode の failed attempt に投影してください")
		}
		return true, nil
	case TimeoutOutcome:
		failed, ok := attempt.(ExpectedFailedAttempt)
		if !ok || failed.ErrorCode() != model.ErrorCodeSourceTimeout {
			return true, fmt.Errorf("timeout は source_timeout の failed attempt に投影してください")
		}
		return true, nil
	default:
		return false, fmt.Errorf("execution outcome の投影種別が定義されていません")
	}
}

func validateCollectionSuccessProjection(
	outcome CollectionSuccessOutcome,
	attempt ExpectedAttempt,
) error {
	if outcome.SourceItemCount() == 0 {
		if _, ok := attempt.(ExpectedEmptyAttempt); !ok {
			return fmt.Errorf("零件の collection_success は empty attempt に投影してください")
		}
		return nil
	}
	completed, ok := attempt.(ExpectedCompletedCollectionAttempt)
	if !ok {
		return fmt.Errorf("非空の collection_success は completed collection attempt に投影してください")
	}
	if completed.PublishedItemCount() > outcome.SourceItemCount() {
		return fmt.Errorf("collection の公開件数は情報源の取得件数を超えられません")
	}
	hasMore := outcome.SourceItemCount() > completed.PublishedItemCount()
	if completed.HasMore() != hasMore {
		return fmt.Errorf("collection の hasMore は取得件数と公開件数から導出してください")
	}
	return nil
}

func cloneExecutionActions(values []ExecutionAction) ([]ExecutionAction, error) {
	cloned := make([]ExecutionAction, 0, len(values))
	for _, value := range values {
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("execution action が有効ではありません: %w", err)
		}
		action, err := NewExecutionAction(ExecutionActionValues{
			MeaningID:    value.MeaningID(),
			StepOrdinal:  value.StepOrdinal(),
			ReleaseOrder: value.ReleaseOrder(),
			Outcome:      value.Outcome(),
		})
		if err != nil {
			return nil, fmt.Errorf("execution action を複製できません: %w", err)
		}
		cloned = append(cloned, action)
	}
	return cloned, nil
}

func mustCloneExecutionActions(values []ExecutionAction) []ExecutionAction {
	cloned, err := cloneExecutionActions(values)
	if err != nil {
		panic(fmt.Sprintf("検証済み ExecutionCase の actions 複製に失敗しました: %v", err))
	}
	return cloned
}
