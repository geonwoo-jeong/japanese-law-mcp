package legalquery

import (
	"errors"
	"reflect"
	"slices"
	"testing"
)

type candidate22MetadataMutation struct {
	field string
	apply func(*QueryProfileMetadata)
	equal bool
}

// SOT-MODEL-026/SOT-ENG-035: private field の変化を元の署名比較と独立に照合する。
func TestCandidate22MetadataEqualEveryPrivateField(t *testing.T) {
	mutations := candidate22MetadataMutations()
	covered := make(map[string]bool, len(mutations))
	for _, mutation := range mutations {
		covered[mutation.field] = true
		t.Run(mutation.field, func(t *testing.T) {
			base := candidate22Metadata(t)
			changed := candidate22CloneMetadata(base)
			mutation.apply(&changed)
			if queryProfileMetadataFieldsEqual(base, changed) {
				t.Fatal("変更した private field を直接比較が見逃しました")
			}
			candidate22AssertOldComparison(t, base, changed, mutation.equal)
			candidate22AssertOldComparison(t, changed, base, mutation.equal)
			candidate22AssertOldComparison(t, changed, changed, true)
		})
	}
	for _, typ := range []reflect.Type{
		reflect.TypeFor[QueryProfileMetadata](),
		reflect.TypeFor[QueryProfileTarget](),
		reflect.TypeFor[QueryScorePolicy](),
		reflect.TypeFor[QueryEvidenceWeight](),
		reflect.TypeFor[QuerySelectionPolicy](),
	} {
		for index := range typ.NumField() {
			field := typ.Name() + "." + typ.Field(index).Name
			if !covered[field] {
				t.Errorf("private field の変更試験がありません: %s", field)
			}
		}
	}
}

func candidate22MetadataMutations() []candidate22MetadataMutation {
	return []candidate22MetadataMutation{
		{"QueryProfileMetadata.schemaVersion", func(m *QueryProfileMetadata) { m.schemaVersion++ }, false},
		{"QueryProfileMetadata.profileID", func(m *QueryProfileMetadata) { m.profileID += "-changed" }, false},
		{"QueryProfileMetadata.profileVersion", func(m *QueryProfileMetadata) { m.profileVersion += "-changed" }, false},
		{"QueryProfileMetadata.rankingVersion", func(m *QueryProfileMetadata) { m.rankingVersion += "-changed" }, false},
		{"QueryProfileMetadata.cueSetVersion", func(m *QueryProfileMetadata) { m.cueSetVersion += "-changed" }, false},
		{"QueryProfileMetadata.lawNameLexiconVersion", func(m *QueryProfileMetadata) { m.lawNameLexiconVersion += "-changed" }, false},
		{"QueryProfileMetadata.legalConceptLexiconVersion", func(m *QueryProfileMetadata) { m.legalConceptLexiconVersion += "-changed" }, false},
		{"QueryProfileMetadata.targets", func(m *QueryProfileMetadata) { m.targets = m.targets[:1] }, false},
		{"QueryProfileMetadata.score", func(m *QueryProfileMetadata) { m.score = QueryScorePolicy{} }, false},
		{"QueryProfileMetadata.selection", func(m *QueryProfileMetadata) { m.selection = QuerySelectionPolicy{} }, false},
		{"QueryProfileMetadata.tieBreak", func(m *QueryProfileMetadata) { m.tieBreak = m.tieBreak[:1] }, false},
		{"QueryProfileMetadata.conditionalTieBreaks", func(m *QueryProfileMetadata) { m.conditionalTieBreaks = nil }, false},
		{"QueryProfileTarget.task", func(m *QueryProfileMetadata) { m.targets[0].task = TaskRead }, false},
		{"QueryProfileTarget.resource", func(m *QueryProfileMetadata) { m.targets[0].resource = ResourceJudicialDecision }, false},
		{"QueryProfileTarget.inputKind", func(m *QueryProfileMetadata) { m.targets[0].inputKind = InputKindLawRead }, false},
		{"QueryScorePolicy.minimum", func(m *QueryProfileMetadata) { m.score.minimum++ }, false},
		{"QueryScorePolicy.maximum", func(m *QueryProfileMetadata) { m.score.maximum++ }, false},
		{"QueryScorePolicy.highConfidenceAt", func(m *QueryProfileMetadata) { m.score.highConfidenceAt++ }, false},
		{"QueryScorePolicy.mediumConfidenceAt", func(m *QueryProfileMetadata) { m.score.mediumConfidenceAt++ }, false},
		{"QueryScorePolicy.evidenceWeights", func(m *QueryProfileMetadata) { m.score.evidenceWeights = m.score.evidenceWeights[:1] }, false},
		{"QueryEvidenceWeight.code", func(m *QueryProfileMetadata) { m.score.evidenceWeights[0].code = EvidenceCode("変更") }, false},
		{"QueryEvidenceWeight.weight", func(m *QueryProfileMetadata) { m.score.evidenceWeights[0].weight++ }, false},
		{"QuerySelectionPolicy.singleThreshold", func(m *QueryProfileMetadata) { m.selection.singleThreshold++ }, false},
		{"QuerySelectionPolicy.minimumExecutionThreshold", func(m *QueryProfileMetadata) { m.selection.minimumExecutionThreshold++ }, false},
		{"QuerySelectionPolicy.singleMargin", func(m *QueryProfileMetadata) { m.selection.singleMargin++ }, false},
		{"QuerySelectionPolicy.hedgeMargin", func(m *QueryProfileMetadata) { m.selection.hedgeMargin++ }, false},
		{"QuerySelectionPolicy.branchRetentionMargin", func(m *QueryProfileMetadata) { m.selection.branchRetentionMargin++ }, false},
		{"QuerySelectionPolicy.branchRetentionPresent", func(m *QueryProfileMetadata) { m.selection.branchRetentionPresent = false }, false},
		{"QuerySelectionPolicy.scoreMinimum", func(m *QueryProfileMetadata) { m.selection.scoreMinimum++ }, true},
		{"QuerySelectionPolicy.scoreMaximum", func(m *QueryProfileMetadata) { m.selection.scoreMaximum++ }, true},
	}
}

