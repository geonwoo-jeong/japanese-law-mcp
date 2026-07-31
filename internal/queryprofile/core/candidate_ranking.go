package core

import (
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type preparedDraft struct {
	draft            candidateDraft
	evidence         []legalquery.EvidenceCode
	score            int
	confidence       legalquery.Confidence
	signature        string
	rankingSignature string
}

func comparePreparedDrafts(
	left preparedDraft,
	right preparedDraft,
	rankAliasCollisionGroupsBySource bool,
) int {
	if left.score != right.score {
		return right.score - left.score
	}
	leftEvidence := evidenceSignature(left.evidence)
	rightEvidence := evidenceSignature(right.evidence)
	if leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	if len(left.draft.steps) != len(right.draft.steps) {
		return len(left.draft.steps) - len(right.draft.steps)
	}
	if rankAliasCollisionGroupsBySource {
		if comparison := compareAliasCollisionGroupPositions(
			left,
			right,
		); comparison != 0 {
			return comparison
		}
	}
	if left.rankingSignature != right.rankingSignature {
		return strings.Compare(left.rankingSignature, right.rankingSignature)
	}
	if left.signature != right.signature {
		return strings.Compare(left.signature, right.signature)
	}
	return sourcePosition(left.draft) - sourcePosition(right.draft)
}

func normalizeEvidence(
	values map[legalquery.EvidenceCode]struct{},
	preserveOfficialAlias bool,
	preserveLegalConcept bool,
	preserveMorphologicalContext bool,
) []legalquery.EvidenceCode {
	result := make(map[legalquery.EvidenceCode]struct{}, len(values))
	for code := range values {
		result[code] = struct{}{}
	}
	if _, exists := result[legalquery.EvidenceOfficialIdentifier]; exists {
		if !preserveOfficialAlias {
			delete(result, legalquery.EvidenceOfficialAlias)
		}
		if !preserveLegalConcept {
			delete(result, legalquery.EvidenceLegalConcept)
		}
		delete(result, legalquery.EvidenceMorphologicalContext)
		delete(result, legalquery.EvidenceUniqueTypoCorrection)
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceOfficialAlias]; exists {
		delete(result, legalquery.EvidenceMorphologicalContext)
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceLegalConcept]; exists {
		if !preserveMorphologicalContext {
			delete(result, legalquery.EvidenceMorphologicalContext)
		}
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceMorphologicalContext]; exists {
		delete(result, legalquery.EvidenceGeneralTerm)
	}
	ordered := make([]legalquery.EvidenceCode, 0, len(result))
	for _, code := range evidenceOrder {
		if _, exists := result[code]; exists {
			ordered = append(ordered, code)
		}
	}
	return ordered
}

func preservesLegalConceptForDistinctStep(draft candidateDraft) bool {
	if len(draft.concepts) == 0 || len(draft.steps) < 2 {
		return false
	}
	for _, step := range draft.steps {
		if step.input.InputKind() ==
			legalquery.InputKindLawContentSearch {
			return true
		}
	}
	return false
}

var evidenceOrder = []legalquery.EvidenceCode{
	legalquery.EvidenceOfficialIdentifier,
	legalquery.EvidenceStructuredReference,
	legalquery.EvidenceExplicitTask,
	legalquery.EvidenceExplicitResource,
	legalquery.EvidenceOfficialAlias,
	legalquery.EvidenceLegalConcept,
	legalquery.EvidenceMorphologicalContext,
	legalquery.EvidenceUniqueTypoCorrection,
	legalquery.EvidenceGeneralTerm,
}

func evidenceSignature(values []legalquery.EvidenceCode) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, "\x00")
}

func uniqueConceptSources(
	values []legalquery.LegalConceptSource,
) []legalquery.LegalConceptSource {
	byID := make(map[string]legalquery.LegalConceptSource, len(values))
	for _, value := range values {
		byID[value.ConceptID()] = value
	}
	ids := make([]string, 0, len(byID))
	for conceptID := range byID {
		ids = append(ids, conceptID)
	}
	slices.Sort(ids)
	result := make([]legalquery.LegalConceptSource, 0, len(ids))
	for _, conceptID := range ids {
		result = append(result, byID[conceptID])
	}
	return result
}

func sourcePosition(value candidateDraft) int {
	if len(value.steps) == 0 {
		return 0
	}
	position := value.steps[0].startByte
	for _, step := range value.steps[1:] {
		if step.startByte < position {
			position = step.startByte
		}
	}
	return position
}
