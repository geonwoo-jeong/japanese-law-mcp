package core

import (
	"reflect"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const (
	coreEvidenceMappingPrivateLifetimeID     = "core-evidence-mapping-private-lifetime"
	coreEvidenceMappingTopicPositiveID       = "core-evidence-mapping-topic-positive"
	coreEvidenceMappingRefNoSpanID           = "core-evidence-mapping-ref-no-span"
	coreEvidenceMappingProviderIndependentID = "core-evidence-mapping-provider-independent"
	coreLawNameContentProjectionID           = "core-law-name-content-projection"
)

func TestCoreEvidenceProfileはTest専用経路だけを有効化する(
	t *testing.T,
) {
	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf(
			"%s: active profile を読み込めません: %v",
			coreEvidenceMappingPrivateLifetimeID,
			err,
		)
	}
	activeMargin, activeMarginPresent :=
		active.Metadata().Selection().BranchRetentionMargin()
	if active.Metadata().SchemaVersion() != 1 ||
		activeMarginPresent ||
		activeMargin != 0 ||
		active.intentEvidenceMode != cueIntentEvidenceLegacy {
		t.Fatalf(
			"%s: active profile が変更されています",
			coreEvidenceMappingPrivateLifetimeID,
		)
	}
	activeGeneration := generateCoreEvidenceQuery(
		t,
		active,
		"永住許可、帰化を教えてください",
		nil,
		coreEvidenceMappingPrivateLifetimeID,
	)
	for _, candidate := range activeGeneration.Candidates() {
		if len(candidate.Steps()) > 1 {
			t.Fatalf(
				"%s: active profile が共有末尾列を複数 step へ使用しました",
				coreEvidenceMappingPrivateLifetimeID,
			)
		}
	}

	next := mustCoreEvidenceProfile(t)
	nextMargin, nextMarginPresent :=
		next.Metadata().Selection().BranchRetentionMargin()
	if next.Metadata().SchemaVersion() != 2 ||
		!nextMarginPresent ||
		nextMargin != 12 {
		t.Fatalf(
			"%s: test 専用 profile の metadata = schema:%d margin:(%d,%t)",
			coreEvidenceMappingPrivateLifetimeID,
			next.Metadata().SchemaVersion(),
			nextMargin,
			nextMarginPresent,
		)
	}

	first := generateCoreEvidenceQuery(
		t,
		next,
		"永住許可、帰化を教えてください",
		nil,
		coreEvidenceMappingPrivateLifetimeID,
	)
	_ = generateCoreEvidenceQuery(
		t,
		next,
		"民法第709条を読んでください。",
		nil,
		coreEvidenceMappingPrivateLifetimeID,
	)
	repeated := generateCoreEvidenceQuery(
		t,
		next,
		"永住許可、帰化を教えてください",
		nil,
		coreEvidenceMappingPrivateLifetimeID,
	)
	if !reflect.DeepEqual(first, repeated) {
		t.Fatalf(
			"%s: request 間で private mapping が残りました",
			coreEvidenceMappingPrivateLifetimeID,
		)
	}

	again, err := LoadEmbedded()
	if err != nil {
		t.Fatalf(
			"%s: active profile を再読込できません: %v",
			coreEvidenceMappingPrivateLifetimeID,
			err,
		)
	}
	if again.Metadata().SchemaVersion() != 1 ||
		again.intentEvidenceMode != cueIntentEvidenceLegacy {
		t.Fatalf(
			"%s: test 専用経路が active profile を変更しました",
			coreEvidenceMappingPrivateLifetimeID,
		)
	}
}

func TestCoreEvidenceProfileは共有末尾の二主題を本文検索Stepへ分離する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	for _, query := range []string{
		"永住許可、帰化を教えてください",
		"永住許可と帰化について教えてください",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			generation := generateCoreEvidenceQuery(
				t,
				profile,
				query,
				nil,
				coreEvidenceMappingTopicPositiveID,
			)
			assertSingleContentCandidate(
				t,
				generation,
				[][]string{{"永住許可"}, {"帰化"}},
				coreEvidenceMappingTopicPositiveID,
			)
		})
	}
}

