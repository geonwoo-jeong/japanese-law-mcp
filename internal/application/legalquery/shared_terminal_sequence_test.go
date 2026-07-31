package legalquery

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestSharedTerminalSequenceはTopic列とTerminalRelationを深く複製する(
	t *testing.T,
) {
	t.Parallel()

	relation := testDirectTaskRelation(
		mustQuerySpan(t, 0, 30),
		mustQuerySpan(t, 24, 30),
	)
	sequence, err := newSharedTerminalSequence(
		[]QuerySpan{
			mustQuerySpan(t, 0, 6),
			mustQuerySpan(t, 9, 15),
		},
		relation,
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-031: sidecar を作成できません: %v", err)
	}

	topics := sequence.TopicSpans()
	terminal := sequence.TerminalTaskRelation()
	topics[0] = QuerySpan{}
	terminal.subject.cueID = "変更済み"
	if sequence.TopicSpans()[0] != mustQuerySpan(t, 0, 6) ||
		sequence.TerminalTaskRelation().Subject().CueID() != "task-terminal" {
		t.Fatal("SOT-MODEL-031: accessor から sidecar を変更できました")
	}

	input := CandidateGenerationInput{
		sharedTerminalSequences: []SharedTerminalSequence{sequence},
	}
	values := input.SharedTerminalSequences()
	values[0].topicSpans[0] = QuerySpan{}
	values[0].terminalTaskRelation.subject.cueID = "変更済み"
	if input.SharedTerminalSequences()[0].TopicSpans()[0] !=
		mustQuerySpan(t, 0, 6) ||
		input.SharedTerminalSequences()[0].
			TerminalTaskRelation().
			Subject().
			CueID() != "task-terminal" {
		t.Fatal("SOT-MODEL-031: input accessor から sidecar を変更できました")
	}
}

func TestSharedTerminalSequenceはTopic件数と構造上限を検証する(t *testing.T) {
	t.Parallel()

	relation := testDirectTaskRelation(
		mustQuerySpan(t, 0, 400),
		mustQuerySpan(t, 390, 400),
	)
	topics := make([]QuerySpan, 256)
	for index := range topics {
		topics[index] = mustQuerySpan(t, index, index+1)
	}
	sequence, err := newSharedTerminalSequence(topics, relation)
	if err != nil {
		t.Fatalf("二百五十六 topic を拒否しました: %v", err)
	}
	if len(sequence.TopicSpans()) != 256 {
		t.Fatalf("topic 件数 = %d", len(sequence.TopicSpans()))
	}

	for name, invalid := range map[string][]QuerySpan{
		"一件": {
			mustQuerySpan(t, 0, 1),
		},
		"二百五十七件": append(
			append([]QuerySpan(nil), topics...),
			mustQuerySpan(t, 256, 257),
		),
		"逆順": {
			mustQuerySpan(t, 2, 3),
			mustQuerySpan(t, 0, 1),
		},
		"重複": {
			mustQuerySpan(t, 0, 1),
			mustQuerySpan(t, 0, 1),
		},
		"重なり": {
			mustQuerySpan(t, 0, 3),
			mustQuerySpan(t, 2, 4),
		},
		"節外": {
			mustQuerySpan(t, 0, 1),
			mustQuerySpan(t, 401, 402),
		},
	} {
		invalid := invalid
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := newSharedTerminalSequence(invalid, relation); err == nil {
				t.Fatalf("%sの topic 列を受理しました", name)
			}
		})
	}

	nonDirect := relation
	nonDirect.kind = CueTaskRelationStandaloneTask
	if _, err := newSharedTerminalSequence(topics[:2], nonDirect); err == nil {
		t.Fatal("direct_task ではない terminal relation を受理しました")
	}
}

