package legalquery

import (
	"fmt"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

const (
	selectorTestRankingVersion = "ranking-2026-07-28-1"
	selectorTestProfileVersion = "profile-2026-07-28-1"
)

type selectorTestProfile struct {
	metadata        QueryProfileMetadata
	candidates      []LegalQueryCandidate
	signals         []CandidateGenerationSignal
	selectionMode   QuerySelectionMode
	hedgePairs      []CandidateHedgePair
	profileID       string
	profileVersion  string
	rankingVersion  string
	generate        func(CandidateIDScope) (QueryProfileContribution, error)
	generationError error
}

func (p selectorTestProfile) Metadata() QueryProfileMetadata {
	return p.metadata
}

func (selectorTestProfile) CueVocabulary() []CueVocabularyEntry {
	return nil
}

func (p selectorTestProfile) Generate(
	_ CandidateGenerationInput,
	scope CandidateIDScope,
) (QueryProfileContribution, error) {
	if p.generationError != nil {
		return QueryProfileContribution{}, p.generationError
	}
	if p.generate != nil {
		return p.generate(scope)
	}
	profileID := p.profileID
	if profileID == "" {
		profileID = p.metadata.ProfileID()
	}
	profileVersion := p.profileVersion
	if profileVersion == "" {
		profileVersion = p.metadata.ProfileVersion()
	}
	rankingVersion := p.rankingVersion
	if rankingVersion == "" {
		rankingVersion = p.metadata.RankingVersion()
	}
	return NewCandidateGeneration(QueryProfileContributionValues{
		ProfileID:      profileID,
		ProfileVersion: profileVersion,
		RankingVersion: rankingVersion,
		Candidates:     p.candidates,
		Signals:        p.signals,
		SelectionMode:  p.selectionMode,
		HedgePairs:     p.hedgePairs,
	})
}

func selectorTestMetadataValues(
	t *testing.T,
	profileID string,
	profileVersion string,
	rankingVersion string,
) QueryProfileMetadataValues {
	t.Helper()
	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task:      TaskSearch,
		Resource:  ResourceLaw,
		InputKind: InputKindLawSearch,
	})
	if err != nil {
		t.Fatalf("試験用 profile target を作成できません: %v", err)
	}
	score := mustTestQueryScorePolicy(t)
	selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold:           120,
		MinimumExecutionThreshold: 80,
		SingleMargin:              25,
		HedgeMargin:               10,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("試験用 selection policy を作成できません: %v", err)
	}
	return QueryProfileMetadataValues{
		SchemaVersion:              1,
		ProfileID:                  profileID,
		ProfileVersion:             profileVersion,
		RankingVersion:             rankingVersion,
		CueSetVersion:              profileID + "-cues-v1",
		LawNameLexiconVersion:      "law-name-v1",
		LegalConceptLexiconVersion: "legal-concept-v1",
		Targets:                    []QueryProfileTarget{target},
		Score:                      score,
		Selection:                  selection,
		TieBreak: []QueryTieBreak{
			QueryTieBreakEvidenceSet,
			QueryTieBreakStepCount,
			QueryTieBreakMeaningSignature,
			QueryTieBreakSourcePosition,
		},
	}
}

func mustSelectorTestMetadata(
	t *testing.T,
	profileID string,
	profileVersion string,
	rankingVersion string,
) QueryProfileMetadata {
	t.Helper()
	metadata, err := NewQueryProfileMetadata(selectorTestMetadataValues(
		t,
		profileID,
		profileVersion,
		rankingVersion,
	))
	if err != nil {
		t.Fatalf("試験用 profile metadata を作成できません: %v", err)
	}
	return metadata
}

func mustSelectorTestCandidate(
	t *testing.T,
	candidateID string,
	score int,
	requiredPacks []string,
	stepCount int,
) LegalQueryCandidate {
	t.Helper()
	return mustSelectorTestCandidateWithEvidence(
		t,
		candidateID,
		score,
		requiredPacks,
		stepCount,
		[]EvidenceCode{EvidenceExplicitTask},
	)
}

