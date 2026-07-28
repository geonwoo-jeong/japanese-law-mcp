package kagome

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
)

type tokenOccurrenceSnapshot struct {
	surface        string
	startByte      int
	endByte        int
	userDictionary bool
	partOfSpeech   []string
}

func TestAnalyzeTokenOccurrencesPreservesUTF8ByteSpansAndDuplicates(
	t *testing.T,
) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"個情法"})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewAnalyzer() のエラー = %v", err)
	}
	const input = "前個情法と個情法後"
	got, err := analyzer.AnalyzeTokenOccurrences(context.Background(), input)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: AnalyzeTokenOccurrences() のエラー = %v", err)
	}

	var reconstructed strings.Builder
	nonUserDictionaryCount := 0
	for index, occurrence := range got {
		startByte := occurrence.StartByte()
		endByte := occurrence.EndByte()
		if startByte < 0 || endByte <= startByte || endByte > len(input) {
			t.Fatalf(
				"SOT-ARCH-021: occurrences[%d] の byte span = [%d,%d)",
				index,
				startByte,
				endByte,
			)
		}
		if input[startByte:endByte] != occurrence.Surface() {
			t.Fatalf(
				"SOT-ARCH-021: occurrences[%d] の原文 = %q, surface = %q",
				index,
				input[startByte:endByte],
				occurrence.Surface(),
			)
		}
		reconstructed.WriteString(occurrence.Surface())
		if !occurrence.UserDictionary() {
			nonUserDictionaryCount++
		}
	}
	if reconstructed.String() != input {
		t.Fatalf(
			"SOT-ARCH-021: token surface の連結 = %q, want %q",
			reconstructed.String(),
			input,
		)
	}
	if nonUserDictionaryCount == 0 {
		t.Fatal("SOT-ARCH-021: user dictionary 外の token がありません")
	}

	registered := make([]tokenOccurrenceSnapshot, 0, 2)
	for _, occurrence := range got {
		if occurrence.UserDictionary() {
			registered = append(
				registered,
				snapshotTokenOccurrence(occurrence),
			)
		}
	}
	want := []tokenOccurrenceSnapshot{
		{
			surface:        "個情法",
			startByte:      len("前"),
			endByte:        len("前個情法"),
			userDictionary: true,
			partOfSpeech:   []string{"検索登録語"},
		},
		{
			surface:        "個情法",
			startByte:      len("前個情法と"),
			endByte:        len("前個情法と個情法"),
			userDictionary: true,
			partOfSpeech:   []string{"検索登録語"},
		},
	}
	if !reflect.DeepEqual(registered, want) {
		t.Fatalf(
			"SOT-ARCH-021: user dictionary occurrence = %#v, want %#v",
			registered,
			want,
		)
	}
}

func TestAnalyzeTokenOccurrencesHonorsCancellationAndInputLimit(t *testing.T) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"個情法"})
	if err != nil {
		t.Fatalf("NewAnalyzer() のエラー = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, analyzeErr := analyzer.AnalyzeTokenOccurrences(
		ctx,
		"個情法",
	); !errors.Is(analyzeErr, context.Canceled) {
		t.Fatalf(
			"SOT-ENG-010: cancel 済み context のエラー = %v",
			analyzeErr,
		)
	}

	withinLimit := strings.Repeat("a", maxAnalyzerInputBytes)
	if _, analyzeErr := analyzer.AnalyzeTokenOccurrences(
		context.Background(),
		withinLimit,
	); analyzeErr != nil {
		t.Fatalf(
			"SOT-ARCH-021: %d byte の入力エラー = %v",
			maxAnalyzerInputBytes,
			analyzeErr,
		)
	}
	overLimit := strings.Repeat("a", maxAnalyzerInputBytes+1)
	if _, analyzeErr := analyzer.AnalyzeTokenOccurrences(
		context.Background(),
		overLimit,
	); analyzeErr == nil {
		t.Fatalf(
			"SOT-ARCH-021: %d byte の入力を受理しました",
			maxAnalyzerInputBytes+1,
		)
	}
}

