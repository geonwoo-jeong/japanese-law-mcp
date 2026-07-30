package core

import (
	"context"
	_ "embed"
	"slices"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

var (
	//go:embed testdata/relation-v2/profile.json
	relationV2ProfileJSON []byte

	//go:embed testdata/relation-v2/cues.json
	relationV2CuesJSON []byte
)

func TestRelationV2ProfileはPositiveCueRoleの完全対応を固定する(
	t *testing.T,
) {
	t.Parallel()

	profile := mustRelationV2Profile(t)
	for cueID, definition := range profile.cueByID {
		if definition.category == "unsupported" {
			continue
		}
		want := legalquery.CueSyntaxRoleNone
		switch {
		case definition.category == "task" &&
			(definition.value == "search" ||
				definition.value == "read" ||
				definition.value == "list_updates"):
			want = legalquery.CueSyntaxRoleTaskExpression
		case definition.category == "syntax" &&
			definition.value == "task_predicate":
			want = legalquery.CueSyntaxRoleTaskPredicate
		}
		if definition.syntaxRole != want {
			t.Fatalf(
				"positive-cue-value-role-mapping: cue %q (%s/%s) の role = %q、期待値は %q",
				cueID,
				definition.category,
				definition.value,
				definition.syntaxRole,
				want,
			)
		}
	}

	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("active profile を読み込めません: %v", err)
	}
	activeMargin, activeMarginPresent :=
		active.Metadata().Selection().BranchRetentionMargin()
	nextMargin, nextMarginPresent :=
		profile.Metadata().Selection().BranchRetentionMargin()
	if active.Metadata().SchemaVersion() != 1 ||
		activeMarginPresent ||
		activeMargin != 0 ||
		active.Metadata().ProfileVersion() != "core-2026-07-30-33" ||
		active.Metadata().CueSetVersion() != "core-cues-2026-07-30-15" ||
		profile.Metadata().SchemaVersion() != 2 ||
		profile.Metadata().ProfileVersion() != "core-2026-07-31-35" ||
		profile.Metadata().RankingVersion() !=
			"legal-query-ranking-2026-07-31-2" ||
		!nextMarginPresent ||
		nextMargin != 12 ||
		profile.Metadata().CueSetVersion() != "core-cues-2026-07-31-16" {
		t.Fatalf(
			"SOT-ARCH-033: active=%q/%q next=%q/%q",
			active.Metadata().ProfileVersion(),
			active.Metadata().CueSetVersion(),
			profile.Metadata().ProfileVersion(),
			profile.Metadata().CueSetVersion(),
		)
	}
}

func TestRelationV2Inputは共有末尾列をCandidateGenerationInputへ渡す(
	t *testing.T,
) {
	t.Parallel()

	profile := mustRelationV2Profile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("relation v2 preprocessor を構築できません: %v", err)
	}
	for _, query := range []string{
		"永住許可、帰化を教えてください",
		"永住許可と帰化について教えてください",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()
			request, requestErr := legalquery.NewRequest(
				legalquery.RequestValues{Query: query},
			)
			if requestErr != nil {
				t.Fatalf("request を構築できません: %v", requestErr)
			}
			preprocessed, preprocessErr := preprocessor.Preprocess(
				context.Background(),
				request,
			)
			if preprocessErr != nil {
				t.Fatalf("Preprocess() のエラー = %v", preprocessErr)
			}
			input, inputErr := legalquery.NewCandidateGenerationInput(
				preprocessed,
			)
			if inputErr != nil {
				t.Fatalf(
					"candidate generation input を構築できません: %v",
					inputErr,
				)
			}
			sequences := input.SharedTerminalSequences()
			if len(sequences) != 1 {
				t.Fatalf(
					"sidecar 件数 = %d、law=%#v concept=%#v term=%#v cue=%#v relation=%#v",
					len(sequences),
					preprocessed.LawNameMentions(),
					preprocessed.LegalConceptMentions(),
					preprocessed.QueryTermMentions(),
					preprocessed.CueMentions(),
					preprocessed.CueTaskRelations(),
				)
			}
			spans := sequences[0].TopicSpans()
			if len(spans) != 2 ||
				query[spans[0].StartByte():spans[0].EndByte()] != "永住許可" ||
				query[spans[1].StartByte():spans[1].EndByte()] != "帰化" {
				t.Fatalf("topic spans = %#v", spans)
			}
			if sequences[0].TerminalTaskRelation().Kind() !=
				legalquery.CueTaskRelationDirectTask {
				t.Fatalf(
					"terminal relation = %#v",
					sequences[0].TerminalTaskRelation(),
				)
			}
		})
	}
}

