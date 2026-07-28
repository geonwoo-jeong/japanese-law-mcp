package searchquery

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"unicode/utf8"
)

// Resolver は、能力別の不変辞書と analyzer から安全な正式検索語を解決する。
type Resolver struct {
	analyzer   Analyzer
	exact      map[string][]target
	normalized map[string][]target
	fuzzy      map[int][]fuzzyTerm
}

// NewResolver は、入力を複製して検索用の不変索引を構築する。
func NewResolver(entries []EntryValues, analyzer Analyzer) (*Resolver, error) {
	if len(entries) == 0 || len(entries) > maxEntryCount {
		return nil, fmt.Errorf(
			"検索語 entry は 1 件以上 %d 件以下でなければなりません",
			maxEntryCount,
		)
	}
	if isNilInterface(analyzer) {
		return nil, fmt.Errorf("検索語 analyzer は必須です")
	}

	exact := make(map[string][]target)
	normalized := make(map[string][]target)
	fuzzyValues := make(map[string][]target)
	resourceIDs := make(map[string]struct{}, len(entries))
	for entryIndex, entry := range entries {
		if err := validateEntry(entry); err != nil {
			return nil, fmt.Errorf("entries[%d]: %w", entryIndex, err)
		}
		if _, duplicated := resourceIDs[entry.ResourceID]; duplicated {
			return nil, fmt.Errorf(
				"resourceId %q が重複しています",
				entry.ResourceID,
			)
		}
		resourceIDs[entry.ResourceID] = struct{}{}
		currentTarget := target{
			resourceID: entry.ResourceID,
			canonical:  entry.Canonical,
		}
		terms := append([]string{entry.Canonical}, entry.Terms...)
		seenTerms := make(map[string]struct{}, len(terms))
		for _, term := range terms {
			if _, duplicated := seenTerms[term]; duplicated {
				continue
			}
			seenTerms[term] = struct{}{}
			exact[term] = appendUniqueTarget(exact[term], currentTarget)
			key := comparisonKey(term)
			normalized[key] = appendUniqueTarget(
				normalized[key],
				currentTarget,
			)
			fuzzyValues[key] = appendUniqueTarget(
				fuzzyValues[key],
				currentTarget,
			)
		}
	}

	return &Resolver{
		analyzer:   analyzer,
		exact:      sortTargetIndex(exact),
		normalized: sortTargetIndex(normalized),
		fuzzy:      buildFuzzyIndex(fuzzyValues),
	}, nil
}

// Resolve は、exact、比較用正規化、解析語、誤記距離の順で一意な正式語を返す。
func (r *Resolver) Resolve(
	ctx context.Context,
	query string,
) (string, bool, error) {
	matches, err := r.ResolveMatches(ctx, query)
	if err != nil {
		return "", false, err
	}
	if len(matches) != 1 {
		return "", false, nil
	}
	return matches[0].canonical, true, nil
}

// ResolveMatches は、最初に成立した照合方法の候補を決定的な順序で返す。
func (r *Resolver) ResolveMatches(
	ctx context.Context,
	query string,
) ([]Match, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if r == nil || isNilInterface(r.analyzer) {
		return nil, fmt.Errorf("検索語 resolver は初期化されていません")
	}
	if !utf8.ValidString(query) ||
		len(query) == 0 ||
		len(query) > maxQueryBytes {
		return nil, fmt.Errorf(
			"検索語は有効な UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			maxQueryBytes,
		)
	}

	if targets := r.exact[query]; len(targets) > 0 {
		return matchesFromTargets(targets, MatchKindExact), nil
	}
	key := comparisonKey(query)
	if targets := r.normalized[key]; len(targets) > 0 {
		return matchesFromTargets(
			targets,
			MatchKindComparisonNormalized,
		), nil
	}

	terms, err := r.analyzer.RegisteredTerms(ctx, query)
	if err != nil {
		return nil, err
	}
	if len(terms) > maxAnalyzedTermCount {
		return nil, fmt.Errorf(
			"解析した登録語は %d 件以下でなければなりません",
			maxAnalyzedTermCount,
		)
	}
	analyzedTargets := make([]target, 0, len(terms))
	for _, term := range terms {
		termKey := comparisonKey(term)
		analyzedTargets = appendUniqueTargets(
			analyzedTargets,
			r.normalized[termKey],
		)
	}
	if len(analyzedTargets) > 0 {
		slices.SortFunc(analyzedTargets, compareTargets)
		return matchesFromTargets(
			analyzedTargets,
			MatchKindRegisteredTerm,
		), nil
	}

	fuzzyTargets, err := r.resolveFuzzyTargets(ctx, key)
	if err != nil {
		return nil, err
	}
	if len(fuzzyTargets) == 0 {
		return nil, nil
	}
	return matchesFromTargets(
		fuzzyTargets,
		MatchKindUniqueTypoCorrection,
	), nil
}

