package core

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreEvidenceFixture struct {
	input  legalquery.CandidateGenerationInput
	cues   resolvedCues
	drafts []candidateDraft
}

type evidencePair struct {
	layer profileevidence.Layer
	code  legalquery.EvidenceCode
}

func mustCoreEvidenceFixture(
	t *testing.T,
	query string,
	ref *model.SourceResourceRef,
) coreEvidenceFixture {
	t.Helper()
	profile := mustRelationV2Profile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("core evidence preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(
		legalquery.RequestValues{Query: query, Ref: ref},
	)
	if err != nil {
		t.Fatalf("core evidence request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("core evidence 前処理に失敗しました: %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("core evidence input を構築できません: %v", err)
	}
	rawCues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf("core evidence cue を解決できません: %v", err)
	}
	cues, err := profile.resolveRelationV2Cues(input, rawCues)
	if err != nil {
		t.Fatalf("core evidence relation cue を解決できません: %v", err)
	}
	targets, err := profile.buildRelationV2MentionTargetDrafts(input, cues)
	if err != nil {
		t.Fatalf("core evidence mention target を構築できません: %v", err)
	}
	drafts, err := profile.generateDrafts(input, cues, targets)
	if err != nil {
		t.Fatalf("core evidence draft を構築できません: %v", err)
	}
	drafts, err = withCoreEvidenceBindings(
		input,
		cues,
		withAsOfEvidence(drafts),
	)
	if err != nil {
		t.Fatalf("core evidence binding を構築できません: %v", err)
	}
	return coreEvidenceFixture{
		input:  input,
		cues:   cues,
		drafts: drafts,
	}
}

func mustCoreSingleKindDraft(
	t *testing.T,
	drafts []candidateDraft,
	kind legalquery.LogicalInputKind,
) candidateDraft {
	t.Helper()
	for _, draft := range drafts {
		if len(draft.steps) == 1 &&
			draft.steps[0].input.InputKind() == kind {
			return cloneCoreEvidenceTestDraft(draft)
		}
	}
	t.Fatalf(
		"%s: input kind %q の一 step draft がありません",
		coreEvidenceMappingInputKindsID,
		kind,
	)
	return candidateDraft{}
}

func mustCoreDraftWithKinds(
	t *testing.T,
	drafts []candidateDraft,
	kinds ...legalquery.LogicalInputKind,
) candidateDraft {
	t.Helper()
	for _, draft := range drafts {
		if len(draft.steps) != len(kinds) {
			continue
		}
		matches := true
		for index, kind := range kinds {
			if draft.steps[index].input.InputKind() != kind {
				matches = false
				break
			}
		}
		if matches {
			return cloneCoreEvidenceTestDraft(draft)
		}
	}
	t.Fatalf(
		"%s: input kind 列 %v の draft がありません",
		coreEvidenceMappingPositiveID,
		kinds,
	)
	return candidateDraft{}
}

func cloneCoreEvidenceTestDraft(value candidateDraft) candidateDraft {
	result := cloneDraft(value)
	result.steps = append([]stepDraft(nil), value.steps...)
	for index := range result.steps {
		result.steps[index].evidenceBindings = append(
			[]profileevidence.EvidenceValues(nil),
			value.steps[index].evidenceBindings...,
		)
	}
	return result
}

func mustCoreEvidenceEvaluation(
	t *testing.T,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
) coreEvidenceEvaluation {
	t.Helper()
	evaluation, err := buildCoreEvidenceEvaluation(input, cues, drafts)
	if err != nil {
		t.Fatalf(
			"%s: core evidence evaluation を構築できません: %v",
			coreEvidenceMappingInputKindsID,
			err,
		)
	}
	if len(evaluation.drafts) != len(drafts) {
		t.Fatalf(
			"%s: 有効な draft 対応数 = %d, want %d",
			coreEvidenceMappingInputKindsID,
			len(evaluation.drafts),
			len(drafts),
		)
	}
	return evaluation
}

