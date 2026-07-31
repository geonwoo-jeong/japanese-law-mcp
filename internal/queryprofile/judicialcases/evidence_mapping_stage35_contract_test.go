package judicialcases

import (
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const (
	judicialEvidenceMappingInputKindsID       = "judicial-evidence-mapping-input-kinds"
	judicialEvidenceMappingPrivateLifetimeID  = "judicial-evidence-mapping-private-lifetime"
	judicialEvidenceMappingTopicPositiveID    = "judicial-evidence-mapping-topic-positive"
	judicialEvidenceMappingRefNoSpanID        = "judicial-evidence-mapping-ref-no-span"
	judicialSharedTerminalRejectedID          = "judicial-shared-terminal-rejected"
	judicialEvidencePackProviderInvariantID   = "judicial-evidence-mapping-pack-provider-invariant"
	judicialEvidenceMappingFailClosedID       = "judicial-evidence-mapping-fail-closed"
	judicialMultiStepEvidenceNormalizationID  = "judicial-multi-step-evidence-step-local-normalization"
	judicialBoundedNonCartesianAlternativesID = "judicial-bounded-non-cartesian-alternatives"
)

func TestJudicialEvidenceMappingInputKinds(t *testing.T) {
	profile := mustJudicialEvidenceProfile(t)

	t.Run("検索は裁判例検索input kindだけを生成する", func(t *testing.T) {
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			"医療過誤の裁判例を検索してください。",
			nil,
			judicialEvidenceMappingInputKindsID,
		)
		candidates := generation.Candidates()
		if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
			t.Fatalf(
				"%s: search candidates = %#v",
				judicialEvidenceMappingInputKindsID,
				candidates,
			)
		}
		step := candidates[0].Steps()[0]
		if step.InputKind() != legalquery.InputKindJudicialDecisionSearch {
			t.Fatalf(
				"%s: step kind = %s",
				judicialEvidenceMappingInputKindsID,
				step.InputKind(),
			)
		}
		if !slices.Equal(candidates[0].RequiredPacks(), []string{requiredPackID}) {
			t.Fatalf(
				"%s: required packs = %#v",
				judicialEvidenceMappingInputKindsID,
				candidates[0].RequiredPacks(),
			)
		}
	})

	t.Run("readは入力refと裁判例readだけを生成する", func(t *testing.T) {
		ref := mustJudicialEvidenceRef(
			t,
			"courts",
			"hanrei",
			"95878/detail3",
		)
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			"この裁判例を読んでください。",
			&ref,
			judicialEvidenceMappingInputKindsID,
		)
		candidates := generation.Candidates()
		if len(candidates) != 1 || len(candidates[0].Steps()) != 1 {
			t.Fatalf(
				"%s: read candidates = %#v",
				judicialEvidenceMappingInputKindsID,
				candidates,
			)
		}
		step := candidates[0].Steps()[0]
		input, ok := step.LogicalInput().(legalquery.JudicialDecisionReadIntentV1)
		if !ok {
			t.Fatalf(
				"%s: read input = %#v",
				judicialEvidenceMappingInputKindsID,
				step.LogicalInput(),
			)
		}
		if step.InputKind() != legalquery.InputKindJudicialDecisionRead ||
			input.Ref() != ref {
			t.Fatalf(
				"%s: read step = %#v",
				judicialEvidenceMappingInputKindsID,
				step,
			)
		}
	})
}

