package core

import (
	"context"
	"slices"
	"sort"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const (
	coreSharedTerminalTaskResourceBindingID = "core-shared-terminal-task-resource-binding"
	coreSharedTerminalEvidenceClusterID     = "core-shared-terminal-evidence-cluster"
	coreBoundedNonCartesianAlternativesID   = "core-bounded-non-cartesian-alternatives"
)

func TestCoreSharedTerminalStage343(t *testing.T) {
	t.Run("正確なterminal relationとtask resourceを束縛する", func(t *testing.T) {
		profile := mustRelationV2Profile(t)
		for _, query := range []string{
			"永住許可、帰化を教えてください",
			"永住許可、帰化を教えて",
			"本文で永住許可、帰化を検索してください",
			"読んでください。永住許可、帰化を教えてください",
			"教えてください。永住許可、帰化を教えてください",
		} {
			input, cues := stage343Input(t, profile, query)
			if len(input.SharedTerminalSequences()) != 1 {
				t.Fatalf(
					"%s: %q の sidecar 件数が一件ではありません",
					coreSharedTerminalTaskResourceBindingID,
					query,
				)
			}
			built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
			if err != nil || !built.handled || built.stepLimitExceeded ||
				len(built.drafts) == 0 {
				t.Fatalf(
					"%s: %q の構築結果 = %#v, err=%v",
					coreSharedTerminalTaskResourceBindingID,
					query,
					built,
					err,
				)
			}
		}

		multipleInput, multipleCues := stage343Input(
			t,
			profile,
			"本文と規定で永住許可、帰化を検索してください",
		)
		if len(multipleInput.SharedTerminalSequences()) != 1 ||
			len(multipleCues.mentions[cueMeaningKey("resource", "law_provision")]) < 2 {
			t.Fatalf(
				"%s: 複数の互換 resource cue fixture を構築できません",
				coreSharedTerminalTaskResourceBindingID,
			)
		}
		multiple, err := profile.buildCoreSharedTerminalDrafts(
			multipleInput,
			multipleCues,
		)
		if err != nil || !multiple.handled || len(multiple.drafts) == 0 {
			t.Fatalf(
				"%s: 複数の互換 law_provision cue を消費できません: %#v, err=%v",
				coreSharedTerminalTaskResourceBindingID,
				multiple,
				err,
			)
		}
		for _, step := range multiple.drafts[0].draft.steps {
			resourceFacts := 0
			for _, evidence := range step.evidenceBindings {
				if evidence.Layer == profileevidence.LayerExplicitTaskResource &&
					evidence.Code == legalquery.EvidenceExplicitResource {
					resourceFacts++
				}
			}
			if resourceFacts < 2 {
				t.Fatalf(
					"%s: 互換 resource cue の根拠を和集合していません: %#v",
					coreSharedTerminalTaskResourceBindingID,
					step.evidenceBindings,
				)
			}
		}

		for _, query := range []string{
			"永住許可、帰化を検索してください",
			"法令で永住許可、帰化を教えてください",
		} {
			input, cues := stage343Input(t, profile, query)
			if len(input.SharedTerminalSequences()) != 1 {
				t.Fatalf(
					"%s: 競合 fixture %q に sidecar がありません",
					coreSharedTerminalTaskResourceBindingID,
					query,
				)
			}
			built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
			if err != nil || built.handled || len(built.drafts) != 0 {
				t.Fatalf(
					"%s: 競合 fixture %q を消費しました: %#v, relations=%#v, cues=%#v, err=%v",
					coreSharedTerminalTaskResourceBindingID,
					query,
					built,
					input.CueTaskRelations(),
					input.CueMentions(),
					err,
				)
			}
		}

		input, cues := stage343InputWithCompetingProfileRelation(t, profile)
		built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
		if err != nil || built.handled || len(built.drafts) != 0 {
			t.Fatalf(
				"%s: 同じ節の別 profile relation を分離せず消費しました: %#v, err=%v",
				coreSharedTerminalTaskResourceBindingID,
				built,
				err,
			)
		}
	})

	t.Run("同一span別意味と異なるspan同値を区別する", func(t *testing.T) {
		profile := mustRelationV2Profile(t)
		concept := stage343TopicOption(
			t,
			"帰化",
			0,
			"concept",
			"legal-concept:test",
			legalquery.EvidenceLegalConcept,
		)
		morphological := stage343TopicOption(
			t,
			"帰化",
			0,
			"morphological",
			"morphological_phrase",
			legalquery.EvidenceMorphologicalContext,
		)
		ordered, err := profile.orderCoreTopicOptions([]coreTopicOption{
			morphological,
			concept,
		})
		if err != nil || len(ordered) != 2 ||
			ordered[0].meaningKey != "legal-concept:test" {
			t.Fatalf(
				"%s: topic-local 完全順序 = %#v, err=%v",
				coreBoundedNonCartesianAlternativesID,
				ordered,
				err,
			)
		}
		reversed := slices.Clone(ordered)
		slices.Reverse(reversed)
		again, err := profile.orderCoreTopicOptions(reversed)
		if err != nil || len(again) != 2 ||
			again[0].meaningKey != ordered[0].meaningKey ||
			again[1].meaningKey != ordered[1].meaningKey {
			t.Fatalf(
				"%s: 入力順で topic-local 順序が変わりました",
				coreBoundedNonCartesianAlternativesID,
			)
		}

		choices := coreBoundedTopicChoices([][]coreTopicOption{ordered})
		if len(choices) != 2 {
			t.Fatalf(
				"%s: 同一 span の別意味を縮約しました",
				coreBoundedNonCartesianAlternativesID,
			)
		}

		first := stage343TopicOption(
			t,
			"帰化",
			0,
			"term-one",
			"morphological_phrase",
			legalquery.EvidenceMorphologicalContext,
		)
		second := stage343TopicOption(
			t,
			"帰化",
			12,
			"term-two",
			"morphological_phrase",
			legalquery.EvidenceMorphologicalContext,
		)
		draft, err := buildCoreSharedTerminalDraft([]coreTopicOption{first, second})
		if err != nil || len(draft.draft.steps) != 1 ||
			len(draft.draft.steps[0].evidenceBindings) != 2 ||
			draft.draft.steps[0].startByte != 0 {
			t.Fatalf(
				"%s: 異なる span の同値縮約 = %#v, err=%v",
				coreSharedTerminalEvidenceClusterID,
				draft,
				err,
			)
		}
	})

	t.Run("限定代替列はCartesian積を作らない", func(t *testing.T) {
		first := []coreTopicOption{
			stage343TopicOption(t, "第一", 0, "first-a", "first-a", legalquery.EvidenceLegalConcept),
			stage343TopicOption(t, "第一別", 0, "first-b", "first-b", legalquery.EvidenceMorphologicalContext),
		}
		second := []coreTopicOption{
			stage343TopicOption(t, "第二", 12, "second-a", "second-a", legalquery.EvidenceLegalConcept),
			stage343TopicOption(t, "第二別", 12, "second-b", "second-b", legalquery.EvidenceMorphologicalContext),
		}
		choices := coreBoundedTopicChoices([][]coreTopicOption{first, second})
		if len(choices) != 3 ||
			(choices[1][0].meaningKey == "first-b" &&
				choices[1][1].meaningKey == "second-b") ||
			(choices[2][0].meaningKey == "first-b" &&
				choices[2][1].meaningKey == "second-b") {
			t.Fatalf(
				"%s: 限定代替列 = %#v",
				coreBoundedNonCartesianAlternativesID,
				choices,
			)
		}
	})

	t.Run("四stepと五stepの境界を分ける", func(t *testing.T) {
		profile := mustRelationV2Profile(t)
		fourInput, fourCues := stage343Input(
			t,
			profile,
			"永住許可、帰化、営業秘密、個人情報を教えてください",
		)
		four, err := profile.buildCoreSharedTerminalDrafts(fourInput, fourCues)
		if err != nil || !four.handled || four.stepLimitExceeded ||
			len(four.drafts) == 0 {
			t.Fatalf(
				"%s: 四 step = %#v, err=%v",
				coreSharedTerminalEvidenceClusterID,
				four,
				err,
			)
		}
		for _, draft := range four.drafts {
			if len(draft.draft.steps) > legalquery.MaxCapabilityCalls {
				t.Fatalf(
					"%s: 四 step fixture が上限を超えました",
					coreSharedTerminalEvidenceClusterID,
				)
			}
		}

		fiveInput, fiveCues := stage343Input(
			t,
			profile,
			"永住許可、帰化、営業秘密、個人情報、育児休業を教えてください",
		)
		five, err := profile.buildCoreSharedTerminalDrafts(fiveInput, fiveCues)
		if err != nil || !five.handled || !five.stepLimitExceeded ||
			len(five.drafts) != 0 {
			t.Fatalf(
				"%s: 五 step を部分保持しました: %#v, err=%v",
				coreSharedTerminalEvidenceClusterID,
				five,
				err,
			)
		}
	})

	t.Run("clusterごとにmarginと三件上限を適用する", func(t *testing.T) {
		profile := mustRelationV2Profile(t)
		values := []coreSharedTerminalPreparedDraft{
			stage343PreparedDraft(t, "same", "fourth", 97, 3),
			stage343PreparedDraft(t, "same", "leader", 100, 0),
			stage343PreparedDraft(t, "same", "third", 98, 2),
			stage343PreparedDraft(t, "same", "second", 99, 1),
			stage343PreparedDraft(t, "same", "outside", 80, 4),
		}
		retained, err := profile.retainPreparedCoreSharedTerminalBranches(values)
		if err != nil || len(retained.drafts) != 3 ||
			!retained.clarificationRequired {
			t.Fatalf(
				"%s: 第四分岐の保持 = %#v, err=%v",
				coreSharedTerminalEvidenceClusterID,
				retained,
				err,
			)
		}

		withoutFourth := []coreSharedTerminalPreparedDraft{
			stage343PreparedDraft(t, "same", "leader", 100, 0),
			stage343PreparedDraft(t, "same", "second", 99, 1),
			stage343PreparedDraft(t, "same", "third", 98, 2),
			stage343PreparedDraft(t, "same", "outside", 80, 3),
		}
		retained, err = profile.retainPreparedCoreSharedTerminalBranches(withoutFourth)
		if err != nil || len(retained.drafts) != 3 ||
			retained.clarificationRequired {
			t.Fatalf(
				"%s: margin 外の第四分岐で明確化しました",
				coreSharedTerminalEvidenceClusterID,
			)
		}
	})

	t.Run("実入力をprivate mappingへ渡してclusterを作る", func(t *testing.T) {
		profile := mustRelationV2Profile(t)
		input, cues := stage343Input(
			t,
			profile,
			"永住許可、帰化を教えてください",
		)
		built, err := profile.buildCoreSharedTerminalDrafts(input, cues)
		if err != nil || !built.handled || len(built.drafts) == 0 {
			t.Fatalf("%s: shared draft を作れません", coreSharedTerminalEvidenceClusterID)
		}
		retained, err := profile.retainCoreSharedTerminalDrafts(
			input,
			cues,
			built.drafts,
		)
		if err != nil || len(retained.drafts) == 0 {
			t.Fatalf(
				"%s: cluster を作れません: %#v, err=%v",
				coreSharedTerminalEvidenceClusterID,
				retained,
				err,
			)
		}
	})
}

func stage343Input(
	t *testing.T,
	profile *Profile,
	query string,
) (legalquery.CandidateGenerationInput, resolvedCues) {
	t.Helper()
	return stage343ResolvedInput(t, profile, stage343Preprocess(t, profile, query))
}

func stage343Preprocess(
	t *testing.T,
	profile *Profile,
	query string,
) legalquery.PreprocessResult {
	t.Helper()
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("stage 3.4.3 preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf("stage 3.4.3 request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("stage 3.4.3 preprocess に失敗しました: %v", err)
	}
	return preprocessed
}

func stage343ResolvedInput(
	t *testing.T,
	profile *Profile,
	preprocessed legalquery.PreprocessResult,
) (legalquery.CandidateGenerationInput, resolvedCues) {
	t.Helper()
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("stage 3.4.3 input を構築できません: %v", err)
	}
	cues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf("stage 3.4.3 cue を解決できません: %v", err)
	}
	cues, err = profile.resolveRelationV2Cues(input, cues)
	if err != nil {
		t.Fatalf("stage 3.4.3 relation cue を解決できません: %v", err)
	}
	return input, cues
}