func (r *Resolver) resolveFuzzyTargets(
	ctx context.Context,
	query string,
) ([]target, error) {
	queryRunes := []rune(query)
	if len(queryRunes) < 3 {
		return nil, nil
	}

	bestDistance := 4
	var bestTerm fuzzyTerm
	ambiguousTerm := false
	checked := 0
	for length := max(3, len(queryRunes)-3); length <= len(queryRunes)+3; length++ {
		for _, term := range r.fuzzy[length] {
			checked++
			if checked%64 == 0 {
				if err := ctx.Err(); err != nil {
					return nil, err
				}
			}
			maximum := fuzzyMaximum(length)
			if maximum == 0 {
				continue
			}
			distance := boundedDamerauLevenshtein(
				queryRunes,
				[]rune(term.value),
				maximum,
			)
			switch {
			case distance > maximum || distance > bestDistance:
				continue
			case distance < bestDistance:
				bestDistance = distance
				bestTerm = term
				ambiguousTerm = false
			default:
				if term.value != bestTerm.value {
					ambiguousTerm = true
				}
			}
		}
	}
	if bestDistance == 4 || ambiguousTerm {
		return nil, nil
	}
	return append([]target(nil), bestTerm.targets...), nil
}

func matchesFromTargets(values []target, kind MatchKind) []Match {
	matches := make([]Match, len(values))
	for index, value := range values {
		matches[index] = Match{
			resourceID: value.resourceID,
			canonical:  value.canonical,
			kind:       kind,
		}
	}
	return matches
}

func validateEntry(entry EntryValues) error {
	if err := validateTerm("resourceId", entry.ResourceID); err != nil {
		return err
	}
	if err := validateTerm("canonical", entry.Canonical); err != nil {
		return err
	}
	if len(entry.Terms) > maxTermsPerEntry {
		return fmt.Errorf(
			"terms は %d 件以下でなければなりません",
			maxTermsPerEntry,
		)
	}
	for index, term := range entry.Terms {
		if err := validateTerm(fmt.Sprintf("terms[%d]", index), term); err != nil {
			return err
		}
	}
	return nil
}

func validateTerm(name string, value string) error {
	if !utf8.ValidString(value) ||
		strings.TrimSpace(value) == "" ||
		len(value) > maxTermBytes ||
		comparisonKey(value) == "" {
		return fmt.Errorf(
			"%s は比較できる UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maxTermBytes,
		)
	}
	return nil
}

func buildFuzzyIndex(values map[string][]target) map[int][]fuzzyTerm {
	index := make(map[int][]fuzzyTerm)
	for value, targets := range values {
		length := len([]rune(value))
		index[length] = append(index[length], fuzzyTerm{
			value:   value,
			targets: append([]target(nil), targets...),
		})
	}
	for length := range index {
		slices.SortFunc(index[length], func(left, right fuzzyTerm) int {
			return strings.Compare(left.value, right.value)
		})
	}
	return index
}

func sortTargetIndex(index map[string][]target) map[string][]target {
	for key := range index {
		slices.SortFunc(index[key], compareTargets)
	}
	return index
}

func appendUniqueTargets(values []target, additions []target) []target {
	for _, addition := range additions {
		values = appendUniqueTarget(values, addition)
	}
	return values
}

func appendUniqueTarget(values []target, addition target) []target {
	for _, value := range values {
		if value.resourceID == addition.resourceID {
			return values
		}
	}
	return append(values, addition)
}

func compareTargets(left target, right target) int {
	if compared := strings.Compare(left.resourceID, right.resourceID); compared != 0 {
		return compared
	}
	return strings.Compare(left.canonical, right.canonical)
}

func fuzzyMaximum(length int) int {
	switch {
	case length < 3:
		return 0
	case length <= 9:
		return 1
	case length <= 15:
		return 2
	default:
		return 3
	}
}

func isNilInterface(value any) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
