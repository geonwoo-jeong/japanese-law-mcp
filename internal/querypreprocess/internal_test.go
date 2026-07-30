package querypreprocess

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
)

type emptyOccurrenceAnalyzer struct{}

func (emptyOccurrenceAnalyzer) RegisteredTerms(
	context.Context,
	string,
) ([]string, error) {
	return nil, nil
}

func (emptyOccurrenceAnalyzer) AnalyzeTokenOccurrences(
	context.Context,
	string,
) ([]kagome.TokenOccurrence, error) {
	return nil, nil
}

func TestPreprocessRejectsAnalyzerThatLosesEntireQuery(t *testing.T) {
	t.Parallel()

	preprocessor, err := New(Values{
		Analyzer: emptyOccurrenceAnalyzer{},
		LawNames: validInternalLawEntries(),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 前処理器を作成できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "民法",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 照会を作成できません: %v", err)
	}

	if _, err := preprocessor.Preprocess(context.Background(), request); err == nil {
		t.Fatal("SOT-MODEL-025: 原文を失った形態素解析結果を受理しました")
	}
}

func TestNewSupportsCueOnlyProfileWithoutLawVocabulary(t *testing.T) {
	t.Parallel()

	analyzer, err := kagome.NewAnalyzer([]string{"裁判例を読む"})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 形態素解析器を作成できません: %v", err)
	}
	preprocessor, err := New(Values{
		Analyzer: analyzer,
		Cues: []legalquery.CueVocabularyEntry{{
			ProfileID:  "judicial-cases",
			CueID:      "decision-read",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"裁判例を読む"},
		}},
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: cue 専用前処理器を作成できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "裁判例を読む",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 照会を作成できません: %v", err)
	}

	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: cue 専用 Preprocess() のエラー = %v", err)
	}
	cues := result.CueMentions()
	if len(cues) != 1 ||
		cues[0].ProfileID() != "judicial-cases" ||
		cues[0].CueID() != "decision-read" {
		t.Fatalf("SOT-MODEL-025: cue mentions = %#v", cues)
	}
	if len(result.LawNameMentions()) != 0 {
		t.Fatalf(
			"SOT-ARCH-021: cue 専用 profile に法令名を作りました: %#v",
			result.LawNameMentions(),
		)
	}
}

func TestPreprocessAnalyzesOnceWhenResolvingTypoCandidates(t *testing.T) {
	t.Parallel()

	base, err := kagome.NewAnalyzer([]string{"労働契約法"})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 形態素解析器を作成できません: %v", err)
	}
	analyzer := &countingAnalyzer{inner: base}
	law := validInternalLawEntries()[0]
	law.ResourceID = "419AC0000000128"
	law.RevisionID = "419AC0000000128_20200401_430AC0000000071"
	law.LawNumber = "平成十九年法律第百二十八号"
	law.Canonical = "労働契約法"
	law.Terms = nil
	preprocessor, err := New(Values{
		Analyzer: analyzer,
		LawNames: []lawnamelexicon.Entry{law},
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 前処理器を作成できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "労契約法",
	})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 照会を作成できません: %v", err)
	}

	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	laws := result.LawNameMentions()
	if len(laws) != 1 ||
		laws[0].MatchKind() != legalquery.PreprocessMatchUniqueTypoCorrection {
		t.Fatalf("SOT-MODEL-025: law mentions = %#v", laws)
	}
	if analyzer.occurrenceCalls != 1 || analyzer.registeredCalls != 0 {
		t.Fatalf(
			"SOT-ARCH-021: analyzer 呼出し = occurrences:%d registered:%d",
			analyzer.occurrenceCalls,
			analyzer.registeredCalls,
		)
	}
}

func TestParsePositiveNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  int
	}{
		{input: "9", want: 9},
		{input: "３９８", want: 398},
		{input: "九", want: 9},
		{input: "十", want: 10},
		{input: "二十二", want: 22},
		{input: "百二十八", want: 128},
		{input: "一〇", want: 10},
		{input: "一万二千三百四十五", want: 12345},
	}
	for _, test := range tests {
		test := test
		t.Run(test.input, func(t *testing.T) {
			t.Parallel()

			got, err := parsePositiveNumber(test.input)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: parsePositiveNumber() のエラー = %v", err)
			}
			if got != test.want {
				t.Fatalf(
					"SOT-MODEL-025: parsePositiveNumber(%q) = %d, want %d",
					test.input,
					got,
					test.want,
				)
			}
		})
	}
}

func TestParsePositiveNumberRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	for _, input := range []string{"", "0", "零", "一A", strings.Repeat("九", 64)} {
		input := input
		t.Run(input, func(t *testing.T) {
			t.Parallel()

			if _, err := parsePositiveNumber(input); err == nil {
				t.Fatalf("SOT-MODEL-025: 不正な正数 %q を受理しました", input)
			}
		})
	}
}

func TestNormalizeComparisonRune(t *testing.T) {
	t.Parallel()

	tests := map[rune]rune{
		'A': 'a',
		'ア': 'あ',
		'ヽ': 'ゝ',
		'ヾ': 'ゞ',
		'法': '法',
	}
	for input, want := range tests {
		if got := normalizeComparisonRune(input); got != want {
			t.Errorf(
				"SOT-ARCH-021: normalizeComparisonRune(%q) = %q, want %q",
				input,
				got,
				want,
			)
		}
	}
}

func TestDeduplicateDraftsKeepStrongestMatchAndDeterministicOrder(t *testing.T) {
	t.Parallel()

	laws := deduplicateLawDrafts([]lawDraft{
		{
			startByte: 3,
			endByte:   9,
			lawID:     "law-b",
			matchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
		{
			startByte: 0,
			endByte:   6,
			lawID:     "law-a",
			matchKind: legalquery.PreprocessMatchUniqueTypoCorrection,
		},
		{
			startByte: 0,
			endByte:   6,
			lawID:     "law-a",
			matchKind: legalquery.PreprocessMatchExact,
		},
		{
			startByte: 0,
			endByte:   9,
			lawID:     "law-c",
			matchKind: legalquery.PreprocessMatchComparisonNormalized,
		},
	})
	if len(laws) != 3 {
		t.Fatalf("SOT-MODEL-025: law drafts = %#v", laws)
	}
	if laws[0].lawID != "law-c" ||
		laws[1].lawID != "law-a" ||
		laws[1].matchKind != legalquery.PreprocessMatchExact ||
		laws[2].lawID != "law-b" {
		t.Fatalf("SOT-MODEL-025: law drafts の順序又は優先度 = %#v", laws)
	}

	concepts := deduplicateConceptDrafts([]conceptDraft{
		{
			startByte: 0,
			endByte:   6,
			conceptID: "concept-b",
			matchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
		{
			startByte: 0,
			endByte:   6,
			conceptID: "concept-a",
			matchKind: legalquery.PreprocessMatchUniqueTypoCorrection,
		},
		{
			startByte: 0,
			endByte:   6,
			conceptID: "concept-a",
			matchKind: legalquery.PreprocessMatchComparisonNormalized,
		},
	})
	if len(concepts) != 2 ||
		concepts[0].conceptID != "concept-a" ||
		concepts[0].matchKind != legalquery.PreprocessMatchComparisonNormalized ||
		concepts[1].conceptID != "concept-b" {
		t.Fatalf("SOT-MODEL-025: concept drafts = %#v", concepts)
	}

	cues := deduplicateCueDrafts([]cueDraft{
		{
			startByte: 0,
			endByte:   6,
			profileID: "core",
			cueID:     "search",
			matchKind: legalquery.PreprocessMatchRegisteredTerm,
		},
		{
			startByte: 0,
			endByte:   6,
			profileID: "core",
			cueID:     "search",
			matchKind: legalquery.PreprocessMatchExact,
		},
	})
	if len(cues) != 1 ||
		cues[0].matchKind != legalquery.PreprocessMatchExact {
		t.Fatalf("SOT-MODEL-025: cue drafts = %#v", cues)
	}
}

func TestPreprocessMatchPriorityRejectsUnknownKind(t *testing.T) {
	t.Parallel()

	if got := preprocessMatchPriority("unknown"); got != 4 {
		t.Fatalf("SOT-MODEL-025: 未知の match kind の優先度 = %d", got)
	}
}

func TestDeduplicateCueDraftsは同じ意味群の最長spanをProfile別に残す(
	t *testing.T,
) {
	t.Parallel()

	values := []cueDraft{
		{
			startByte:  0,
			endByte:    6,
			profileID:  "core",
			cueID:      "short",
			matchGroup: "same-tuple",
		},
		{
			startByte:  0,
			endByte:    18,
			profileID:  "core",
			cueID:      "long",
			matchGroup: "same-tuple",
		},
		{
			startByte:  0,
			endByte:    6,
			profileID:  "judicial-cases",
			cueID:      "short",
			matchGroup: "same-tuple",
		},
	}
	got := deduplicateCueDrafts(values)
	if len(got) != 2 ||
		got[0].profileID != "core" ||
		got[0].cueID != "long" ||
		got[1].profileID != "judicial-cases" ||
		got[1].cueID != "short" {
		t.Fatalf(
			"cue-loader-longest-same-tuple: cue drafts = %#v",
			got,
		)
	}
}

func TestNewRejectsInvalidVocabularyEntries(t *testing.T) {
	t.Parallel()

	validLaws := validInternalLawEntries()
	validConcept := validInternalConceptEntries()[0]
	validCue := legalquery.CueVocabularyEntry{
		ProfileID:  "core",
		CueID:      "read",
		SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
		Terms:      []string{"読む"},
	}
	tests := []struct {
		name   string
		values Values
	}{
		{
			name: "法令 ID 欠落",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: []lawnamelexicon.Entry{{
					RevisionID: validLaws[0].RevisionID,
					LawNumber:  validLaws[0].LawNumber,
					Canonical:  validLaws[0].Canonical,
				}},
			},
		},
		{
			name: "法令 ID 重複",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: append(validLaws, validLaws[0]),
			},
		},
		{
			name: "概念 ID 形式",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				LegalConcepts: []legalconceptlexicon.Entry{{
					ConceptID: "Invalid_ID",
					Canonical: validConcept.Canonical,
					Terms:     validConcept.Terms,
				}},
			},
		},
		{
			name: "概念語欠落",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				LegalConcepts: []legalconceptlexicon.Entry{{
					ConceptID: "concept",
					Canonical: "法概念",
				}},
			},
		},
		{
			name: "概念 ID 重複",
			values: Values{
				Analyzer:      emptyOccurrenceAnalyzer{},
				LawNames:      validLaws,
				LegalConcepts: []legalconceptlexicon.Entry{validConcept, validConcept},
			},
		},
		{
			name: "cue profile ID 形式",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  "Core",
					CueID:      validCue.CueID,
					SyntaxRole: validCue.SyntaxRole,
					Terms:      validCue.Terms,
				}},
			},
		},
		{
			name: "cue ID 形式",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      "Read_Value",
					SyntaxRole: validCue.SyntaxRole,
					Terms:      validCue.Terms,
				}},
			},
		},
		{
			name: "cue match group 形式",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      validCue.CueID,
					MatchGroup: "Invalid_Group",
					SyntaxRole: validCue.SyntaxRole,
					Terms:      validCue.Terms,
				}},
			},
		},
		{
			name: "cue match group 長超過",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      validCue.CueID,
					MatchGroup: strings.Repeat("a", maxCueMatchGroupBytes+1),
					SyntaxRole: validCue.SyntaxRole,
					Terms:      validCue.Terms,
				}},
			},
		},
		{
			name: "cue 構文 role 欠落",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID: validCue.ProfileID,
					CueID:     validCue.CueID,
					Terms:     validCue.Terms,
				}},
			},
		},
		{
			name: "cue 構文 role が未知",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      validCue.CueID,
					SyntaxRole: legalquery.CueSyntaxRole("unknown"),
					Terms:      validCue.Terms,
				}},
			},
		},
		{
			name: "cue 語欠落",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      validCue.CueID,
					SyntaxRole: validCue.SyntaxRole,
				}},
			},
		},
		{
			name: "cue 語重複",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues: []legalquery.CueVocabularyEntry{{
					ProfileID:  validCue.ProfileID,
					CueID:      validCue.CueID,
					SyntaxRole: validCue.SyntaxRole,
					Terms:      []string{"読む", "読む"},
				}},
			},
		},
		{
			name: "cue 識別子重複",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues:     []legalquery.CueVocabularyEntry{validCue, validCue},
			},
		},
		{
			name: "cue 件数超過",
			values: Values{
				Analyzer: emptyOccurrenceAnalyzer{},
				LawNames: validLaws,
				Cues:     make([]legalquery.CueVocabularyEntry, maxCueEntries+1),
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := New(test.values); err == nil {
				t.Fatal("SOT-ARCH-021: 不正な辞書又は cue を受理しました")
			}
		})
	}
}

