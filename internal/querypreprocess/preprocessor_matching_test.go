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

func TestNewEmbeddedは法令略称の文字幅差を誤記にしない(t *testing.T) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "JAS法という法令略称を検索してください。"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	mentions := result.LawNameMentions()
	if len(mentions) != 1 ||
		mentions[0].Canonical() != "日本農林規格等に関する法律" ||
		mentions[0].MatchKind() !=
			legalquery.PreprocessMatchComparisonNormalized {
		t.Fatalf(
			"SOT-MODEL-025: 文字幅差の lawNameMentions = %#v",
			mentions,
		)
	}
}

func TestNewEmbeddedは自然文中の長い正式法令名の一意な誤記を補正する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	query := "個人情報の保護に関係する法律という名前で法令を探して。"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	mentions := result.LawNameMentions()
	if len(mentions) != 1 ||
		mentions[0].LawID() != "415AC0000000057" ||
		mentions[0].Canonical() != "個人情報の保護に関する法律" ||
		mentions[0].Surface() != "個人情報の保護に関係する法律" ||
		mentions[0].MatchKind() !=
			legalquery.PreprocessMatchUniqueTypoCorrection {
		t.Fatalf(
			"SOT-MODEL-025: 長い正式法令名の誤記 = %#v",
			mentions,
		)
	}
}

func TestPreprocessは誤記候補窓の拡張前に受理した入力境界を維持する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	query := strings.Repeat("法令", 87)
	if _, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	); err != nil {
		t.Fatalf(
			"SOT-ARCH-021: 旧六 token 窓で受理できた照会のエラー = %v",
			err,
		)
	}
}

func TestPreprocessDoesNotCorrectPostpositionEndedPhraseToLawName(
	t *testing.T,
) {
	t.Parallel()

	laws := []lawnamelexicon.Entry{
		{
			ResourceID: "132AC0000000015",
			RevisionID: "132AC0000000015_20160401_426AC0000000069",
			LawNumber:  "明治三十二年法律第十五号",
			Canonical:  "供託法",
		},
		{
			ResourceID: "132AC0000000046",
			RevisionID: "132AC0000000046_20250601_504AC0000000068",
			LawNumber:  "明治三十二年法律第四十六号",
			Canonical:  "船舶法",
		},
	}
	preprocessor := mustNewPreprocessor(
		t,
		laws,
		nil,
		[]legalquery.CueVocabularyEntry{
			{
				ProfileID:  testCueProfileID,
				CueID:      "task-search",
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索"},
			},
			{
				ProfileID:  testCueProfileID,
				CueID:      "resource-provision",
				SyntaxRole: legalquery.CueSyntaxRoleNone,
				Terms:      []string{"条文"},
			},
		},
	)
	for _, query := range []string{
		"供託を検索",
		"船舶を含む条文を検索",
	} {
		result, err := preprocessor.Preprocess(
			context.Background(),
			mustRequest(t, query),
		)
		if err != nil {
			t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
		}
		if mentions := result.LawNameMentions(); len(mentions) != 0 {
			t.Fatalf(
				"SOT-MODEL-025: 助詞を法令名の誤記として補正しました: query=%q mentions=%#v",
				query,
				mentions,
			)
		}
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

func TestNewEmbeddedは口語引用境界直前の曖昧な略称誤記を保持する(
	t *testing.T,
) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	query := "民訴去っていう法令を検索してもらえますか。"
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, query),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	mentions := result.LawNameMentions()
	if len(mentions) != 2 {
		t.Fatalf(
			"SOT-MODEL-025: 同じ略称の対象を保持できません: %#v",
			mentions,
		)
	}
	wantIDs := []string{"215AC0000000062", "408AC0000000109"}
	wantCanonicals := []string{
		"民事訴訟法中改正法律施行法",
		"民事訴訟法",
	}
	for index, mention := range mentions {
		if mention.LawID() != wantIDs[index] ||
			mention.Canonical() != wantCanonicals[index] ||
			mention.Surface() != "民訴去" ||
			mention.MatchKind() !=
				legalquery.PreprocessMatchUniqueTypoCorrection {
			t.Fatalf(
				"SOT-MODEL-025: lawNameMentions[%d] = %#v",
				index,
				mention,
			)
		}
		assertSpan(t, query, mention.Span(), "民訴去", 0)
	}
}

func TestPreprocessは口語引用でない促音を法令名境界にしない(
	t *testing.T,
) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "民訴去ってしまう"),
	)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}
	if mentions := result.LawNameMentions(); len(mentions) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: 通常の促音から法令名を作りました: %#v",
			mentions,
		)
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
			ProfileID:  "judicial-cases-test",
			CueID:      "decision-read",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"裁判例を読む"},
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

