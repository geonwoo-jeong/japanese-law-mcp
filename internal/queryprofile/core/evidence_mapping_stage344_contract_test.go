package core

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const (
	coreEvidenceMappingLifetimeID        = "core-evidence-mapping-private-lifetime"
	coreEvidenceMappingProviderID        = "core-evidence-mapping-provider-independent"
	coreEvidenceMappingFailClosedID      = "core-evidence-mapping-fail-closed"
	coreMultiStepEvidenceNormalizationID = "core-multi-step-evidence-step-local-normalization"
	coreEvidenceProductionNeutralID      = "core-evidence-production-neutral"
)

func TestCoreMultiStepEvidenceはStep内正規化後に候補全体へ統合する(
	t *testing.T,
) {
	fixture := mustCoreEvidenceFixture(
		t,
		"民法（法令ID 129AC0000000089）という法令を読んでください。"+
			"法令本文から「営業秘密」を検索してください。",
		nil,
	)
	draft := mustCoreDraftWithKinds(
		t,
		fixture.drafts,
		legalquery.InputKindLawRead,
		legalquery.InputKindLawContentSearch,
	)
	evaluation := mustCoreEvidenceEvaluation(
		t,
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
	)
	assertCoreStepContainsCodes(
		t,
		mustCoreStepEvidence(t, evaluation, 0, 0),
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceOfficialAlias,
	)
	assertCoreStepContainsCodes(
		t,
		mustCoreStepEvidence(t, evaluation, 0, 1),
		legalquery.EvidenceGeneralTerm,
	)

	profile := mustCoreEvidenceProfile(t)
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf(
			"%s: candidate scope を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	candidates, _, forced, err := profile.materializeCoreEvidenceCandidates(
		fixture.input,
		fixture.cues,
		[]candidateDraft{draft},
		scope,
	)
	if err != nil {
		t.Fatalf(
			"%s: candidate を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if len(candidates) != 1 || forced {
		t.Fatalf(
			"%s: candidates=%d forced=%t",
			coreMultiStepEvidenceNormalizationID,
			len(candidates),
			forced,
		)
	}
	wantEvidence := []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceGeneralTerm,
	}
	wantScore, err := profile.Metadata().Score().Score(wantEvidence)
	if err != nil {
		t.Fatalf(
			"%s: 期待 score を計算できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if !slices.Equal(candidates[0].EvidenceCodes(), wantEvidence) ||
		candidates[0].SemanticScore() != wantScore {
		t.Fatalf(
			"%s: evidence=%v score=%d、期待値は %v/%d",
			coreMultiStepEvidenceNormalizationID,
			candidates[0].EvidenceCodes(),
			candidates[0].SemanticScore(),
			wantEvidence,
			wantScore,
		)
	}
}

func TestCoreMultiStepEvidenceは三Stepと根拠順を決定的に統合する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	preprocessed := preprocessCoreEvidenceQuery(
		t,
		profile,
		"永住許可、帰化、営業秘密を教えてください",
		nil,
		coreMultiStepEvidenceNormalizationID,
	)
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf(
			"%s: candidate input を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	rawCues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf(
			"%s: cue を解決できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	cues, err := profile.resolveRelationV2Cues(input, rawCues)
	if err != nil {
		t.Fatalf(
			"%s: relation cue を解決できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	drafts, err := profile.generateCoreEvidenceDrafts(input, cues, nil)
	if err != nil || len(drafts) == 0 {
		t.Fatalf(
			"%s: 三 step draft を構築できません: drafts=%d err=%v",
			coreMultiStepEvidenceNormalizationID,
			len(drafts),
			err,
		)
	}
	base := cloneCoreEvidenceTestDraft(drafts[0])
	if len(base.steps) < 3 {
		t.Fatalf(
			"%s: 三 step 未満です: %d",
			coreMultiStepEvidenceNormalizationID,
			len(base.steps),
		)
	}
	first := mustStage344MaterializedCandidate(
		t,
		profile,
		input,
		cues,
		base,
	)

	reordered := cloneCoreEvidenceTestDraft(base)
	slices.Reverse(reordered.steps)
	for index := range reordered.steps {
		slices.Reverse(reordered.steps[index].evidenceBindings)
	}
	slices.Reverse(reordered.concepts)
	second := mustStage344MaterializedCandidate(
		t,
		profile,
		input,
		cues,
		reordered,
	)

	if !slices.Equal(first.EvidenceCodes(), second.EvidenceCodes()) ||
		first.SemanticScore() != second.SemanticScore() ||
		!reflect.DeepEqual(first.ConceptSources(), second.ConceptSources()) {
		t.Fatalf(
			"%s: 入力順で統合結果が変わりました: %#v/%#v",
			coreMultiStepEvidenceNormalizationID,
			first,
			second,
		)
	}
}

func TestCoreEvidenceProductionはActiveProfileから到達しない(
	t *testing.T,
) {
	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf(
			"%s: active profile を読み込めません: %v",
			coreEvidenceProductionNeutralID,
			err,
		)
	}
	margin, marginPresent :=
		active.Metadata().Selection().BranchRetentionMargin()
	if active.Metadata().SchemaVersion() != 1 ||
		active.intentEvidenceMode != cueIntentEvidenceLegacy ||
		marginPresent || margin != 0 {
		t.Fatalf(
			"%s: active profile が test 専用 mode を採用しています",
			coreEvidenceProductionNeutralID,
		)
	}
	generation := generateCoreEvidenceQuery(
		t,
		active,
		"永住許可、帰化を教えてください",
		nil,
		coreEvidenceProductionNeutralID,
	)
	for _, candidate := range generation.Candidates() {
		if len(candidate.Steps()) > 1 {
			t.Fatalf(
				"%s: active profile が core evidence path の複数 step を返しました",
				coreEvidenceProductionNeutralID,
			)
		}
	}
}

func TestCoreEvidenceMappingPrivateLifetime(t *testing.T) {
	t.Run("長寿命modelへprivate mappingを追加しない", func(t *testing.T) {
		types := []reflect.Type{
			reflect.TypeOf(legalquery.PreprocessResult{}),
			reflect.TypeOf(legalquery.CandidateGenerationInput{}),
			reflect.TypeOf(legalquery.LegalQueryCandidate{}),
			reflect.TypeOf(legalquery.CandidateGeneration{}),
			reflect.TypeOf(legalquery.QueryProfileSetResult{}),
			reflect.TypeOf(legalquery.LegalQueryPlan{}),
		}
		mappingType := reflect.TypeOf(profileevidence.Mapping{})
		for _, current := range types {
			for index := range current.NumField() {
				field := current.Field(index)
				name := strings.ToLower(field.Name)
				if field.Type == mappingType ||
					strings.Contains(name, "evidencemapping") ||
					strings.Contains(name, "evidencespan") ||
					strings.Contains(name, "clusterkey") ||
					strings.Contains(name, "topicordinal") {
					t.Fatalf(
						"%s: %s に一時 field %q があります",
						coreEvidenceMappingLifetimeID,
						current.Name(),
						field.Name,
					)
				}
			}
		}
	})
}

func TestCoreEvidenceMappingIsProviderIndependent(t *testing.T) {
	first := mustCoreProviderEvaluation(
		t,
		mustContractLawRef(t, "provider-one", "source-one"),
	)
	second := mustCoreProviderEvaluation(
		t,
		mustContractLawRef(t, "provider-two", "source-two"),
	)
	firstEvidence := coreEvidenceSignatures(
		mustCoreStepEvidence(t, first, 0, 0),
	)
	secondEvidence := coreEvidenceSignatures(
		mustCoreStepEvidence(t, second, 0, 0),
	)
	firstKey := assertCoreDraftClusterEligible(t, first, 0)
	secondKey := assertCoreDraftClusterEligible(t, second, 0)
	if !slices.Equal(firstEvidence, secondEvidence) ||
		firstKey.Canonical() != secondKey.Canonical() {
		t.Fatalf(
			"%s: provider/source で mapping が変わりました: %v/%q != %v/%q",
			coreEvidenceMappingProviderID,
			firstEvidence,
			firstKey.Canonical(),
			secondEvidence,
			secondKey.Canonical(),
		)
	}

	firstCandidate := mustCoreProviderCandidate(
		t,
		mustContractLawRef(t, "provider-one", "source-one"),
	)
	secondCandidate := mustCoreProviderCandidate(
		t,
		mustContractLawRef(t, "provider-two", "source-two"),
	)
	if !slices.Equal(
		firstCandidate.EvidenceCodes(),
		secondCandidate.EvidenceCodes(),
	) || firstCandidate.SemanticScore() != secondCandidate.SemanticScore() ||
		firstCandidate.Confidence() != secondCandidate.Confidence() ||
		!reflect.DeepEqual(
			firstCandidate.ConceptSources(),
			secondCandidate.ConceptSources(),
		) {
		t.Fatalf(
			"%s: provider/source で最終 candidate の意味結果が変わりました",
			coreEvidenceMappingProviderID,
		)
	}
}

func TestCoreEvidenceMappingFailsClosed(t *testing.T) {
	t.Run("一つのfactを二stepへ束縛しない", func(t *testing.T) {
		fixture := mustCoreEvidenceFixture(
			t,
			"民法の正式な法令を検索してください。",
			nil,
		)
		draft := cloneCoreEvidenceTestDraft(
			mustCoreSingleKindDraft(
				t,
				fixture.drafts,
				legalquery.InputKindLawSearch,
			),
		)
		second := draft.steps[0]
		second.evidenceBindings = append(
			[]profileevidence.EvidenceValues(nil),
			second.evidenceBindings...,
		)
		draft.steps = append(draft.steps, second)
		assertCoreDraftNotEligible(
			t,
			fixture.input,
			fixture.cues,
			draft,
			coreEvidenceMappingFailClosedID,
		)
	})

	fixture := mustCoreEvidenceFixture(
		t,
		"法令本文から「営業秘密」を含む条文を検索してください。",
		nil,
	)
	base := mustCoreSingleKindDraft(
		t,
		fixture.drafts,
		legalquery.InputKindLawContentSearch,
	)
	tests := []struct {
		name   string
		change func(profileevidence.EvidenceValues) profileevidence.EvidenceValues
	}{
		{
			name: "同じfactとcodeのlayerが競合する",
			change: func(value profileevidence.EvidenceValues) profileevidence.EvidenceValues {
				if value.Layer == profileevidence.LayerTargetAnchor {
					value.Layer = profileevidence.LayerSemanticExpansion
				} else {
					value.Layer = profileevidence.LayerTargetAnchor
				}
				return value
			},
		},
		{
			name: "同じfactとcodeのcluster可否が競合する",
			change: func(value profileevidence.EvidenceValues) profileevidence.EvidenceValues {
				value.ClusterSpan = !value.ClusterSpan
				return value
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			draft := cloneCoreEvidenceTestDraft(base)
			index := generalTermBindingIndex(
				t,
				draft.steps[0].evidenceBindings,
			)
			conflict := test.change(
				draft.steps[0].evidenceBindings[index],
			)
			draft.steps[0].evidenceBindings = append(
				draft.steps[0].evidenceBindings,
				conflict,
			)
			if _, err := buildCoreEvidenceEvaluation(
				fixture.input,
				fixture.cues,
				[]candidateDraft{draft},
			); err == nil {
				t.Fatalf(
					"%s: 競合する mapping を受理しました",
					coreEvidenceMappingFailClosedID,
				)
			}
		})
	}

	t.Run("requestとlogical inputのref完全一致", func(t *testing.T) {
		testCoreEvidenceRefIdentity(t)
	})
}

func testCoreEvidenceRefIdentity(t *testing.T) {
	t.Helper()
	const query = "これを読んでください。"
	requestRef := mustStage344LawRef(
		t,
		"provider-one",
		"source-one",
		"001-AbC/Ａ",
		"Rev-001/Ａ",
	)
	fixture := mustCoreEvidenceFixture(t, query, &requestRef)
	base := mustCoreRefReadDraft(t, requestRef)
	if evaluation, err := buildCoreEvidenceEvaluation(
		fixture.input,
		fixture.cues,
		[]candidateDraft{withCoreReadRef(t, base, requestRef)},
	); err != nil || len(evaluation.drafts) != 1 {
		t.Fatalf(
			"%s: 完全一致する ref を拒否しました: drafts=%d err=%v",
			coreEvidenceMappingFailClosedID,
			len(evaluation.drafts),
			err,
		)
	}

	tests := []struct {
		name string
		ref  model.SourceResourceRef
	}{
		{
			name: "provider差",
			ref: mustStage344LawRef(
				t, "provider-two", "source-one", "001-AbC/Ａ", "Rev-001/Ａ",
			),
		},
		{
			name: "source差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-two", "001-AbC/Ａ", "Rev-001/Ａ",
			),
		},
		{
			name: "resource差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001-AbD/Ａ", "Rev-001/Ａ",
			),
		},
		{
			name: "version差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001-AbC/Ａ", "Rev-002/Ａ",
			),
		},
		{
			name: "version存在状態差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001-AbC/Ａ", "",
			),
		},
		{
			name: "大文字小文字差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001-abc/Ａ", "Rev-001/Ａ",
			),
		},
		{
			name: "先頭零差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "0001-AbC/Ａ", "Rev-001/Ａ",
			),
		},
		{
			name: "Unicode byte差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001-AbC/A", "Rev-001/Ａ",
			),
		},
		{
			name: "区切り文字差",
			ref: mustStage344LawRef(
				t, "provider-one", "source-one", "001:AbC/Ａ", "Rev-001/Ａ",
			),
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			assertCoreRefDraftRejected(
				t,
				fixture.input,
				fixture.cues,
				withCoreReadRef(t, base, test.ref),
			)
		})
	}

	withoutRef := mustCoreEvidenceFixture(t, query, nil)
	t.Run("request ref欠落", func(t *testing.T) {
		assertCoreRefDraftRejected(
			t,
			withoutRef.input,
			withoutRef.cues,
			withCoreReadRef(t, base, requestRef),
		)
	})
}