func TestNewCandidateGenerationInputは共有末尾Cueの二主題列を構築する(
	t *testing.T,
) {
	t.Parallel()

	for _, query := range []string{
		"永住許可、帰化を教えてください",
		"永住許可と帰化について教えてください",
	} {
		query := query
		t.Run(query, func(t *testing.T) {
			t.Parallel()

			result := mustSharedTerminalPreprocess(
				t,
				query,
				[]string{"永住許可", "帰化"},
			)
			input, err := NewCandidateGenerationInput(result)
			if err != nil {
				t.Fatalf("candidate generation input のエラー = %v", err)
			}
			sequences := input.SharedTerminalSequences()
			if len(sequences) != 1 {
				t.Fatalf("sidecar 件数 = %d", len(sequences))
			}
			topics := sequences[0].TopicSpans()
			if len(topics) != 2 ||
				query[topics[0].StartByte():topics[0].EndByte()] != "永住許可" ||
				query[topics[1].StartByte():topics[1].EndByte()] != "帰化" {
				t.Fatalf("topic spans = %#v", topics)
			}
			if got := sequences[0].TerminalTaskRelation(); got.Kind() !=
				CueTaskRelationDirectTask ||
				got.Subject() != got.Predicate() {
				t.Fatalf("terminal relation = %#v", got)
			}
			if err := input.Validate(); err != nil {
				t.Fatalf("candidate generation input の Validate() = %v", err)
			}
		})
	}
}

func TestNewCandidateGenerationInputは閉じたSeparatorと末尾接続だけを受理する(
	t *testing.T,
) {
	t.Parallel()

	for _, separator := range []string{
		"、", ",", "，", "と", "及び", "および", "並びに", "ならびに",
	} {
		separator := separator
		t.Run("separator="+separator, func(t *testing.T) {
			t.Parallel()
			query := "甲号 \u3000" + separator + "\u3000乙号 を 教えてください"
			input := mustCandidateGenerationInput(
				t,
				mustSharedTerminalPreprocess(
					t,
					query,
					[]string{"甲号", "乙号"},
				),
			)
			if got := len(input.SharedTerminalSequences()); got != 1 {
				t.Fatalf("sidecar 件数 = %d", got)
			}
		})
	}

	for _, value := range []struct {
		name  string
		query string
	}{
		{name: "separatorなし", query: "甲号乙号を教えてください"},
		{name: "未知separator", query: "甲号や乙号を教えてください"},
		{name: "separator反復", query: "甲号、、乙号を教えてください"},
		{name: "separator連結", query: "甲号と及び乙号を教えてください"},
		{name: "tail反復", query: "甲号、乙号をについて教えてください"},
		{name: "tail連結", query: "甲号、乙号についてを教えてください"},
	} {
		value := value
		t.Run(value.name, func(t *testing.T) {
			t.Parallel()
			input := mustCandidateGenerationInput(
				t,
				mustSharedTerminalPreprocess(
					t,
					value.query,
					[]string{"甲号", "乙号"},
				),
			)
			if got := input.SharedTerminalSequences(); len(got) != 0 {
				t.Fatalf("違反した列から sidecar を作成しました: %#v", got)
			}
		})
	}
}

func TestNewCandidateGenerationInputは最大Topic列だけを一件保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "甲号、乙号、丙号、丁号、戊号を教えてください"
	input := mustCandidateGenerationInput(
		t,
		mustSharedTerminalPreprocess(
			t,
			query,
			[]string{"甲号", "乙号", "丙号", "丁号", "戊号"},
		),
	)
	sequences := input.SharedTerminalSequences()
	if len(sequences) != 1 {
		t.Fatalf("sidecar 件数 = %d", len(sequences))
	}
	if got := len(sequences[0].TopicSpans()); got != 5 {
		t.Fatalf("最大 topic 件数 = %d", got)
	}
}