func TestCoreEvidenceProfileは共有末尾の四主題と五主題上限を区別する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	four := generateCoreEvidenceQuery(
		t,
		profile,
		"永住許可、帰化、営業秘密、個人情報を教えてください",
		nil,
		coreSharedTerminalEvidenceClusterID,
	)
	candidates := four.Candidates()
	if len(candidates) != 1 || len(candidates[0].Steps()) != 4 {
		t.Fatalf(
			"%s: 四主題の候補 = %#v",
			coreSharedTerminalEvidenceClusterID,
			candidates,
		)
	}

	five := generateCoreEvidenceQuery(
		t,
		profile,
		"永住許可、帰化、営業秘密、個人情報、育児休業を教えてください",
		nil,
		coreSharedTerminalEvidenceClusterID,
	)
	if len(five.Candidates()) != 0 ||
		five.CompositionConstraint() !=
			legalquery.QueryCompositionConstraintStepLimitExceeded {
		t.Fatalf(
			"%s: 五主題の上限結果 = candidates:%#v constraint:%q",
			coreSharedTerminalEvidenceClusterID,
			five.Candidates(),
			five.CompositionConstraint(),
		)
	}
}

func TestCoreEvidenceProfileは異なるSpanの同じ意味を一Stepへ縮約する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	two := generateCoreEvidenceQuery(
		t,
		profile,
		"帰化、帰化を教えてください",
		nil,
		coreSharedTerminalEvidenceClusterID,
	)
	assertSingleContentCandidate(
		t,
		two,
		[][]string{{"帰化"}},
		coreSharedTerminalEvidenceClusterID,
	)

	fiveWithRepeatedMeaning := generateCoreEvidenceQuery(
		t,
		profile,
		"量子相続、月面抵当、火星登記、海底供託、月面抵当について教えてください",
		nil,
		coreSharedTerminalEvidenceClusterID,
	)
	assertSingleContentCandidate(
		t,
		fiveWithRepeatedMeaning,
		[][]string{
			{"量子相続"},
			{"月面抵当"},
			{"火星登記"},
			{"海底供託"},
		},
		coreSharedTerminalEvidenceClusterID,
	)
	if fiveWithRepeatedMeaning.SelectionMode() !=
		legalquery.QuerySelectionModeAutomatic {
		t.Fatalf(
			"%s: 同値縮約後の selectionMode = %q",
			coreSharedTerminalEvidenceClusterID,
			fiveWithRepeatedMeaning.SelectionMode(),
		)
	}
}

func TestCoreEvidenceProfileは法令名を許可した三経路だけで本文検索語にする(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	tests := []struct {
		name      string
		query     string
		wantTerms [][]string
	}{
		{
			name:      "同じspanの引用句",
			query:     "法令本文から「民法」を検索してください。",
			wantTerms: [][]string{{"民法"}},
		},
		{
			name:      "明示した個別主題",
			query:     "民法と商法を個別に法令本文で検索してください。",
			wantTerms: [][]string{{"民法"}, {"商法"}},
		},
		{
			name:      "共有末尾主題",
			query:     "民法、商法を教えてください",
			wantTerms: [][]string{{"民法"}, {"商法"}},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			generation := generateCoreEvidenceQuery(
				t,
				profile,
				test.query,
				nil,
				coreLawNameContentProjectionID,
			)
			assertSingleContentCandidate(
				t,
				generation,
				test.wantTerms,
				coreLawNameContentProjectionID,
			)
		})
	}
}

func TestCoreEvidenceProfileは法令名投影の禁止対象を本文検索語にしない(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	tests := []struct {
		name          string
		query         string
		forbiddenTerm string
	}{
		{
			name:          "裸の法令名",
			query:         "民法を法令本文で検索してください。",
			forbiddenTerm: "民法",
		},
		{
			name:          "一意な誤記補正だけ",
			query:         "労契去を個別に法令本文で検索してください。",
			forbiddenTerm: "労契去",
		},
		{
			name:          "法令ID",
			query:         "法令ID 129AC0000000089 を個別に法令本文で検索してください。",
			forbiddenTerm: "129AC0000000089",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			generation := generateCoreEvidenceQuery(
				t,
				profile,
				test.query,
				nil,
				coreLawNameContentProjectionID,
			)
			for _, candidate := range generation.Candidates() {
				for _, step := range candidate.Steps() {
					content, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
					if ok && slices.Contains(
						content.AllTerms(),
						test.forbiddenTerm,
					) {
						t.Fatalf(
							"%s: 禁止対象 %q を本文検索語へ投影しました",
							coreLawNameContentProjectionID,
							test.forbiddenTerm,
						)
					}
				}
			}
		})
	}
}