func TestCandidate22MetadataEqualOrderAndConditionalKeys(t *testing.T) {
	mutations := []struct {
		name  string
		apply func(*QueryProfileMetadata)
	}{
		{"target順", func(m *QueryProfileMetadata) {
			m.targets[0], m.targets[1] = m.targets[1], m.targets[0]
		}},
		{"weight順", func(m *QueryProfileMetadata) {
			m.score.evidenceWeights[0], m.score.evidenceWeights[1] = m.score.evidenceWeights[1], m.score.evidenceWeights[0]
		}},
		{"tie順", func(m *QueryProfileMetadata) {
			m.tieBreak[0], m.tieBreak[1] = m.tieBreak[1], m.tieBreak[0]
		}},
		{"条件名", func(m *QueryProfileMetadata) {
			name := ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit
			order := m.conditionalTieBreaks[name]
			delete(m.conditionalTieBreaks, name)
			m.conditionalTieBreaks["別条件"] = order
		}},
		{"条件順", func(m *QueryProfileMetadata) {
			order := m.conditionalTieBreaks[ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit]
			order[0], order[1] = order[1], order[0]
		}},
		{"条件値", func(m *QueryProfileMetadata) {
			m.conditionalTieBreaks[ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit][0] = QueryTieBreak("変更")
		}},
		{"条件要素数", func(m *QueryProfileMetadata) {
			name := ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit
			m.conditionalTieBreaks[name] = m.conditionalTieBreaks[name][:1]
		}},
		{"空値を持つ条件追加", func(m *QueryProfileMetadata) {
			m.conditionalTieBreaks["別条件"] = nil
		}},
	}
	for _, mutation := range mutations {
		t.Run(mutation.name, func(t *testing.T) {
			base := candidate22Metadata(t)
			changed := candidate22CloneMetadata(base)
			mutation.apply(&changed)
			if queryProfileMetadataFieldsEqual(base, changed) {
				t.Fatal("順序または条件の変更を直接比較が見逃しました")
			}
			candidate22AssertOldComparison(t, base, changed, false)
			candidate22AssertOldComparison(t, changed, base, false)
		})
	}
	left := candidate22Metadata(t)
	right := candidate22CloneMetadata(left)
	left.conditionalTieBreaks = map[ConditionalTieBreakName][]QueryTieBreak{}
	right.conditionalTieBreaks = map[ConditionalTieBreakName][]QueryTieBreak{}
	// 未知条件を含む値も比較自体は従来署名と同値にする。妥当性検証は別の責務。
	for _, name := range []ConditionalTieBreakName{"alpha", "omega"} {
		left.conditionalTieBreaks[name] = []QueryTieBreak{QueryTieBreakEvidenceSet}
	}
	for _, name := range []ConditionalTieBreakName{"omega", "alpha"} {
		right.conditionalTieBreaks[name] = []QueryTieBreak{QueryTieBreakEvidenceSet}
	}
	if !queryProfileMetadataFieldsEqual(left, right) {
		t.Fatal("map の挿入順だけで直接比較が異なりました")
	}
	candidate22AssertOldComparison(t, left, right, true)
}

