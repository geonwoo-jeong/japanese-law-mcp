package core

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func buildCompositionMembers(
	candidates []legalquery.LegalQueryCandidate,
	stepStartBytes [][]int,
	mode legalquery.QuerySelectionMode,
) ([]legalquery.QueryCandidateCompositionMember, error) {
	if mode != legalquery.QuerySelectionModeAutomatic {
		return nil, nil
	}
	if len(stepStartBytes) != len(candidates) {
		return nil, fmt.Errorf(
			"core composition member の候補と位置情報が一致しません",
		)
	}
	result := make(
		[]legalquery.QueryCandidateCompositionMember,
		0,
		len(candidates),
	)
	for candidateIndex, candidate := range candidates {
		steps := candidate.Steps()
		if len(stepStartBytes[candidateIndex]) != len(steps) {
			return nil, fmt.Errorf(
				"core composition member の step と位置情報が一致しません",
			)
		}
		origins := make(
			[]legalquery.QueryCandidateStepOrigin,
			0,
			len(steps),
		)
		positioned := true
		for stepIndex, step := range steps {
			startByte := stepStartBytes[candidateIndex][stepIndex]
			if startByte < 0 {
				positioned = false
				break
			}
			origin, err := legalquery.NewQueryCandidateStepOrigin(
				legalquery.QueryCandidateStepOriginValues{
					StepID:          step.StepID(),
					SourceStartByte: startByte,
				},
			)
			if err != nil {
				return nil, fmt.Errorf(
					"core composition member の step origin を作成できません: %w",
					err,
				)
			}
			origins = append(origins, origin)
		}
		if !positioned {
			continue
		}
		member, err := legalquery.NewQueryCandidateCompositionMember(
			legalquery.QueryCandidateCompositionMemberValues{
				CandidateID: candidate.CandidateID(),
				Role: legalquery.
					QueryCandidateCompositionRoleRequiredMember,
				StepOrigins: origins,
			},
		)
		if err != nil {
			return nil, fmt.Errorf(
				"core composition member を作成できません: %w",
				err,
			)
		}
		result = append(result, member)
	}
	return result, nil
}
