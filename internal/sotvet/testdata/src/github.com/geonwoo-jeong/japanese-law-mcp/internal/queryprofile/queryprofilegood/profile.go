package queryprofilegood

import (
	applicationfixture "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/testfixture"
	modelfixture "github.com/geonwoo-jeong/japanese-law-mcp/internal/model/testfixture"
)

type Profile struct {
	Source applicationfixture.Source
	Record modelfixture.Record
}
