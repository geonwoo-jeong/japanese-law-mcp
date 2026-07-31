package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func validateCoreDraftInputs(
	input legalquery.CandidateGenerationInput,
	drafts []candidateDraft,
) error {
	requestRef, requestHasRef := input.Ref()
	for draftIndex, draft := range drafts {
		for stepIndex, step := range draft.steps {
			if step.input == nil {
				return fmt.Errorf(
					"drafts[%d].steps[%d].logical input は必須です",
					draftIndex,
					stepIndex,
				)
			}
			var logicalRef *model.SourceResourceRef
			switch value := step.input.(type) {
			case legalquery.LawReadIntentV1:
				if ref, exists := value.Ref(); exists {
					logicalRef = &ref
				}
			case legalquery.LawArticleReadIntentV1:
				if ref, exists := value.Ref(); exists {
					logicalRef = &ref
				}
			}
			if logicalRef != nil &&
				(!requestHasRef || *logicalRef != requestRef) {
				return fmt.Errorf(
					"drafts[%d].steps[%d].ref は request の ref と完全一致しません",
					draftIndex,
					stepIndex,
				)
			}
		}
	}
	return nil
}

func removeAmbiguousCoreEvidence(
	value profileevidence.DraftValues,
	draft candidateDraft,
	facts map[string]coreEvidenceFact,
) (profileevidence.DraftValues, bool) {
	uses := make(map[string]map[int]struct{})
	for stepIndex, step := range value.Steps {
		for _, evidence := range step.Evidence {
			if _, registered := facts[evidence.FactID]; !registered {
				continue
			}
			if uses[evidence.FactID] == nil {
				uses[evidence.FactID] = make(map[int]struct{})
			}
			uses[evidence.FactID][stepIndex] = struct{}{}
		}
	}
	ambiguous := make(map[string]bool)
	for factID, stepIndexes := range uses {
		if len(stepIndexes) < 2 {
			continue
		}
		if coreSharedTerminalExplicitTaskFactReusable(factID, value, draft) {
			continue
		}
		if coreSharedTerminalExplicitResourceRepeated(factID, value, draft) {
			return profileevidence.DraftValues{}, false
		}
		ambiguous[factID] = true
	}
	for index := range value.Steps {
		filtered := make([]profileevidence.EvidenceValues, 0, len(value.Steps[index].Evidence))
		for _, evidence := range value.Steps[index].Evidence {
			if !ambiguous[evidence.FactID] {
				filtered = append(filtered, evidence)
			}
		}
		value.Steps[index].Evidence = filtered
	}
	return value, true
}

func coreSharedTerminalExplicitTaskFactReusable(
	factID string,
	value profileevidence.DraftValues,
	draft candidateDraft,
) bool {
	if !draft.sharedTerminal ||
		len(value.Steps) != len(draft.steps) ||
		len(value.Steps) < 2 {
		return false
	}
	usedSteps := 0
	for stepIndex, step := range value.Steps {
		if draft.steps[stepIndex].input == nil ||
			draft.steps[stepIndex].input.InputKind() !=
				legalquery.InputKindLawContentSearch {
			return false
		}
		usedInStep := false
		for _, evidence := range step.Evidence {
			if evidence.FactID != factID {
				continue
			}
			usedInStep = true
			if evidence.Layer != profileevidence.LayerExplicitTaskResource ||
				evidence.Code != legalquery.EvidenceExplicitTask {
				return false
			}
		}
		if usedInStep {
			usedSteps++
		}
	}
	return usedSteps == len(value.Steps)
}

func coreSharedTerminalExplicitResourceRepeated(
	factID string,
	value profileevidence.DraftValues,
	draft candidateDraft,
) bool {
	if !draft.sharedTerminal {
		return false
	}
	usedSteps := 0
	for _, step := range value.Steps {
		usedInStep := false
		for _, evidence := range step.Evidence {
			if evidence.FactID != factID ||
				evidence.Layer != profileevidence.LayerExplicitTaskResource ||
				evidence.Code != legalquery.EvidenceExplicitResource {
				continue
			}
			usedInStep = true
		}
		if usedInStep {
			usedSteps++
		}
	}
	return usedSteps > 1
}
