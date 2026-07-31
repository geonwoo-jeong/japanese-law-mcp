package judicialcases

import (
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type judicialEvidenceFact struct {
	values      profileevidence.FactValues
	conceptID   string
	cueCategory string
	inputRef    bool
}

type judicialEvidenceFactSet struct {
	values []profileevidence.FactValues
	byID   map[string]judicialEvidenceFact
}

func buildJudicialEvidenceFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (judicialEvidenceFactSet, error) {
	result := judicialEvidenceFactSet{
		byID: make(map[string]judicialEvidenceFact),
	}
	if _, exists := input.Ref(); exists {
		result.add(judicialEvidenceFact{
			values:   profileevidence.FactValues{FactID: "input-ref"},
			inputRef: true,
		})
	}
	if err := result.addCueFacts(input, cues); err != nil {
		return judicialEvidenceFactSet{}, err
	}
	for index, mention := range input.CaseNumberMentions() {
		result.add(judicialEvidenceFact{
			values: profileevidence.FactValues{
				FactID: fmt.Sprintf("case-number-%03d", index+1),
				Span:   cloneJudicialSpan(mention.Span()),
			},
		})
	}
	for index, mention := range input.DateMentions() {
		result.add(judicialEvidenceFact{
			values: profileevidence.FactValues{
				FactID: fmt.Sprintf("date-%03d", index+1),
				Span:   cloneJudicialSpan(mention.Span()),
			},
		})
	}
	for index, mention := range input.QueryTermMentions() {
		result.add(judicialEvidenceFact{
			values: profileevidence.FactValues{
				FactID: fmt.Sprintf("query-term-%03d", index+1),
				Span:   cloneJudicialSpan(mention.Span()),
			},
		})
	}
	for index, mention := range input.LegalConceptMentions() {
		result.add(judicialEvidenceFact{
			values: profileevidence.FactValues{
				FactID: fmt.Sprintf("legal-concept-%03d", index+1),
				Span:   cloneJudicialSpan(mention.Span()),
			},
			conceptID: mention.ConceptID(),
		})
	}
	return result, nil
}

func (s *judicialEvidenceFactSet) addCueFacts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) error {
	metadata, err := buildJudicialCueFactMetadata(cues)
	if err != nil {
		return err
	}
	for index, mention := range input.CueMentions() {
		if mention.ProfileID() != profileID {
			continue
		}
		current := metadata[cueRelationRefKeyFromMention(mention)]
		s.add(judicialEvidenceFact{
			values: profileevidence.FactValues{
				FactID: fmt.Sprintf("cue-%03d", index+1),
				Span:   cloneJudicialSpan(mention.Span()),
			},
			cueCategory: current.category,
		})
	}
	return nil
}

type judicialCueFactMetadata struct {
	category string
	value    string
}

func buildJudicialCueFactMetadata(
	cues resolvedCues,
) (map[cueRelationRefKey]judicialCueFactMetadata, error) {
	result := make(map[cueRelationRefKey]judicialCueFactMetadata)
	for meaning, mentions := range cues.mentions {
		category, value, valid := strings.Cut(meaning, "\x00")
		if !valid {
			return nil, fmt.Errorf("judicial cue の意味 key が有効ではありません")
		}
		for _, mention := range mentions {
			key := cueRelationRefKeyFromMention(mention)
			previous, exists := result[key]
			if exists &&
				(previous.category != category || previous.value != value) {
				return nil, fmt.Errorf("同じ judicial cue 出現に複数の意味があります")
			}
			result[key] = judicialCueFactMetadata{
				category: category,
				value:    value,
			}
		}
	}
	return result, nil
}

func (s *judicialEvidenceFactSet) add(fact judicialEvidenceFact) {
	s.values = append(s.values, fact.values)
	s.byID[fact.values.FactID] = fact
}

func cloneJudicialSpan(
	span legalquery.QuerySpan,
) *legalquery.QuerySpan {
	cloned := span
	return &cloned
}
