package judicialcases

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type conceptDefinition struct {
	entry  legalconceptlexicon.Entry
	source legalquery.LegalConceptSource
}

func buildConceptDefinitions(
	entries []legalconceptlexicon.Entry,
) (map[string]conceptDefinition, error) {
	result := make(map[string]conceptDefinition, len(entries))
	for _, entry := range entries {
		confirmedOn, err := model.NewDate(entry.ConfirmedAt)
		if err != nil {
			return nil, fmt.Errorf(
				"concept %q の確認日が不正です: %w",
				entry.ConceptID,
				err,
			)
		}
		source, err := legalquery.NewLegalConceptSource(
			legalquery.LegalConceptSourceValues{
				ConceptID:   entry.ConceptID,
				Title:       entry.SourceName,
				URL:         entry.SourceURL,
				ConfirmedOn: confirmedOn,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"concept %q の出典が不正です: %w",
				entry.ConceptID,
				err,
			)
		}
		result[entry.ConceptID] = conceptDefinition{
			entry:  cloneConceptEntry(entry),
			source: source,
		}
	}
	return result, nil
}

func cloneConceptEntry(
	entry legalconceptlexicon.Entry,
) legalconceptlexicon.Entry {
	candidates := make(
		[]legalconceptlexicon.Candidate,
		0,
		len(entry.Candidates),
	)
	for _, candidate := range entry.Candidates {
		candidate.RequiredPacks = append(
			[]string(nil),
			candidate.RequiredPacks...,
		)
		candidates = append(candidates, candidate)
	}
	entry.Terms = append([]string(nil), entry.Terms...)
	entry.ComparisonTerms = append([]string(nil), entry.ComparisonTerms...)
	entry.Candidates = candidates
	return entry
}