func TestPreprocessKeepsLongestCueMentionForSameMatchGroup(t *testing.T) {
	t.Parallel()

	const matchGroup = "cue-tuple-test-search"
	preprocessor := mustNewPreprocessor(
		t,
		nil,
		nil,
		[]legalquery.CueVocabularyEntry{
			{
				ProfileID:  testCueProfileID,
				CueID:      "task-search-short",
				MatchGroup: matchGroup,
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索"},
			},
			{
				ProfileID:  testCueProfileID,
				CueID:      "task-search-long",
				MatchGroup: matchGroup,
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索してください"},
			},
		},
	)

	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "検 索 してください"),
	)
	if err != nil {
		t.Fatalf("cue-loader-longest-same-tuple: Preprocess() のエラー = %v", err)
	}
	mentions := result.CueMentions()
	if len(mentions) != 1 ||
		mentions[0].CueID() != "task-search-long" ||
		mentions[0].Span().StartByte() != 0 ||
		mentions[0].Span().EndByte() != len("検 索 してください") {
		t.Fatalf(
			"cue-loader-longest-same-tuple: cue mentions = %#v",
			mentions,
		)
	}
}

func TestPreprocessDoesNotCombineSameMatchGroupAcrossProfiles(t *testing.T) {
	t.Parallel()

	preprocessor := mustNewPreprocessor(
		t,
		nil,
		nil,
		[]legalquery.CueVocabularyEntry{
			{
				ProfileID:  "core-test",
				CueID:      "task-search",
				MatchGroup: "cue-tuple-test-search",
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索"},
			},
			{
				ProfileID:  "judicial-test",
				CueID:      "task-search",
				MatchGroup: "cue-tuple-test-search",
				SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
				Terms:      []string{"検索してください"},
			},
		},
	)

	result, err := preprocessor.Preprocess(
		context.Background(),
		mustRequest(t, "検 索 してください"),
	)
	if err != nil {
		t.Fatalf("cue-loader-cross-profile-reuse: Preprocess() のエラー = %v", err)
	}
	mentions := result.CueMentions()
	if len(mentions) != 2 ||
		mentions[0].ProfileID() == mentions[1].ProfileID() {
		t.Fatalf(
			"cue-loader-cross-profile-reuse: cue mentions = %#v",
			mentions,
		)
	}
}

func TestNewEmbeddedLoadsOneImmutableSharedVocabulary(t *testing.T) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded([]legalquery.CueVocabularyEntry{
		{
			ProfileID:  testCueProfileID,
			CueID:      "task-search",
			SyntaxRole: legalquery.CueSyntaxRoleTaskExpression,
			Terms:      []string{"検索"},
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
		len(result.LegalConceptMentions()) != 2 ||
		len(result.ArticleMentions()) != 1 ||
		len(result.CueMentions()) != 1 {
		t.Fatalf(
			"SOT-MODEL-025: 組込み語彙の前処理結果 = %#v",
			snapshotResult(result),
		)
	}
	concepts := result.LegalConceptMentions()
	if concepts[0].ConceptID() != "permanent-residence" ||
		concepts[1].ConceptID() != "permanent-residence-permission" {
		t.Fatalf(
			"SOT-ENG-023: 衝突する法概念 = %#v",
			snapshotResult(result).concepts,
		)
	}
}

func TestNewEmbeddedPreservesEveryGroundedConceptForOneSurface(t *testing.T) {
	t.Parallel()

	preprocessor, err := querypreprocess.NewEmbedded(nil)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewEmbedded() のエラー = %v", err)
	}
	tests := []struct {
		query      string
		conceptIDs []string
	}{
		{
			query: "育休",
			conceptIDs: []string{
				"childcare-leave",
				"childcare-leave-benefit",
			},
		},
		{
			query:      "ネット中傷",
			conceptIDs: []string{"online-defamation"},
		},
		{
			query: "永住権",
			conceptIDs: []string{
				"permanent-residence",
				"permanent-residence-permission",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.query, func(t *testing.T) {
			t.Parallel()

			result, preprocessErr := preprocessor.Preprocess(
				context.Background(),
				mustRequest(t, test.query),
			)
			if preprocessErr != nil {
				t.Fatalf(
					"SOT-MODEL-025: Preprocess() のエラー = %v",
					preprocessErr,
				)
			}
			mentions := result.LegalConceptMentions()
			conceptIDs := make([]string, 0, len(mentions))
			for _, mention := range mentions {
				conceptIDs = append(conceptIDs, mention.ConceptID())
				if mention.Surface() != test.query ||
					mention.MatchKind() != legalquery.PreprocessMatchExact {
					t.Fatalf(
						"SOT-MODEL-025: 法概念 mention = %#v",
						mention,
					)
				}
			}
			if !slices.Equal(conceptIDs, test.conceptIDs) {
				t.Fatalf(
					"SOT-ENG-023: concept IDs = %#v, want %#v",
					conceptIDs,
					test.conceptIDs,
				)
			}
		})
	}
}