func TestJudicialEvidenceMappingPrivateLifetime(t *testing.T) {
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
						judicialEvidenceMappingPrivateLifetimeID,
						current.Name(),
						field.Name,
					)
				}
			}
		}
	})

	t.Run("active profile を変更しない", func(t *testing.T) {
		active, err := LoadEmbedded()
		if err != nil {
			t.Fatalf(
				"%s: active profile を読み込めません: %v",
				judicialEvidenceMappingPrivateLifetimeID,
				err,
			)
		}
		margin, present := active.Metadata().Selection().BranchRetentionMargin()
		if active.Metadata().SchemaVersion() != 1 ||
			present || margin != 0 ||
			active.intentEvidenceMode != cueIntentEvidenceLegacy {
			t.Fatalf(
				"%s: active profile が 3.5 経路へ切り替わっています",
				judicialEvidenceMappingPrivateLifetimeID,
			)
		}
	})

	t.Run("next profile はschema v2とmarginを要求する", func(t *testing.T) {
		next := mustJudicialEvidenceProfile(t)
		margin, present := next.Metadata().Selection().BranchRetentionMargin()
		if next.Metadata().SchemaVersion() != 2 ||
			!present ||
			margin <= 0 ||
			next.intentEvidenceMode != cueIntentEvidenceJudicial {
			t.Fatalf(
				"%s: next profile の境界が不足しています",
				judicialEvidenceMappingPrivateLifetimeID,
			)
		}
	})

	t.Run("request間でprivate mappingを保持しない", func(t *testing.T) {
		profile := mustJudicialEvidenceProfile(t)
		first := generateJudicialEvidenceQuery(
			t,
			profile,
			"「医療過誤」の裁判例を検索してください。",
			nil,
			judicialEvidenceMappingPrivateLifetimeID,
		)
		_ = generateJudicialEvidenceQuery(
			t,
			profile,
			"「養育費」の裁判例を検索してください。",
			nil,
			judicialEvidenceMappingPrivateLifetimeID,
		)
		again := generateJudicialEvidenceQuery(
			t,
			profile,
			"「医療過誤」の裁判例を検索してください。",
			nil,
			judicialEvidenceMappingPrivateLifetimeID,
		)
		if !reflect.DeepEqual(first, again) {
			t.Fatalf(
				"%s: request間で生成結果が変わりました",
				judicialEvidenceMappingPrivateLifetimeID,
			)
		}
	})
}

func TestJudicialEvidenceMappingは主題とRefの位置根拠を分ける(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)

	t.Run("明示した個別主題はそれぞれ正の根拠を持つ", func(t *testing.T) {
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			"「医療過誤」と「損害賠償」を個別に裁判例検索してください。",
			nil,
			judicialEvidenceMappingTopicPositiveID,
		)
		candidate := mustSingleJudicialEvidenceCandidate(
			t,
			generation,
			judicialEvidenceMappingTopicPositiveID,
		)
		if got := judicialEvidenceSearchQueries(
			t,
			candidate,
			judicialEvidenceMappingTopicPositiveID,
		); !slices.Equal(got, []string{"医療過誤", "損害賠償"}) {
			t.Fatalf(
				"%s: 個別主題 = %#v",
				judicialEvidenceMappingTopicPositiveID,
				got,
			)
		}
	})

	t.Run("spanのないrefはread cueだけをcluster位置にする", func(t *testing.T) {
		const query = "この参照を読んでください。"
		ref := mustJudicialEvidenceRef(
			t,
			"provider-one",
			"source-one",
			"95878/detail3",
		)
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			query,
			&ref,
			judicialEvidenceMappingRefNoSpanID,
		)
		candidate := mustSingleJudicialEvidenceCandidate(
			t,
			generation,
			judicialEvidenceMappingRefNoSpanID,
		)
		if !slices.Equal(candidate.EvidenceCodes(), []legalquery.EvidenceCode{
			legalquery.EvidenceOfficialIdentifier,
			legalquery.EvidenceExplicitTask,
		}) {
			t.Fatalf(
				"%s: ref read evidence = %#v",
				judicialEvidenceMappingRefNoSpanID,
				candidate.EvidenceCodes(),
			)
		}
		members := generation.CompositionMembers()
		wantStart := strings.Index(query, "読んでください")
		if len(members) != 1 ||
			len(members[0].StepOrigins()) != 1 ||
			members[0].StepOrigins()[0].SourceStartByte() != wantStart {
			t.Fatalf(
				"%s: read の位置 = %#v、期待値は %d",
				judicialEvidenceMappingRefNoSpanID,
				members,
				wantStart,
			)
		}
	})
}

