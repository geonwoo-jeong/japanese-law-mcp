package legalqueryplanning

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const stage43ExpectedCalibrationFingerprint = "92d53008b5804f38dae39488ea8cc5a79f80ade931d88d4a8481abd4b9858999"

func stage43ExpectedCalibrationScorecard() stage43CalibrationScorecard {
	return stage43CalibrationScorecard{
		CaseCount:               43,
		RequestErrorCount:       2,
		RequestErrorMatched:     2,
		PlanCount:               41,
		PlanOutcomeMatched:      28,
		MeaningCount:            43,
		MeaningSignatureMatched: 38,
		EvidenceApplicable:      38,
		EvidenceMatched:         11,
		ConceptApplicable:       1,
		ConceptMatched:          1,
	}
}

type stage43CalibrationArtifact struct {
	CalibrationID     string                          `json:"calibrationId"`
	CorpusVersion     string                          `json:"corpusVersion"`
	DevelopmentDigest string                          `json:"developmentDigest"`
	ProfileSetVersion string                          `json:"profileSetVersion"`
	Cases             []stage43CalibrationObservation `json:"cases"`
	Scorecard         stage43CalibrationScorecard     `json:"scorecard"`
}

type stage43CalibrationScorecard struct {
	CaseCount               int `json:"caseCount"`
	RequestErrorCount       int `json:"requestErrorCount"`
	RequestErrorMatched     int `json:"requestErrorMatched"`
	PlanCount               int `json:"planCount"`
	PlanOutcomeMatched      int `json:"planOutcomeMatched"`
	MeaningCount            int `json:"meaningCount"`
	MeaningSignatureMatched int `json:"meaningSignatureMatched"`
	EvidenceApplicable      int `json:"evidenceApplicable"`
	EvidenceMatched         int `json:"evidenceMatched"`
	ConceptApplicable       int `json:"conceptApplicable"`
	ConceptMatched          int `json:"conceptMatched"`
}

type stage43CalibrationObservation struct {
	CaseID       string                                `json:"caseId"`
	ExpectedKind legalquerycorpus.SemanticExpectedKind `json:"expectedKind"`
	RequestError *stage43RequestErrorObservation       `json:"requestError,omitempty"`
	Plan         *stage43PlanObservation               `json:"plan,omitempty"`
	Evaluation   *stage43EvaluationObservation         `json:"evaluation,omitempty"`
}

type stage43RequestErrorObservation struct {
	Code  model.ErrorCode `json:"code"`
	Field string          `json:"field"`
}

type stage43PlanObservation struct {
	ProfileVersion string                        `json:"profileVersion"`
	Decision       legalquery.PlanDecision       `json:"decision"`
	ReasonCodes    []legalquery.ReasonCode       `json:"reasonCodes"`
	Candidates     []stage43CandidateObservation `json:"candidates"`
	Selected       []stage43SelectionObservation `json:"selected"`
	Budget         stage43BudgetObservation      `json:"budget"`
}

type stage43CandidateObservation struct {
	Rank          int                       `json:"rank"`
	SemanticScore int                       `json:"semanticScore"`
	Confidence    legalquery.Confidence     `json:"confidence"`
	EvidenceCodes []legalquery.EvidenceCode `json:"evidenceCodes"`
	ConceptIDs    []string                  `json:"conceptIds"`
	RequiredPacks []string                  `json:"requiredPacks"`
	Steps         []stage43StepObservation  `json:"steps"`
}

type stage43SelectionObservation struct {
	CandidateRank int                              `json:"candidateRank"`
	Availability  legalquery.SelectionAvailability `json:"availability"`
	RequiredPacks []string                         `json:"requiredPacks"`
}

type stage43BudgetObservation struct {
	LimitPerAttempt    int  `json:"limitPerAttempt"`
	MaxCapabilityCalls int  `json:"maxCapabilityCalls"`
	MaxReturnedItems   int  `json:"maxReturnedItems"`
	FirstPageOnly      bool `json:"firstPageOnly"`
}

type stage43StepObservation struct {
	Task         legalquery.Task                `json:"task"`
	Resource     legalquery.Resource            `json:"resource"`
	InputKind    legalquery.LogicalInputKind    `json:"inputKind"`
	LogicalInput stage43LogicalInputObservation `json:"logicalInput"`
}

