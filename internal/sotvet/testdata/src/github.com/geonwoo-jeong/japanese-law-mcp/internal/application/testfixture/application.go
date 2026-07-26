package testfixture

import (
	"context"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model/testfixture"
)

type Source interface {
	Lookup(context.Context) testfixture.Record
}