func TestAnalyzeTokenOccurrencesCopiesPartOfSpeech(t *testing.T) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"検索"})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewAnalyzer() のエラー = %v", err)
	}
	occurrences, err := analyzer.AnalyzeTokenOccurrences(
		context.Background(),
		"営業秘密を検索",
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: AnalyzeTokenOccurrences() のエラー = %v", err)
	}

	var nounFound bool
	var registeredFound bool
	for _, occurrence := range occurrences {
		partOfSpeech := occurrence.PartOfSpeech()
		if len(partOfSpeech) == 0 {
			t.Fatalf(
				"SOT-ARCH-021: %q の品詞がありません",
				occurrence.Surface(),
			)
		}
		switch occurrence.Surface() {
		case "営業", "秘密":
			if partOfSpeech[0] == "名詞" {
				nounFound = true
			}
		case "検索":
			if occurrence.UserDictionary() &&
				partOfSpeech[0] == "検索登録語" {
				registeredFound = true
			}
		}

		original := append([]string(nil), partOfSpeech...)
		partOfSpeech[0] = "変更済み"
		if !reflect.DeepEqual(occurrence.PartOfSpeech(), original) {
			t.Fatalf(
				"SOT-ARCH-021: %q の品詞 getter から値を変更できました",
				occurrence.Surface(),
			)
		}
	}
	if !nounFound || !registeredFound {
		t.Fatalf(
			"SOT-ARCH-021: 名詞または user dictionary 品詞を確認できません: %#v",
			snapshotTokenOccurrences(occurrences),
		)
	}
}

func TestAnalyzeTokenOccurrencesIsDeterministicUnderConcurrentUse(
	t *testing.T,
) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"個情法", "道交法"})
	if err != nil {
		t.Fatalf("NewAnalyzer() のエラー = %v", err)
	}
	const input = "個情法と道交法と個情法"
	baseline, err := analyzer.AnalyzeTokenOccurrences(
		context.Background(),
		input,
	)
	if err != nil {
		t.Fatalf("AnalyzeTokenOccurrences() のエラー = %v", err)
	}
	want := snapshotTokenOccurrences(baseline)
	if len(want) == 0 {
		t.Fatal("SOT-ARCH-021: token occurrence が空です")
	}

	// 呼出し元による返却 slice の変更が後続の解析結果へ漏れないことを確認する。
	baseline[0] = TokenOccurrence{}
	again, err := analyzer.AnalyzeTokenOccurrences(
		context.Background(),
		input,
	)
	if err != nil {
		t.Fatalf("AnalyzeTokenOccurrences() の再実行エラー = %v", err)
	}
	if got := snapshotTokenOccurrences(again); !reflect.DeepEqual(got, want) {
		t.Fatalf(
			"SOT-ARCH-021: 返却 slice 変更後 = %#v, want %#v",
			got,
			want,
		)
	}

	const goroutines = 24
	results := make(chan []tokenOccurrenceSnapshot, goroutines)
	errorsChannel := make(chan error, goroutines)
	var waitGroup sync.WaitGroup
	for range goroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			occurrences, analyzeErr := analyzer.AnalyzeTokenOccurrences(
				context.Background(),
				input,
			)
			if analyzeErr != nil {
				errorsChannel <- analyzeErr
				return
			}
			results <- snapshotTokenOccurrences(occurrences)
		}()
	}
	waitGroup.Wait()
	close(results)
	close(errorsChannel)

	for analyzeErr := range errorsChannel {
		t.Errorf("AnalyzeTokenOccurrences() の並行実行エラー = %v", analyzeErr)
	}
	for got := range results {
		if !reflect.DeepEqual(got, want) {
			t.Errorf(
				"SOT-ARCH-021: 並行実行結果 = %#v, want %#v",
				got,
				want,
			)
		}
	}
}

func snapshotTokenOccurrences(
	occurrences []TokenOccurrence,
) []tokenOccurrenceSnapshot {
	snapshots := make([]tokenOccurrenceSnapshot, 0, len(occurrences))
	for _, occurrence := range occurrences {
		snapshots = append(snapshots, snapshotTokenOccurrence(occurrence))
	}
	return snapshots
}

func snapshotTokenOccurrence(
	occurrence TokenOccurrence,
) tokenOccurrenceSnapshot {
	return tokenOccurrenceSnapshot{
		surface:        occurrence.Surface(),
		startByte:      occurrence.StartByte(),
		endByte:        occurrence.EndByte(),
		userDictionary: occurrence.UserDictionary(),
		partOfSpeech:   occurrence.PartOfSpeech(),
	}
}