type stage43LogicalInputObservation struct {
	Kind         legalquery.LogicalInputKind `json:"kind"`
	Query        string                      `json:"query,omitempty"`
	AllTerms     []string                    `json:"allTerms,omitempty"`
	AnyTerms     []string                    `json:"anyTerms,omitempty"`
	ExcludeTerms []string                    `json:"excludeTerms,omitempty"`
	LawID        string                      `json:"lawId,omitempty"`
	RevisionID   string                      `json:"revisionId,omitempty"`
	AsOf         string                      `json:"asOf,omitempty"`
	Date         string                      `json:"date,omitempty"`
	Ref          *model.SourceResourceRef    `json:"ref,omitempty"`
	Location     *model.LawArticleLocation   `json:"location,omitempty"`
}

type stage43EvaluationObservation struct {
	PlanOutcomeMatched bool                        `json:"planOutcomeMatched"`
	PrimaryTop1Matched bool                        `json:"primaryTop1Matched"`
	PrimaryTop2Matched bool                        `json:"primaryTop2Matched"`
	HighConfidence     stage43AssertionObservation `json:"highConfidence"`
	Meanings           []stage43MeaningObservation `json:"meanings"`
}

type stage43MeaningObservation struct {
	MeaningID        string                      `json:"meaningId"`
	CandidateRank    int                         `json:"candidateRank"`
	SignatureMatched bool                        `json:"signatureMatched"`
	Evidence         stage43AssertionObservation `json:"evidence"`
	Concept          stage43AssertionObservation `json:"concept"`
}

type stage43AssertionObservation struct {
	Applicable bool `json:"applicable"`
	Matched    bool `json:"matched"`
}

func assertStage43CalibrationFingerprint(
	t *testing.T,
	firstDependencies Dependencies,
	firstDevelopment legalquerycorpus.DevelopmentCorpus,
	secondDependencies Dependencies,
	secondDevelopment legalquerycorpus.DevelopmentCorpus,
) {
	t.Helper()

	first := buildStage43CalibrationArtifact(
		t,
		firstDependencies,
		firstDevelopment,
	)
	second := buildStage43CalibrationArtifact(
		t,
		secondDependencies,
		secondDevelopment,
	)
	firstFingerprint := stage43CalibrationFingerprint(t, first)
	secondFingerprint := stage43CalibrationFingerprint(t, second)
	if firstFingerprint != secondFingerprint {
		t.Fatalf(
			"%s: development 校正が再現しません: %s/%s",
			nextProfileSetDevelopmentOnlyCalibrationID,
			firstFingerprint,
			secondFingerprint,
		)
	}
	if firstFingerprint != stage43ExpectedCalibrationFingerprint ||
		first.Scorecard != stage43ExpectedCalibrationScorecard() {
		t.Fatalf(
			"%s: fingerprint/scorecard = %q/%+v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			firstFingerprint,
			first.Scorecard,
		)
	}
}

func buildStage43CalibrationArtifact(
	t *testing.T,
	dependencies Dependencies,
	development legalquerycorpus.DevelopmentCorpus,
) stage43CalibrationArtifact {
	t.Helper()

	cases := development.Cases()
	artifact := stage43CalibrationArtifact{
		CalibrationID:     nextProfileSetDevelopmentOnlyCalibrationID,
		CorpusVersion:     development.CorpusVersion(),
		DevelopmentDigest: development.ContentDigest(),
		ProfileSetVersion: dependencies.Profiles().ProfileVersion(),
		Cases:             make([]stage43CalibrationObservation, 0, len(cases)),
	}
	for _, semanticCase := range cases {
		observation := observeStage43CalibrationCase(
			t,
			dependencies,
			semanticCase,
			&artifact.Scorecard,
		)
		artifact.Cases = append(artifact.Cases, observation)
	}
	artifact.Scorecard.CaseCount = len(cases)
	return artifact
}

func observeStage43CalibrationCase(
	t *testing.T,
	dependencies Dependencies,
	semanticCase legalquerycorpus.SemanticCase,
	scorecard *stage43CalibrationScorecard,
) stage43CalibrationObservation {
	t.Helper()

	observation := stage43CalibrationObservation{
		CaseID:       semanticCase.CaseID(),
		ExpectedKind: semanticCase.Expected().Kind(),
	}
	if semanticCase.Expected().Kind() ==
		legalquerycorpus.SemanticExpectedKindRequestError {
		return observeStage43RequestError(t, semanticCase, scorecard, observation)
	}
	plan, err := evaluateStage43DevelopmentCase(
		t.Context(),
		dependencies,
		semanticCase,
	)
	if err != nil {
		t.Fatalf(
			"%s: caseId %q の runtime error = %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			err,
		)
	}
	assertStage43DevelopmentCaseIsRunnable(t, semanticCase, plan)
	planObservation, err := newStage43PlanObservation(plan)
	if err != nil {
		t.Fatalf(
			"%s: caseId %q の plan を観測できません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			err,
		)
	}
	evaluation, err := legalqueryeval.EvaluateSemanticPlanCase(semanticCase, plan)
	if err != nil {
		t.Fatalf(
			"%s: caseId %q の expected 比較に失敗しました: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			err,
		)
	}
	evaluationObservation := newStage43EvaluationObservation(evaluation, scorecard)
	scorecard.PlanCount++
	observation.Plan = &planObservation
	observation.Evaluation = &evaluationObservation
	return observation
}

