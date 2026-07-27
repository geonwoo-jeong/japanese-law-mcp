package kagome

import (
	"context"
	"sync"
	"testing"
)

func TestAnalyzerExtractsOnlyRegisteredTerms(t *testing.T) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{
		"個情法",
		"個人情報の保護に関する法律",
	})
	if err != nil {
		t.Fatalf("SOT-ARCH-021: NewAnalyzer() のエラー = %v", err)
	}
	got, err := analyzer.RegisteredTerms(
		context.Background(),
		"個情法について教えてください。",
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-021: RegisteredTerms() のエラー = %v", err)
	}
	if len(got) != 1 || got[0] != "個情法" {
		t.Fatalf("SOT-ARCH-021: RegisteredTerms() = %#v", got)
	}
}

func TestAnalyzerDeduplicatesDictionaryAndResults(t *testing.T) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"道交法", "道交法"})
	if err != nil {
		t.Fatalf("SOT-ENG-022: NewAnalyzer() のエラー = %v", err)
	}
	got, err := analyzer.RegisteredTerms(
		context.Background(),
		"道交法と道交法",
	)
	if err != nil {
		t.Fatalf("RegisteredTerms() のエラー = %v", err)
	}
	if len(got) != 1 || got[0] != "道交法" {
		t.Fatalf("SOT-ENG-022: RegisteredTerms() = %#v", got)
	}
}

func TestAnalyzerRejectsEmptyDictionaryAndHonorsCancellation(t *testing.T) {
	t.Parallel()

	if _, err := NewAnalyzer(nil); err == nil {
		t.Fatal("SOT-ENG-022: 空の user dictionary を受理しました")
	}
	analyzer, err := NewAnalyzer([]string{"独禁法"})
	if err != nil {
		t.Fatalf("NewAnalyzer() のエラー = %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := analyzer.RegisteredTerms(ctx, "独禁法"); err == nil {
		t.Fatal("SOT-ENG-010: cancel 済み context を受理しました")
	}
}

func TestAnalyzerSupportsConcurrentUse(t *testing.T) {
	t.Parallel()

	analyzer, err := NewAnalyzer([]string{"個情法"})
	if err != nil {
		t.Fatalf("NewAnalyzer() のエラー = %v", err)
	}

	const goroutines = 24
	var waitGroup sync.WaitGroup
	for range goroutines {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			got, analyzeErr := analyzer.RegisteredTerms(
				context.Background(),
				"個情法を確認する",
			)
			if analyzeErr != nil {
				t.Errorf("RegisteredTerms() のエラー = %v", analyzeErr)
				return
			}
			if len(got) != 1 || got[0] != "個情法" {
				t.Errorf("RegisteredTerms() = %#v", got)
			}
		}()
	}
	waitGroup.Wait()
}
