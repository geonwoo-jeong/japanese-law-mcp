package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestCoreSharedTerminalの明示Resource再利用はDraftを棄却する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	input, cues := stage343Input(
		t,
		profile,
		"本文で永住許可、帰化を検索してください",
	)
	built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
	if err != nil || !built.handled || len(built.drafts) == 0 {
		t.Fatalf(
			"%s: 明示 resource を持つ共有末尾 draft を構築できません: %#v, err=%v",
			coreEvidenceMappingFailClosedID,
			built,
			err,
		)
	}
	draft := built.drafts[0].draft
	if !coreStage344HasRepeatedEvidenceCode(
		draft,
		legalquery.EvidenceExplicitResource,
	) {
		t.Fatalf(
			"%s: 明示 resource の複数 step 再利用 fixture ではありません",
			coreEvidenceMappingFailClosedID,
		)
	}

	evaluation, err := buildCoreEvidenceEvaluation(
		input,
		cues,
		[]candidateDraft{draft},
	)
	if err != nil {
		t.Fatalf(
			"%s: core evidence evaluation に失敗しました: %v",
			coreEvidenceMappingFailClosedID,
			err,
		)
	}
	if len(evaluation.drafts) != 0 {
		t.Fatalf(
			"%s: 複数 step で再利用した明示 resource を持つ draft を保持しました",
			coreEvidenceMappingFailClosedID,
		)
	}
}

func TestCoreSharedTerminalの明示Task共有は保持する(t *testing.T) {
	profile := mustCoreEvidenceProfile(t)
	input, cues := stage343Input(
		t,
		profile,
		"永住許可、帰化を教えてください",
	)
	built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
	if err != nil || !built.handled || len(built.drafts) == 0 {
		t.Fatalf(
			"%s: 明示 task を持つ共有末尾 draft を構築できません: %#v, err=%v",
			coreMultiStepEvidenceNormalizationID,
			built,
			err,
		)
	}
	draft := built.drafts[0].draft
	if !coreStage344HasRepeatedEvidenceCode(
		draft,
		legalquery.EvidenceExplicitTask,
	) {
		t.Fatalf(
			"%s: 明示 task の複数 step 共有 fixture ではありません",
			coreMultiStepEvidenceNormalizationID,
		)
	}

	evaluation, err := buildCoreEvidenceEvaluation(
		input,
		cues,
		[]candidateDraft{draft},
	)
	if err != nil {
		t.Fatalf(
			"%s: core evidence evaluation に失敗しました: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if len(evaluation.drafts) != 1 {
		t.Fatalf(
			"%s: terminalTaskRelation の明示 task 共有を棄却しました",
			coreMultiStepEvidenceNormalizationID,
		)
	}
}

func coreStage344HasRepeatedEvidenceCode(
	draft candidateDraft,
	code legalquery.EvidenceCode,
) bool {
	stepsByFact := make(map[string]map[int]struct{})
	for stepIndex, step := range draft.steps {
		for _, evidence := range step.evidenceBindings {
			if evidence.Code != code {
				continue
			}
			if stepsByFact[evidence.FactID] == nil {
				stepsByFact[evidence.FactID] = make(map[int]struct{})
			}
			stepsByFact[evidence.FactID][stepIndex] = struct{}{}
		}
	}
	for _, stepIndexes := range stepsByFact {
		if len(stepIndexes) > 1 {
			return true
		}
	}
	return false
}
