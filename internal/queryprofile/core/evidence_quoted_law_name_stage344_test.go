package core

import (
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const coreQuotedLawNameEvidenceID = "core-law-name-quoted-phrase-evidence"

func TestCoreEvidenceは法令名と同じSpanの引用句を一般検索語だけにする(
	t *testing.T,
) {
	fixture := mustCoreEvidenceFixture(
		t,
		"法令本文から「民法」を検索してください。",
		nil,
	)
	contentInput, err := legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms: []string{"民法"},
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: 本文検索入力を構築できません: %v",
			coreQuotedLawNameEvidenceID,
			err,
		)
	}
	lawNames := fixture.input.LawNameMentions()
	if len(lawNames) == 0 {
		t.Fatalf("%s: 法令名出現がありません", coreQuotedLawNameEvidenceID)
	}
	draft := newCandidateDraft()
	draft.steps = []stepDraft{{
		startByte: lawNames[0].Span().StartByte(),
		input:     contentInput,
	}}
	bound, err := withCoreEvidenceBindings(
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
	)
	if err != nil {
		t.Fatalf(
			"%s: evidence binding を構築できません: %v",
			coreQuotedLawNameEvidenceID,
			err,
		)
	}
	draft = bound[0]
	evaluation := mustCoreEvidenceEvaluation(
		t,
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
	)
	evidence := mustCoreStepEvidence(t, evaluation, 0, 0)

	var quotedSpan legalquery.QuerySpan
	var quotedSpanExists bool
	for _, value := range evidence {
		if value.Code() != legalquery.EvidenceGeneralTerm {
			continue
		}
		span, exists := value.Span()
		if !exists {
			continue
		}
		quotedSpan = span
		quotedSpanExists = true
		break
	}
	if !quotedSpanExists {
		t.Fatalf(
			"%s: 引用句の general_term 根拠がありません: %#v",
			coreQuotedLawNameEvidenceID,
			evidence,
		)
	}

	positiveCount := 0
	clusterSpanCount := 0
	for _, value := range evidence {
		if value.Layer() != profileevidence.LayerTargetAnchor {
			continue
		}
		span, exists := value.Span()
		if !exists || span != quotedSpan {
			continue
		}
		if value.IndependentPositive() {
			positiveCount++
		}
		if value.ClusterSpan() {
			clusterSpanCount++
		}
	}
	if positiveCount != 1 || clusterSpanCount != 1 {
		t.Fatalf(
			"%s: 同じ span の positive=%d clusterSpan=%d、期待値はいずれも1: %#v",
			coreQuotedLawNameEvidenceID,
			positiveCount,
			clusterSpanCount,
			evidence,
		)
	}

	profile := mustCoreEvidenceProfile(t)
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf(
			"%s: candidate scope を構築できません: %v",
			coreQuotedLawNameEvidenceID,
			err,
		)
	}
	candidates, _, _, err := profile.materializeCoreEvidenceCandidates(
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
		scope,
	)
	if err != nil {
		t.Fatalf(
			"%s: candidate を構築できません: %v",
			coreQuotedLawNameEvidenceID,
			err,
		)
	}
	if len(candidates) != 1 {
		t.Fatalf(
			"%s: candidates=%d、期待値は1",
			coreQuotedLawNameEvidenceID,
			len(candidates),
		)
	}
	finalEvidence := candidates[0].EvidenceCodes()
	if !slices.Contains(finalEvidence, legalquery.EvidenceGeneralTerm) ||
		slices.Contains(finalEvidence, legalquery.EvidenceOfficialAlias) {
		t.Fatalf(
			"%s: final evidence=%v、general_term だけを検索対象根拠にする必要があります",
			coreQuotedLawNameEvidenceID,
			finalEvidence,
		)
	}
}
