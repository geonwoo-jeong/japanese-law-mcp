package judicialdecisionread_test

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
)

func TestErrNotFoundSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(
		errors.New("裁判例詳細の取得に失敗しました"),
		judicialdecisionread.ErrNotFound,
	)
	if !errors.Is(wrapped, judicialdecisionread.ErrNotFound) {
		t.Fatal("SOT-IF-042: ErrNotFound を errors.Is で判定できません")
	}
}