func mustSelectorTestCandidateWithEvidence(
	t *testing.T,
	candidateID string,
	score int,
	requiredPacks []string,
	stepCount int,
	evidenceCodes []EvidenceCode,
) LegalQueryCandidate {
	t.Helper()
	steps := make([]LegalQueryCandidateStep, 0, stepCount)
	specification, exists := stepSpecificationFor(InputKindLawSearch)
	if !exists {
		t.Fatal("試験用 law search の能力対応がありません")
	}
	for index := 0; index < stepCount; index++ {
		input, err := NewLawSearchIntentV1(LawSearchIntentV1Values{
			Query: fmt.Sprintf("試験候補%sの主題%d", candidateID, index+1),
		})
		if err != nil {
			t.Fatalf("試験用 law search input を作成できません: %v", err)
		}
		step, err := NewLegalQueryCandidateStep(
			LegalQueryCandidateStepValues{
				StepID: fmt.Sprintf(
					"step-%s-%d",
					candidateID,
					index+1,
				),
				Task:                   specification.task,
				Resource:               specification.resource,
				CapabilityID:           specification.capabilityID,
				CapabilityMajorVersion: specification.majorVersion,
				InputKind:              InputKindLawSearch,
				LogicalInput:           input,
			},
		)
		if err != nil {
			t.Fatalf("試験用 candidate step を作成できません: %v", err)
		}
		steps = append(steps, step)
	}
	confidence := ConfidenceLow
	switch {
	case score >= 130:
		confidence = ConfidenceHigh
	case score >= 80:
		confidence = ConfidenceMedium
	}
	candidate, err := NewLegalQueryCandidate(LegalQueryCandidateValues{
		CandidateID:    candidateID,
		SemanticScore:  score,
		Confidence:     confidence,
		EvidenceCodes:  append([]EvidenceCode{}, evidenceCodes...),
		ConceptSources: []LegalConceptSource{},
		RequiredPacks:  requiredPacks,
		Steps:          steps,
	})
	if err != nil {
		t.Fatalf("試験用 candidate を作成できません: %v", err)
	}
	return candidate
}

func mustSelectorTestHedgePair(
	t *testing.T,
	first LegalQueryCandidate,
	second LegalQueryCandidate,
) CandidateHedgePair {
	t.Helper()
	pair, err := NewCandidateHedgePair(CandidateHedgePairValues{
		FirstCandidateID:  first.CandidateID(),
		SecondCandidateID: second.CandidateID(),
	})
	if err != nil {
		t.Fatalf("試験用 hedge pair を作成できません: %v", err)
	}
	return pair
}

func mustSelectorTestPreprocessResult(t *testing.T) PreprocessResult {
	t.Helper()
	const query = "法情報を検索してください"
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:         query,
		ComparisonKey: querynormalization.ComparisonKey(query),
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	return result
}

func selectorTestProfileSetResult(
	t *testing.T,
	candidates []LegalQueryCandidate,
	signals []CandidateGenerationSignal,
	selectionMode QuerySelectionMode,
	hedgePairs []CandidateHedgePair,
) (QueryProfileSetResult, error) {
	t.Helper()
	profile := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			selectorTestProfileVersion,
			selectorTestRankingVersion,
		),
		candidates:    candidates,
		signals:       signals,
		selectionMode: selectionMode,
		hedgePairs:    hedgePairs,
	}
	profileSet, err := NewQueryProfileSet([]QueryProfile{profile})
	if err != nil {
		return QueryProfileSetResult{}, err
	}
	return profileSet.Collect(mustSelectorTestPreprocessResult(t))
}

func mustSelectorTestProfileSetResult(
	t *testing.T,
	candidates []LegalQueryCandidate,
	signals []CandidateGenerationSignal,
	selectionMode QuerySelectionMode,
	hedgePairs []CandidateHedgePair,
) QueryProfileSetResult {
	t.Helper()
	result, err := selectorTestProfileSetResult(
		t,
		candidates,
		signals,
		selectionMode,
		hedgePairs,
	)
	if err != nil {
		t.Fatalf("試験用 profile set result を作成できません: %v", err)
	}
	return result
}

func mustSelectorTestPackState(
	t *testing.T,
	adopted []string,
	enabled []string,
) PackState {
	t.Helper()
	state, err := NewStaticPackState(adopted, enabled)
	if err != nil {
		t.Fatalf("試験用 pack state を作成できません: %v", err)
	}
	return state
}

func mustSelectTestPlan(
	t *testing.T,
	result QueryProfileSetResult,
	packState PackState,
	limitPerAttempt int,
) LegalQueryPlan {
	t.Helper()
	plan, err := SelectLegalQueryPlan(SelectorInput{
		ProfileSetResult: result,
		PackState:        packState,
		LimitPerAttempt:  limitPerAttempt,
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-023: selector のエラー = %v", err)
	}
	return plan
}