func mustStage344LawRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "law",
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("%s: ref key を構築できません: %v", coreEvidenceMappingFailClosedID, err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("%s: ref を構築できません: %v", coreEvidenceMappingFailClosedID, err)
	}
	return ref
}

func withCoreReadRef(
	t *testing.T,
	draft candidateDraft,
	ref model.SourceResourceRef,
) candidateDraft {
	t.Helper()
	result := cloneCoreEvidenceTestDraft(draft)
	if len(result.steps) != 1 ||
		result.steps[0].input.InputKind() != legalquery.InputKindLawRead {
		t.Fatalf("%s: law_read draft が必要です", coreEvidenceMappingFailClosedID)
	}
	input, err := legalquery.NewLawReadIntentV1(
		legalquery.LawReadIntentV1Values{Ref: &ref},
	)
	if err != nil {
		t.Fatalf("%s: law_read input を構築できません: %v", coreEvidenceMappingFailClosedID, err)
	}
	result.steps[0].input = input
	return result
}

func assertCoreRefDraftRejected(
	t *testing.T,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
) {
	t.Helper()
	evaluation, err := buildCoreEvidenceEvaluation(
		input,
		cues,
		[]candidateDraft{draft},
	)
	if err == nil && len(evaluation.drafts) != 0 {
		t.Fatalf(
			"%s: request と異なる logical input ref を受理しました",
			coreEvidenceMappingFailClosedID,
		)
	}
}