func observeStage43RequestError(
	t *testing.T,
	semanticCase legalquerycorpus.SemanticCase,
	scorecard *stage43CalibrationScorecard,
	observation stage43CalibrationObservation,
) stage43CalibrationObservation {
	t.Helper()

	_, err := stage43ProductRequest(semanticCase.Request())
	var argumentError legalquery.ArgumentError
	if !errors.As(err, &argumentError) {
		t.Fatalf(
			"%s: caseId %q が request error になりません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			err,
		)
	}
	expected := semanticCase.Expected().(legalquerycorpus.ExpectedRequestError)
	matched := expected.ErrorCode() == argumentError.Code() &&
		string(expected.Field()) == argumentError.Field()
	if !matched {
		t.Fatalf(
			"%s: caseId %q の request error = %s/%s",
			nextProfileSetDevelopmentOnlyCalibrationID,
			semanticCase.CaseID(),
			argumentError.Code(),
			argumentError.Field(),
		)
	}
	scorecard.RequestErrorCount++
	scorecard.RequestErrorMatched++
	observation.RequestError = &stage43RequestErrorObservation{
		Code:  argumentError.Code(),
		Field: argumentError.Field(),
	}
	return observation
}

func newStage43PlanObservation(
	plan legalquery.LegalQueryPlan,
) (stage43PlanObservation, error) {
	ranked := plan.RankedCandidates()
	candidates := make([]stage43CandidateObservation, 0, len(ranked))
	ranks := make(map[string]int, len(ranked))
	for index, candidate := range ranked {
		ranks[candidate.CandidateID()] = index + 1
		observed, err := newStage43CandidateObservation(index+1, candidate)
		if err != nil {
			return stage43PlanObservation{}, err
		}
		candidates = append(candidates, observed)
	}
	selected := make([]stage43SelectionObservation, 0, len(plan.Selected()))
	for _, selection := range plan.Selected() {
		rank, exists := ranks[selection.CandidateID()]
		if !exists {
			return stage43PlanObservation{}, fmt.Errorf(
				"selected candidate %q の順位がありません",
				selection.CandidateID(),
			)
		}
		selected = append(selected, stage43SelectionObservation{
			CandidateRank: rank,
			Availability:  selection.Availability(),
			RequiredPacks: selection.RequiredPacks(),
		})
	}
	budget := plan.Budget()
	return stage43PlanObservation{
		ProfileVersion: plan.ProfileVersion(),
		Decision:       plan.Decision(),
		ReasonCodes:    plan.ReasonCodes(),
		Candidates:     candidates,
		Selected:       selected,
		Budget: stage43BudgetObservation{
			LimitPerAttempt:    budget.LimitPerAttempt(),
			MaxCapabilityCalls: budget.MaxCapabilityCalls(),
			MaxReturnedItems:   budget.MaxReturnedItems(),
			FirstPageOnly:      budget.FirstPageOnly(),
		},
	}, nil
}

func newStage43CandidateObservation(
	rank int,
	candidate legalquery.LegalQueryCandidate,
) (stage43CandidateObservation, error) {
	concepts := candidate.ConceptSources()
	conceptIDs := make([]string, 0, len(concepts))
	for _, source := range concepts {
		conceptIDs = append(conceptIDs, source.ConceptID())
	}
	slices.Sort(conceptIDs)
	steps := make([]stage43StepObservation, 0, len(candidate.Steps()))
	for _, step := range candidate.Steps() {
		logicalInput, err := newStage43LogicalInputObservation(step.LogicalInput())
		if err != nil {
			return stage43CandidateObservation{}, err
		}
		steps = append(steps, stage43StepObservation{
			Task:         step.Task(),
			Resource:     step.Resource(),
			InputKind:    step.InputKind(),
			LogicalInput: logicalInput,
		})
	}
	return stage43CandidateObservation{
		Rank:          rank,
		SemanticScore: candidate.SemanticScore(),
		Confidence:    candidate.Confidence(),
		EvidenceCodes: candidate.EvidenceCodes(),
		ConceptIDs:    conceptIDs,
		RequiredPacks: candidate.RequiredPacks(),
		Steps:         steps,
	}, nil
}