func TestRelationV2ProfileはTaskPredicateRoleを拒否する(
	t *testing.T,
) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	profile, err := Load(
		relationV2ProfileJSON,
		relationV2CuesJSON,
		lawNames,
		concepts,
	)
	if err != nil {
		t.Fatalf("relation v2 profile を読み込めません: %v", err)
	}
	definitions := make(map[string]cueDefinition, len(profile.cueByID))
	for cueID, definition := range profile.cueByID {
		definitions[cueID] = definition
	}
	search := definitions["task-search"]
	search.syntaxRole = legalquery.CueSyntaxRoleTaskPredicate
	definitions["task-search"] = search
	changed := *profile
	changed.cueByID = definitions
	if _, err := newCueTaskRelationV2Profile(&changed); err == nil {
		t.Fatal(
			"positive-cue-value-role-mapping: task_predicate を受理しました",
		)
	}
	if profile.cueByID["task-search"].syntaxRole !=
		legalquery.CueSyntaxRoleTaskExpression {
		t.Fatal("元の profile を変更しました")
	}
}

func TestRelationV2Profileは元Profileの内部状態を共有しない(
	t *testing.T,
) {
	t.Parallel()

	lawNames, concepts := mustEmbeddedLexicons(t)
	original, err := Load(
		relationV2ProfileJSON,
		relationV2CuesJSON,
		lawNames,
		concepts,
	)
	if err != nil {
		t.Fatalf("relation v2 profile を読み込めません: %v", err)
	}
	cloned, err := newCueTaskRelationV2Profile(original)
	if err != nil {
		t.Fatalf("relation v2 profile を準備できません: %v", err)
	}

	originalCueID := original.cues[0].CueID
	originalTerm := original.cues[0].Terms[0]
	originalDefinition := original.cueByID["task-search"]
	var conceptID string
	for current := range original.concepts {
		conceptID = current
		break
	}
	if conceptID == "" {
		t.Fatal("clone 検証に必要な concept がありません")
	}
	cloned.cues[0].CueID = ""
	cloned.cues[0].Terms[0] = ""
	delete(cloned.cueByID, "task-search")
	delete(cloned.concepts, conceptID)

	if original.cues[0].CueID != originalCueID ||
		original.cues[0].Terms[0] != originalTerm {
		t.Fatal("next profile の cue slice が元 profile と共有されています")
	}
	if original.cueByID["task-search"] != originalDefinition {
		t.Fatal("next profile の cue map が元 profile と共有されています")
	}
	if _, exists := original.concepts[conceptID]; !exists {
		t.Fatal("next profile の concept map が元 profile と共有されています")
	}
}

