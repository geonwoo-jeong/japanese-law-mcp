package judicialcasecitationextract_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
)

func TestErrNotFoundIsStableAndDoesNotExposeInput(t *testing.T) {
	t.Parallel()

	wrapped := errors.Join(
		errors.New("取得に失敗しました"),
		judicialcasecitationextract.ErrNotFound,
	)
	if !errors.Is(wrapped, judicialcasecitationextract.ErrNotFound) {
		t.Fatal("ErrNotFound を errors.Is で判定できません")
	}
	if strings.Contains(judicialcasecitationextract.ErrNotFound.Error(), "https://") {
		t.Fatal("ErrNotFound に取得 URL が含まれています")
	}
}