func assertCoreStepContainsCodes(
	t *testing.T,
	evidence []profileevidence.Evidence,
	codes ...legalquery.EvidenceCode,
) {
	t.Helper()
	for _, code := range codes {
		if !slices.ContainsFunc(evidence, func(value profileevidence.Evidence) bool {
			return value.Code() == code
		}) {
			t.Fatalf(
				"%s: step evidence に %q がありません: %#v",
				coreMultiStepEvidenceNormalizationID,
				code,
				evidence,
			)
		}
	}
}

func mustCoreProviderEvaluation(
	t *testing.T,
	ref model.SourceResourceRef,
) coreEvidenceEvaluation {
	t.Helper()
	fixture := mustCoreEvidenceFixture(
		t,
		"これを読んでください。",
		&ref,
	)
	return mustCoreEvidenceEvaluation(
		t,
		fixture.input,
		fixture.cues,
		[]candidateDraft{mustCoreRefReadDraft(t, ref)},
	)
}

func mustCoreProviderCandidate(
	t *testing.T,
	ref model.SourceResourceRef,
) legalquery.LegalQueryCandidate {
	t.Helper()
	fixture := mustCoreEvidenceFixture(
		t,
		"これを読んでください。",
		&ref,
	)
	return mustStage344MaterializedCandidate(
		t,
		mustCoreEvidenceProfile(t),
		fixture.input,
		fixture.cues,
		mustCoreRefReadDraft(t, ref),
	)
}