func mustCoreStepEvidence(
	t *testing.T,
	evaluation coreEvidenceEvaluation,
	draftIndex int,
	stepIndex int,
) []profileevidence.Evidence {
	t.Helper()
	reference := evaluation.drafts[draftIndex]
	mapping, err := evaluation.mappingFor(reference.draftID)
	if err != nil {
		t.Fatalf("core evidence mapping を取得できません: %v", err)
	}
	evidence, err := mapping.StepEvidence(
		reference.draftID,
		reference.stepIDs[stepIndex],
	)
	if err != nil {
		t.Fatalf("core evidence mapping から step を取得できません: %v", err)
	}
	return evidence
}

func assertCoreDraftClusterEligible(
	t *testing.T,
	evaluation coreEvidenceEvaluation,
	draftIndex int,
) profileevidence.ClusterKey {
	t.Helper()
	reference := evaluation.drafts[draftIndex]
	mapping, err := evaluation.mappingFor(reference.draftID)
	if err != nil {
		t.Fatalf("%s: mapping を取得できません: %v", coreEvidenceMappingPositiveID, err)
	}
	key, eligible, err := mapping.ClusterKey(reference.draftID)
	if err != nil || !eligible {
		t.Fatalf(
			"%s: cluster eligibility = %t, key=%q, err=%v",
			coreEvidenceMappingPositiveID,
			eligible,
			key.Canonical(),
			err,
		)
	}
	return key
}

func assertCoreDraftNotEligible(
	t *testing.T,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
	verificationID string,
) {
	t.Helper()
	evaluation, err := buildCoreEvidenceEvaluation(
		input,
		cues,
		[]candidateDraft{draft},
	)
	if err != nil {
		t.Fatalf(
			"%s: 非適格draftの評価でエラーになりました: %v",
			verificationID,
			err,
		)
	}
	if len(evaluation.drafts) == 0 {
		return
	}
	reference := evaluation.drafts[0]
	mapping, mappingErr := evaluation.mappingFor(reference.draftID)
	if mappingErr != nil {
		t.Fatalf("%s: mapping を取得できません: %v", verificationID, mappingErr)
	}
	_, eligible, clusterErr := mapping.ClusterKey(reference.draftID)
	if clusterErr != nil {
		t.Fatalf("%s: cluster 検証に失敗しました: %v", verificationID, clusterErr)
	}
	if eligible {
		t.Fatalf("%s: 根拠のない draft を実行適格にしました", verificationID)
	}
}

func assertCoreAllowedEvidence(
	t *testing.T,
	kind legalquery.LogicalInputKind,
	evidence []profileevidence.Evidence,
) {
	t.Helper()
	allowed := coreAllowedEvidencePairs(kind)
	for _, value := range evidence {
		pair := evidencePair{layer: value.Layer(), code: value.Code()}
		if !slices.Contains(allowed, pair) {
			t.Fatalf(
				"%s: %s が禁止対応 %s/%s を持ちます",
				coreEvidenceMappingInputKindsID,
				kind,
				value.Layer(),
				value.Code(),
			)
		}
	}
}

func assertCoreResourceCueValues(
	t *testing.T,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	evidence []profileevidence.Evidence,
	allowed []string,
	require bool,
) {
	t.Helper()
	facts, err := buildCoreEvidenceFacts(input, cues)
	if err != nil {
		t.Fatalf(
			"%s: resource cue factを構築できません: %v",
			coreEvidenceMappingInputKindsID,
			err,
		)
	}
	var actual []string
	for _, value := range evidence {
		if value.Layer() != profileevidence.LayerExplicitTaskResource ||
			value.Code() != legalquery.EvidenceExplicitResource {
			continue
		}
		fact, exists := facts.byID[value.FactID()]
		if !exists || fact.cueCategory != "resource" {
			t.Fatalf(
				"%s: explicit_resource fact %q がresource cueではありません",
				coreEvidenceMappingInputKindsID,
				value.FactID(),
			)
		}
		actual = append(actual, fact.cueValue)
		if !slices.Contains(allowed, fact.cueValue) {
			t.Fatalf(
				"%s: 許可されないresource意味 %q を束縛しました",
				coreEvidenceMappingInputKindsID,
				fact.cueValue,
			)
		}
	}
	if require && len(actual) == 0 {
		t.Fatalf(
			"%s: 必須resource cue %v がありません",
			coreEvidenceMappingInputKindsID,
			allowed,
		)
	}
}

