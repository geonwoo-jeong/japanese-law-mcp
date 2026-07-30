package cueartifact

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

const (
	maximumASCIIBytes = 128
)

var canonicalIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type meaningTuple struct {
	category    string
	value       string
	intentGroup optionalValue
	signal      optionalValue
	syntaxRole  legalquery.CueSyntaxRole
}

type normalizedTerm struct {
	cueID      string
	comparison string
	tuple      meaningTuple
}

func buildArtifact(document rawDocument) (*Artifact, error) {
	if document.schemaVersion != SchemaVersion {
		return nil, fmt.Errorf(
			"cues.json の schemaVersion は %d でなければなりません",
			SchemaVersion,
		)
	}
	if err := validateCanonicalID("profileId", document.profileID); err != nil {
		return nil, err
	}
	if err := validateCanonicalID(
		"cueSetVersion",
		document.cueSetVersion,
	); err != nil {
		return nil, err
	}
	if len(document.cues) == 0 {
		return nil, fmt.Errorf("cues は一件以上必要です")
	}

	entries := make([]Entry, 0, len(document.cues))
	registered := make([]normalizedTerm, 0)
	owners := make(map[string]string)
	previousCueID := ""
	for index, raw := range document.cues {
		entry, terms, err := buildEntry(
			index,
			previousCueID,
			raw,
		)
		if err != nil {
			return nil, err
		}
		for _, term := range terms {
			if owner, exists := owners[term.comparison]; exists {
				return nil, fmt.Errorf(
					"cue %q と %q に同じ比較用正規化語があります",
					owner,
					term.cueID,
				)
			}
			owners[term.comparison] = term.cueID
			registered = append(registered, term)
		}
		entries = append(entries, entry)
		previousCueID = raw.cueID
	}
	if err := validateTupleCollisions(registered); err != nil {
		return nil, err
	}
	return &Artifact{
		schemaVersion: document.schemaVersion,
		profileID:     document.profileID,
		cueSetVersion: document.cueSetVersion,
		entries:       entries,
	}, nil
}

func buildEntry(
	index int,
	previousCueID string,
	raw rawEntry,
) (Entry, []normalizedTerm, error) {
	if err := validateCanonicalID("cueId", raw.cueID); err != nil {
		return Entry{}, nil, fmt.Errorf("cues[%d]: %w", index, err)
	}
	if previousCueID != "" && previousCueID >= raw.cueID {
		return Entry{}, nil, fmt.Errorf(
			"cues は cueId の byte 昇順でなければなりません",
		)
	}
	for name, value := range map[string]string{
		"category": raw.category,
		"value":    raw.value,
	} {
		if err := validateASCIIValue(name, value); err != nil {
			return Entry{}, nil, fmt.Errorf("cues[%d]: %w", index, err)
		}
	}
	intentGroup, err := buildOptionalASCIIValue("intentGroup", raw.intentGroup)
	if err != nil {
		return Entry{}, nil, fmt.Errorf("cues[%d]: %w", index, err)
	}
	signal, err := buildOptionalASCIIValue("signal", raw.signal)
	if err != nil {
		return Entry{}, nil, fmt.Errorf("cues[%d]: %w", index, err)
	}
	syntaxRole, err := validateSyntaxRole(raw.syntaxRole)
	if err != nil {
		return Entry{}, nil, fmt.Errorf("cues[%d]: %w", index, err)
	}
	if len(raw.terms) == 0 {
		return Entry{}, nil, fmt.Errorf("cues[%d].terms は一件以上必要です", index)
	}

	tuple := meaningTuple{
		category:    raw.category,
		value:       raw.value,
		intentGroup: intentGroup,
		signal:      signal,
		syntaxRole:  syntaxRole,
	}
	terms := append([]string(nil), raw.terms...)
	normalized := make([]normalizedTerm, 0, len(terms))
	seen := make(map[string]struct{}, len(terms))
	previousComparison := ""
	previousTerm := ""
	for termIndex, term := range terms {
		if !utf8.ValidString(term) || term == "" {
			return Entry{}, nil, fmt.Errorf(
				"cues[%d].terms[%d] は有効な UTF-8 の非空文字列でなければなりません",
				index,
				termIndex,
			)
		}
		comparison := querynormalization.ComparisonKey(term)
		if comparison == "" {
			return Entry{}, nil, fmt.Errorf(
				"cues[%d].terms[%d] の比較用正規化値は必須です",
				index,
				termIndex,
			)
		}
		if termIndex > 0 &&
			(previousComparison > comparison ||
				(previousComparison == comparison && previousTerm >= term)) {
			return Entry{}, nil, fmt.Errorf(
				"cues[%d].terms は比較用正規化値と原文字列の昇順でなければなりません",
				index,
			)
		}
		if _, exists := seen[comparison]; exists {
			return Entry{}, nil, fmt.Errorf(
				"cues[%d].terms に同じ比較用正規化語があります",
				index,
			)
		}
		seen[comparison] = struct{}{}
		normalized = append(normalized, normalizedTerm{
			cueID:      raw.cueID,
			comparison: comparison,
			tuple:      tuple,
		})
		previousComparison = comparison
		previousTerm = term
	}
	return Entry{
		cueID:       raw.cueID,
		category:    raw.category,
		value:       raw.value,
		intentGroup: intentGroup,
		signal:      signal,
		syntaxRole:  syntaxRole,
		matchGroup:  matchGroupForTuple(tuple),
		terms:       terms,
	}, normalized, nil
}

