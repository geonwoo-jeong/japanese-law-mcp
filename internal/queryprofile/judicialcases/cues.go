package judicialcases

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

var cueIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

type cuesDocument struct {
	SchemaVersion int           `json:"schemaVersion"`
	ProfileID     string        `json:"profileId"`
	CueSetVersion string        `json:"cueSetVersion"`
	Cues          []cueDocument `json:"cues"`
}

type cueDocument struct {
	CueID    string   `json:"cueId"`
	Category string   `json:"category"`
	Value    string   `json:"value"`
	Terms    []string `json:"terms"`
}

type cueDefinition struct {
	category string
	value    string
}

type resolvedCues struct {
	mentions map[string][]legalquery.CueMention
}

func buildCues(
	document cuesDocument,
) (
	[]legalquery.CueVocabularyEntry,
	map[string]cueDefinition,
	error,
) {
	if len(document.Cues) == 0 || len(document.Cues) > maximumCueCount {
		return nil, nil, fmt.Errorf(
			"cues は一件以上 %d 件以下必要です",
			maximumCueCount,
		)
	}
	cues := make([]legalquery.CueVocabularyEntry, 0, len(document.Cues))
	definitions := make(map[string]cueDefinition, len(document.Cues))
	meaningIDs := make(map[string]string, len(document.Cues))
	termOwners := make(map[string]string)
	previousID := ""
	for index, raw := range document.Cues {
		if !cueIDPattern.MatchString(raw.CueID) ||
			(index > 0 && previousID >= raw.CueID) {
			return nil, nil, fmt.Errorf(
				"cues は有効な cueId の昇順でなければなりません",
			)
		}
		if !validCueMeaning(raw.Category, raw.Value) {
			return nil, nil, fmt.Errorf(
				"cues[%d] の category/value が未対応です",
				index,
			)
		}
		meaningKey := cueMeaningKey(raw.Category, raw.Value)
		if previousCueID, exists := meaningIDs[meaningKey]; exists {
			return nil, nil, fmt.Errorf(
				"cue %q と %q の category/value が重複しています",
				previousCueID,
				raw.CueID,
			)
		}
		if len(raw.Terms) == 0 || len(raw.Terms) > 64 {
			return nil, nil, fmt.Errorf(
				"cues[%d].terms の件数が有効ではありません",
				index,
			)
		}
		terms := append([]string(nil), raw.Terms...)
		for termIndex, term := range terms {
			if strings.TrimSpace(term) == "" {
				return nil, nil, fmt.Errorf(
					"cues[%d].terms[%d] は必須です",
					index,
					termIndex,
				)
			}
			if previousCueID, exists := termOwners[term]; exists {
				return nil, nil, fmt.Errorf(
					"cue %q と %q で term %q が重複しています",
					previousCueID,
					raw.CueID,
					term,
				)
			}
			termOwners[term] = raw.CueID
		}
		slices.Sort(terms)
		if len(slices.Compact(terms)) != len(terms) {
			return nil, nil, fmt.Errorf(
				"cues[%d].terms を重複させることはできません",
				index,
			)
		}
		cues = append(cues, legalquery.CueVocabularyEntry{
			ProfileID: document.ProfileID,
			CueID:     raw.CueID,
			Terms:     terms,
		})
		definitions[raw.CueID] = cueDefinition{
			category: raw.Category,
			value:    raw.Value,
		}
		meaningIDs[meaningKey] = raw.CueID
		previousID = raw.CueID
	}
	requiredMeanings := []string{
		cueMeaningKey("operator", "individual"),
		cueMeaningKey("resource", "judicial_decision"),
		cueMeaningKey("resource_scope", "legal_information"),
		cueMeaningKey("task", "read"),
		cueMeaningKey("task", "search"),
	}
	for _, meaning := range requiredMeanings {
		if _, exists := meaningIDs[meaning]; !exists {
			return nil, nil, fmt.Errorf(
				"裁判例 profile に必要な cue category/value が不足しています",
			)
		}
	}
	return cues, definitions, nil
}

func validCueMeaning(category string, value string) bool {
	switch category {
	case "task":
		return value == "search" || value == "read"
	case "resource":
		return value == "judicial_decision"
	case "resource_scope":
		return value == "legal_information"
	case "operator":
		return value == "individual"
	default:
		return false
	}
}

func (p *Profile) resolveCues(
	values []legalquery.CueMention,
) (resolvedCues, error) {
	result := resolvedCues{
		mentions: make(map[string][]legalquery.CueMention),
	}
	for _, mention := range values {
		if mention.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		definition, exists := p.cueByID[mention.CueID()]
		if !exists {
			return resolvedCues{}, fmt.Errorf(
				"judicial-cases profile に未登録の cueId %q があります",
				mention.CueID(),
			)
		}
		key := cueMeaningKey(definition.category, definition.value)
		result.mentions[key] = append(result.mentions[key], mention)
	}
	return result, nil
}

func (c resolvedCues) has(category string, value string) bool {
	return len(c.mentions[cueMeaningKey(category, value)]) > 0
}

func cueMeaningKey(category string, value string) string {
	return category + "\x00" + value
}
