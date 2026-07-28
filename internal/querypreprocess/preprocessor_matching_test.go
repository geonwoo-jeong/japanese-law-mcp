package querypreprocess_test

import (
	"context"
	"slices"
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestPreprocessDistinguishesFourLawNameMatchKinds(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	tests := []struct {
		name      string
		query     string
		lawID     string
		canonical string
		kind      legalquery.PreprocessMatchKind
	}{
		{
			name:      "正式名称の完全一致",
			query:     "民法",
			lawID:     civilLawID,
			canonical: "民法",
			kind:      legalquery.PreprocessMatchExact,
		},
		{
			name:      "読みの比較用正規化一致",
			query:     "ミンポウ",
			lawID:     civilLawID,
			canonical: "民法",
			kind:      legalquery.PreprocessMatchComparisonNormalized,
		},
		{
			name:      "自然文からの Kagome 登録語抽出",
			query:     "民法を調べてください",
			lawID:     civilLawID,
			canonical: "民法",
			kind:      legalquery.PreprocessMatchRegisteredTerm,
		},
		{
			name:      "一意な別名誤記",
			query:     "労契去",
			lawID:     laborLawID,
			canonical: "労働契約法",
			kind:      legalquery.PreprocessMatchUniqueTypoCorrection,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			laws := result.LawNameMentions()
			if len(laws) != 1 {
				t.Fatalf("SOT-MODEL-025: lawNameMentions = %#v", laws)
			}
			if laws[0].LawID() != test.lawID ||
				laws[0].Canonical() != test.canonical ||
				laws[0].MatchKind() != test.kind {
				t.Fatalf(
					"SOT-MODEL-025: lawNameMention = %#v, want %s/%s/%s",
					laws[0],
					test.lawID,
					test.canonical,
					test.kind,
				)
			}
			startByte := strings.Index(test.query, laws[0].Surface())
			assertSpan(
				t,
				test.query,
				laws[0].Span(),
				laws[0].Surface(),
				startByte,
			)
		})
	}
}

func TestPreprocessAcceptsConceptAndCueOnlyProfile(t *testing.T) {
	t.Parallel()

	preprocessor := mustNewPreprocessor(
		t,
		nil,
		testConceptEntries(),
		testCueEntries(),
	)
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "永住権を検索"),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: Preprocess() のエラー = %v", err)
	}
	if len(result.LawNameMentions()) != 0 {
		t.Fatalf("SOT-ARCH-021: lawNameMentions = %#v", result.LawNameMentions())
	}
	concepts := result.LegalConceptMentions()
	if len(concepts) != 1 ||
		concepts[0].ConceptID() != "permanent-residence" ||
		concepts[0].MatchKind() != legalquery.PreprocessMatchRegisteredTerm {
		t.Fatalf("SOT-ARCH-021: concept mentions = %#v", concepts)
	}
	cues := result.CueMentions()
	if len(cues) != 1 ||
		cues[0].CueID() != "task-search" ||
		cues[0].MatchKind() != legalquery.PreprocessMatchRegisteredTerm {
		t.Fatalf("SOT-ARCH-021: cue mentions = %#v", cues)
	}
}

