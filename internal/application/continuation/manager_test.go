package continuation

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

type singleByteReader struct {
	data  []byte
	total int
}

func (r *singleByteReader) Read(destination []byte) (int, error) {
	if r.total >= len(r.data) {
		return 0, io.EOF
	}
	destination[0] = r.data[r.total]
	r.total++
	return 1, nil
}

type failingRandomReader struct {
	remaining int
}

func (r *failingRandomReader) Read(destination []byte) (int, error) {
	if r.remaining == 0 {
		return 0, errors.New("乱数生成に失敗しました")
	}
	count := len(destination)
	if count > r.remaining {
		count = r.remaining
	}
	for index := 0; index < count; index++ {
		destination[index] = byte(index)
	}
	r.remaining -= count
	return count, nil
}

func TestManagerReadsExactly32BytesForProcessKey(t *testing.T) {
	t.Parallel()

	random := &singleByteReader{data: append(fixedContinuationKey(), 0xff)}
	manager, err := newManager(random, func() time.Time {
		return goldenNow
	})
	if err != nil {
		t.Fatalf("SOT-IF-016: newManager() のエラー = %v", err)
	}
	if manager == nil {
		t.Fatalf("SOT-IF-016: newManager() が nil を返した")
	}
	if random.total != 32 {
		t.Fatalf("SOT-IF-016: CSPRNG から読んだ byte 数 = %d、期待値 = 32", random.total)
	}
}

func TestManagerFailsWhenProcessKeyCannotBeGenerated(t *testing.T) {
	t.Parallel()

	for _, remaining := range []int{0, 31} {
		random := &failingRandomReader{remaining: remaining}
		manager, err := newManager(random, func() time.Time {
			return goldenNow
		})
		if err == nil || manager != nil {
			t.Fatalf(
				"SOT-IF-016: %d byte しか得られない CSPRNG の結果 = %#v, %v",
				remaining,
				manager,
				err,
			)
		}
	}
}

func TestNewManagerCreatesUsableIndependentProcessKeys(t *testing.T) {
	t.Parallel()

	first, err := NewManager()
	if err != nil {
		t.Fatalf("SOT-IF-016: 一つ目の NewManager() のエラー = %v", err)
	}
	second, err := NewManager()
	if err != nil {
		t.Fatalf("SOT-IF-016: 二つ目の NewManager() のエラー = %v", err)
	}
	if first == nil || second == nil || first == second {
		t.Fatalf("SOT-IF-016: process ごとの manager が独立していない")
	}
}

func TestTokenFromPreviousProcessIsRejected(t *testing.T) {
	t.Parallel()

	previous := newContinuationFixtureWithKey(t, fixedContinuationKey())
	current := newContinuationFixtureWithKey(t, alternateContinuationKey())
	token, err := previous.manager.Issue(goldenIssueInput(t, previous))
	if err != nil {
		t.Fatalf("SOT-IF-016: 再起動前 process の token 発行エラー = %v", err)
	}

	_, err = current.manager.Verify(goldenVerifyInput(current, token))
	if !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("SOT-IF-016: 再起動前 token の検証エラー = %v", err)
	}
}

func TestManagerFormattingDoesNotRevealProcessKey(t *testing.T) {
	t.Parallel()

	manager := newFixedManager(t, fixedContinuationKey(), goldenNow)
	for _, format := range []string{"%s", "%v", "%+v", "%#v"} {
		formatted := fmt.Sprintf(format, manager)
		if strings.Contains(formatted, "0 1 2 3 4") ||
			strings.Contains(formatted, "0x0, 0x1") {
			t.Fatalf("SOT-IF-016: manager の formatting が process key を公開した: %s", formatted)
		}
	}
}
