package legalquery

import "fmt"

// CandidateAssemblyValues は、logical input から候補を組み立てる値である。
type CandidateAssemblyValues struct {
	IDScope          CandidateIDScope
	CandidateOrdinal int
	SemanticScore    int
	Confidence       Confidence
	EvidenceCodes    []EvidenceCode
	ConceptSources   []LegalConceptSource
	RequiredPacks    []string
	LogicalInputs    []LogicalInput
}

// AssembleLegalQueryCandidate は、能力対応と ID を一元的に導出する。
func AssembleLegalQueryCandidate(
	values CandidateAssemblyValues,
) (LegalQueryCandidate, error) {
	candidateID, err := values.IDScope.CandidateID(
		values.CandidateOrdinal,
	)
	if err != nil {
		return LegalQueryCandidate{}, err
	}
	steps := make([]LegalQueryCandidateStep, 0, len(values.LogicalInputs))
	for index, logicalInput := range values.LogicalInputs {
		if logicalInput == nil {
			return LegalQueryCandidate{}, fmt.Errorf(
				"logicalInputs[%d] は必須です",
				index,
			)
		}
		specification, exists := stepSpecificationFor(logicalInput.InputKind())
		if !exists {
			return LegalQueryCandidate{}, fmt.Errorf(
				"logicalInputs[%d] の inputKind が定義されていません",
				index,
			)
		}
		stepID, stepIDErr := values.IDScope.StepID(
			values.CandidateOrdinal,
			index+1,
		)
		if stepIDErr != nil {
			return LegalQueryCandidate{}, stepIDErr
		}
		step, stepErr := NewLegalQueryCandidateStep(
			LegalQueryCandidateStepValues{
				StepID:                 stepID,
				Task:                   specification.task,
				Resource:               specification.resource,
				CapabilityID:           specification.capabilityID,
				CapabilityMajorVersion: specification.majorVersion,
				InputKind:              logicalInput.InputKind(),
				LogicalInput:           logicalInput,
			},
		)
		if stepErr != nil {
			return LegalQueryCandidate{}, fmt.Errorf(
				"logicalInputs[%d] から step を作成できません: %w",
				index,
				stepErr,
			)
		}
		steps = append(steps, step)
	}
	return NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    candidateID,
		SemanticScore:  values.SemanticScore,
		Confidence:     values.Confidence,
		EvidenceCodes:  append([]EvidenceCode(nil), values.EvidenceCodes...),
		ConceptSources: append([]LegalConceptSource(nil), values.ConceptSources...),
		RequiredPacks:  append([]string(nil), values.RequiredPacks...),
		Steps:          steps,
	})
}
