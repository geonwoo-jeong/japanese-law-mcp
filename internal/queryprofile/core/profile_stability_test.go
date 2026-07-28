package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
)

func TestProfileは照会間で状態を持たず同じ候補を再生成する(t *testing.T) {
	t.Parallel()

	profile := mustProfile(t)
	firstInput := prepareGenerationInput(
		t,
		profile,
		"法令本文から「営業秘密」を含む条文を検索してください。",
	)
	secondInput := prepareGenerationInput(
		t,
		profile,
		"商法第512条第1項を読んでください。",
	)
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	first, err := profile.Generate(firstInput, scope)
	if err != nil {
		t.Fatalf("一回目の Generate() のエラー = %v", err)
	}
	if _, err := profile.Generate(secondInput, scope); err != nil {
		t.Fatalf("別照会の Generate() のエラー = %v", err)
	}
	repeated, err := profile.Generate(firstInput, scope)
	if err != nil {
		t.Fatalf("再実行した Generate() のエラー = %v", err)
	}
	if !reflect.DeepEqual(first, repeated) {
		t.Fatalf(
			"SOT-ARCH-023: 同じ入力の候補が照会間で変わりました: first=%#v repeated=%#v",
			first,
			repeated,
		)
	}
}

func TestProfileは共有インスタンスで並行生成しても決定的である(t *testing.T) {
	t.Parallel()

	profile := mustProfile(t)
	inputs := []legalquery.CandidateGenerationInput{
		prepareGenerationInput(
			t,
			profile,
			"独禁法の正式な法令を検索してください。",
		),
		prepareGenerationInput(
			t,
			profile,
			"商法第512条第1項と第2項を読んでください。",
		),
	}
	scope, err := legalquery.NewCandidateIDScope(1)
	if err != nil {
		t.Fatalf("ID scope を作成できません: %v", err)
	}
	expected := make([]legalquery.CandidateGeneration, len(inputs))
	for index, input := range inputs {
		generation, generateErr := profile.Generate(input, scope)
		if generateErr != nil {
			t.Fatalf("期待候補の Generate() のエラー = %v", generateErr)
		}
		expected[index] = generation
	}

	const workerCount = 32
	failures := make(chan error, workerCount)
	var wait sync.WaitGroup
	for worker := 0; worker < workerCount; worker++ {
		index := worker % len(inputs)
		wait.Add(1)
		go func() {
			defer wait.Done()
			generation, generateErr := profile.Generate(inputs[index], scope)
			if generateErr != nil {
				failures <- generateErr
				return
			}
			if !reflect.DeepEqual(expected[index], generation) {
				failures <- fmt.Errorf(
					"worker %d の生成結果が決定的ではありません",
					index,
				)
			}
		}()
	}
	wait.Wait()
	close(failures)
	for failure := range failures {
		t.Error(failure)
	}
}

func prepareGenerationInput(
	t *testing.T,
	profile *Profile,
	query string,
) legalquery.CandidateGenerationInput {
	t.Helper()
	preprocessor, err := querypreprocess.NewEmbedded(profile.CueVocabulary())
	if err != nil {
		t.Fatalf("preprocessor を構築できません: %v", err)
	}
	request, err := legalquery.NewRequest(
		legalquery.RequestValues{Query: query},
	)
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	result, err := preprocessor.Preprocess(context.Background(), request)
	if err != nil {
		t.Fatalf("Preprocess() のエラー = %v", err)
	}
	input, err := legalquery.NewCandidateGenerationInput(result)
	if err != nil {
		t.Fatalf("generation input を構築できません: %v", err)
	}
	return input
}
