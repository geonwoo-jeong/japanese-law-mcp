package sourcegood

import (
	"context"

	applicationfixture "github.com/japanese-law-mcp/japanese-law-mcp/internal/application/testfixture"
	modelfixture "github.com/japanese-law-mcp/japanese-law-mcp/internal/model/testfixture"
)

type Adapter struct{}

func (Adapter) Lookup(context.Context) modelfixture.Record {
	return modelfixture.Record{}
}

var _ applicationfixture.Source = Adapter{}
