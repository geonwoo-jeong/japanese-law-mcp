package continuation

import (
	"fmt"
	"strconv"
	"sync"
	"testing"
)

func TestManagerAndImmutableValuesSupportConcurrentUse(t *testing.T) {
	t.Parallel()

	fixture := newContinuationFixture(t)
	base := goldenIssueInput(t, fixture)
	const workers = 64
	failures := make(chan error, workers)
	var wait sync.WaitGroup
	wait.Add(workers)
	for index := 0; index < workers; index++ {
		index := index
		go func() {
			defer wait.Done()

			position, err := NewJSONValue([]byte(strconv.Itoa(index)))
			if err != nil {
				failures <- fmt.Errorf("SOT-IF-016: position の作成エラー = %w", err)
				return
			}
			input := base
			input.Position = position
			token, err := fixture.manager.Issue(input)
			if err != nil {
				failures <- fmt.Errorf("SOT-IF-016: token の発行エラー = %w", err)
				return
			}
			cursor, err := fixture.manager.Verify(goldenVerifyInput(fixture, token))
			if err != nil {
				failures <- fmt.Errorf("SOT-IF-016: token の検証エラー = %w", err)
				return
			}
			if got := string(cursor.Position().Bytes()); got != strconv.Itoa(index) {
				failures <- fmt.Errorf(
					"SOT-IF-016: position = %s、期待値 = %d",
					got,
					index,
				)
				return
			}
			returned := cursor.Position().Bytes()
			returned[0] = 'x'
			if string(cursor.Position().Bytes()) != strconv.Itoa(index) {
				failures <- fmt.Errorf("SOT-IF-016: cursor が並行処理中に変更された")
			}
		}()
	}
	wait.Wait()
	close(failures)
	for err := range failures {
		t.Error(err)
	}
}