func assertNoSharedExplicitFact(
	t *testing.T,
	left []profileevidence.Evidence,
	right []profileevidence.Evidence,
) {
	t.Helper()
	leftFacts := make(map[string]struct{})
	for _, value := range left {
		if value.Layer() == profileevidence.LayerExplicitTaskResource {
			leftFacts[value.FactID()] = struct{}{}
		}
	}
	for _, value := range right {
		if value.Layer() != profileevidence.LayerExplicitTaskResource {
			continue
		}
		if _, shared := leftFacts[value.FactID()]; shared {
			t.Fatalf(
				"%s: 明示fact %qを複数stepへ共有しました",
				coreEvidenceMappingInputKindsID,
				value.FactID(),
			)
		}
	}
}

func mustCoreEvidenceByFactPrefix(
	t *testing.T,
	evidence []profileevidence.Evidence,
	prefix string,
) profileevidence.Evidence {
	t.Helper()
	for _, value := range evidence {
		if strings.HasPrefix(value.FactID(), prefix) {
			return value
		}
	}
	t.Fatalf(
		"%s: fact prefix %q のevidenceがありません",
		coreEvidenceMappingInputKindsID,
		prefix,
	)
	return profileevidence.Evidence{}
}

func countCoreEvidenceByFactPrefix(
	evidence []profileevidence.Evidence,
	prefix string,
) int {
	var count int
	for _, value := range evidence {
		if strings.HasPrefix(value.FactID(), prefix) {
			count++
		}
	}
	return count
}

func mustCoreEvidenceByCode(
	t *testing.T,
	evidence []profileevidence.Evidence,
	code legalquery.EvidenceCode,
) profileevidence.Evidence {
	t.Helper()
	for _, value := range evidence {
		if value.Code() == code {
			return value
		}
	}
	t.Fatalf(
		"%s: evidence code %q がありません",
		coreEvidenceMappingInputKindsID,
		code,
	)
	return profileevidence.Evidence{}
}

func assertCoreDateEvidence(
	t *testing.T,
	value profileevidence.Evidence,
	positive bool,
	clusterSpan bool,
) {
	t.Helper()
	if value.Layer() != profileevidence.LayerTargetAnchor ||
		value.Code() != legalquery.EvidenceStructuredReference ||
		value.IndependentPositive() != positive ||
		value.ClusterSpan() != clusterSpan {
		t.Fatalf(
			"%s: 日付evidence = %#v",
			coreEvidenceMappingInputKindsID,
			value,
		)
	}
}

func assertCoreRefEvidence(
	t *testing.T,
	evidence []profileevidence.Evidence,
) {
	t.Helper()
	value := mustCoreEvidenceByFactPrefix(t, evidence, "input-ref")
	_, hasSpan := value.Span()
	if hasSpan ||
		value.Layer() != profileevidence.LayerBoundary ||
		value.Code() != legalquery.EvidenceOfficialIdentifier ||
		!value.IndependentPositive() ||
		value.ClusterSpan() {
		t.Fatalf(
			"%s: ref evidence = %#v",
			coreEvidenceMappingRefSpanID,
			value,
		)
	}
}

func coreAllowedEvidencePairs(
	kind legalquery.LogicalInputKind,
) []evidencePair {
	explicit := []evidencePair{
		{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitTask},
		{profileevidence.LayerExplicitTaskResource, legalquery.EvidenceExplicitResource},
	}
	switch kind {
	case legalquery.InputKindLawSearch:
		return append(explicit,
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialIdentifier},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceUniqueTypoCorrection},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceGeneralTerm},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceLegalConcept},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceUniqueTypoCorrection},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext},
		)
	case legalquery.InputKindLawContentSearch:
		return append(explicit,
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceGeneralTerm},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceLegalConcept},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceUniqueTypoCorrection},
			evidencePair{profileevidence.LayerSemanticExpansion, legalquery.EvidenceMorphologicalContext},
		)
	case legalquery.InputKindLawRead, legalquery.InputKindLawArticleRead:
		return append(explicit,
			evidencePair{profileevidence.LayerBoundary, legalquery.EvidenceOfficialIdentifier},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialIdentifier},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceOfficialAlias},
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceUniqueTypoCorrection},
		)
	case legalquery.InputKindLawUpdates:
		return append(explicit,
			evidencePair{profileevidence.LayerTargetAnchor, legalquery.EvidenceStructuredReference},
		)
	default:
		return nil
	}
}