func TestCoreEvidenceProfileは同じ表記の複数LawIdentityを一検索に縮約する(
	t *testing.T,
) {
	const query = "民法、商法を教えてください"

	profile := mustCoreEvidenceProfile(t)
	preprocessed := preprocessCoreEvidenceQuery(
		t,
		profile,
		query,
		nil,
		coreLawNameContentProjectionID,
	)
	laws := preprocessed.LawNameMentions()
	if len(laws) != 2 {
		t.Fatalf(
			"%s: fixture の lawNameMentions = %#v",
			coreLawNameContentProjectionID,
			laws,
		)
	}
	first := laws[0]
	other, err := legalquery.NewLawNameMention(
		legalquery.LawNameMentionValues{
			Span:       first.Span(),
			Surface:    first.Surface(),
			LawID:      "test-second-law",
			RevisionID: "test-second-revision",
			LawNumber:  "令和八年法律第百号",
			Canonical:  "試験用第二民法",
			MatchKind:  legalquery.PreprocessMatchRegisteredTerm,
		},
	)
	if err != nil {
		t.Fatalf(
			"%s: 複数 identity fixture を作成できません: %v",
			coreLawNameContentProjectionID,
			err,
		)
	}
	laws = append(laws, other)
	sort.Slice(laws, func(left int, right int) bool {
		if laws[left].Span() != laws[right].Span() {
			if laws[left].Span().StartByte() !=
				laws[right].Span().StartByte() {
				return laws[left].Span().StartByte() <
					laws[right].Span().StartByte()
			}
			return laws[left].Span().EndByte() >
				laws[right].Span().EndByte()
		}
		return laws[left].LawID() < laws[right].LawID()
	})

	input := rebuildCoreEvidenceInput(
		t,
		preprocessed,
		laws,
		preprocessed.LegalConceptMentions(),
		preprocessed.QueryTermMentions(),
		coreLawNameContentProjectionID,
	)
	generation := generateCoreEvidenceInput(
		t,
		profile,
		input,
		coreLawNameContentProjectionID,
	)
	assertSingleContentCandidate(
		t,
		generation,
		[][]string{{"民法"}, {"商法"}},
		coreLawNameContentProjectionID,
	)
}

func TestCoreEvidenceProfileは五InputKindの代表候補を生成する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	tests := []struct {
		name  string
		query string
		kind  legalquery.LogicalInputKind
	}{
		{
			name:  "法令検索",
			query: "民法という法令を検索してください。",
			kind:  legalquery.InputKindLawSearch,
		},
		{
			name:  "法令本文検索",
			query: "法令本文から「営業秘密」を検索してください。",
			kind:  legalquery.InputKindLawContentSearch,
		},
		{
			name:  "法令読取り",
			query: "法令ID 129AC0000000089 の本文を読んでください。",
			kind:  legalquery.InputKindLawRead,
		},
		{
			name:  "条文読取り",
			query: "法令ID 129AC0000000089 の第709条を読んでください。",
			kind:  legalquery.InputKindLawArticleRead,
		},
		{
			name:  "更新一覧",
			query: "2024年4月1日の法令更新一覧を列挙して",
			kind:  legalquery.InputKindLawUpdates,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			generation := generateCoreEvidenceQuery(
				t,
				profile,
				test.query,
				nil,
				coreEvidenceMappingInputKindsID,
			)
			if !generationContainsInputKind(generation, test.kind) {
				t.Fatalf(
					"%s: %q の候補に inputKind %q がありません: %#v",
					coreEvidenceMappingInputKindsID,
					test.query,
					test.kind,
					generation.Candidates(),
				)
			}
		})
	}
}

