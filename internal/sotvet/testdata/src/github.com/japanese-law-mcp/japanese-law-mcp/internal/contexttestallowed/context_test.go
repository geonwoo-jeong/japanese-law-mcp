package contexttestallowed

import (
	"context"
	"testing"
)

func TestFixture(t *testing.T) {
	t.Parallel()

	_ = context.Background()
	_ = context.TODO()
}