func TestNewCandidateGenerationInputは非同一最大列が複数なら選ばない(
	t *testing.T,
) {
	t.Parallel()

	const query = "日本民法、帰化を教えてください"
	result := mustSharedTerminalPreprocess(
		t,
		query,
		[]string{"日本民法", "帰化"},
	)
	nested, err := NewQueryTermMention(QueryTermMentionValues{
		Span: mustQuerySpan(
			t,
			len("日本"),
			len("日本民法"),
		),
		Surface: "民法",
		Kind:    QueryTermMentionMorphologicalPhrase,
	})
	if err != nil {
		t.Fatalf("重なり query term を作成できません: %v", err)
	}
	result.queryTermMentions = append(result.queryTermMentions, nested)
	slices.SortFunc(result.queryTermMentions, func(
		left QueryTermMention,
		right QueryTermMention,
	) int {
		return compareMention(
			left.Span(),
			string(left.Kind()),
			right.Span(),
			string(right.Kind()),
		)
	})
	if err := result.Validate(); err != nil {
		t.Fatalf("重なり前処理結果が無効です: %v", err)
	}

	input := mustCandidateGenerationInput(t, result)
	if got := input.SharedTerminalSequences(); len(got) != 0 {
		t.Fatalf("非同一の最大列から一件を選びました: %#v", got)
	}
}

func TestNewCandidateGenerationInputは同一Spanの複数意味を一Topicにする(
	t *testing.T,
) {
	t.Parallel()

	const query = "永住許可、帰化を教えてください"
	result := mustSharedTerminalPreprocess(
		t,
		query,
		[]string{"永住許可", "帰化"},
	)
	concept, err := NewLegalConceptMention(LegalConceptMentionValues{
		Span:      mustQuerySpan(t, 0, len("永住許可")),
		Surface:   "永住許可",
		ConceptID: "permanent-residence",
		Canonical: "永住許可",
		MatchKind: PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("同一 span の法概念を作成できません: %v", err)
	}
	result.legalConceptMentions = []LegalConceptMention{concept}
	if err := result.Validate(); err != nil {
		t.Fatalf("複数意味の前処理結果が無効です: %v", err)
	}

	input := mustCandidateGenerationInput(t, result)
	sequences := input.SharedTerminalSequences()
	if len(sequences) != 1 || len(sequences[0].TopicSpans()) != 2 {
		t.Fatalf("同一 span を重複加算しました: %#v", sequences)
	}
}

func TestNewCandidateGenerationInputは構造条件を満たさない列を保持しない(
	t *testing.T,
) {
	t.Parallel()

	const query = "甲号、乙号を教えてください"
	valid := mustSharedTerminalPreprocess(
		t,
		query,
		[]string{"甲号", "乙号"},
	)
	tests := map[string]func(*testing.T, PreprocessResult) PreprocessResult{
		"relationなし": func(_ *testing.T, value PreprocessResult) PreprocessResult {
			value.cueTaskRelations = nil
			return value
		},
		"direct_task以外": func(_ *testing.T, value PreprocessResult) PreprocessResult {
			value.cueTaskRelations = cloneCueTaskRelations(
				value.cueTaskRelations,
			)
			value.cueTaskRelations[0].kind = CueTaskRelationStandaloneTask
			return value
		},
		"別の節": func(t *testing.T, value PreprocessResult) PreprocessResult {
			value.cueTaskRelations = cloneCueTaskRelations(
				value.cueTaskRelations,
			)
			value.cueTaskRelations[0].clauseSpan = mustQuerySpan(
				t,
				len("甲号、"),
				len(query),
			)
			return value
		},
		"文末でないcue": func(t *testing.T, value PreprocessResult) PreprocessResult {
			value.query += "追加"
			value.comparisonKey = querynormalization.ComparisonKey(value.query)
			value.cueTaskRelations = cloneCueTaskRelations(
				value.cueTaskRelations,
			)
			value.cueTaskRelations[0].clauseSpan = mustQuerySpan(
				t,
				0,
				len(value.query),
			)
			return value
		},
		"途中の別cue": func(t *testing.T, value PreprocessResult) PreprocessResult {
			otherCue, err := NewCueMention(CueMentionValues{
				Span: mustQuerySpan(
					t,
					0,
					len("甲号"),
				),
				Surface:   "甲号",
				ProfileID: "core",
				CueID:     "other-task",
				MatchKind: PreprocessMatchRegisteredTerm,
			})
			if err != nil {
				t.Fatalf("途中 cue を作成できません: %v", err)
			}
			value.cueMentions = append(
				[]CueMention{otherCue},
				value.cueMentions...,
			)
			return value
		},
	}
	for name, mutate := range tests {
		name := name
		mutate := mutate
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			result := mutate(t, valid)
			if err := result.Validate(); err != nil {
				t.Fatalf("試験用前処理結果が無効です: %v", err)
			}
			input := mustCandidateGenerationInput(t, result)
			if got := input.SharedTerminalSequences(); len(got) != 0 {
				t.Fatalf("構造条件違反から sidecar を作成しました: %#v", got)
			}
		})
	}
}