func TestCoreEvidenceProfileはRefにSpanを捏造せずProviderにも依存しない(
	t *testing.T,
) {
	const query = "この参照を読んでください。"

	profile := mustCoreEvidenceProfile(t)
	firstRef := mustCoreEvidenceLawRef(
		t,
		"provider-one",
		"source-one",
		coreEvidenceMappingRefNoSpanID,
	)
	secondRef := mustCoreEvidenceLawRef(
		t,
		"provider-two",
		"source-two",
		coreEvidenceMappingProviderIndependentID,
	)
	first := generateCoreEvidenceQuery(
		t,
		profile,
		query,
		&firstRef,
		coreEvidenceMappingRefNoSpanID,
	)
	second := generateCoreEvidenceQuery(
		t,
		profile,
		query,
		&secondRef,
		coreEvidenceMappingProviderIndependentID,
	)

	firstCandidate, firstOrigin := singleRefReadCandidate(
		t,
		first,
		coreEvidenceMappingRefNoSpanID,
	)
	secondCandidate, secondOrigin := singleRefReadCandidate(
		t,
		second,
		coreEvidenceMappingProviderIndependentID,
	)
	wantOrigin := strings.Index(query, "読んでください")
	if firstOrigin != wantOrigin || secondOrigin != wantOrigin {
		t.Fatalf(
			"%s: ref の代替 span = first:%d second:%d want:%d",
			coreEvidenceMappingRefNoSpanID,
			firstOrigin,
			secondOrigin,
			wantOrigin,
		)
	}
	if firstCandidate.SemanticScore() != secondCandidate.SemanticScore() ||
		firstCandidate.Confidence() != secondCandidate.Confidence() ||
		!slices.Equal(
			firstCandidate.EvidenceCodes(),
			secondCandidate.EvidenceCodes(),
		) ||
		first.SelectionMode() != second.SelectionMode() {
		t.Fatalf(
			"%s: provider metadata で候補評価が変わりました",
			coreEvidenceMappingProviderIndependentID,
		)
	}
}

func TestCoreEvidenceProfileは共有末尾ClusterへBranchRetentionMarginを適用する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	margin, present :=
		profile.Metadata().Selection().BranchRetentionMargin()
	if !present || margin != 12 {
		t.Fatalf(
			"%s: branchRetentionMargin = (%d,%t)",
			coreSharedTerminalEvidenceClusterID,
			margin,
			present,
		)
	}

	insideInput := syntheticSharedTerminalConceptInput(
		t,
		profile,
		2,
		-1,
		coreSharedTerminalEvidenceClusterID,
	)
	inside := generateCoreEvidenceInput(
		t,
		profile,
		insideInput,
		coreSharedTerminalEvidenceClusterID,
	)
	insideCandidates := inside.Candidates()
	if len(insideCandidates) != 2 ||
		scoreRange(insideCandidates) > margin {
		t.Fatalf(
			"%s: margin 内候補 = %#v",
			coreSharedTerminalEvidenceClusterID,
			insideCandidates,
		)
	}

	outsideInput := syntheticSharedTerminalConceptInput(
		t,
		profile,
		2,
		1,
		coreSharedTerminalEvidenceClusterID,
	)
	outside := generateCoreEvidenceInput(
		t,
		profile,
		outsideInput,
		coreSharedTerminalEvidenceClusterID,
	)
	outsideCandidates := outside.Candidates()
	if len(outsideCandidates) != 1 ||
		!slices.Contains(
			outsideCandidates[0].EvidenceCodes(),
			legalquery.EvidenceUniqueTypoCorrection,
		) {
		t.Fatalf(
			"%s: margin 外候補を保持しました: %#v",
			coreSharedTerminalEvidenceClusterID,
			outsideCandidates,
		)
	}
}

func TestCoreEvidenceProfileは共有末尾Clusterの第四分岐で明確化する(
	t *testing.T,
) {
	profile := mustCoreEvidenceProfile(t)
	input := syntheticSharedTerminalConceptInput(
		t,
		profile,
		4,
		-1,
		coreSharedTerminalEvidenceClusterID,
	)
	generation := generateCoreEvidenceInput(
		t,
		profile,
		input,
		coreSharedTerminalEvidenceClusterID,
	)
	if len(generation.Candidates()) != 3 ||
		generation.SelectionMode() !=
			legalquery.QuerySelectionModeClarificationRequired ||
		len(generation.HedgePairs()) != 0 {
		t.Fatalf(
			"%s: 第四分岐の結果 = candidates:%#v mode:%q hedge:%#v",
			coreSharedTerminalEvidenceClusterID,
			generation.Candidates(),
			generation.SelectionMode(),
			generation.HedgePairs(),
		)
	}
}
