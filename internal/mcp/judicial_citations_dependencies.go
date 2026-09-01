package mcp

import (
	"fmt"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
)

// JudicialCitationsDependencies は、判例引用追跡ツールの公開単位を保持する。
type JudicialCitationsDependencies struct {
	trace       judicialcitationtrace.Port
	initialized bool
}

// NewJudicialCitationsDependencies は、引用追跡ポートの公開単位を返す。
func NewJudicialCitationsDependencies(
	trace judicialcitationtrace.Port,
) (JudicialCitationsDependencies, error) {
	if isNilJudicialCitationTracePort(trace) {
		return JudicialCitationsDependencies{},
			fmt.Errorf("judicial-citations の追跡ポートは必須です")
	}
	return JudicialCitationsDependencies{
		trace:       trace,
		initialized: true,
	}, nil
}

func (d JudicialCitationsDependencies) addTools(server toolRegistrar) {
	if !d.ready() {
		return
	}
	addTraceJudicialCitationsTool(server, d.trace)
}

func (d JudicialCitationsDependencies) ready() bool {
	return d.initialized && !isNilJudicialCitationTracePort(d.trace)
}

func (d JudicialCitationsDependencies) configured() bool {
	return d.initialized || !isNilJudicialCitationTracePort(d.trace)
}

func isNilJudicialCitationTracePort(trace judicialcitationtrace.Port) bool {
	if trace == nil {
		return true
	}
	value := reflect.ValueOf(trace)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