func TestNewCandidateGenerationInputは異なるSpanの同じ意味を保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "帰化、帰化を教えてください"
	result := mustSharedTerminalPreprocess(
		t,
		query,
		[]string{"帰化", "帰化"},
	)
	concepts := make([]LegalConceptMention, 0, 2)
	cursor := 0
	for range 2 {
		startByte := cursor + strings.Index(query[cursor:], "帰化")
		endByte := startByte + len("帰化")
		mention, err := NewLegalConceptMention(LegalConceptMentionValues{
			Span:      mustQuerySpan(t, startByte, endByte),
			Surface:   "帰化",
			ConceptID: "naturalization",
			Canonical: "帰化",
			MatchKind: PreprocessMatchRegisteredTerm,
		})
		if err != nil {
			t.Fatalf("反復法概念を作成できません: %v", err)
		}
		concepts = append(concepts, mention)
		cursor = endByte
	}
	result.legalConceptMentions = concepts
	if err := result.Validate(); err != nil {
		t.Fatalf("反復法概念の前処理結果が無効です: %v", err)
	}
	input := mustCandidateGenerationInput(t, result)
	sequences := input.SharedTerminalSequences()
	if len(sequences) != 1 || len(sequences[0].TopicSpans()) != 2 ||
		sequences[0].TopicSpans()[0] == sequences[0].TopicSpans()[1] {
		t.Fatalf("異なる位置を縮約しました: %#v", sequences)
	}
}

func TestNewCandidateGenerationInputは百二十八Sidecarを保持する(
	t *testing.T,
) {
	t.Parallel()

	const query = "甲号、乙号を教えてください"
	base := mustSharedTerminalPreprocess(
		t,
		query,
		[]string{"甲号", "乙号"},
	)
	cueSpan := base.cueMentions[0].Span()
	cues := make([]CueMention, 0, maximumSharedTerminalSequences)
	relations := make([]CueTaskRelation, 0, maximumSharedTerminalSequences)
	for index := range maximumSharedTerminalSequences {
		cueID := fmt.Sprintf("task-%03d", index)
		cue, err := NewCueMention(CueMentionValues{
			Span:      cueSpan,
			Surface:   "教えてください",
			ProfileID: "core",
			CueID:     cueID,
			MatchKind: PreprocessMatchRegisteredTerm,
		})
		if err != nil {
			t.Fatalf("cue[%d] を作成できません: %v", index, err)
		}
		relation, err := NewCueTaskRelation(CueTaskRelationValues{
			Query:         query,
			Subject:       cue,
			Predicate:     cue,
			SubjectRole:   CueSyntaxRoleTaskExpression,
			PredicateRole: CueSyntaxRoleTaskExpression,
			ClauseSpan:    mustQuerySpan(t, 0, len(query)),
			Kind:          CueTaskRelationDirectTask,
		})
		if err != nil {
			t.Fatalf("relation[%d] を作成できません: %v", index, err)
		}
		cues = append(cues, cue)
		relations = append(relations, relation)
	}
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:             query,
		ComparisonKey:     querynormalization.ComparisonKey(query),
		QueryTermMentions: base.QueryTermMentions(),
		CueMentions:       cues,
		CueTaskRelations:  relations,
	})
	if err != nil {
		t.Fatalf("百二十八 relation の前処理結果を作成できません: %v", err)
	}
	input := mustCandidateGenerationInput(t, result)
	if got := len(input.SharedTerminalSequences()); got !=
		maximumSharedTerminalSequences {
		t.Fatalf("sidecar 件数 = %d", got)
	}
}