func mustStage344MaterializedCandidate(
	t *testing.T,
	profile *Profile,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	draft candidateDraft,
) legalquery.LegalQueryCandidate {
	t.Helper()
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf(
			"%s: candidate scope を構築できません: %v",
			coreMultiStepEvidenceNormalizationID,
			err,
		)
	}
	candidates, _, forced, err := profile.materializeCoreEvidenceCandidates(
		input,
		cues,
		[]candidateDraft{draft},
		scope,
	)
	if err != nil || forced || len(candidates) != 1 {
		t.Fatalf(
			"%s: candidate を一意に構築できません: candidates=%d forced=%t err=%v",
			coreMultiStepEvidenceNormalizationID,
			len(candidates),
			forced,
			err,
		)
	}
	return candidates[0]
}

func coreEvidenceSignatures(
	values []profileevidence.Evidence,
) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		span, exists := value.Span()
		result = append(result, fmt.Sprintf(
			"%s|%s|%s|%t|%d:%d|%t|%t",
			value.FactID(),
			value.Layer(),
			value.Code(),
			exists,
			span.StartByte(),
			span.EndByte(),
			value.IndependentPositive(),
			value.ClusterSpan(),
		))
	}
	return result
}

func generalTermBindingIndex(
	t *testing.T,
	values []profileevidence.EvidenceValues,
) int {
	t.Helper()
	for index, value := range values {
		if value.Code == legalquery.EvidenceGeneralTerm {
			return index
		}
	}
	t.Fatalf(
		"%s: general_term binding がありません",
		coreEvidenceMappingFailClosedID,
	)
	return -1
}
