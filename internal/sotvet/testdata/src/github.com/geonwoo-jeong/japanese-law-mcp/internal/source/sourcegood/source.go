package sourcegood

import (
	"context"

	applicationfixture "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/testfixture"
	modelfixture "github.com/geonwoo-jeong/japanese-law-mcp/internal/model/testfixture"
)

type Adapter struct{}

func (Adapter) Lookup(context.Context) modelfixture.Record {
	return modelfixture.Record{}
}

var _ applicationfixture.Source = Adapter{}
