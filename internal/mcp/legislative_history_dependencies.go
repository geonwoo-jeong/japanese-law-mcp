package mcp

import (
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/parliamentspeechsearch"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// LegislativeHistoryDependencies は、国会発言検索を一つの公開単位として保持する。
type LegislativeHistoryDependencies struct {
	search      parliamentspeechsearch.Port
	initialized bool
}

// NewLegislativeHistoryDependencies は、国会発言検索ポートの公開単位を返す。
func NewLegislativeHistoryDependencies(
	search parliamentspeechsearch.Port,
) (LegislativeHistoryDependencies, error) {
	if isNilParliamentSpeechSearchPort(search) {
		return LegislativeHistoryDependencies{},
			fmt.Errorf("legislative-history の検索ポートは必須です")
	}
	return LegislativeHistoryDependencies{
		search:      search,
		initialized: true,
	}, nil
}

func (d LegislativeHistoryDependencies) addTools(server *sdk.Server) {
	if !d.ready() {
		return
	}
	addSearchDietSpeechesTool(server, d.search)
}

func (d LegislativeHistoryDependencies) ready() bool {
	return d.initialized && !isNilParliamentSpeechSearchPort(d.search)
}

func isNilParliamentSpeechSearchPort(search parliamentspeechsearch.Port) bool {
	if search == nil {
		return true
	}
	value := reflect.ValueOf(search)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