func TestRelationV2ProfileはRelationだけを意図根拠にする(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name              string
		query             string
		wantSignal        legalquery.CandidateGenerationSignal
		wantCandidates    int
		wantExplicitTask  bool
		wantRelationKinds []legalquery.CueTaskRelationKind
		wantContentTerms  []string
		wantCueTargetTerm bool
	}{
		{
			name:           "positive task と対象外 object の両方",
			query:          "EDINETを検索してください。",
			wantSignal:     legalquery.CandidateSignalUnsupportedTaskOrResource,
			wantCandidates: 0,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationObjectPredicate,
				legalquery.CueTaskRelationDirectTask,
			},
		},
		{
			name:             "positive read",
			query:            "民法第709条を見せてください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
		},
		{
			name:             "topic の translation は対象外にしない",
			query:            "翻訳に関する規定を検索してください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantContentTerms:  []string{"翻訳"},
			wantCueTargetTerm: true,
		},
		{
			name:             "という語の object は検索対象にする",
			query:            "影響グラフという語を含む条文を検索してください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantContentTerms:  []string{"影響グラフ"},
			wantCueTargetTerm: true,
		},
		{
			name:             "非終端の object は検索対象にする",
			query:            "差分を説明する規定を検索してください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantContentTerms:  []string{"差分"},
			wantCueTargetTerm: true,
		},
		{
			name:             "という文言の expression は検索対象にする",
			query:            "英語に翻訳してくださいという文言を含む条文を検索してください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantContentTerms:  []string{"英語に翻訳してください"},
			wantCueTargetTerm: true,
		},
		{
			name:             "引用内の object は引用検索対象にする",
			query:            "「比較」を含む条文を検索してください。",
			wantCandidates:   1,
			wantExplicitTask: true,
			wantRelationKinds: []legalquery.CueTaskRelationKind{
				legalquery.CueTaskRelationDirectTask,
			},
			wantContentTerms: []string{"比較"},
		},
		{
			name:           "構文 predicate だけでは task にしない",
			query:          "作成してください。",
			wantCandidates: 0,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			preprocessed, generation :=
				generateRelationV2Query(t, test.query)
			relations := preprocessed.CueTaskRelations()
			kinds := make([]legalquery.CueTaskRelationKind, 0, len(relations))
			for _, relation := range relations {
				kinds = append(kinds, relation.Kind())
			}
			if !slices.Equal(kinds, test.wantRelationKinds) {
				t.Fatalf(
					"SOT-MODEL-030: relation kinds = %#v、期待値は %#v",
					kinds,
					test.wantRelationKinds,
				)
			}
			if test.wantSignal != "" &&
				!slices.Contains(generation.Signals(), test.wantSignal) {
				t.Fatalf(
					"cue-relation-positive-dual-role: signals = %#v",
					generation.Signals(),
				)
			}
			if test.wantSignal == "" &&
				hasUnsupportedSignal(generation.Signals()) {
				t.Fatalf(
					"cue-relation-task-and-mention: signals = %#v",
					generation.Signals(),
				)
			}
			candidates := generation.Candidates()
			if len(candidates) != test.wantCandidates {
				t.Fatalf(
					"SOT-ARCH-031: candidates = %#v、期待件数は %d、queryTerms = %#v、cueMentions = %#v、relations = %#v",
					candidates,
					test.wantCandidates,
					preprocessed.QueryTermMentions(),
					preprocessed.CueMentions(),
					preprocessed.CueTaskRelations(),
				)
			}
			for _, candidate := range candidates {
				if slices.Contains(
					candidate.EvidenceCodes(),
					legalquery.EvidenceExplicitTask,
				) != test.wantExplicitTask {
					t.Fatalf(
						"positive-cue-bare-mention-rejected: evidence = %#v",
						candidate.EvidenceCodes(),
					)
				}
				if len(test.wantContentTerms) == 0 {
					continue
				}
				steps := candidate.Steps()
				if len(steps) != 1 {
					t.Fatalf(
						"cue-relation-task-and-mention: steps = %#v",
						steps,
					)
				}
				input, ok := steps[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
				if !ok ||
					!slices.Equal(input.AllTerms(), test.wantContentTerms) {
					t.Fatalf(
						"cue-relation-task-and-mention: logicalInput = %#v",
						steps[0].LogicalInput(),
					)
				}
				if test.wantCueTargetTerm &&
					(!slices.Contains(
						candidate.EvidenceCodes(),
						legalquery.EvidenceGeneralTerm,
					) ||
						slices.Contains(
							candidate.EvidenceCodes(),
							legalquery.EvidenceMorphologicalContext,
						)) {
					t.Fatalf(
						"cue 由来検索語の evidence = %#v",
						candidate.EvidenceCodes(),
					)
				}
			}
		})
	}
}