func TestJudicialEvidenceProfileは共有末尾とPackProviderを意味根拠にしない(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)

	t.Run("共有末尾sidecarを裁判例stepへfan-outしない", func(t *testing.T) {
		preprocessed := preprocessJudicialEvidenceQuery(
			t,
			profile,
			"成年後見、養育費を教えてください",
			nil,
			judicialSharedTerminalRejectedID,
		)
		input, err := legalquery.NewCandidateGenerationInput(preprocessed)
		if err != nil {
			t.Fatalf(
				"%s: candidate input を作成できません: %v",
				judicialSharedTerminalRejectedID,
				err,
			)
		}
		if len(input.SharedTerminalSequences()) == 0 {
			t.Fatalf(
				"%s: 検証に必要なsidecarがありません",
				judicialSharedTerminalRejectedID,
			)
		}
		generation := generateJudicialEvidenceInput(
			t,
			profile,
			input,
			judicialSharedTerminalRejectedID,
		)
		for _, candidate := range generation.Candidates() {
			if len(candidate.Steps()) > 1 {
				t.Fatalf(
					"%s: sidecar から複数 step を作りました: %#v",
					judicialSharedTerminalRejectedID,
					candidate,
				)
			}
		}
	})

	t.Run("providerとsourceの値で根拠scoreを変えない", func(t *testing.T) {
		firstRef := mustJudicialEvidenceRef(
			t, "provider-one", "source-one", "95878/detail3",
		)
		secondRef := mustJudicialEvidenceRef(
			t, "provider-two", "source-two", "95878/detail3",
		)
		first := mustSingleJudicialEvidenceCandidate(
			t,
			generateJudicialEvidenceQuery(
				t,
				profile,
				"この参照を読んでください。",
				&firstRef,
				judicialEvidencePackProviderInvariantID,
			),
			judicialEvidencePackProviderInvariantID,
		)
		second := mustSingleJudicialEvidenceCandidate(
			t,
			generateJudicialEvidenceQuery(
				t,
				profile,
				"この参照を読んでください。",
				&secondRef,
				judicialEvidencePackProviderInvariantID,
			),
			judicialEvidencePackProviderInvariantID,
		)
		if !slices.Equal(first.EvidenceCodes(), second.EvidenceCodes()) ||
			first.SemanticScore() != second.SemanticScore() ||
			first.Confidence() != second.Confidence() ||
			!slices.Equal(first.RequiredPacks(), []string{requiredPackID}) ||
			!slices.Equal(second.RequiredPacks(), []string{requiredPackID}) {
			t.Fatalf(
				"%s: provider/source で意味結果が変わりました",
				judicialEvidencePackProviderInvariantID,
			)
		}
	})
}

func TestJudicialEvidenceMappingは曖昧な束縛をFailClosedにする(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)
	ref := mustJudicialEvidenceRef(
		t,
		"provider-one",
		"source-one",
		"95878/detail3",
	)

	t.Run("二件のread relationから一件を選ばない", func(t *testing.T) {
		const query = "この参照を読んでください。もう一度読んでください。"
		preprocessed := preprocessJudicialEvidenceQuery(
			t,
			profile,
			query,
			&ref,
			judicialEvidenceMappingRefNoSpanID,
		)
		readRelations := 0
		for _, relation := range preprocessed.CueTaskRelations() {
			if relation.Kind() == legalquery.CueTaskRelationDirectTask &&
				relation.Subject().ProfileID() == profileID &&
				relation.Subject().CueID() == "task-read" {
				readRelations++
			}
		}
		if readRelations != 2 {
			t.Fatalf(
				"%s: fixture の read relation = %d",
				judicialEvidenceMappingRefNoSpanID,
				readRelations,
			)
		}
		input, err := legalquery.NewCandidateGenerationInput(preprocessed)
		if err != nil {
			t.Fatalf(
				"%s: candidate input を作成できません: %v",
				judicialEvidenceMappingFailClosedID,
				err,
			)
		}
		generation := generateJudicialEvidenceInput(
			t,
			profile,
			input,
			judicialEvidenceMappingFailClosedID,
		)
		if generationHasJudicialRead(generation) {
			t.Fatalf(
				"%s: 二件のread relationから候補を作りました",
				judicialEvidenceMappingFailClosedID,
			)
		}
	})

	t.Run("法令名と条項を裁判例検索語へ転用しない", func(t *testing.T) {
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			"民法第709条の裁判例を検索してください。",
			nil,
			judicialEvidenceMappingFailClosedID,
		)
		if len(generation.Candidates()) != 0 {
			t.Fatalf(
				"%s: 禁止factから候補を作りました: %#v",
				judicialEvidenceMappingFailClosedID,
				generation.Candidates(),
			)
		}
	})

	t.Run("入力refのないreadはsearchに置き換えない", func(t *testing.T) {
		generation := generateJudicialEvidenceQuery(
			t,
			profile,
			"この裁判例を読んでください。",
			nil,
			judicialEvidenceMappingFailClosedID,
		)
		if generationHasJudicialRead(generation) {
			t.Fatalf(
				"%s: refのないread候補を作りました",
				judicialEvidenceMappingFailClosedID,
			)
		}
	})

	t.Run("logical inputのref不一致はdraft全体を拒否する", func(t *testing.T) {
		preprocessed := preprocessJudicialEvidenceQuery(
			t,
			profile,
			"この参照を読んでください。",
			&ref,
			judicialEvidenceMappingFailClosedID,
		)
		input, err := legalquery.NewCandidateGenerationInput(preprocessed)
		if err != nil {
			t.Fatalf(
				"%s: candidate input を作成できません: %v",
				judicialEvidenceMappingFailClosedID,
				err,
			)
		}
		other := mustJudicialEvidenceRef(
			t,
			"provider-two",
			"source-one",
			"95878/detail3",
		)
		read, err := legalquery.NewJudicialDecisionReadIntentV1(
			legalquery.JudicialDecisionReadIntentV1Values{Ref: other},
		)
		if err != nil {
			t.Fatalf(
				"%s: read input を作成できません: %v",
				judicialEvidenceMappingFailClosedID,
				err,
			)
		}
		if err := validateJudicialEvidenceDraftInputs(
			input,
			[]candidateDraft{{steps: []stepDraft{{input: read}}}},
		); err == nil {
			t.Fatalf(
				"%s: request と異なる ref を受理しました",
				judicialEvidenceMappingFailClosedID,
			)
		}
	})
}