func TestPreprocessPreservesEveryTargetOfAmbiguousAliasAndTypo(
	t *testing.T,
) {
	t.Parallel()

	laws := []lawnamelexicon.Entry{
		{
			ResourceID: "501AC0000000001",
			RevisionID: "501AC0000000001_20260401_508AC0000000001",
			LawNumber:  "令和元年法律第一号",
			Canonical:  "開示手続法甲",
			Terms:      []string{"開示法"},
		},
		{
			ResourceID: "501AC0000000002",
			RevisionID: "501AC0000000002_20260401_508AC0000000002",
			LawNumber:  "令和元年法律第二号",
			Canonical:  "開示手続法乙",
			Terms:      []string{"開示法"},
		},
	}
	preprocessor := mustNewPreprocessor(t, laws, nil, testCueEntries())
	for _, test := range []struct {
		query string
		kind  legalquery.PreprocessMatchKind
	}{
		{query: "開示法", kind: legalquery.PreprocessMatchExact},
		{query: "開示去", kind: legalquery.PreprocessMatchUniqueTypoCorrection},
	} {
		test := test
		t.Run(test.query, func(t *testing.T) {
			t.Parallel()

			result, err := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if err != nil {
				t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
			}
			mentions := result.LawNameMentions()
			if len(mentions) != 2 {
				t.Fatalf(
					"SOT-MODEL-025: 同じ表記の対象を縮約しました: %#v",
					mentions,
				)
			}
			gotIDs := []string{mentions[0].LawID(), mentions[1].LawID()}
			wantIDs := []string{"501AC0000000001", "501AC0000000002"}
			if !slices.Equal(gotIDs, wantIDs) {
				t.Fatalf("SOT-MODEL-025: law IDs = %#v, want %#v", gotIDs, wantIDs)
			}
			for _, mention := range mentions {
				if mention.MatchKind() != test.kind {
					t.Fatalf(
						"SOT-MODEL-025: match kind = %s, want %s",
						mention.MatchKind(),
						test.kind,
					)
				}
				assertSpan(t, test.query, mention.Span(), mention.Surface(), 0)
			}
		})
	}
}

func TestPreprocessSuppressesSameSpanConceptOnlyForExactLawName(
	t *testing.T,
) {
	t.Parallel()

	laws := []lawnamelexicon.Entry{
		{
			ResourceID: "501AC0000000003",
			RevisionID: "501AC0000000003_20260401_508AC0000000003",
			LawNumber:  "令和元年法律第三号",
			Canonical:  "永住権",
		},
	}
	preprocessor := mustNewPreprocessor(
		t,
		laws,
		testConceptEntries(),
		testCueEntries(),
	)

	lawResult, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "永住権"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	if len(lawResult.LawNameMentions()) != 1 {
		t.Fatalf(
			"SOT-MODEL-025: 完全一致の法令名がありません: %#v",
			lawResult.LawNameMentions(),
		)
	}
	if concepts := lawResult.LegalConceptMentions(); len(concepts) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: 同じ span の法概念を残しました: %#v",
			concepts,
		)
	}

	conceptResult, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "取消訴訟"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	concepts := conceptResult.LegalConceptMentions()
	if len(concepts) != 1 || concepts[0].ConceptID() != "revocation-action" {
		t.Fatalf(
			"SOT-MODEL-025: 別の根拠を持つ法概念まで削除しました: %#v",
			concepts,
		)
	}
}

func TestPreprocessUsesInjectedProfileCuesWithoutTypoCorrection(t *testing.T) {
	t.Parallel()

	cues := []legalquery.CueVocabularyEntry{
		{
			ProfileID: "judicial-cases-test",
			CueID:     "decision-read",
			Terms:     []string{"裁判例を読む"},
		},
	}
	preprocessor := mustNewPreprocessor(t, testLawEntries(), nil, cues)

	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "裁判例を読む"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	mentions := result.CueMentions()
	if len(mentions) != 1 ||
		mentions[0].ProfileID() != "judicial-cases-test" ||
		mentions[0].CueID() != "decision-read" ||
		mentions[0].MatchKind() != legalquery.PreprocessMatchExact {
		t.Fatalf("SOT-MODEL-025: cueMentions = %#v", mentions)
	}

	typoResult, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "裁判例を読も"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: cue 誤記の Preprocess() エラー = %v", err)
	}
	if mentions := typoResult.CueMentions(); len(mentions) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: cue の誤記から task を作りました: %#v",
			mentions,
		)
	}
}

func TestNewEmbeddedLoadsOneImmutableSharedVocabulary(t *testing.T) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded([]legalquery.CueVocabularyEntry{
		{
			ProfileID: testCueProfileID,
			CueID:     "task-search",
			Terms:     []string{"検索"},
		},
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	const query = "民法第709条と永住権を検索"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	if len(result.LawNameMentions()) != 1 ||
		len(result.LegalConceptMentions()) != 1 ||
		len(result.ArticleMentions()) != 1 ||
		len(result.CueMentions()) != 1 {
		t.Fatalf(
			"SOT-MODEL-025: 組込み語彙の前処理結果 = %#v",
			snapshotResult(result),
		)
	}
}