func TestCandidate22MetadataEqualPreservesCanonicalFallback(t *testing.T) {
	tests := []struct {
		name string
		pair func() (QueryProfileMetadata, QueryProfileMetadata)
	}{
		{"targetのnilと空", func() (QueryProfileMetadata, QueryProfileMetadata) {
			return QueryProfileMetadata{}, QueryProfileMetadata{targets: []QueryProfileTarget{}}
		}},
		{"weightのnilと空", func() (QueryProfileMetadata, QueryProfileMetadata) {
			return QueryProfileMetadata{}, QueryProfileMetadata{score: QueryScorePolicy{evidenceWeights: []QueryEvidenceWeight{}}}
		}},
		{"tieのnilと空", func() (QueryProfileMetadata, QueryProfileMetadata) {
			return QueryProfileMetadata{}, QueryProfileMetadata{tieBreak: []QueryTieBreak{}}
		}},
		{"条件mapのnilと空", func() (QueryProfileMetadata, QueryProfileMetadata) {
			return QueryProfileMetadata{}, QueryProfileMetadata{conditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{}}
		}},
		{"条件値のnilと空", func() (QueryProfileMetadata, QueryProfileMetadata) {
			return QueryProfileMetadata{conditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{"条件": nil}},
				QueryProfileMetadata{conditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{"条件": {}}}
		}},
		{"非存在marginの内部値", func() (QueryProfileMetadata, QueryProfileMetadata) {
			left := candidate22Metadata(t)
			left.schemaVersion = 1
			left.selection.branchRetentionPresent = false
			left.selection.branchRetentionMargin = 0
			right := candidate22CloneMetadata(left)
			right.selection.branchRetentionMargin = 99
			return left, right
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left, right := test.pair()
			if queryProfileMetadataFieldsEqual(left, right) {
				t.Fatal("従来経路へ戻す異なる内部値を直接受理しました")
			}
			candidate22AssertOldComparison(t, left, right, true)
			candidate22AssertOldComparison(t, right, left, true)
		})
	}
	missing := QueryProfileMetadata{conditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{"first": nil}}
	other := QueryProfileMetadata{conditionalTieBreaks: map[ConditionalTieBreakName][]QueryTieBreak{"second": nil}}
	candidate22AssertOldComparison(t, missing, other, false)
}

func TestCandidate22MetadataEqualIndependentCopiesAndZero(t *testing.T) {
	for _, base := range []QueryProfileMetadata{candidate22Metadata(t), {}} {
		copy := candidate22CloneMetadata(base)
		if !queryProfileMetadataFieldsEqual(base, copy) {
			t.Fatal("同じ値を別の backing array に複製しても直接比較が一致しません")
		}
		candidate22AssertOldComparison(t, base, copy, true)
	}
}

func candidate22AssertOldComparison(t *testing.T, left, right QueryProfileMetadata, want bool) {
	t.Helper()
	// 新 helper を呼ばない、変更前から存在する署名生成器を oracle とする。
	old := queryProfileMetadataSignature(left) == queryProfileMetadataSignature(right)
	if old != want {
		t.Fatalf("固定した従来比較の期待値が異なります: got=%t want=%t", old, want)
	}
	if got := queryProfileMetadataEqual(left, right); got != old {
		t.Fatalf("候補の比較が従来署名と一致しません: got=%t old=%t", got, old)
	}
}

func candidate22Metadata(t *testing.T) QueryProfileMetadata {
	t.Helper()
	values := selectorTestMetadataValues(t, "core", "candidate22-v1", selectorTestRankingVersion)
	values.SchemaVersion = 2
	selection, err := NewQuerySelectionPolicy(QuerySelectionPolicyValues{
		SingleThreshold: 120, MinimumExecutionThreshold: 80, SingleMargin: 25, HedgeMargin: 10,
		BranchRetentionMargin: 12, BranchRetentionPresent: true,
		ScoreMinimum: values.Score.Minimum(), ScoreMaximum: values.Score.Maximum(),
	})
	if err != nil {
		t.Fatalf("試験用 selection を作成できません: %v", err)
	}
	values.Selection = selection
	target, err := NewQueryProfileTarget(QueryProfileTargetValues{
		Task: TaskRead, Resource: ResourceLaw, InputKind: InputKindLawRead,
	})
	if err != nil {
		t.Fatalf("試験用 target を作成できません: %v", err)
	}
	values.Targets = append(values.Targets, target)
	values.ConditionalTieBreaks = map[ConditionalTieBreakName][]QueryTieBreak{
		ConditionalTieBreakLawAliasCollisionGroupsOverCandidateLimit: {
			QueryTieBreakEvidenceSet, QueryTieBreakStepCount,
			QueryTieBreakSourcePosition, QueryTieBreakMeaningSignature,
		},
	}
	metadata, err := NewQueryProfileMetadata(values)
	if err != nil {
		t.Fatalf("試験用 metadata を作成できません: %v", err)
	}
	return metadata
}

