package core

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

func TestCoreNormalizationGroupは複数本文検索語へのConcept束縛を拒否する(
	t *testing.T,
) {
	input := mustStage344ContentNormalizationInput(
		t,
		[]string{"永住許可"},
		[]string{"帰化"},
		nil,
	)
	value, draft, facts := stage344ContentNormalizationFixture(
		t,
		input,
		[]stage344NormalizationFact{{
			id:      "legal-concept-1",
			kind:    coreEvidenceFactLegalConcept,
			surface: "永住許可",
			code:    legalquery.EvidenceLegalConcept,
		}},
	)

	got, err := withCoreNormalizationGroups(value, draft, facts)
	if err == nil {
		t.Fatalf(
			"%s: 複数 target へ束縛できる法概念 fact を受理しました: %#v",
			coreMultiStepEvidenceNormalizationID,
			got,
		)
	}
	if got.DraftID != "" || len(got.Steps) != 0 {
		t.Fatalf(
			"%s: 拒否時に部分 draft が残りました: %#v",
			coreMultiStepEvidenceNormalizationID,
			got,
		)
	}
	if !strings.Contains(err.Error(), "一意に正規化できません") {
		t.Fatalf(
			"%s: エラーが曖昧束縛を識別できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
}

func TestCoreNormalizationGroupは一意なConcept束縛を保持する(
	t *testing.T,
) {
	input := mustStage344ContentNormalizationInput(
		t,
		[]string{"永住許可"},
		nil,
		nil,
	)
	value, draft, facts := stage344ContentNormalizationFixture(
		t,
		input,
		[]stage344NormalizationFact{
			{
				id:      "legal-concept-1",
				kind:    coreEvidenceFactLegalConcept,
				surface: "永住許可",
				code:    legalquery.EvidenceLegalConcept,
			},
			{
				id:      "query-term-1",
				kind:    coreEvidenceFactQueryTerm,
				surface: "永住許可",
				code:    legalquery.EvidenceMorphologicalContext,
			},
		},
	)

	got, err := withCoreNormalizationGroups(value, draft, facts)
	if err != nil {
		t.Fatalf(
			"%s: 一意な法概念束縛を拒否しました: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	evidence := got.Steps[0].Evidence
	if len(evidence) != 2 ||
		evidence[0].NormalizationGroup == "" ||
		evidence[0].NormalizationGroup != evidence[1].NormalizationGroup {
		t.Fatalf(
			"%s: 一意な同値 fact の group = %#v",
			coreMultiStepEvidenceNormalizationID,
			evidence,
		)
	}
}

func TestCoreNormalizationGroupは複数語でも一意な非Concept束縛を保つ(
	t *testing.T,
) {
	input := mustStage344ContentNormalizationInput(
		t,
		[]string{"永住許可"},
		[]string{"帰化"},
		nil,
	)
	value, draft, facts := stage344ContentNormalizationFixture(
		t,
		input,
		[]stage344NormalizationFact{
			{
				id:      "query-term-1",
				kind:    coreEvidenceFactQueryTerm,
				surface: "永住許可",
				code:    legalquery.EvidenceMorphologicalContext,
			},
			{
				id:      "query-term-2",
				kind:    coreEvidenceFactQueryTerm,
				surface: "永住許可",
				code:    legalquery.EvidenceGeneralTerm,
			},
		},
	)

	got, err := withCoreNormalizationGroups(value, draft, facts)
	if err != nil {
		t.Fatalf(
			"%s: 一意な非法概念束縛を拒否しました: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	evidence := got.Steps[0].Evidence
	if len(evidence) != 2 ||
		evidence[0].NormalizationGroup == "" ||
		evidence[0].NormalizationGroup != evidence[1].NormalizationGroup {
		t.Fatalf(
			"%s: 一意な非法概念 fact の group = %#v",
			coreMultiStepEvidenceNormalizationID,
			evidence,
		)
	}
}

type stage344NormalizationFact struct {
	id      string
	kind    coreEvidenceFactKind
	surface string
	code    legalquery.EvidenceCode
}

func mustStage344ContentNormalizationInput(
	t *testing.T,
	allTerms []string,
	anyTerms []string,
	excludeTerms []string,
) legalquery.LawContentSearchIntentV1 {
	t.Helper()
	input, err := legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     allTerms,
			AnyTerms:     anyTerms,
			ExcludeTerms: excludeTerms,
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: 本文検索 input を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	return input
}

func stage344ContentNormalizationFixture(
	t *testing.T,
	input legalquery.LawContentSearchIntentV1,
	definitions []stage344NormalizationFact,
) (
	profileevidence.DraftValues,
	candidateDraft,
	map[string]coreEvidenceFact,
) {
	t.Helper()
	signature, err := logicalInputSignature(input)
	if err != nil {
		t.Fatalf(
			"%s: logical input 署名を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	step := profileevidence.StepValues{
		StepID:               "step-1",
		SourceOrdinal:        1,
		TopicOrdinal:         1,
		StepMeaningSignature: signature,
	}
	facts := make(map[string]coreEvidenceFact, len(definitions))
	for index, definition := range definitions {
		span := mustStage344NormalizationSpan(t, index*4, index*4+3)
		values := profileevidence.FactValues{
			FactID: definition.id,
			Span:   &span,
		}
		facts[definition.id] = coreEvidenceFact{
			values:  values,
			kind:    definition.kind,
			surface: definition.surface,
		}
		step.Evidence = append(step.Evidence, profileevidence.EvidenceValues{
			FactID:              definition.id,
			Layer:               profileevidence.LayerSemanticExpansion,
			Code:                definition.code,
			IndependentPositive: true,
			ClusterSpan:         true,
		})
	}
	return profileevidence.DraftValues{
			DraftID: "draft-1",
			Steps:   []profileevidence.StepValues{step},
		}, candidateDraft{
			steps: []stepDraft{{input: input}},
		}, facts
}