func TestCandidateGenerationInputは不正Sidecar参照を拒否する(t *testing.T) {
	t.Parallel()

	const query = "甲号、乙号を教えてください"
	input := mustCandidateGenerationInput(
		t,
		mustSharedTerminalPreprocess(
			t,
			query,
			[]string{"甲号", "乙号"},
		),
	)
	if len(input.sharedTerminalSequences) != 1 {
		t.Fatal("試験前提の sidecar がありません")
	}

	unknownTopic := input
	unknownTopic.sharedTerminalSequences = cloneSharedTerminalSequences(
		input.sharedTerminalSequences,
	)
	unknownTopic.sharedTerminalSequences[0].topicSpans[0] =
		mustQuerySpan(t, 1, len("甲号"))
	if err := unknownTopic.Validate(); err == nil {
		t.Fatal("既存出現と完全一致しない topic span を受理しました")
	}

	unknownRelation := input
	unknownRelation.sharedTerminalSequences = cloneSharedTerminalSequences(
		input.sharedTerminalSequences,
	)
	unknownRelation.sharedTerminalSequences[0].
		terminalTaskRelation.
		subject.
		cueID = "unknown"
	unknownRelation.sharedTerminalSequences[0].
		terminalTaskRelation.
		predicate.
		cueID = "unknown"
	if err := unknownRelation.Validate(); err == nil {
		t.Fatal("既存 relation と一致しない terminal relation を受理しました")
	}

	tooMany := input
	tooMany.sharedTerminalSequences = make(
		[]SharedTerminalSequence,
		maximumSharedTerminalSequences+1,
	)
	for index := range tooMany.sharedTerminalSequences {
		tooMany.sharedTerminalSequences[index] = input.sharedTerminalSequences[0]
	}
	if err := tooMany.Validate(); err == nil {
		t.Fatal("百二十九件の sidecar を受理しました")
	}
}

func TestSharedTerminalSequenceの消費を非公開準備経路に閉じる(
	t *testing.T,
) {
	t.Parallel()

	for _, profileDirectory := range []string{"core", "judicialcases"} {
		profileDirectory := profileDirectory
		t.Run(profileDirectory, func(t *testing.T) {
			t.Parallel()
			sidecarAccesses := 0
			builderReferences := 0
			files, err := filepath.Glob(filepath.Join(
				"..",
				"..",
				"queryprofile",
				profileDirectory,
				"*.go",
			))
			if err != nil {
				t.Fatalf("profile source を列挙できません: %v", err)
			}
			productionFiles := 0
			for _, file := range files {
				if strings.HasSuffix(file, "_test.go") {
					continue
				}
				productionFiles++
				parsed, parseErr := parser.ParseFile(
					token.NewFileSet(),
					file,
					nil,
					parser.SkipObjectResolution,
				)
				if parseErr != nil {
					t.Fatalf("%s を解析できません: %v", file, parseErr)
				}
				ast.Inspect(parsed, func(node ast.Node) bool {
					selector, ok := node.(*ast.SelectorExpr)
					if ok && selector.Sel.Name == "SharedTerminalSequences" {
						if profileDirectory == "core" &&
							filepath.Base(file) == "shared_terminal_v2.go" {
							sidecarAccesses++
							return true
						}
						t.Errorf(
							"SOT-MODEL-031: active profile から到達する %s が sidecar を消費しています",
							file,
						)
					}
					if ok && selector.Sel.Name == "buildCoreSharedTerminalDrafts" {
						if profileDirectory == "core" &&
							filepath.Base(file) == "evidence_v2.go" {
							builderReferences++
							return true
						}
						builderReferences++
						t.Errorf(
							"SOT-ENG-039: 3.4.3 の準備関数を許可外の %s から選択しています",
							file,
						)
					}
					return true
				})
			}
			if productionFiles == 0 {
				t.Fatal("検査対象の production profile source がありません")
			}
			if profileDirectory == "core" && sidecarAccesses != 1 {
				t.Fatalf(
					"SOT-ENG-039: core の準備 accessor 数 = %d",
					sidecarAccesses,
				)
			}
			wantBuilderReferences := 0
			if profileDirectory == "core" {
				wantBuilderReferences = 1
			}
			if builderReferences != wantBuilderReferences {
				t.Fatalf(
					"SOT-ENG-039: 3.4.3 の準備関数参照数 = %d, want %d",
					builderReferences,
					wantBuilderReferences,
				)
			}
		})
	}
}