func matchGroupForTuple(tuple meaningTuple) string {
	var value strings.Builder
	appendTupleComponent(&value, true, tuple.category)
	appendTupleComponent(&value, true, tuple.value)
	appendTupleComponent(
		&value,
		tuple.intentGroup.present,
		tuple.intentGroup.value,
	)
	appendTupleComponent(&value, tuple.signal.present, tuple.signal.value)
	appendTupleComponent(&value, true, string(tuple.syntaxRole))
	sum := sha256.Sum256([]byte(value.String()))
	return fmt.Sprintf("cue-tuple-sha256-%x", sum)
}

func appendTupleComponent(
	target *strings.Builder,
	present bool,
	value string,
) {
	if present {
		target.WriteByte('1')
	} else {
		target.WriteByte('0')
	}
	target.WriteString(strconv.Itoa(len(value)))
	target.WriteByte(':')
	target.WriteString(value)
	target.WriteByte(';')
}

func validateCanonicalID(name string, value string) error {
	if len(value) == 0 ||
		len(value) > maximumASCIIBytes ||
		!canonicalIDPattern.MatchString(value) {
		return fmt.Errorf(
			"%s は 1 byte 以上 %d byte 以下の正規形 ID でなければなりません",
			name,
			maximumASCIIBytes,
		)
	}
	return nil
}

func validateASCIIValue(name string, value string) error {
	if len(value) == 0 {
		return fmt.Errorf("%s は空でない ASCII 値でなければなりません", name)
	}
	for index := range len(value) {
		if value[index] > 0x7f {
			return fmt.Errorf("%s は ASCII 値でなければなりません", name)
		}
	}
	return nil
}

func buildOptionalASCIIValue(
	name string,
	value *string,
) (optionalValue, error) {
	if value == nil {
		return optionalValue{}, nil
	}
	if err := validateASCIIValue(name, *value); err != nil {
		return optionalValue{}, err
	}
	return optionalValue{value: *value, present: true}, nil
}

func validateSyntaxRole(value string) (legalquery.CueSyntaxRole, error) {
	role := legalquery.CueSyntaxRole(value)
	switch role {
	case legalquery.CueSyntaxRoleNone,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate:
		return role, nil
	default:
		return "", fmt.Errorf("syntaxRole %q は未対応です", value)
	}
}

func validateTupleCollisions(terms []normalizedTerm) error {
	ordered := append([]normalizedTerm(nil), terms...)
	slices.SortFunc(ordered, func(left, right normalizedTerm) int {
		return strings.Compare(left.comparison, right.comparison)
	})
	active := make([]normalizedTerm, 0)
	for _, current := range ordered {
		for len(active) > 0 &&
			!strings.HasPrefix(current.comparison, active[len(active)-1].comparison) {
			active = active[:len(active)-1]
		}
		for _, prefix := range active {
			if prefix.tuple != current.tuple {
				return fmt.Errorf(
					"cue %q と %q の比較用正規化語を異なる tuple で包含させることはできません",
					prefix.cueID,
					current.cueID,
				)
			}
		}
		active = append(active, current)
	}
	return nil
}