func TestNewRejectsTypedNilAnalyzer(t *testing.T) {
	t.Parallel()

	var analyzer *kagome.Analyzer
	if _, err := New(Values{
		Analyzer: analyzer,
		LawNames: validInternalLawEntries(),
	}); err == nil {
		t.Fatal("SOT-ARCH-021: typed nil analyzer を受理しました")
	}
}

func TestPreprocessPropagatesAnalyzerError(t *testing.T) {
	t.Parallel()

	want := errors.New("試験用解析エラー")
	analyzer := errorOccurrenceAnalyzer{err: want}
	preprocessor, err := New(Values{
		Analyzer: analyzer,
		LawNames: validInternalLawEntries(),
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: 前処理器を作成できません: %v", err)
	}
	request, err := legalquery.NewRequest(legalquery.RequestValues{Query: "民法"})
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 照会を作成できません: %v", err)
	}

	if _, err := preprocessor.Preprocess(context.Background(), request); !errors.Is(
		err,
		want,
	) {
		t.Fatalf("SOT-MODEL-025: 解析エラー = %v, want %v", err, want)
	}
}

type errorOccurrenceAnalyzer struct {
	err error
}

func (a errorOccurrenceAnalyzer) RegisteredTerms(
	context.Context,
	string,
) ([]string, error) {
	return nil, a.err
}

