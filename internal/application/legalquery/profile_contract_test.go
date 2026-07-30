package legalquery

import (
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

func TestCandidateGenerationInputは原文を再解析せず位置付き事実を複製する(
	t *testing.T,
) {
	t.Parallel()

	query := "民法を検索"
	lawMention, err := NewLawNameMention(LawNameMentionValues{
		Span:       mustQuerySpan(t, 0, len("民法")),
		Surface:    "民法",
		LawID:      "129AC0000000089",
		RevisionID: "129AC0000000089_20250601_504AC0000000068",
		LawNumber:  "明治二十九年法律第八十九号",
		Canonical:  "民法",
		MatchKind:  PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("試験用 law mention を作成できません: %v", err)
	}
	cueMention, err := NewCueMention(CueMentionValues{
		Span:      mustQuerySpan(t, len("民法を"), len(query)),
		Surface:   "検索",
		ProfileID: "core",
		CueID:     "task-search",
		MatchKind: PreprocessMatchRegisteredTerm,
	})
	if err != nil {
		t.Fatalf("試験用 cue mention を作成できません: %v", err)
	}
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:           query,
		ComparisonKey:   "民法を検索",
		LawNameMentions: []LawNameMention{lawMention},
		CueMentions:     []CueMention{cueMention},
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}

	input, err := NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("SOT-ARCH-022: candidate generation input のエラー = %v", err)
	}
	if input.Language() != QueryLanguageJapanese {
		t.Fatalf("language = %q", input.Language())
	}
	laws := input.LawNameMentions()
	cues := input.CueMentions()
	if len(laws) != 1 || laws[0].LawID() != "129AC0000000089" {
		t.Fatalf("law mentions = %#v", laws)
	}
	if len(cues) != 1 || cues[0].CueID() != "task-search" {
		t.Fatalf("cue mentions = %#v", cues)
	}

	laws[0] = LawNameMention{}
	cues[0] = CueMention{}
	if input.LawNameMentions()[0].LawID() != "129AC0000000089" ||
		input.CueMentions()[0].CueID() != "task-search" {
		t.Fatal("SOT-ARCH-022: getter から generation input を変更できました")
	}
}

func TestCandidateGenerationInputは非日本語を型付き事実にする(t *testing.T) {
	t.Parallel()

	for _, query := range []string{
		"search Japanese statutes",
		"Civil Code を read",
		"请阅读日本民法",
		"请民法を検索",
	} {
		result, err := NewPreprocessResult(PreprocessResultValues{
			Query:         query,
			ComparisonKey: string(querynormalization.ComparisonKey(query)),
		})
		if err != nil {
			t.Fatalf("試験用 preprocess result を作成できません: %v", err)
		}
		input, err := NewCandidateGenerationInput(result)
		if err != nil {
			t.Fatalf("candidate generation input のエラー = %v", err)
		}
		if input.Language() != QueryLanguageNonJapanese {
			t.Fatalf("query=%q language = %q", query, input.Language())
		}
	}
}

func TestCandidateGenerationInputは仮名なしでも全文の型付き事実を日本語とする(
	t *testing.T,
) {
	t.Parallel()

	const query = "民法"
	mention, err := NewLawNameMention(LawNameMentionValues{
		Span:       mustQuerySpan(t, 0, len(query)),
		Surface:    query,
		LawID:      "129AC0000000089",
		RevisionID: "129AC0000000089_20250601_504AC0000000068",
		LawNumber:  "明治二十九年法律第八十九号",
		Canonical:  query,
		MatchKind:  PreprocessMatchExact,
	})
	if err != nil {
		t.Fatalf("試験用 law mention を作成できません: %v", err)
	}
	result, err := NewPreprocessResult(PreprocessResultValues{
		Query:           query,
		ComparisonKey:   query,
		LawNameMentions: []LawNameMention{mention},
	})
	if err != nil {
		t.Fatalf("試験用 preprocess result を作成できません: %v", err)
	}
	input, err := NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("candidate generation input のエラー = %v", err)
	}
	if input.Language() != QueryLanguageJapanese {
		t.Fatalf("language = %q", input.Language())
	}
}

func TestCandidateIDScopeは入力断片を含まない決定的IDを作る(t *testing.T) {
	t.Parallel()

	scope, err := NewCandidateIDScope(2)
	if err != nil {
		t.Fatalf("SOT-MODEL-022: ID scope のエラー = %v", err)
	}
	candidateID, err := scope.CandidateID(3)
	if err != nil {
		t.Fatalf("candidate ID のエラー = %v", err)
	}
	stepID, err := scope.StepID(3, 4)
	if err != nil {
		t.Fatalf("step ID のエラー = %v", err)
	}
	if candidateID != "candidate-2-3" || stepID != "step-2-3-4" {
		t.Fatalf("IDs = %q, %q", candidateID, stepID)
	}
	for _, value := range []string{candidateID, stepID} {
		if strings.Contains(value, "民法") {
			t.Fatalf("SOT-MODEL-022: ID に入力断片が入りました: %q", value)
		}
	}

	for _, invalid := range []int{-1, 0, 10000} {
		if _, err := NewCandidateIDScope(invalid); err == nil {
			t.Fatalf("不正な profile ordinal %d を受理しました", invalid)
		}
	}
	if _, err := scope.CandidateID(0); err == nil {
		t.Fatal("candidate ordinal 0 を受理しました")
	}
	if _, err := scope.StepID(1, 5); err == nil {
		t.Fatal("step ordinal 5 を受理しました")
	}
}

