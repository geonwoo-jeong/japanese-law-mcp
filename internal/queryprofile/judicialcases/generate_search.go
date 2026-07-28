package judicialcases

import (
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

func (p *Profile) buildSearchDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) ([]candidateDraft, bool, bool, error) {
	if !cues.has("task", "search") ||
		!cues.has("resource", "judicial_decision") {
		return nil, false, false, nil
	}
	drafts, ambiguous, err := p.buildSearchSubjects(input)
	if err != nil {
		return nil, false, false, err
	}
	if !cues.has("operator", "individual") || len(drafts) < 2 {
		return drafts, false, ambiguous, nil
	}
	if len(drafts) > 4 {
		return nil, true, ambiguous, nil
	}
	sort.SliceStable(drafts, func(left, right int) bool {
		return drafts[left].steps[0].startByte <
			drafts[right].steps[0].startByte
	})
	combined := candidateDraft{
		evidence: []legalquery.EvidenceCode{
			legalquery.EvidenceExplicitTask,
			legalquery.EvidenceExplicitResource,
		},
	}
	for _, draft := range drafts {
		combined.evidence = append(combined.evidence, draft.evidence...)
		combined.concepts = append(combined.concepts, draft.concepts...)
		combined.steps = append(combined.steps, draft.steps...)
	}
	return []candidateDraft{combined}, false, ambiguous, nil
}

func (p *Profile) buildSearchSubjects(
	input legalquery.CandidateGenerationInput,
) ([]candidateDraft, bool, error) {
	result := make([]candidateDraft, 0)
	for _, term := range input.QueryTermMentions() {
		searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{
				Query: term.Surface(),
			},
		)
		if err != nil {
			return nil, false, err
		}
		result = append(result, candidateDraft{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceMorphologicalContext,
			},
			steps: []stepDraft{{
				startByte: term.Span().StartByte(),
				input:     searchInput,
			}},
		})
	}
	for _, date := range input.DateMentions() {
		searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{
				Query: date.Surface(),
			},
		)
		if err != nil {
			return nil, false, err
		}
		result = append(result, candidateDraft{
			evidence: []legalquery.EvidenceCode{
				legalquery.EvidenceStructuredReference,
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
			},
			steps: []stepDraft{{
				startByte: date.Span().StartByte(),
				input:     searchInput,
			}},
		})
	}

	ambiguous := false
	for _, mention := range input.LegalConceptMentions() {
		definition, exists := p.concepts[mention.ConceptID()]
		if !exists {
			return nil, false, fmt.Errorf(
				"judicial-cases profile に未登録の conceptId %q があります",
				mention.ConceptID(),
			)
		}
		if mention.Canonical() != definition.entry.Canonical {
			return nil, false, fmt.Errorf(
				"conceptId %q の canonical が profile snapshot と一致しません",
				mention.ConceptID(),
			)
		}
		for _, candidate := range definition.entry.Candidates {
			if !isJudicialConceptCandidate(candidate) {
				continue
			}
			searchInput, err := legalquery.NewJudicialDecisionSearchIntentV1(
				legalquery.JudicialDecisionSearchIntentV1Values{
					Query: candidate.OfficialTerm,
				},
			)
			if err != nil {
				return nil, false, fmt.Errorf(
					"conceptId %q の裁判例検索条件を構築できません: %w",
					mention.ConceptID(),
					err,
				)
			}
			evidence := []legalquery.EvidenceCode{
				legalquery.EvidenceExplicitTask,
				legalquery.EvidenceExplicitResource,
				legalquery.EvidenceLegalConcept,
			}
			if mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
				evidence = append(
					evidence,
					legalquery.EvidenceUniqueTypoCorrection,
				)
			}
			result = append(result, candidateDraft{
				evidence: evidence,
				concepts: []legalquery.LegalConceptSource{
					definition.source,
				},
				steps: []stepDraft{{
					startByte: mention.Span().StartByte(),
					input:     searchInput,
				}},
			})
			if definition.entry.SelectionPolicy ==
				legalconceptlexicon.SelectionPolicyAmbiguousNoAutoExecute {
				ambiguous = true
			}
		}
	}
	sort.SliceStable(result, func(left, right int) bool {
		return result[left].steps[0].startByte <
			result[right].steps[0].startByte
	})
	return result, ambiguous, nil
}

func isJudicialConceptCandidate(
	candidate legalconceptlexicon.Candidate,
) bool {
	return candidate.Task == legalquery.TaskSearch &&
		candidate.Resource == legalquery.ResourceJudicialDecision &&
		candidate.InputKind == legalquery.InputKindJudicialDecisionSearch &&
		len(candidate.RequiredPacks) == 1 &&
		candidate.RequiredPacks[0] == requiredPackID
}
