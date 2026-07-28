package core

import (
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type lawTarget struct {
	startByte  int
	endByte    int
	lawID      string
	revisionID string
	searchTerm string
	viaRef     *model.SourceResourceRef
	evidence   legalquery.EvidenceCode
	typo       bool
}

func buildLawTargets(
	input legalquery.CandidateGenerationInput,
) []lawTarget {
	targets := make([]lawTarget, 0)
	if ref, exists := input.Ref(); exists && ref.Key().ResourceType() == "law" {
		return []lawTarget{{
			viaRef:   &ref,
			evidence: legalquery.EvidenceOfficialIdentifier,
		}}
	}
	for _, mention := range input.IdentifierMentions() {
		revisionID, _ := mention.RevisionID()
		evidence := legalquery.EvidenceOfficialIdentifier
		searchTerm := mention.Surface()
		if mention.Kind() == legalquery.IdentifierMentionLawNumber {
			evidence = legalquery.EvidenceStructuredReference
			if lawNumber, exists := mention.LawNumber(); exists {
				searchTerm = lawNumber
			}
		}
		targets = append(targets, lawTarget{
			startByte:  mention.Span().StartByte(),
			endByte:    mention.Span().EndByte(),
			lawID:      mention.LawID(),
			revisionID: revisionID,
			searchTerm: searchTerm,
			evidence:   evidence,
		})
	}
	for _, mention := range input.LawNameMentions() {
		if shouldIgnoreTypoLawMention(mention) {
			continue
		}
		targets = append(targets, lawTarget{
			startByte:  mention.Span().StartByte(),
			endByte:    mention.Span().EndByte(),
			lawID:      mention.LawID(),
			searchTerm: searchTermForLawMention(mention),
			evidence:   legalquery.EvidenceOfficialAlias,
			typo:       mention.MatchKind() == legalquery.PreprocessMatchUniqueTypoCorrection,
		})
	}
	sort.SliceStable(targets, func(left, right int) bool {
		if targets[left].startByte != targets[right].startByte {
			return targets[left].startByte < targets[right].startByte
		}
		if targets[left].endByte != targets[right].endByte {
			return targets[left].endByte > targets[right].endByte
		}
		if targets[left].lawID != targets[right].lawID {
			return targets[left].lawID < targets[right].lawID
		}
		return evidenceIndex(targets[left].evidence) <
			evidenceIndex(targets[right].evidence)
	})
	return dedupeLawTargets(targets)
}

func dedupeLawTargets(values []lawTarget) []lawTarget {
	result := make([]lawTarget, 0, len(values))
	for _, value := range values {
		duplicate := false
		for _, existing := range result {
			if existing.startByte == value.startByte &&
				existing.endByte == value.endByte &&
				existing.lawID == value.lawID {
				duplicate = true
				break
			}
		}
		if !duplicate {
			result = append(result, value)
		}
	}
	return result
}

func buildLawSearchCandidates(
	input legalquery.CandidateGenerationInput,
	targets []lawTarget,
	searchRequested bool,
	explicitTask bool,
	documentRequested bool,
	individual bool,
	asOf *model.Date,
) ([]candidateDraft, error) {
	if !searchRequested ||
		documentRequested ||
		len(input.ArticleMentions()) > 0 {
		return nil, nil
	}
	groups := groupLawTargets(targets)
	if individual && len(groups) > 4 {
		return nil, nil
	}
	if individual && len(groups) > 1 && allGroupsUnique(groups) {
		draft := newCandidateDraft()
		if explicitTask {
			draft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
		}
		draft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
		for _, group := range groups {
			target := group[0]
			if target.viaRef != nil || target.searchTerm == "" {
				return nil, nil
			}
			searchInput, err := legalquery.NewLawSearchIntentV1(
				legalquery.LawSearchIntentV1Values{
					Query: target.searchTerm,
					AsOf:  asOf,
				},
			)
			if err != nil {
				return nil, err
			}
			addTargetEvidence(&draft, target, asOf != nil)
			draft.steps = append(draft.steps, stepDraft{
				startByte: target.startByte,
				input:     searchInput,
			})
		}
		return []candidateDraft{draft}, nil
	}

	result := make([]candidateDraft, 0, len(targets))
	for _, group := range groups {
		for _, target := range group {
			if target.viaRef != nil || target.searchTerm == "" {
				continue
			}
			searchInput, err := legalquery.NewLawSearchIntentV1(
				legalquery.LawSearchIntentV1Values{
					Query: target.searchTerm,
					AsOf:  asOf,
				},
			)
			if err != nil {
				return nil, err
			}
			draft := newCandidateDraft()
			if explicitTask {
				draft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
			}
			draft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
			addTargetEvidence(&draft, target, asOf != nil)
			draft.steps = append(draft.steps, stepDraft{
				startByte: target.startByte,
				input:     searchInput,
			})
			result = append(result, draft)
		}
	}
	return result, nil
}

func buildReadCandidates(
	input legalquery.CandidateGenerationInput,
	targets []lawTarget,
	readRequested bool,
	explicitTask bool,
	documentRequested bool,
	asOf *model.Date,
) ([]candidateDraft, error) {
	if !readRequested || len(targets) == 0 {
		return nil, nil
	}
	groups := groupLawTargets(targets)
	result := make([]candidateDraft, 0, len(targets))
	for groupIndex, group := range groups {
		locations, err := articleLocationsForTargetGroup(
			input,
			groups,
			groupIndex,
		)
		if err != nil {
			return nil, err
		}
		if len(locations) == 0 && !documentRequested {
			continue
		}
		for _, target := range group {
			draft := newCandidateDraft()
			addTargetEvidence(&draft, target, asOf != nil)
			if explicitTask {
				draft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
			}
			if documentRequested || len(locations) > 0 {
				draft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
			}
			if documentRequested {
				readInput, err := buildLawReadInput(target, asOf)
				if err != nil {
					return nil, err
				}
				draft.steps = append(draft.steps, stepDraft{
					startByte: target.startByte,
					input:     readInput,
				})
			}
			for _, location := range locations {
				if target.revisionID != "" {
					continue
				}
				articleInput, err := buildLawArticleInput(target, location, asOf)
				if err != nil {
					return nil, err
				}
				draft.evidence[legalquery.EvidenceStructuredReference] = struct{}{}
				draft.steps = append(draft.steps, stepDraft{
					startByte: location.startByte,
					input:     articleInput,
				})
			}
			if len(draft.steps) > 0 {
				result = append(result, draft)
			}
		}
	}
	return result, nil
}

func buildUpdateCandidate(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (*candidateDraft, error) {
	if !cues.has("task", "list_updates") &&
		!cues.has("resource", "updates") {
		return nil, nil
	}
	dates := input.DateMentions()
	if len(dates) == 0 {
		return nil, nil
	}
	updateInput, err := legalquery.NewLawUpdateListIntentV1(
		legalquery.LawUpdateListIntentV1Values{
			Date: dates[0].Date(),
		},
	)
	if err != nil {
		return nil, err
	}
	draft := newCandidateDraft()
	draft.evidence[legalquery.EvidenceStructuredReference] = struct{}{}
	draft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	draft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
	draft.steps = append(draft.steps, stepDraft{
		startByte: dates[0].Span().StartByte(),
		input:     updateInput,
	})
	return &draft, nil
}

func addTargetEvidence(
	draft *candidateDraft,
	target lawTarget,
	hasAsOf bool,
) {
	draft.evidence[target.evidence] = struct{}{}
	if target.typo {
		draft.evidence[legalquery.EvidenceUniqueTypoCorrection] = struct{}{}
	}
	if hasAsOf {
		draft.evidence[legalquery.EvidenceStructuredReference] = struct{}{}
	}
}

func newCandidateDraft() candidateDraft {
	return candidateDraft{
		evidence: make(map[legalquery.EvidenceCode]struct{}),
	}
}

func groupLawTargets(values []lawTarget) [][]lawTarget {
	result := make([][]lawTarget, 0)
	for _, value := range values {
		if len(result) == 0 ||
			!sameLawTargetGroup(result[len(result)-1][0], value) {
			result = append(result, []lawTarget{value})
			continue
		}
		result[len(result)-1] = append(result[len(result)-1], value)
	}
	return result
}

func sameLawTargetGroup(left lawTarget, right lawTarget) bool {
	if left.viaRef != nil || right.viaRef != nil {
		return left.viaRef != nil && right.viaRef != nil
	}
	return left.startByte == right.startByte && left.endByte == right.endByte
}

func allGroupsUnique(values [][]lawTarget) bool {
	for _, value := range values {
		if len(value) != 1 {
			return false
		}
	}
	return true
}

func buildLawReadInput(
	target lawTarget,
	asOf *model.Date,
) (legalquery.LawReadIntentV1, error) {
	if target.viaRef != nil {
		return legalquery.NewLawReadIntentV1(
			legalquery.LawReadIntentV1Values{Ref: target.viaRef},
		)
	}
	if target.revisionID != "" {
		asOf = nil
	}
	return legalquery.NewLawReadIntentV1(legalquery.LawReadIntentV1Values{
		LawID:      target.lawID,
		RevisionID: target.revisionID,
		AsOf:       asOf,
	})
}

func buildLawArticleInput(
	target lawTarget,
	location articleLocation,
	asOf *model.Date,
) (legalquery.LawArticleReadIntentV1, error) {
	if target.viaRef != nil {
		return legalquery.NewLawArticleReadIntentV1(
			legalquery.LawArticleReadIntentV1Values{
				Ref:      target.viaRef,
				Location: location.location,
				AsOf:     asOf,
			},
		)
	}
	return legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    target.lawID,
			Location: location.location,
			AsOf:     asOf,
		},
	)
}