func candidate22CloneMetadata(value QueryProfileMetadata) QueryProfileMetadata {
	cloned := value
	cloned.targets = slices.Clone(value.targets)
	cloned.score.evidenceWeights = slices.Clone(value.score.evidenceWeights)
	cloned.tieBreak = slices.Clone(value.tieBreak)
	if value.conditionalTieBreaks != nil {
		cloned.conditionalTieBreaks = make(map[ConditionalTieBreakName][]QueryTieBreak, len(value.conditionalTieBreaks))
		for name, order := range value.conditionalTieBreaks {
			cloned.conditionalTieBreaks[name] = slices.Clone(order)
		}
	}
	return cloned
}

type candidate22ObservedProfile struct {
	metadata    QueryProfileMetadata
	name        string
	events      *[]string
	calls       int
	changeAt    int
	generateErr error
}

func (p *candidate22ObservedProfile) Metadata() QueryProfileMetadata {
	p.calls++
	*p.events = append(*p.events, p.name+".metadata")
	value := p.metadata
	if p.calls == p.changeAt {
		// ID は維持したまま、有効な private scalar だけを変える。
		value.cueSetVersion += "-changed"
	}
	return value
}

func (*candidate22ObservedProfile) CueVocabulary() []CueVocabularyEntry {
	return nil
}

func (p *candidate22ObservedProfile) Generate(
	CandidateGenerationInput,
	CandidateIDScope,
) (CandidateGeneration, error) {
	*p.events = append(*p.events, p.name+".generate")
	if p.generateErr != nil {
		return CandidateGeneration{}, p.generateErr
	}
	return NewCandidateGeneration(CandidateGenerationValues{
		ProfileID: p.metadata.ProfileID(), ProfileVersion: p.metadata.ProfileVersion(),
		RankingVersion: p.metadata.RankingVersion(), SelectionMode: QuerySelectionModeAutomatic,
	})
}

func TestCandidate22MetadataCallbacksAndErrorsStayOrdered(t *testing.T) {
	cause := errors.New("試験用生成失敗")
	tests := []struct {
		name        string
		changeAt    int
		generateErr error
		wantError   string
		wantEvents  []string
	}{
		{"正常", 0, nil, "", []string{"core.metadata", "core.metadata", "core.metadata", "core.generate", "core.metadata"}},
		{"set再検証", 2, nil,
			"profile set が有効ではありません: profiles[0] の metadata が構築時と一致しません",
			[]string{"core.metadata", "core.metadata"}},
		{"生成直前", 3, nil,
			"profiles[0] の metadata が構築後に変更されました",
			[]string{"core.metadata", "core.metadata", "core.metadata"}},
		{"生成直後", 4, nil,
			"profiles[0] の contribution を回収できません: profile metadata が候補生成中に変更されました",
			[]string{"core.metadata", "core.metadata", "core.metadata", "core.generate", "core.metadata"}},
		{"生成失敗", 0, cause,
			"profiles[0] の contribution を回収できません: 試験用生成失敗",
			[]string{"core.metadata", "core.metadata", "core.metadata", "core.generate"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var events []string
			profile := &candidate22ObservedProfile{
				metadata: candidate22Metadata(t), name: "core", events: &events,
				changeAt: test.changeAt, generateErr: test.generateErr,
			}
			set, err := NewQueryProfileSet([]QueryProfile{profile})
			if err != nil {
				t.Fatalf("試験用 set を構築できません: %v", err)
			}
			_, err = set.Collect(mustSelectorTestPreprocessResult(t))
			actual := ""
			if err != nil {
				actual = err.Error()
			}
			if actual != test.wantError || !slices.Equal(events, test.wantEvents) {
				t.Fatalf("失敗箇所または callback 順が変わりました: error=%q events=%v", actual, events)
			}
			if test.generateErr != nil && !errors.Is(err, cause) {
				t.Fatal("生成失敗の error chain を失いました")
			}
		})
	}
}

func TestCandidate22MetadataCallbacksKeepProfileOrder(t *testing.T) {
	var events []string
	first := &candidate22ObservedProfile{metadata: candidate22Metadata(t), name: "first", events: &events}
	second := &candidate22ObservedProfile{metadata: candidate22Metadata(t), name: "second", events: &events}
	second.metadata.profileID = "judicial"
	set, err := NewQueryProfileSet([]QueryProfile{first, second})
	if err != nil {
		t.Fatalf("試験用の二 profile set を構築できません: %v", err)
	}
	if _, err := set.Collect(mustSelectorTestPreprocessResult(t)); err != nil {
		t.Fatalf("二 profile の回収に失敗しました: %v", err)
	}
	want := []string{
		"first.metadata", "second.metadata",
		"first.metadata", "second.metadata",
		"first.metadata", "first.generate", "first.metadata",
		"second.metadata", "second.generate", "second.metadata",
	}
	if !slices.Equal(events, want) {
		t.Fatalf("profile の callback 順が変わりました: %v", events)
	}
}
