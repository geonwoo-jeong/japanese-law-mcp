package judicialcases

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/cueartifact"
)

type cueDefinition struct {
	category   string
	value      string
	syntaxRole legalquery.CueSyntaxRole
}

type resolvedCues struct {
	mentions map[string][]legalquery.CueMention
}

func buildCues(
	document *cueartifact.Artifact,
) (
	[]legalquery.CueVocabularyEntry,
	map[string]cueDefinition,
	error,
) {
	if err := document.ValidateEntries(validateCueEntry); err != nil {
		return nil, nil, err
	}
	entries := document.Entries()
	definitions := make(map[string]cueDefinition, len(entries))
	meaningIDs := make(map[string]string, len(entries))
	for _, entry := range entries {
		meaningKey := cueMeaningKey(entry.Category(), entry.Value())
		definitions[entry.CueID()] = cueDefinition{
			category:   entry.Category(),
			value:      entry.Value(),
			syntaxRole: entry.SyntaxRole(),
		}
		if _, exists := meaningIDs[meaningKey]; !exists {
			meaningIDs[meaningKey] = entry.CueID()
		}
	}
	requiredMeanings := []string{
		cueMeaningKey("operator", "individual"),
		cueMeaningKey("operator", "resource_choice"),
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
	return document.Vocabulary(), definitions, nil
}

func validateCueEntry(document cueartifact.Entry) error {
	if !validCueMeaning(document.Category(), document.Value()) {
		return fmt.Errorf("category/value が未対応です")
	}
	if _, exists := document.IntentGroup(); exists {
		return fmt.Errorf("intentGroup は指定できません")
	}
	if _, exists := document.Signal(); exists {
		return fmt.Errorf("signal は指定できません")
	}
	return validateCueSyntaxRole(document)
}

func validateCueSyntaxRole(document cueartifact.Entry) error {
	role := document.SyntaxRole()
	if document.Category() == "task" {
		if role != legalquery.CueSyntaxRoleTaskExpression &&
			role != legalquery.CueSyntaxRoleTaskPredicate {
			return fmt.Errorf(
				"task cue の syntaxRole は task_expression または task_predicate でなければなりません",
			)
		}
		return nil
	}
	if role != legalquery.CueSyntaxRoleNone {
		return fmt.Errorf(
			"task relation に使わない cue の syntaxRole は none でなければなりません",
		)
	}
	return nil
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
		return value == "individual" ||
			value == "resource_choice"
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
