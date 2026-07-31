package profileevidence

import (
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func normalizeStepEvidence(values []Evidence) []Evidence {
	present := make(map[string]map[legalquery.EvidenceCode]struct{})
	for _, value := range values {
		if value.normalizationGroup == "" {
			continue
		}
		if present[value.normalizationGroup] == nil {
			present[value.normalizationGroup] =
				make(map[legalquery.EvidenceCode]struct{})
		}
		present[value.normalizationGroup][value.code] = struct{}{}
	}

	result := make([]Evidence, 0, len(values))
	for _, value := range values {
		if value.normalizationGroup != "" &&
			evidenceCodeDominated(
				value.code,
				present[value.normalizationGroup],
			) {
			continue
		}
		result = append(result, value)
	}
	slices.SortFunc(result, compareEvidence)
	return result
}

func evidenceCodeDominated(
	code legalquery.EvidenceCode,
	present map[legalquery.EvidenceCode]struct{},
) bool {
	if _, exists := present[legalquery.EvidenceOfficialIdentifier]; exists {
		switch code {
		case legalquery.EvidenceOfficialAlias,
			legalquery.EvidenceLegalConcept,
			legalquery.EvidenceMorphologicalContext,
			legalquery.EvidenceUniqueTypoCorrection,
			legalquery.EvidenceGeneralTerm:
			return true
		}
	}
	if _, exists := present[legalquery.EvidenceOfficialAlias]; exists {
		if code == legalquery.EvidenceMorphologicalContext ||
			code == legalquery.EvidenceGeneralTerm {
			return true
		}
	}
	if _, exists := present[legalquery.EvidenceLegalConcept]; exists {
		if code == legalquery.EvidenceMorphologicalContext ||
			code == legalquery.EvidenceGeneralTerm {
			return true
		}
	}
	if _, exists := present[legalquery.EvidenceMorphologicalContext]; exists &&
		code == legalquery.EvidenceGeneralTerm {
		return true
	}
	return false
}