func TestRelationV2ProfileはReservedPackの言及を取得要求にしない(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name  string
		query string
	}{
		{
			name:  "引用句",
			query: "「裁判例」を含む条文を検索してください。",
		},
		{
			name:  "という語",
			query: "裁判例という語を含む条文を検索してください。",
		},
		{
			name:  "topic",
			query: "裁判例に関する規定を検索してください。",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, generation := generateRelationV2Query(t, test.query)
			if slices.Contains(
				generation.Signals(),
				legalquery.CandidateSignalReservedPackRequest,
			) {
				t.Fatalf(
					"reserved_pack の言及から signal = %#v",
					generation.Signals(),
				)
			}
			candidates := generation.Candidates()
			if len(candidates) != 1 ||
				len(candidates[0].Steps()) != 1 {
				t.Fatalf(
					"reserved_pack の言及候補 = %#v",
					candidates,
				)
			}
			input, ok := candidates[0].Steps()[0].LogicalInput().(legalquery.LawContentSearchIntentV1)
			if !ok || !slices.Equal(input.AllTerms(), []string{"裁判例"}) {
				t.Fatalf(
					"reserved_pack の言及検索条件 = %#v",
					candidates[0].Steps()[0].LogicalInput(),
				)
			}
		})
	}
}

func TestRelationV2Profileは明示ReservedPack要求を保持する(
	t *testing.T,
) {
	t.Parallel()

	_, generation := generateRelationV2Query(
		t,
		"裁判例を検索してください。",
	)
	if !slices.Contains(
		generation.Signals(),
		legalquery.CandidateSignalReservedPackRequest,
	) {
		t.Fatalf(
			"明示 reserved_pack request の signals = %#v",
			generation.Signals(),
		)
	}
}

func TestRelationV2Profileは対象外意図と候補のScopeを分離する(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name           string
		query          string
		wantCandidates int
	}{
		{
			name:           "別節の裸法令を保持しない",
			query:          "影響グラフを作成してください。民法。",
			wantCandidates: 0,
		},
		{
			name:           "別節で完結した法令検索だけを保持する",
			query:          "民法を検索してください。影響グラフを作成してください。",
			wantCandidates: 1,
		},
		{
			name:           "対象外 relation と同じ節の強い条文根拠を保持する",
			query:          "民法第103条の影響グラフを作成してください。",
			wantCandidates: 1,
		},
		{
			name:           "弱い step を強い step へ縮約しない",
			query:          "民法と架空語をそれぞれ検索してください。影響グラフを作成してください。",
			wantCandidates: 0,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			preprocessed, generation := generateRelationV2Query(t, test.query)
			if !slices.Contains(
				generation.Signals(),
				legalquery.CandidateSignalUnsupportedTaskOrResource,
			) {
				t.Fatalf(
					"cue-relation-candidate-scope: signals = %#v",
					generation.Signals(),
				)
			}
			if len(generation.Candidates()) != test.wantCandidates {
				t.Fatalf(
					"cue-relation-candidate-scope: candidates = %#v、期待件数は %d、queryTerms = %#v、cueMentions = %#v、relations = %#v",
					generation.Candidates(),
					test.wantCandidates,
					preprocessed.QueryTermMentions(),
					preprocessed.CueMentions(),
					preprocessed.CueTaskRelations(),
				)
			}
		})
	}
}