func stage343InputWithCompetingProfileRelation(
	t *testing.T,
	profile *Profile,
) (legalquery.CandidateGenerationInput, resolvedCues) {
	t.Helper()
	const query = "永住許可、帰化を教えてください"
	base := stage343Preprocess(t, profile, query)
	relations := base.CueTaskRelations()
	cues := base.CueMentions()
	if len(cues) != 1 || len(relations) != 1 {
		t.Fatalf("%s: 競合 relation fixture の基底が不正です", coreSharedTerminalTaskResourceBindingID)
	}
	otherCue, err := legalquery.NewCueMention(legalquery.CueMentionValues{
		Span:      cues[0].Span(),
		Surface:   cues[0].Surface(),
		ProfileID: "other-profile",
		CueID:     "task-search",
		MatchKind: legalquery.PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("%s: 別 profile cue を構築できません: %v", coreSharedTerminalTaskResourceBindingID, err)
	}
	otherRelation, err := legalquery.NewCueTaskRelation(
		legalquery.CueTaskRelationValues{
			Query:         query,
			Subject:       otherCue,
			Predicate:     otherCue,
			SubjectRole:   legalquery.CueSyntaxRoleTaskExpression,
			PredicateRole: legalquery.CueSyntaxRoleTaskExpression,
			ClauseSpan:    relations[0].ClauseSpan(),
			Kind:          legalquery.CueTaskRelationDirectTask,
		},
	)
	if err != nil {
		t.Fatalf("%s: 別 profile relation を構築できません: %v", coreSharedTerminalTaskResourceBindingID, err)
	}
	cues = append(cues, otherCue)
	sort.Slice(cues, func(left, right int) bool {
		if cues[left].Span().StartByte() != cues[right].Span().StartByte() {
			return cues[left].Span().StartByte() < cues[right].Span().StartByte()
		}
		leftIdentity := cues[left].ProfileID() + "\x00" + cues[left].CueID()
		rightIdentity := cues[right].ProfileID() + "\x00" + cues[right].CueID()
		return leftIdentity < rightIdentity
	})
	relations = append(relations, otherRelation)
	sort.Slice(relations, func(left, right int) bool {
		leftSubject := relations[left].Subject()
		rightSubject := relations[right].Subject()
		if relations[left].ClauseSpan().StartByte() !=
			relations[right].ClauseSpan().StartByte() {
			return relations[left].ClauseSpan().StartByte() <
				relations[right].ClauseSpan().StartByte()
		}
		if leftSubject.Span().StartByte() != rightSubject.Span().StartByte() {
			return leftSubject.Span().StartByte() < rightSubject.Span().StartByte()
		}
		return leftSubject.ProfileID() < rightSubject.ProfileID()
	})
	rebuilt, err := legalquery.NewPreprocessResult(
		legalquery.PreprocessResultValues{
			Query:                base.Query(),
			ComparisonKey:        base.ComparisonKey(),
			LawNameMentions:      base.LawNameMentions(),
			LegalConceptMentions: base.LegalConceptMentions(),
			CueMentions:          cues,
			IdentifierMentions:   base.IdentifierMentions(),
			DateMentions:         base.DateMentions(),
			ArticleMentions:      base.ArticleMentions(),
			ParagraphMentions:    base.ParagraphMentions(),
			CaseNumberMentions:   base.CaseNumberMentions(),
			QueryTermMentions:    base.QueryTermMentions(),
			CueTaskRelations:     relations,
		},
	)
	if err != nil {
		t.Fatalf("%s: 競合 relation fixture を再構築できません: %v", coreSharedTerminalTaskResourceBindingID, err)
	}
	return stage343ResolvedInput(t, profile, rebuilt)
}

func stage343TopicOption(
	t *testing.T,
	term string,
	startByte int,
	factID string,
	meaningKey string,
	code legalquery.EvidenceCode,
) coreTopicOption {
	t.Helper()
	layer := profileevidence.LayerSemanticExpansion
	if code == legalquery.EvidenceGeneralTerm ||
		code == legalquery.EvidenceOfficialAlias {
		layer = profileevidence.LayerTargetAnchor
	}
	option, err := newCoreTopicOption(
		term,
		startByte,
		[]profileevidence.EvidenceValues{{
			FactID:              factID,
			Layer:               layer,
			Code:                code,
			IndependentPositive: true,
			ClusterSpan:         true,
		}},
		nil,
		meaningKey,
	)
	if err != nil {
		t.Fatalf("stage 3.4.3 topic option を構築できません: %v", err)
	}
	return option
}

func stage343PreparedDraft(
	t *testing.T,
	cluster string,
	meaning string,
	score int,
	startByte int,
) coreSharedTerminalPreparedDraft {
	t.Helper()
	option := stage343TopicOption(
		t,
		meaning,
		startByte,
		"fact-"+meaning,
		meaning,
		legalquery.EvidenceGeneralTerm,
	)
	draft, err := buildCoreSharedTerminalDraft([]coreTopicOption{option})
	if err != nil {
		t.Fatalf("stage 3.4.3 prepared draft を構築できません: %v", err)
	}
	return coreSharedTerminalPreparedDraft{
		value:    draft,
		cluster:  cluster,
		evidence: []legalquery.EvidenceCode{legalquery.EvidenceGeneralTerm},
		score:    score,
	}
}
