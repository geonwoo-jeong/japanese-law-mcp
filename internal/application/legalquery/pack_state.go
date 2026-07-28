package legalquery

import "fmt"

// PackState は、採用済み拡張 pack の有効状態を照会する。
type PackState interface {
	State(packID string) (enabled bool, adopted bool)
}

// StaticPackState は、起動時に確定した拡張 pack 状態の不変な snapshot である。
type StaticPackState struct {
	adopted map[string]struct{}
	enabled map[string]struct{}
}

var adoptedPackIDs = map[string]struct{}{
	"judicial-cases":      {},
	"legislative-history": {},
	"tax":                 {},
	"labor":               {},
}

// NewStaticPackState は、採用済み pack とその部分集合である有効 pack を返す。
func NewStaticPackState(
	adopted []string,
	enabled []string,
) (StaticPackState, error) {
	adoptedSet, err := newPackIDSet("adopted", adopted)
	if err != nil {
		return StaticPackState{}, err
	}
	enabledSet, err := newPackIDSet("enabled", enabled)
	if err != nil {
		return StaticPackState{}, err
	}
	for packID := range enabledSet {
		if _, exists := adoptedSet[packID]; !exists {
			return StaticPackState{}, fmt.Errorf(
				"有効な pack %q は採用済みでなければなりません",
				packID,
			)
		}
	}
	return StaticPackState{
		adopted: adoptedSet,
		enabled: enabledSet,
	}, nil
}

// State は、指定した pack の有効状態と採用状態を返す。
func (s StaticPackState) State(packID string) (bool, bool) {
	_, adopted := s.adopted[packID]
	_, enabled := s.enabled[packID]
	return enabled, adopted
}

func newPackIDSet(field string, values []string) (map[string]struct{}, error) {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, exists := adoptedPackIDs[value]; !exists {
			return nil, fmt.Errorf("%s に未定義の pack ID %q があります", field, value)
		}
		if _, exists := result[value]; exists {
			return nil, fmt.Errorf("%s の pack ID を重複させることはできません", field)
		}
		result[value] = struct{}{}
	}
	return result, nil
}