func TestJudicialMultiStepEvidenceはStep内正規化後に和集合する(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)
	ref := mustJudicialEvidenceRef(
		t,
		"provider-one",
		"source-one",
		"95878/detail3",
	)
	generation := generateJudicialEvidenceQuery(
		t,
		profile,
		"この参照を読んでください。「駅構内転倒」の裁判例も検索してください。",
		&ref,
		judicialMultiStepEvidenceNormalizationID,
	)
	candidate := mustSingleJudicialEvidenceCandidate(
		t,
		generation,
		judicialMultiStepEvidenceNormalizationID,
	)
	wantEvidence := []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceGeneralTerm,
	}
	wantScore, err := profile.Metadata().Score().Score(wantEvidence)
	if err != nil {
		t.Fatalf(
			"%s: 期待scoreを計算できません: %v",
			judicialMultiStepEvidenceNormalizationID,
			err,
		)
	}
	if len(candidate.Steps()) != 2 ||
		!slices.Equal(candidate.EvidenceCodes(), wantEvidence) ||
		candidate.SemanticScore() != wantScore {
		t.Fatalf(
			"%s: steps/evidence/score = %d/%v/%d、期待値は 2/%v/%d",
			judicialMultiStepEvidenceNormalizationID,
			len(candidate.Steps()),
			candidate.EvidenceCodes(),
			candidate.SemanticScore(),
			wantEvidence,
			wantScore,
		)
	}
}

func TestJudicialEvidenceProfileはNonCartesianな限定代替列だけを保持する(
	t *testing.T,
) {
	profile := mustJudicialEvidenceProfile(t)
	generation := generateJudicialEvidenceQuery(
		t,
		profile,
		"「ネット中傷」と「永住権」を個別に裁判例検索してください。",
		nil,
		judicialBoundedNonCartesianAlternativesID,
	)
	want := map[string]struct{}{
		"名誉毀損\x00永住許可":  {},
		"ネット中傷\x00永住許可": {},
		"名誉毀損\x00永住権":   {},
	}
	actual := make(map[string]struct{})
	for _, candidate := range generation.Candidates() {
		queries := judicialEvidenceSearchQueries(
			t,
			candidate,
			judicialBoundedNonCartesianAlternativesID,
		)
		if len(queries) != 2 {
			t.Fatalf(
				"%s: 限定代替候補のstep = %#v",
				judicialBoundedNonCartesianAlternativesID,
				queries,
			)
		}
		actual[strings.Join(queries, "\x00")] = struct{}{}
	}
	if !reflect.DeepEqual(actual, want) {
		t.Fatalf(
			"%s: 限定代替列 = %#v、期待値は %#v",
			judicialBoundedNonCartesianAlternativesID,
			actual,
			want,
		)
	}
	if _, cartesian := actual["ネット中傷\x00永住権"]; cartesian {
		t.Fatalf(
			"%s: Cartesian組合せを保持しました",
			judicialBoundedNonCartesianAlternativesID,
		)
	}
}

func mustJudicialEvidenceRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: "judicial-decision",
		ResourceID:   resourceID,
	})
	if err != nil {
		t.Fatalf(
			"%s: ref key を構築できません: %v",
			judicialEvidenceMappingInputKindsID,
			err,
		)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf(
			"%s: ref を構築できません: %v",
			judicialEvidenceMappingInputKindsID,
			err,
		)
	}
	return ref
}
