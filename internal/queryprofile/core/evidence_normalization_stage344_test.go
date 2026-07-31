package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func TestCoreSharedTerminalExplicitEvidenceは各Stepで保持する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	generation := generateCoreEvidenceQuery(
		t,
		profile,
		"永住許可、帰化を教えてください",
		nil,
		coreMultiStepEvidenceNormalizationID,
	)
	candidates := generation.Candidates()
	if len(candidates) != 1 ||
		!slices.Contains(
			candidates[0].EvidenceCodes(),
			legalquery.EvidenceExplicitTask,
		) {
		t.Fatalf(
			"%s: 共有末尾の明示 task 根拠が失われました: %#v",
			coreMultiStepEvidenceNormalizationID,
			candidates,
		)
	}
}

func TestCoreNormalizedConceptSourcesは保持したFactだけを公開する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	conceptIDs := make([]string, 0, len(profile.concepts))
	for conceptID := range profile.concepts {
		conceptIDs = append(conceptIDs, conceptID)
	}
	slices.Sort(conceptIDs)
	if len(conceptIDs) < 2 {
		t.Fatalf(
			"%s: concept source fixture が不足しています",
			coreMultiStepEvidenceNormalizationID,
		)
	}
	removedID := conceptIDs[0]
	retainedID := conceptIDs[1]
	identifierSpan := mustStage344NormalizationSpan(t, 0, 3)
	removedSpan := mustStage344NormalizationSpan(t, 4, 7)
	retainedSpan := mustStage344NormalizationSpan(t, 8, 11)
	mapping, err := profileevidence.NewMapping(profileevidence.MappingValues{
		ProfileID: profileID,
		Facts: []profileevidence.FactValues{
			{FactID: "identifier", Span: &identifierSpan},
			{FactID: "removed-concept", Span: &removedSpan},
			{FactID: "retained-concept", Span: &retainedSpan},
		},
		Drafts: []profileevidence.DraftValues{{
			DraftID: "draft-one",
			Steps: []profileevidence.StepValues{
				{
					StepID:               "step-one",
					SourceOrdinal:        1,
					TopicOrdinal:         1,
					StepMeaningSignature: "law-target",
					Evidence: []profileevidence.EvidenceValues{
						{
							FactID:              "identifier",
							Layer:               profileevidence.LayerTargetAnchor,
							Code:                legalquery.EvidenceOfficialIdentifier,
							IndependentPositive: true,
							ClusterSpan:         true,
							NormalizationGroup:  "target-one",
						},
						{
							FactID:              "removed-concept",
							Layer:               profileevidence.LayerSemanticExpansion,
							Code:                legalquery.EvidenceLegalConcept,
							IndependentPositive: true,
							ClusterSpan:         true,
							NormalizationGroup:  "target-one",
						},
					},
				},
				{
					StepID:               "step-two",
					SourceOrdinal:        2,
					TopicOrdinal:         2,
					StepMeaningSignature: "content-target",
					Evidence: []profileevidence.EvidenceValues{{
						FactID:              "retained-concept",
						Layer:               profileevidence.LayerSemanticExpansion,
						Code:                legalquery.EvidenceLegalConcept,
						IndependentPositive: true,
						ClusterSpan:         true,
					}},
				},
			},
		}},
	})
	if err != nil {
		t.Fatalf(
			"%s: mapping を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	reference := coreEvidenceDraftRef{
		draftID: "draft-one",
		stepIDs: []string{"step-one", "step-two"},
	}
	summary, err := coreNormalizedEvidenceFor(
		mapping,
		reference,
		map[string]coreEvidenceFact{
			"identifier": {kind: coreEvidenceFactIdentifier},
			"removed-concept": {
				kind:      coreEvidenceFactLegalConcept,
				conceptID: removedID,
			},
			"retained-concept": {
				kind:      coreEvidenceFactLegalConcept,
				conceptID: retainedID,
			},
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: 正規化済み根拠を集約できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if !slices.Equal(summary.codes, []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceLegalConcept,
	}) {
		t.Fatalf(
			"%s: 正規化済み code = %v",
			coreMultiStepEvidenceNormalizationID,
			summary.codes,
		)
	}
	sources, err := coreConceptSourcesForEvidence(
		[]legalquery.LegalConceptSource{
			profile.concepts[removedID].source,
			profile.concepts[retainedID].source,
		},
		summary.conceptIDs,
	)
	if err != nil {
		t.Fatalf(
			"%s: conceptSources を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if len(sources) != 1 || sources[0].ConceptID() != retainedID {
		t.Fatalf(
			"%s: 除去済み fact の source が残りました: %#v",
			coreMultiStepEvidenceNormalizationID,
			sources,
		)
	}
}

func TestCoreNormalizedConceptSourcesは同じConceptIDのTuple競合を拒否する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	conceptIDs := make([]string, 0, len(profile.concepts))
	for conceptID := range profile.concepts {
		conceptIDs = append(conceptIDs, conceptID)
	}
	slices.Sort(conceptIDs)
	if len(conceptIDs) == 0 {
		t.Fatalf(
			"%s: concept source fixture がありません",
			coreMultiStepEvidenceNormalizationID,
		)
	}
	base := profile.concepts[conceptIDs[0]].source
	conflict, err := legalquery.NewLegalConceptSource(
		legalquery.LegalConceptSourceValues{
			ConceptID:   base.ConceptID(),
			Title:       base.Title() + "（競合）",
			URL:         base.URL(),
			ConfirmedOn: base.ConfirmedOn(),
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: 競合 source fixture を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if _, err := coreConceptSourcesForEvidence(
		[]legalquery.LegalConceptSource{base, conflict},
		map[string]struct{}{base.ConceptID(): {}},
	); err == nil {
		t.Fatalf(
			"%s: 同じ conceptId の source tuple 競合を受理しました",
			coreMultiStepEvidenceNormalizationID,
		)
	}
}

func mustStage344NormalizationSpan(
	t *testing.T,
	startByte int,
	endByte int,
) legalquery.QuerySpan {
	t.Helper()
	span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf(
			"%s: span を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	return span
}