func newStage43LogicalInputObservation(
	input legalquery.LogicalInput,
) (stage43LogicalInputObservation, error) {
	observation := stage43LogicalInputObservation{Kind: input.InputKind()}
	switch value := input.(type) {
	case legalquery.LawSearchIntentV1:
		observation.Query = value.Query()
		observation.AsOf = stage43OptionalDate(value.AsOf())
	case legalquery.LawContentSearchIntentV1:
		observation.AllTerms = value.AllTerms()
		observation.AnyTerms = value.AnyTerms()
		observation.ExcludeTerms = value.ExcludeTerms()
		observation.AsOf = stage43OptionalDate(value.AsOf())
	case legalquery.LawReadIntentV1:
		if ref, exists := value.Ref(); exists {
			observation.Ref = &ref
		} else {
			observation.LawID, _ = value.LawID()
			observation.RevisionID, _ = value.RevisionID()
		}
		observation.AsOf = stage43OptionalDate(value.AsOf())
	case legalquery.LawArticleReadIntentV1:
		if ref, exists := value.Ref(); exists {
			observation.Ref = &ref
		} else {
			observation.LawID, _ = value.LawID()
		}
		location := value.Location()
		observation.Location = &location
		observation.AsOf = stage43OptionalDate(value.AsOf())
	case legalquery.LawUpdateListIntentV1:
		observation.Date = value.Date().String()
	case legalquery.JudicialDecisionSearchIntentV1:
		observation.Query = value.Query()
	case legalquery.JudicialDecisionReadIntentV1:
		ref := value.Ref()
		observation.Ref = &ref
	default:
		return stage43LogicalInputObservation{}, fmt.Errorf(
			"input kind %q は校正観測に未対応です",
			input.InputKind(),
		)
	}
	return observation, nil
}

func stage43OptionalDate(date model.Date, exists bool) string {
	if !exists {
		return ""
	}
	return date.String()
}

func newStage43EvaluationObservation(
	evaluation legalqueryeval.SemanticCaseEvaluation,
	scorecard *stage43CalibrationScorecard,
) stage43EvaluationObservation {
	if evaluation.PlanOutcomeMatched() {
		scorecard.PlanOutcomeMatched++
	}
	highMatched, highApplicable := evaluation.HighConfidencePrecision()
	result := stage43EvaluationObservation{
		PlanOutcomeMatched: evaluation.PlanOutcomeMatched(),
		PrimaryTop1Matched: evaluation.PrimaryTop1Matched(),
		PrimaryTop2Matched: evaluation.PrimaryTop2Matched(),
		HighConfidence: stage43AssertionObservation{
			Applicable: highApplicable,
			Matched:    highMatched,
		},
	}
	for _, meaning := range evaluation.Meanings() {
		evidenceMatched, evidenceApplicable := meaning.EvidenceAssertion()
		conceptMatched, conceptApplicable := meaning.ConceptAssertion()
		scorecard.MeaningCount++
		if meaning.SignatureMatched() {
			scorecard.MeaningSignatureMatched++
		}
		if evidenceApplicable {
			scorecard.EvidenceApplicable++
			if evidenceMatched {
				scorecard.EvidenceMatched++
			}
		}
		if conceptApplicable {
			scorecard.ConceptApplicable++
			if conceptMatched {
				scorecard.ConceptMatched++
			}
		}
		result.Meanings = append(result.Meanings, stage43MeaningObservation{
			MeaningID:        meaning.MeaningID(),
			CandidateRank:    meaning.MatchedCandidateRank(),
			SignatureMatched: meaning.SignatureMatched(),
			Evidence: stage43AssertionObservation{
				Applicable: evidenceApplicable,
				Matched:    evidenceMatched,
			},
			Concept: stage43AssertionObservation{
				Applicable: conceptApplicable,
				Matched:    conceptMatched,
			},
		})
	}
	return result
}

func stage43CalibrationFingerprint(
	t *testing.T,
	artifact stage43CalibrationArtifact,
) string {
	t.Helper()

	// query 本文を含む development 校正観測は外部へ出力せず、digest だけを固定する。
	data, err := json.Marshal(artifact)
	if err != nil {
		t.Fatalf(
			"%s: 校正観測を canonical JSON にできません: %v",
			nextProfileSetDevelopmentOnlyCalibrationID,
			err,
		)
	}
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