func TestAssembleLegalQueryCandidateはlogicalInputから能力対応を一元生成する(
	t *testing.T,
) {
	t.Parallel()

	scope, err := NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	inputs := []LogicalInput{
		mustLawSearchIntent(t),
		mustLawContentSearchIntent(t),
		mustLawReadByIDIntent(t),
		mustLawArticleReadByIDIntent(t),
		mustLawUpdateIntent(t),
	}
	candidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: 1,
		SemanticScore:    700,
		Confidence:       ConfidenceHigh,
		EvidenceCodes:    []EvidenceCode{EvidenceOfficialIdentifier},
		LogicalInputs:    inputs[:4],
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-022: candidate assembly のエラー = %v", err)
	}
	steps := candidate.Steps()
	wantKinds := []LogicalInputKind{
		InputKindLawSearch,
		InputKindLawContentSearch,
		InputKindLawRead,
		InputKindLawArticleRead,
	}
	for index, step := range steps {
		if step.InputKind() != wantKinds[index] ||
			step.StepID() != "step-1-1-"+string(rune('1'+index)) {
			t.Fatalf("steps[%d] = %#v", index, step)
		}
	}

	updateCandidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: 2,
		SemanticScore:    200,
		Confidence:       ConfidenceMedium,
		EvidenceCodes:    []EvidenceCode{EvidenceStructuredReference},
		LogicalInputs:    inputs[4:],
	})
	if err != nil {
		t.Fatalf("update candidate assembly のエラー = %v", err)
	}
	if got := updateCandidate.Steps()[0]; got.Task() != TaskListUpdates ||
		got.Resource() != ResourceLaw ||
		got.CapabilityID() != "law.update.list" ||
		got.CapabilityMajorVersion() != 1 {
		t.Fatalf("update step = %#v", got)
	}
}

func TestCandidateGenerationは候補と安全信号を不変に保持する(t *testing.T) {
	t.Parallel()

	scope, err := NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	candidate, err := AssembleLegalQueryCandidate(CandidateAssemblyValues{
		IDScope:          scope,
		CandidateOrdinal: 1,
		SemanticScore:    100,
		Confidence:       ConfidenceMedium,
		EvidenceCodes:    []EvidenceCode{EvidenceExplicitTask},
		LogicalInputs:    []LogicalInput{mustLawSearchIntent(t)},
	})
	if err != nil {
		t.Fatalf("candidate を作成できません: %v", err)
	}
	generation, err := NewCandidateGeneration(CandidateGenerationValues{
		ProfileID:      "core",
		ProfileVersion: "core-2026-07-28-1",
		RankingVersion: "ranking-2026-07-28-1",
		Candidates:     []LegalQueryCandidate{candidate},
		Signals: []CandidateGenerationSignal{
			CandidateSignalUnsupportedLegalAdvice,
			CandidateSignalUnsupportedTaskOrResource,
		},
		SelectionMode: QuerySelectionModeAutomatic,
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-023: generation のエラー = %v", err)
	}

	candidates := generation.Candidates()
	signals := generation.Signals()
	candidates[0] = LegalQueryCandidate{}
	signals[0] = CandidateSignalNonJapaneseQuery
	if generation.Candidates()[0].CandidateID() != "candidate-1-1" ||
		!slices.Equal(generation.Signals(), []CandidateGenerationSignal{
			CandidateSignalUnsupportedLegalAdvice,
			CandidateSignalUnsupportedTaskOrResource,
		}) {
		t.Fatal("SOT-ARCH-023: getter から generation を変更できました")
	}

	for _, invalid := range [][]CandidateGenerationSignal{
		{
			CandidateSignalUnsupportedTaskOrResource,
			CandidateSignalUnsupportedLegalAdvice,
		},
		{
			CandidateSignalUnsupportedLegalAdvice,
			CandidateSignalUnsupportedLegalAdvice,
		},
		{"unknown"},
	} {
		values := CandidateGenerationValues{
			ProfileID:      "core",
			ProfileVersion: "core-2026-07-28-1",
			RankingVersion: "ranking-2026-07-28-1",
			Candidates:     []LegalQueryCandidate{candidate},
			Signals:        invalid,
			SelectionMode:  QuerySelectionModeAutomatic,
		}
		if _, err := NewCandidateGeneration(values); err == nil {
			t.Fatalf("不正な signals を受理しました: %#v", invalid)
		}
	}
}

func mustQuerySpan(t *testing.T, startByte int, endByte int) QuerySpan {
	t.Helper()
	span, err := NewQuerySpan(QuerySpanValues{
		StartByte: startByte,
		EndByte:   endByte,
	})
	if err != nil {
		t.Fatalf("試験用 span を作成できません: %v", err)
	}
	return span
}

func mustLawReadByIDIntent(t *testing.T) LawReadIntentV1 {
	t.Helper()
	intent, err := NewLawReadIntentV1(LawReadIntentV1Values{
		LawID: "129AC0000000089",
	})
	if err != nil {
		t.Fatalf("試験用 law read intent を作成できません: %v", err)
	}
	return intent
}

func mustLawArticleReadByIDIntent(t *testing.T) LawArticleReadIntentV1 {
	t.Helper()
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "709",
	})
	if err != nil {
		t.Fatalf("試験用 article location を作成できません: %v", err)
	}
	intent, err := NewLawArticleReadIntentV1(LawArticleReadIntentV1Values{
		LawID:    "129AC0000000089",
		Location: location,
	})
	if err != nil {
		t.Fatalf("試験用 article read intent を作成できません: %v", err)
	}
	return intent
}
