package querypreprocess_test

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestPreprocessCopiesRefWithoutProviderOrCapabilityDecision(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	ref := mustLawRef(t)
	request := mustRequestWithRef(t, "刑法第199条を読む", ref)
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: Preprocess() のエラー = %v", err)
	}

	gotRef, exists := result.Ref()
	if !exists {
		t.Fatal("SOT-MODEL-025: 入力 ref が失われました")
	}
	if gotRef.ProviderID() != ref.ProviderID() ||
		gotRef.Key().SourceID() != ref.Key().SourceID() ||
		gotRef.Key().ResourceType() != ref.Key().ResourceType() ||
		gotRef.Key().ResourceID() != ref.Key().ResourceID() {
		t.Fatalf("SOT-MODEL-025: ref = %#v, want %#v", gotRef, ref)
	}
	laws := result.LawNameMentions()
	if len(laws) != 1 || laws[0].LawID() != penalLawID {
		t.Fatalf("SOT-MODEL-025: query 自体の法令名 = %#v", laws)
	}
	if gotRef.Key().ResourceID() == laws[0].LawID() {
		t.Fatal("試験条件が ref と照会中の法令名を区別できていません")
	}
}

func TestPreprocessRejectsNilAndCanceledContext(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	request := mustRequest(t, "民法")
	var nilContext context.Context
	if _, err := preprocessor.Preprocess(nilContext, request); err == nil {
		t.Fatal("SOT-MODEL-025: nil context を受理しました")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := preprocessor.Preprocess(ctx, request); !errors.Is(
		err,
		context.Canceled,
	) {
		t.Fatalf("SOT-MODEL-025: cancel 済み context のエラー = %v", err)
	}
}

func TestPreprocessorCopiesConstructionValues(t *testing.T) {
	t.Parallel()

	laws := testLawEntries()
	concepts := testConceptEntries()
	cues := testCueEntries()
	preprocessor := mustNewPreprocessor(t, laws, concepts, cues)

	laws[0].Canonical = "変更済み法令名"
	laws[0].Terms[0] = "変更済み読み"
	concepts[0].ConceptID = "changed-concept"
	concepts[0].Terms[0] = "変更済み概念"
	cues[0].ProfileID = "changed-profile"
	cues[0].Terms[0] = "変更済み cue"

	for _, test := range []struct {
		query  string
		assert func(*testing.T, legalquery.PreprocessResult)
	}{
		{
			query: "民法",
			assert: func(t *testing.T, result legalquery.PreprocessResult) {
				t.Helper()
				mentions := result.LawNameMentions()
				if len(mentions) != 1 || mentions[0].Canonical() != "民法" {
					t.Fatalf("SOT-ARCH-021: 法令辞書の変更が漏れました: %#v", mentions)
				}
			},
		},
		{
			query: "永住権",
			assert: func(t *testing.T, result legalquery.PreprocessResult) {
				t.Helper()
				mentions := result.LegalConceptMentions()
				if len(mentions) != 1 ||
					mentions[0].ConceptID() != "permanent-residence" {
					t.Fatalf("SOT-ARCH-021: 法概念辞書の変更が漏れました: %#v", mentions)
				}
			},
		},
		{
			query: "読む",
			assert: func(t *testing.T, result legalquery.PreprocessResult) {
				t.Helper()
				mentions := result.CueMentions()
				if len(mentions) != 1 ||
					mentions[0].ProfileID() != testCueProfileID {
					t.Fatalf("SOT-ARCH-021: cue 語彙の変更が漏れました: %#v", mentions)
				}
			},
		},
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
			test.assert(t, result)
		})
	}
}

func TestPreprocessorIsDeterministicUnderConcurrentUse(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	const query = "2026年7月1日時点の民法第709条第1項を読んで"
	request := mustRequest(t, query)
	baseline, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("SOT-MODEL-025: 基準 Preprocess() のエラー = %v", err)
	}
	want := snapshotResult(baseline)

	const goroutineCount = 24
	results := make(chan preprocessSnapshot, goroutineCount)
	errorChannel := make(chan error, goroutineCount)
	var waitGroup sync.WaitGroup
	for range goroutineCount {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			result, preprocessErr := preprocessor.Preprocess(
				context.Background(),
				request,
			)
			if preprocessErr != nil {
				errorChannel <- preprocessErr
				return
			}
			results <- snapshotResult(result)
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorChannel)

	for preprocessErr := range errorChannel {
		t.Errorf("SOT-ARCH-021: 並行 Preprocess() のエラー = %v", preprocessErr)
	}
	for got := range results {
		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"SOT-MODEL-025: 並行実行結果 = %#v, want %#v",
				got,
				want,
			)
		}
	}
}

func TestPreprocessFailsInsteadOfTruncatingMentionOverflow(t *testing.T) {
	t.Parallel()

	preprocessor := mustDefaultPreprocessor(t)
	request := mustRequest(t, strings.Repeat("民法", 65))
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err == nil {
		t.Fatalf(
			"SOT-MODEL-025: 六十四件を超える出現を切り捨てて受理しました: %#v",
			snapshotResult(result),
		)
	}
	if len(result.LawNameMentions()) != 0 ||
		len(result.LegalConceptMentions()) != 0 ||
		len(result.CueMentions()) != 0 ||
		len(result.IdentifierMentions()) != 0 ||
		len(result.DateMentions()) != 0 ||
		len(result.ArticleMentions()) != 0 ||
		len(result.ParagraphMentions()) != 0 {
		t.Fatalf(
			"SOT-MODEL-025: エラー時に途中結果を返しました: %#v",
			snapshotResult(result),
		)
	}
}

func TestNewRejectsMissingAnalyzerAndEmptyVocabulary(t *testing.T) {
	t.Parallel()

	for name, values := range map[string]querypreprocess.Values{
		"analyzer 欠落": {
			LawNames: testLawEntries(),
			Cues:     testCueEntries(),
		},
		"全語彙欠落": {},
	} {
		name := name
		values := values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := querypreprocess.New(values); err == nil {
				t.Fatal("SOT-ARCH-021: 不完全な起動構成を受理しました")
			}
		})
	}
}
