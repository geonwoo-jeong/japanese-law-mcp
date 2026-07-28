package core

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryeval"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileはcorpusV4Developmentのcore意味を製品辞書から生成する(t *testing.T) {
	t.Parallel()

	profile := mustProfile(t)
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	corpus, err := legalquerycorpus.Load(
		context.Background(),
		repositoryRoot(t),
		"testdata/legalquery/corpus-v4",
	)
	if err != nil {
		t.Fatalf("corpus-v4 を読み込めません: %v", err)
	}
	profileSet, err := legalquery.NewQueryProfileSet(
		[]legalquery.QueryProfile{profile},
	)
	if err != nil {
		t.Fatalf("core profile set を構築できません: %v", err)
	}
	packState, err := legalquery.NewStaticPackState(nil, nil)
	if err != nil {
		t.Fatalf("core pack state を構築できません: %v", err)
	}

	processed := 0
	planned := 0
	for _, semanticCase := range corpus.Development() {
		expected, isPlan := semanticCase.Expected().(legalquerycorpus.ExpectedPlan)
		if !isPlan {
			continue
		}
		processed++
		currentCase := semanticCase
		t.Run(currentCase.CaseID(), func(t *testing.T) {
			request, err := productRequest(currentCase.Request())
			if err != nil {
				t.Fatalf("product request のエラー = %v", err)
			}
			preprocessed, err := preprocessor.Preprocess(context.Background(), request)
			if err != nil {
				t.Fatalf("Preprocess() のエラー = %v", err)
			}
			input, err := legalquery.NewCandidateGenerationInput(preprocessed)
			if err != nil {
				t.Fatalf("generation input のエラー = %v", err)
			}
			scope, err := legalquery.NewCandidateIDScope(1)
			if err != nil {
				t.Fatalf("ID scope のエラー = %v", err)
			}
			generation, err := profile.Generate(input, scope)
			if err != nil {
				t.Fatalf("Generate() のエラー = %v", err)
			}

			coreMeanings := expectedCoreMeanings(expected.Meanings())
			candidates := generation.Candidates()
			if len(candidates) != len(coreMeanings) {
				t.Fatalf(
					"candidate count = %d, want %d; signals=%#v; laws=%#v; concepts=%#v; cues=%#v; terms=%#v; candidates=%#v",
					len(candidates),
					len(coreMeanings),
					generation.Signals(),
					preprocessed.LawNameMentions(),
					preprocessed.LegalConceptMentions(),
					preprocessed.CueMentions(),
					preprocessed.QueryTermMentions(),
					candidates,
				)
			}
			assertCoreMeanings(t, coreMeanings, candidates)
			assertDevelopmentSignals(t, expected, generation.Signals())
			if len(coreMeanings) != len(expected.Meanings()) {
				return
			}
			profileSetResult, err := profileSet.Collect(preprocessed)
			if err != nil {
				t.Fatalf("profile set の集約エラー = %v", err)
			}
			plan, err := legalquery.SelectLegalQueryPlan(
				legalquery.SelectorInput{
					ProfileSetResult: profileSetResult,
					PackState:        packState,
					LimitPerAttempt:  request.LimitPerAttempt(),
				},
			)
			if err != nil {
				t.Fatalf("selector のエラー = %v", err)
			}
			evaluation, err := legalqueryeval.EvaluateSemanticPlanCase(
				currentCase,
				plan,
			)
			if err != nil {
				t.Fatalf("plan 評価のエラー = %v", err)
			}
			if !evaluation.PlanOutcomeMatched() {
				t.Fatalf(
					"decision/reason/selection が期待値と一致しません: decision=%q reasons=%#v selected=%#v",
					plan.Decision(),
					plan.ReasonCodes(),
					plan.Selected(),
				)
			}
			planned++
		})
	}
	if processed != 29 {
		t.Fatalf("development plan case count = %d, want 29", processed)
	}
	if planned != 27 {
		t.Fatalf("core plan evaluation count = %d, want 27", planned)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("repository root を解決できません: %v", err)
	}
	return root
}

func productRequest(raw legalquerycorpus.Request) (legalquery.Request, error) {
	values := legalquery.RequestValues{Query: raw.Query()}
	if limit, exists := raw.LimitPerAttempt(); exists {
		values.LimitPerAttempt = &limit
	}
	if rawRef, exists := raw.Ref(); exists {
		keyValues := model.SourceResourceKeyValues{
			SourceID:     rawRef.Key().SourceID(),
			ResourceType: rawRef.Key().ResourceType(),
			ResourceID:   rawRef.Key().ResourceID(),
		}
		if versionID, versioned := rawRef.Key().VersionID(); versioned {
			keyValues.VersionID = versionID
		}
		key, err := model.NewSourceResourceKey(keyValues)
		if err != nil {
			return legalquery.Request{}, err
		}
		ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
			ProviderID: rawRef.ProviderID(),
			Key:        key,
		})
		if err != nil {
			return legalquery.Request{}, err
		}
		values.Ref = &ref
	}
	return legalquery.NewRequest(values)
}

func expectedCoreMeanings(
	values []legalquerycorpus.ExpectedMeaning,
) []legalquerycorpus.ExpectedMeaning {
	result := make([]legalquerycorpus.ExpectedMeaning, 0, len(values))
	for _, meaning := range values {
		if len(meaning.RequiredPacks()) > 0 {
			continue
		}
		core := true
		for _, step := range meaning.Steps() {
			if step.Resource() != legalquery.ResourceLaw &&
				step.Resource() != legalquery.ResourceLawProvision {
				core = false
				break
			}
		}
		if core {
			result = append(result, meaning)
		}
	}
	return result
}

func assertCoreMeanings(
	t *testing.T,
	expected []legalquerycorpus.ExpectedMeaning,
	actual []legalquery.LegalQueryCandidate,
) {
	t.Helper()
	used := make([]bool, len(actual))
	for _, meaning := range expected {
		found := -1
		for index, candidate := range actual {
			if used[index] {
				continue
			}
			comparison, err := legalqueryeval.CompareMeaning(meaning, candidate)
			if err != nil {
				t.Fatalf("meaning 比較のエラー = %v", err)
			}
			if comparison.AllMatched() {
				found = index
				break
			}
		}
		if found < 0 {
			t.Fatalf(
				"meaning %q に一致する candidate がありません: %#v",
				meaning.MeaningID(),
				actual,
			)
		}
		used[found] = true
	}
}

func assertDevelopmentSignals(
	t *testing.T,
	expected legalquerycorpus.ExpectedPlan,
	signals []legalquery.CandidateGenerationSignal,
) {
	t.Helper()
	for _, reason := range expected.ReasonCodes() {
		switch reason {
		case legalquery.ReasonCodeNonJapaneseQuery:
			if !containsSignal(signals, legalquery.CandidateSignalNonJapaneseQuery) {
				t.Fatalf("non-Japanese signal がありません: %#v", signals)
			}
		case legalquery.ReasonCodeUnsupportedTaskOrResource:
			if len(signals) == 0 {
				t.Fatal("unsupported signal がありません")
			}
		case legalquery.ReasonCodeRequiredPackDisabled:
			if !containsSignal(signals, legalquery.CandidateSignalReservedPackRequest) {
				t.Fatalf("reserved pack signal がありません: %#v", signals)
			}
		}
	}
}

func containsSignal(
	values []legalquery.CandidateGenerationSignal,
	expected legalquery.CandidateGenerationSignal,
) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}
