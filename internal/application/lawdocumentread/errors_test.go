package lawdocumentread_test

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
)

func TestErrNotFoundSupportsErrorsIs(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(errors.New("取得に失敗しました"), lawdocumentread.ErrNotFound)
	if !errors.Is(wrapped, lawdocumentread.ErrNotFound) {
		t.Fatal("ErrNotFound を errors.Is で判定できません")
	}
}
