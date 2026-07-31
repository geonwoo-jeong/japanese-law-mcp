package judicialcases

import (
	"fmt"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type judicialEvidenceEvaluation struct {
	mapping profileevidence.Mapping
	drafts  []judicialEvidenceDraftRef
	facts   judicialEvidenceFactSet
}

type judicialEvidenceDraftRef struct {
	draftID    string
	stepIDs    []string
	draftIndex int
}

func buildJudicialEvidenceEvaluation(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
) (judicialEvidenceEvaluation, error) {
	if err := input.Validate(); err != nil {
		return judicialEvidenceEvaluation{}, fmt.Errorf(
			"judicial evidence input が有効ではありません: %w",
			err,
		)
	}
	if err := validateJudicialEvidenceDraftInputs(input, drafts); err != nil {
		return judicialEvidenceEvaluation{}, err
	}
	bound, err := withJudicialEvidenceBindings(input, cues, drafts)
	if err != nil {
		return judicialEvidenceEvaluation{}, fmt.Errorf(
			"judicial evidence binding を構築できません: %w",
			err,
		)
	}
	facts, err := buildJudicialEvidenceFacts(input, cues)
	if err != nil {
		return judicialEvidenceEvaluation{}, err
	}
	values := profileevidence.MappingValues{
		ProfileID: profileID,
		Facts:     facts.values,
	}
	references := make([]judicialEvidenceDraftRef, 0, len(bound))
	for index, draft := range bound {
		value, reference, retained, buildErr :=
			buildJudicialEvidenceDraftValue(index, draft, facts, cues)
		if buildErr != nil {
			return judicialEvidenceEvaluation{}, buildErr
		}
		if !retained {
			continue
		}
		values.Drafts = append(values.Drafts, value)
		references = append(references, reference)
	}
	mapping, err := profileevidence.NewMapping(values)
	if err != nil {
		return judicialEvidenceEvaluation{}, fmt.Errorf(
			"judicial evidence mapping を構築できません: %w",
			err,
		)
	}
	return judicialEvidenceEvaluation{
		mapping: mapping,
		drafts:  references,
		facts:   facts,
	}, nil
}

func validateJudicialEvidenceDraftInputs(
	input legalquery.CandidateGenerationInput,
	drafts []candidateDraft,
) error {
	requestRef, requestHasRef := input.Ref()
	for draftIndex, draft := range drafts {
		if len(draft.steps) < 1 ||
			len(draft.steps) > legalquery.MaxCapabilityCalls {
			return fmt.Errorf(
				"drafts[%d] の step 件数が有効ではありません",
				draftIndex,
			)
		}
		for stepIndex, step := range draft.steps {
			if step.input == nil {
				return fmt.Errorf(
					"drafts[%d].steps[%d] の logical input は必須です",
					draftIndex,
					stepIndex,
				)
			}
			switch value := step.input.(type) {
			case legalquery.JudicialDecisionSearchIntentV1:
				if err := value.Validate(); err != nil {
					return fmt.Errorf(
						"drafts[%d].steps[%d]: %w",
						draftIndex,
						stepIndex,
						err,
					)
				}
			case legalquery.JudicialDecisionReadIntentV1:
				ref := value.Ref()
				_, versionExists := ref.Key().VersionID()
				if !requestHasRef || ref != requestRef ||
					ref.Key().ResourceType() != "judicial-decision" ||
					versionExists {
					return fmt.Errorf(
						"drafts[%d].steps[%d] の read ref が request と一致しません",
						draftIndex,
						stepIndex,
					)
				}
			default:
				return fmt.Errorf(
					"drafts[%d].steps[%d] の input kind は未対応です",
					draftIndex,
					stepIndex,
				)
			}
		}
	}
	return nil
}

func buildJudicialEvidenceDraftValue(
	index int,
	draft candidateDraft,
	facts judicialEvidenceFactSet,
	cues resolvedCues,
) (
	profileevidence.DraftValues,
	judicialEvidenceDraftRef,
	bool,
	error,
) {
	draftID := fmt.Sprintf("draft-%d", index+1)
	value := profileevidence.DraftValues{DraftID: draftID}
	reference := judicialEvidenceDraftRef{
		draftID:    draftID,
		draftIndex: index,
	}
	bindings := removeAmbiguousJudicialBindings(draft, facts, cues)
	for stepIndex, step := range draft.steps {
		signature, err := judicialLogicalInputSignature(step.input)
		if err != nil {
			return profileevidence.DraftValues{},
				judicialEvidenceDraftRef{}, false, err
		}
		stepID := fmt.Sprintf("step-%d", stepIndex+1)
		value.Steps = append(value.Steps, profileevidence.StepValues{
			StepID:               stepID,
			SourceOrdinal:        stepIndex + 1,
			TopicOrdinal:         step.topicOrdinal,
			StepMeaningSignature: signature,
			Evidence:             bindings[stepIndex],
		})
		reference.stepIDs = append(reference.stepIDs, stepID)
	}
	if !judicialEvidenceDraftIsComplete(value, draft, facts) {
		return profileevidence.DraftValues{},
			judicialEvidenceDraftRef{}, false, nil
	}
	if _, err := profileevidence.NewMapping(profileevidence.MappingValues{
		ProfileID: profileID,
		Facts:     facts.values,
		Drafts:    []profileevidence.DraftValues{value},
	}); err != nil {
		return profileevidence.DraftValues{},
			judicialEvidenceDraftRef{}, false,
			fmt.Errorf("%s: %w", draftID, err)
	}
	return value, reference, true, nil
}

func removeAmbiguousJudicialBindings(
	draft candidateDraft,
	facts judicialEvidenceFactSet,
	cues resolvedCues,
) [][]profileevidence.EvidenceValues {
	result := make([][]profileevidence.EvidenceValues, len(draft.steps))
	stepsByFact := make(map[string]map[int]struct{})
	for stepIndex, step := range draft.steps {
		for _, value := range step.evidenceBindings {
			if stepsByFact[value.FactID] == nil {
				stepsByFact[value.FactID] = make(map[int]struct{})
			}
			stepsByFact[value.FactID][stepIndex] = struct{}{}
		}
	}
	for stepIndex, step := range draft.steps {
		for _, value := range step.evidenceBindings {
			fact := facts.byID[value.FactID]
			ambiguous := len(stepsByFact[value.FactID]) > 1
			individualCue := cues.has("operator", "individual") &&
				fact.cueCategory != ""
			if ambiguous && !individualCue {
				continue
			}
			result[stepIndex] = append(result[stepIndex], value)
		}
		sort.Slice(result[stepIndex], func(left int, right int) bool {
			if result[stepIndex][left].FactID != result[stepIndex][right].FactID {
				return result[stepIndex][left].FactID < result[stepIndex][right].FactID
			}
			return result[stepIndex][left].Code < result[stepIndex][right].Code
		})
	}
	return result
}

func judicialEvidenceDraftIsComplete(
	value profileevidence.DraftValues,
	draft candidateDraft,
	facts judicialEvidenceFactSet,
) bool {
	if len(value.Steps) != len(draft.steps) || len(value.Steps) == 0 {
		return false
	}
	for index, step := range value.Steps {
		if step.TopicOrdinal < 1 || len(step.Evidence) == 0 {
			return false
		}
		var positive bool
		var hasRef bool
		var hasTask bool
		for _, evidence := range step.Evidence {
			positive = positive || evidence.IndependentPositive
			hasTask = hasTask ||
				evidence.Code == legalquery.EvidenceExplicitTask
			fact := facts.byID[evidence.FactID]
			hasRef = hasRef || fact.inputRef
		}
		switch draft.steps[index].input.InputKind() {
		case legalquery.InputKindJudicialDecisionRead:
			if !positive || !hasRef || !hasTask {
				return false
			}
		case legalquery.InputKindJudicialDecisionSearch:
			if !positive {
				return false
			}
		default:
			return false
		}
	}
	return true
}