func assertCoreForbiddenFactRejected(
	t *testing.T,
	kind legalquery.LogicalInputKind,
) {
	t.Helper()
	var fact coreEvidenceFact
	var value profileevidence.EvidenceValues
	switch kind {
	case legalquery.InputKindLawSearch:
		fact.kind = coreEvidenceFactArticle
		value = targetEvidence(
			"forbidden",
			legalquery.EvidenceStructuredReference,
			true,
		)
	case legalquery.InputKindLawContentSearch:
		fact.kind = coreEvidenceFactIdentifier
		fact.identifierKind = legalquery.IdentifierMentionLawID
		value = targetEvidence(
			"forbidden",
			legalquery.EvidenceOfficialIdentifier,
			true,
		)
	case legalquery.InputKindLawRead:
		fact.kind = coreEvidenceFactQueryTerm
		fact.queryTermKind = legalquery.QueryTermMentionQuotedPhrase
		value = targetEvidence(
			"forbidden",
			legalquery.EvidenceGeneralTerm,
			true,
		)
	case legalquery.InputKindLawArticleRead:
		fact.kind = coreEvidenceFactLegalConcept
		value = profileevidence.EvidenceValues{
			FactID:              "forbidden",
			Layer:               profileevidence.LayerSemanticExpansion,
			Code:                legalquery.EvidenceLegalConcept,
			IndependentPositive: true,
			ClusterSpan:         true,
		}
	case legalquery.InputKindLawUpdates:
		fact.kind = coreEvidenceFactLawName
		value = targetEvidence(
			"forbidden",
			legalquery.EvidenceOfficialAlias,
			true,
		)
	default:
		t.Fatalf(
			"%s: 未対応のinput kind %qです",
			coreEvidenceMappingInputKindsID,
			kind,
		)
	}
	if coreEvidenceBindingAllowed(kind, value, fact) {
		t.Fatalf(
			"%s: %sへ禁止fact kind %dを許可しました",
			coreEvidenceMappingInputKindsID,
			kind,
			fact.kind,
		)
	}
}

func assertEvidencePair(
	t *testing.T,
	evidence []profileevidence.Evidence,
	want evidencePair,
) {
	t.Helper()
	for _, value := range evidence {
		if value.Layer() == want.layer && value.Code() == want.code {
			return
		}
	}
	t.Fatalf(
		"%s: 必須対応 %s/%s がありません",
		coreEvidenceMappingInputKindsID,
		want.layer,
		want.code,
	)
}

func mustContractLawRef(
	t *testing.T,
	providerID string,
	sourceID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   "129AC0000000089",
	})
	if err != nil {
		t.Fatalf("core evidence ref key を構築できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("core evidence ref を構築できません: %v", err)
	}
	return ref
}

func mustCoreLawIDReadDraft(
	t *testing.T,
	lawID string,
) candidateDraft {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{LawID: lawID},
	)
	if err != nil {
		t.Fatalf("core law read inputを構築できません: %v", err)
	}
	return candidateDraft{
		evidence: make(map[legalquery.EvidenceCode]struct{}),
		steps:    []stepDraft{{input: input}},
	}
}

func mustCoreRefReadDraft(
	t *testing.T,
	ref model.SourceResourceRef,
) candidateDraft {
	t.Helper()
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{Ref: &ref},
	)
	if err != nil {
		t.Fatalf("core ref read inputを構築できません: %v", err)
	}
	return candidateDraft{
		evidence: make(map[legalquery.EvidenceCode]struct{}),
		steps:    []stepDraft{{input: input}},
	}
}

func mustCoreRefArticleReadDraft(
	t *testing.T,
	ref model.SourceResourceRef,
	articleNumber string,
) candidateDraft {
	t.Helper()
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: articleNumber,
	})
	if err != nil {
		t.Fatalf("core ref article locationを構築できません: %v", err)
	}
	input, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			Ref:      &ref,
			Location: location,
		},
	)
	if err != nil {
		t.Fatalf("core ref article read inputを構築できません: %v", err)
	}
	return candidateDraft{
		evidence: make(map[legalquery.EvidenceCode]struct{}),
		steps:    []stepDraft{{input: input}},
	}
}
