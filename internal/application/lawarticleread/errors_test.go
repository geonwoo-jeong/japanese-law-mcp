package lawarticleread_test

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
)

func TestSentinelErrorsSupportErrorsIs(t *testing.T) {
	t.Parallel()

	for name, target := range map[string]error{
		"not_found":          lawarticleread.ErrNotFound,
		"ambiguous_location": lawarticleread.ErrAmbiguousLocation,
	} {
		name, target := name, target
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			wrapped := errors.Join(errors.New("条文取得に失敗しました"), target)
			if !errors.Is(wrapped, target) {
				t.Fatalf("SOT-IF-025: %s を errors.Is で判定できない", name)
			}
		})
	}
}
