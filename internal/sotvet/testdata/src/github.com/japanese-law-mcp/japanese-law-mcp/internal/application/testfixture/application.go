package testfixture

import (
	"context"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model/testfixture"
)

type Source interface {
	Lookup(context.Context) testfixture.Record
}
