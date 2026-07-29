package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func isWeakLawResourceAmbiguity(
	cues resolvedCues,
	hasLawTargets bool,
	conceptCandidateCount int,
	termCount int,
) bool {
	if hasLawTargets ||
		conceptCandidateCount != 0 ||
		!cues.has("task", "search") ||
		cues.has("resource", "law_provision") {
		return false
	}
	if cues.has("resource", "law") {
		if !weakGeneralResourceIsImplicit(cues) {
			return true
		}
	}
	return termCount == 1 &&
		!hasUnsupportedWeakGeneralExpansion(cues) &&
		!cues.has("operator", "dual_candidate") &&
		!cues.has("resource", "updates") &&
		!cues.has("reserved_pack", "judicial-cases")
}

func hasUnsupportedWeakGeneralExpansion(cues resolvedCues) bool {
	return cues.has("unsupported", "legal_advice") ||
		cues.has("unsupported", "translation") ||
		cues.has("unsupported", "task_or_resource")
}

func buildWeakGeneralDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	terms []legalquery.QueryTermMention,
) ([]candidateDraft, error) {
	if len(terms) != 1 {
		return nil, nil
	}
	term := terms[0]
	asOf := selectedAsOfDate(input, cues, false)
	searchInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: term.Surface(),
			AsOf:  asOf,
		},
	)
	if err != nil {
		return nil, fmt.Errorf(
			"一般検索語から法令名検索条件を構築できません: %w",
			err,
		)
	}
	contentInput, err := newContentInput(
		[]string{term.Surface()},
		nil,
		nil,
		asOf,
	)
	if err != nil {
		return nil, err
	}

	if weakGeneralResourceIsImplicit(cues) {
		return buildImplicitResourceWeakGeneralDrafts(
			term,
			searchInput,
			contentInput,
		), nil
	}
	return buildExplicitResourceWeakGeneralDrafts(
		term,
		searchInput,
		contentInput,
	), nil
}

func weakGeneralResourceIsImplicit(cues resolvedCues) bool {
	resources := cues.mentions[cueMeaningKey("resource", "law")]
	if len(resources) == 0 {
		return true
	}
	scopes := cues.mentions[cueMeaningKey("syntax", "related_law_scope")]
	if len(scopes) == 0 {
		return false
	}
	for _, resource := range resources {
		covered := false
		for _, scope := range scopes {
			if scope.Span().StartByte() <= resource.Span().StartByte() &&
				resource.Span().EndByte() <= scope.Span().EndByte() {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func buildExplicitResourceWeakGeneralDrafts(
	term legalquery.QueryTermMention,
	searchInput legalquery.LawSearchIntentV1,
	contentInput legalquery.LawContentSearchIntentV1,
) []candidateDraft {
	lawDraft := newCandidateDraft()
	lawDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	lawDraft.evidence[legalquery.EvidenceExplicitResource] = struct{}{}
	lawDraft.evidence[legalquery.EvidenceMorphologicalContext] = struct{}{}
	lawDraft.steps = append(lawDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     searchInput,
	})

	contentDraft := newCandidateDraft()
	contentDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	contentDraft.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	contentDraft.steps = append(contentDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     contentInput,
	})
	return []candidateDraft{lawDraft, contentDraft}
}

func buildImplicitResourceWeakGeneralDrafts(
	term legalquery.QueryTermMention,
	searchInput legalquery.LawSearchIntentV1,
	contentInput legalquery.LawContentSearchIntentV1,
) []candidateDraft {
	contentDraft := newCandidateDraft()
	contentDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	contentDraft.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	contentDraft.implicitResourceWeakGeneral = true
	contentDraft.steps = append(contentDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     contentInput,
	})

	lawDraft := newCandidateDraft()
	lawDraft.evidence[legalquery.EvidenceExplicitTask] = struct{}{}
	lawDraft.evidence[legalquery.EvidenceGeneralTerm] = struct{}{}
	lawDraft.implicitResourceWeakGeneral = true
	lawDraft.steps = append(lawDraft.steps, stepDraft{
		startByte: term.Span().StartByte(),
		input:     searchInput,
	})
	return []candidateDraft{contentDraft, lawDraft}
}

func weakGeneralRankingSignature(
	draft candidateDraft,
	signature string,
) string {
	if !draft.implicitResourceWeakGeneral || len(draft.steps) != 1 {
		return signature
	}
	switch draft.steps[0].input.InputKind() {
	case legalquery.InputKindLawContentSearch:
		return "01-implicit-resource-content|" + signature
	case legalquery.InputKindLawSearch:
		return "02-implicit-resource-law|" + signature
	default:
		return signature
	}
}