func normalizedCanonicalLawQuery(value string) string {
	open := strings.IndexRune(value, '（')
	close := strings.LastIndex(value, "）")
	if open >= 0 && close > open {
		inside := strings.TrimSpace(value[open+len("（") : close])
		if inside != "" {
			return inside
		}
	}
	return value
}

func shouldIgnoreTypoLawMention(mention legalquery.LawNameMention) bool {
	if mention.MatchKind() != legalquery.PreprocessMatchUniqueTypoCorrection {
		return false
	}
	runes := []rune(mention.Surface())
	if len(runes) == 0 {
		return false
	}
	switch runes[len(runes)-1] {
	case 'を', 'が', 'は', 'に', 'へ', 'と', 'で', 'も', 'や', 'の':
		return true
	default:
		return false
	}
}

func searchTermForLawMention(mention legalquery.LawNameMention) string {
	canonical := normalizedCanonicalLawQuery(mention.Canonical())
	surface := strings.TrimSpace(mention.Surface())
	if shouldPreferSurfaceLawQuery(surface, canonical) {
		return surface
	}
	return canonical
}

func shouldPreferSurfaceLawQuery(surface string, canonical string) bool {
	if surface == "" || canonical == "" || surface == canonical {
		return false
	}
	if strings.HasSuffix(surface, "法") &&
		strings.HasSuffix(canonical, "法") &&
		utf8.RuneCountInString(surface) > utf8.RuneCountInString(canonical) {
		return true
	}
	return false
}

func evidenceIndex(value legalquery.EvidenceCode) int {
	for index, current := range evidenceOrder {
		if current == value {
			return index
		}
	}
	return len(evidenceOrder)
}