func (a errorOccurrenceAnalyzer) AnalyzeTokenOccurrences(
	context.Context,
	string,
) ([]kagome.TokenOccurrence, error) {
	return nil, a.err
}

type countingAnalyzer struct {
	inner           *kagome.Analyzer
	registeredCalls int
	occurrenceCalls int
}

func (a *countingAnalyzer) RegisteredTerms(
	ctx context.Context,
	input string,
) ([]string, error) {
	a.registeredCalls++
	return a.inner.RegisteredTerms(ctx, input)
}

func (a *countingAnalyzer) AnalyzeTokenOccurrences(
	ctx context.Context,
	input string,
) ([]kagome.TokenOccurrence, error) {
	a.occurrenceCalls++
	return a.inner.AnalyzeTokenOccurrences(ctx, input)
}

func validInternalLawEntries() []lawnamelexicon.Entry {
	return []lawnamelexicon.Entry{{
		ResourceID: "129AC0000000089",
		RevisionID: "129AC0000000089_20260624_508AC0000000045",
		LawNumber:  "明治二十九年法律第八十九号",
		Canonical:  "民法",
		Terms:      []string{"みんぽう"},
	}}
}

func validInternalConceptEntries() []legalconceptlexicon.Entry {
	return []legalconceptlexicon.Entry{{
		ConceptID: "permanent-residence",
		Canonical: "永住権",
		Terms:     []string{"永住権"},
	}}
}

func TestCompareDraftCoversEveryTieBreak(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		got  int
		want int
	}{
		{
			name: "開始位置",
			got:  compareDraft(0, 3, "b", 1, 9, "a"),
			want: -1,
		},
		{
			name: "開始位置の逆順",
			got:  compareDraft(2, 3, "a", 1, 9, "b"),
			want: 1,
		},
		{
			name: "長い span",
			got:  compareDraft(0, 9, "b", 0, 3, "a"),
			want: -1,
		},
		{
			name: "短い span",
			got:  compareDraft(0, 3, "a", 0, 9, "b"),
			want: 1,
		},
		{
			name: "識別子",
			got:  compareDraft(0, 3, "a", 0, 3, "b"),
			want: -1,
		},
	}
	for _, test := range tests {
		if got := test.got; got != test.want {
			t.Errorf("SOT-MODEL-025: %s = %d, want %d", test.name, got, test.want)
		}
	}
	if !slices.Equal(
		[]int{preprocessMatchPriority(legalquery.PreprocessMatchExact),
			preprocessMatchPriority(legalquery.PreprocessMatchComparisonNormalized),
			preprocessMatchPriority(legalquery.PreprocessMatchRegisteredTerm),
			preprocessMatchPriority(legalquery.PreprocessMatchUniqueTypoCorrection)},
		[]int{0, 1, 2, 3},
	) {
		t.Fatal("SOT-MODEL-025: match kind の優先順位が一致しません")
	}
}
