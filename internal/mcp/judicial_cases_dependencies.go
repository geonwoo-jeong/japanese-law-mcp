package mcp

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
)

// JudicialCasesDependencies は、裁判例検索と詳細取得を一つの公開単位として保持する。
type JudicialCasesDependencies struct {
	search      judicialdecisionsearch.Port
	read        judicialdecisionread.Port
	initialized bool
}

// NewJudicialCasesDependencies は、二つの裁判例ポートがそろった公開単位を返す。
func NewJudicialCasesDependencies(
	search judicialdecisionsearch.Port,
	read judicialdecisionread.Port,
) (JudicialCasesDependencies, error) {
	if isNilJudicialSearchPort(search) {
		return JudicialCasesDependencies{},
			fmt.Errorf("judicial-cases の検索ポートは必須です")
	}
	if isNilJudicialReadPort(read) {
		return JudicialCasesDependencies{},
			fmt.Errorf("judicial-cases の詳細取得ポートは必須です")
	}
	return JudicialCasesDependencies{
		search:      search,
		read:        read,
		initialized: true,
	}, nil
}

func (d JudicialCasesDependencies) addTools(server toolRegistrar) {
	if !d.ready() {
		return
	}
	addSearchJudicialCasesTool(server, d.search)
	addGetJudicialCaseTool(server, d.read)
}

func (d JudicialCasesDependencies) ready() bool {
	return d.initialized &&
		!isNilJudicialSearchPort(d.search) &&
		!isNilJudicialReadPort(d.read)
}

func (d JudicialCasesDependencies) configured() bool {
	return d.initialized ||
		!isNilJudicialSearchPort(d.search) ||
		!isNilJudicialReadPort(d.read)
}