func mustSharedTerminalPreprocess(
	t *testing.T,
	query string,
	topicSurfaces []string,
) PreprocessResult {
	t.Helper()

	queryTerms := make([]QueryTermMention, 0, len(topicSurfaces))
	cursor := 0
	for _, surface := range topicSurfaces {
		relative := strings.Index(query[cursor:], surface)
		if relative < 0 {
			t.Fatalf("query %q に topic %q がありません", query, surface)
		}
		startByte := cursor + relative
		endByte := startByte + len(surface)
		mention, err := NewQueryTermMention(QueryTermMentionValues{
			Span:    mustQuerySpan(t, startByte, endByte),
			Surface: surface,
			Kind:    QueryTermMentionMorphologicalPhrase,
		})
		if err != nil {
			t.Fatalf("試験用 query term を作成できません: %v", err)
		}
		queryTerms = append(queryTerms, mention)
		cursor = endByte
	}

	const cueSurface = "教えてください"
	cueStart := strings.LastIndex(query, cueSurface)
	if cueStart < 0 {
		t.Fatalf("query %q に terminal cue がありません", query)
	}
	cue, err := NewCueMention(CueMentionValues{
		Span:      mustQuerySpan(t, cueStart, cueStart+len(cueSurface)),
		Surface:   cueSurface,
		ProfileID: "core",
		CueID:     "task-terminal",
		MatchKind: PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("試験用 cue mention を作成できません: %v", err)
	}
	relation, err := NewCueTaskRelation(CueTaskRelationValues{
		Query:         query,
		Subject:       cue,
		Predicate:     cue,
		SubjectRole:   CueSyntaxRoleTaskExpression,
		PredicateRole: CueSyntaxRoleTaskExpression,
		ClauseSpan:    mustQuerySpan(t, 0, len(query)),
		Kind:          CueTaskRelationDirectTask,
	})
	if err != nil {
		t.Fatalf("試験用 cue task relation を作成できません: %v", err)
	}
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:             query,
		ComparisonKey:     querynormalization.ComparisonKey(query),
		QueryTermMentions: queryTerms,
		CueMentions:       []CueMention{cue},
		CueTaskRelations:  []CueTaskRelation{relation},
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	return result
}

func mustCandidateGenerationInput(
	t *testing.T,
	result PreprocessResult,
) CandidateGenerationInput {
	t.Helper()
	input, err := NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("candidate generation input を作成できません: %v", err)
	}
	return input
}

func testDirectTaskRelation(
	clauseSpan QuerySpan,
	cueSpan QuerySpan,
) CueTaskRelation {
	ref := CueTaskRelationRef{
		profileID: "core",
		cueID:     "task-terminal",
		span:      cueSpan,
	}
	return CueTaskRelation{
		subject:    ref,
		predicate:  ref,
		clauseSpan: clauseSpan,
		kind:       CueTaskRelationDirectTask,
	}
}