func TestRelationV2Profileは別節の言及名詞検索を削除しない(
	t *testing.T,
) {
	t.Parallel()

	_, generation := generateRelationV2Query(
		t,
		"「文言」を含む条文を検索してください。翻訳に関する規定を検索してください。",
	)
	terms := make([]string, 0)
	for _, candidate := range generation.Candidates() {
		for _, step := range candidate.Steps() {
			input, ok := step.LogicalInput().(legalquery.LawContentSearchIntentV1)
			if !ok {
				continue
			}
			terms = append(terms, input.AllTerms()...)
		}
	}
	if !slices.Contains(terms, "文言") ||
		!slices.Contains(terms, "翻訳") {
		t.Fatalf(
			"cue-relation-clause-scope: content terms = %#v",
			terms,
		)
	}
}

func TestRelationV2Profileは全Stepを独立した強い根拠で検証する(
	t *testing.T,
) {
	t.Parallel()

	profile := mustRelationV2Profile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("relation v2 preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "民法と架空語をそれぞれ検索してください。影響グラフを作成してください。",
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("candidate generation input を構築できません: %v", err)
	}
	rawCues, err := profile.resolveCues(input.CueMentions())
	if err != nil {
		t.Fatalf("cue を解決できません: %v", err)
	}
	cues, err := profile.resolveRelationV2Cues(input, rawCues)
	if err != nil {
		t.Fatalf("relation v2 cue を解決できません: %v", err)
	}
	if len(input.LawNameMentions()) != 1 ||
		len(input.QueryTermMentions()) != 1 {
		t.Fatalf(
			"test fixture の target = law:%#v term:%#v",
			input.LawNameMentions(),
			input.QueryTermMentions(),
		)
	}
	lawInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "民法"},
	)
	if err != nil {
		t.Fatalf("law search input を構築できません: %v", err)
	}
	weakInput, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: "架空語"},
	)
	if err != nil {
		t.Fatalf("weak search input を構築できません: %v", err)
	}
	draft := newCandidateDraft()
	for _, code := range []legalquery.EvidenceCode{
		legalquery.EvidenceExplicitTask,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceGeneralTerm,
	} {
		draft.evidence[code] = struct{}{}
	}
	draft.steps = []stepDraft{
		{
			startByte: input.LawNameMentions()[0].Span().StartByte(),
			input:     lawInput,
		},
		{
			startByte: input.QueryTermMentions()[0].Span().StartByte(),
			input:     weakInput,
		},
	}
	retained := profile.retainRelationV2SupportedDrafts(
		input,
		cues,
		[]candidateDraft{draft},
		profile.generationSignals(input, cues),
	)
	if len(retained) != 0 {
		t.Fatalf(
			"cue-relation-candidate-scope: 一部の step だけが強い候補 = %#v",
			retained,
		)
	}
	if len(draft.steps) != 2 {
		t.Fatalf(
			"cue-relation-candidate-scope: 元の draft を変更しました: %#v",
			draft.steps,
		)
	}
}

func mustRelationV2Profile(t *testing.T) *Profile {
	t.Helper()

	lawNames, concepts := mustEmbeddedLexicons(t)
	profile, err := Load(
		relationV2ProfileJSON,
		relationV2CuesJSON,
		lawNames,
		concepts,
	)
	if err != nil {
		t.Fatalf("relation v2 profile を読み込めません: %v", err)
	}
	profile, err = newCueTaskRelationV2Profile(profile)
	if err != nil {
		t.Fatalf("relation v2 profile を準備できません: %v", err)
	}
	return profile
}

func generateRelationV2Query(
	t *testing.T,
	query string,
) (legalquery.PreprocessResult, legalquery.CandidateGeneration) {
	t.Helper()

	profile := mustRelationV2Profile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("relation v2 preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: query})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	preprocessed, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(preprocessed)
	if err != nil {
		t.Fatalf("candidate generation input を構築できません: %v", err)
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("candidate scope を構築できません: %v", err)
	}
	generation, err := profile.Generate(input, scope)
	if err != nil {
		t.Fatalf("Generate() のエラー = %v", err)
	}
	return preprocessed, generation
}
