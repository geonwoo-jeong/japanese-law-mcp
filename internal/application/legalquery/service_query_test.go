package legalquery

import (
	"context"
	"errors"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
)

func TestServiceQueryは無効な入力で後段を呼ばない(t *testing.T) {
	t.Parallel()

	preprocessor := &servicePreprocessorFake{}
	executor, recorder, _ := mustServiceExecutor(t)
	service := mustService(
		t,
		preprocessor,
		mustServiceSingleProfileSet(t),
		mustSelectorTestPackState(t, nil, nil),
		executor,
		time.Second,
	)

	//nolint:staticcheck // SOT-ARCH-022: application 境界の nil context 拒否を確認する。
	if _, err := service.Query(nil, mustServiceRequest(t, nil)); err == nil {
		t.Fatal("nil context を受理しました")
	}
	var typedNil *executorTestContext
	if _, err := service.Query(typedNil, mustServiceRequest(t, nil)); err == nil {
		t.Fatal("typed nil context を受理しました")
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.Query(
		cancelled,
		mustServiceRequest(t, nil),
	); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled context error = %v", err)
	}
	if _, err := service.Query(context.Background(), Request{}); err == nil {
		t.Fatal("zero-value Request を受理しました")
	}
	if preprocessor.callCount() != 0 || recorder.callCount() != 0 {
		t.Fatalf(
			"無効入力後の call count = preprocess:%d, capability:%d",
			preprocessor.callCount(),
			recorder.callCount(),
		)
	}
}

func TestServiceQueryは三種類の非実行を正常結果として返す(
	t *testing.T,
) {
	t.Parallel()

	tests := []struct {
		name      string
		profile   selectorTestProfile
		packState PackState
		status    LegalQueryResultStatus
	}{
		{
			name: "needs clarification",
			profile: selectorTestProfile{
				selectionMode: QuerySelectionModeAutomatic,
			},
			packState: mustSelectorTestPackState(t, nil, nil),
			status:    LegalQueryResultStatusNeedsClarification,
		},
		{
			name: "unsupported",
			profile: selectorTestProfile{
				signals: []CandidateGenerationSignal{
					CandidateSignalNonJapaneseQuery,
				},
				selectionMode: QuerySelectionModeAutomatic,
			},
			packState: mustSelectorTestPackState(t, nil, nil),
			status:    LegalQueryResultStatusUnsupported,
		},
		{
			name: "capability unavailable",
			profile: selectorTestProfile{
				candidates: []LegalQueryCandidate{
					mustSelectorTestCandidate(
						t,
						"candidate-pack",
						200,
						[]string{"judicial-cases"},
						1,
					),
				},
				selectionMode: QuerySelectionModeAutomatic,
			},
			packState: mustSelectorTestPackState(
				t,
				[]string{"judicial-cases"},
				nil,
			),
			status: LegalQueryResultStatusCapabilityUnavailable,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			preprocessor := &servicePreprocessorFake{}
			executor, recorder, _ := mustServiceExecutor(t)
			service := mustService(
				t,
				preprocessor,
				mustServiceProfileSet(t, test.profile),
				test.packState,
				executor,
				time.Second,
			)
			result, err := service.Query(
				context.Background(),
				mustServiceRequest(t, nil),
			)
			if err != nil {
				t.Fatalf("SOT-IF-051: 非実行 result error = %v", err)
			}
			if got := serviceResultStatus(t, result); got != test.status {
				t.Fatalf("status = %q, want %q", got, test.status)
			}
			if recorder.callCount() != 0 {
				t.Fatalf(
					"SOT-IF-051: 非実行 capability call count = %d",
					recorder.callCount(),
				)
			}
		})
	}
}

func TestServiceQueryは実行結果の状態を保持する(t *testing.T) {
	t.Parallel()

	t.Run("completed", func(t *testing.T) {
		t.Parallel()
		executor, recorder, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		result, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		if err != nil {
			t.Fatalf("completed query error = %v", err)
		}
		if got := serviceResultStatus(t, result); got !=
			LegalQueryResultStatusCompleted {
			t.Fatalf("status = %q", got)
		}
		if recorder.callCount() != 1 {
			t.Fatalf("capability call count = %d", recorder.callCount())
		}
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()
		executor, _, core := mustServiceExecutor(t)
		page, err := lawsearch.NewPage(lawsearch.PageValues{
			Page: mustExecutorTestSourcePage(t, 0),
		})
		if err != nil {
			t.Fatalf("空の law search page を作成できません: %v", err)
		}
		core.payloads.lawSearch = page
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		result, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		if err != nil {
			t.Fatalf("empty query error = %v", err)
		}
		if got := serviceResultStatus(t, result); got !=
			LegalQueryResultStatusEmpty {
			t.Fatalf("status = %q", got)
		}
	})

	t.Run("partial", func(t *testing.T) {
		t.Parallel()
		executor, recorder, _ := mustServiceExecutor(t)
		recorder.errs[InputKindLawContentSearch] =
			mustExecutorExecutedError(t, errors.New("試験用情報源エラー"))
		candidate := mustExecutorCandidate(
			t,
			"candidate-partial",
			200,
			mustExecutorStep(t, "法令検索", "step-success"),
			mustExecutorStep(t, "法令本文検索", "step-failed"),
		)
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceProfileSet(t, selectorTestProfile{
				candidates:     []LegalQueryCandidate{candidate},
				selectionMode:  QuerySelectionModeAutomatic,
				profileVersion: selectorTestProfileVersion,
				rankingVersion: selectorTestRankingVersion,
			}),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		result, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		if err != nil {
			t.Fatalf("partial query error = %v", err)
		}
		if got := serviceResultStatus(t, result); got !=
			LegalQueryResultStatusPartial {
			t.Fatalf("status = %q", got)
		}
	})
}

func TestServiceQueryは段階順と一つの期限と取得上限を渡す(
	t *testing.T,
) {
	t.Parallel()

	recordStage, stages := serviceStageRecorder()
	preprocessor := &servicePreprocessorFake{
		hook: func(
			ctx context.Context,
			request Request,
		) (PreprocessResult, error) {
			recordStage("preprocess")
			if ctx.Value(serviceContextKey{}) != "request-value" {
				t.Error("前処理 context に親の値がありません")
			}
			return servicePreprocessResultForRequest(request)
		},
	}
	candidate := mustServiceSingleCandidate(t)
	delegate := selectorTestProfile{
		metadata: mustSelectorTestMetadata(
			t,
			"core",
			selectorTestProfileVersion,
			selectorTestRankingVersion,
		),
		candidates:     []LegalQueryCandidate{candidate},
		selectionMode:  QuerySelectionModeAutomatic,
		profileVersion: selectorTestProfileVersion,
		rankingVersion: selectorTestRankingVersion,
	}
	profiles, err := NewQueryProfileSet([]QueryProfile{
		serviceRecordingProfile{
			delegate: delegate,
			onGenerate: func() {
				recordStage("profile")
			},
		},
	})
	if err != nil {
		t.Fatalf("記録用 profile set を作成できません: %v", err)
	}
	executor, recorder, _ := mustServiceExecutor(t)
	var capabilityDeadline time.Time
	recorder.hook = func(
		ctx context.Context,
		_ LogicalInputKind,
		budget LegalQueryStepBudget,
	) error {
		recordStage("executor")
		if ctx.Value(serviceContextKey{}) != "request-value" {
			t.Error("能力 context に親の値がありません")
		}
		var ok bool
		capabilityDeadline, ok = ctx.Deadline()
		if !ok {
			t.Error("能力 context に deadline がありません")
		}
		limit, hasLimit := budget.EffectiveLimit()
		if !hasLimit || limit != 7 {
			t.Errorf("effectiveLimit = %d, %t", limit, hasLimit)
		}
		return nil
	}
	service := mustService(
		t,
		preprocessor,
		profiles,
		mustSelectorTestPackState(t, nil, nil),
		executor,
		time.Second,
	)
	limit := 7
	root := context.WithValue(
		context.Background(),
		serviceContextKey{},
		"request-value",
	)
	result, err := service.Query(root, mustServiceRequest(t, &limit))
	if err != nil {
		t.Fatalf("統合照会 error = %v", err)
	}
	if serviceResultStatus(t, result) != LegalQueryResultStatusCompleted {
		t.Fatalf("status = %q", result.Status())
	}
	calls := preprocessor.callsSnapshot()
	if len(calls) != 1 {
		t.Fatalf("preprocess call count = %d", len(calls))
	}
	preprocessDeadline, ok := calls[0].ctx.Deadline()
	if !ok || !preprocessDeadline.Equal(capabilityDeadline) {
		t.Fatalf(
			"前処理と能力の deadline = %v, %v",
			preprocessDeadline,
			capabilityDeadline,
		)
	}
	if got := stages(); !slices.Equal(
		got,
		[]string{"preprocess", "profile", "executor"},
	) {
		t.Fatalf("pipeline stages = %#v", got)
	}
}

func TestServiceQueryは期限とエラーの同一性を保持する(t *testing.T) {
	t.Parallel()

	t.Run("service timeout", func(t *testing.T) {
		t.Parallel()
		preprocessor := &servicePreprocessorFake{
			hook: func(
				ctx context.Context,
				request Request,
			) (PreprocessResult, error) {
				<-ctx.Done()
				return servicePreprocessResultForRequest(request)
			},
		}
		executor, recorder, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			preprocessor,
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			20*time.Millisecond,
		)
		if _, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout error = %v", err)
		}
		if recorder.callCount() != 0 {
			t.Fatalf("timeout 後の capability call count = %d", recorder.callCount())
		}
	})

	t.Run("service timeoutが前処理エラーより優先される", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("期限後の前処理エラー")
		preprocessor := &servicePreprocessorFake{
			hook: func(
				ctx context.Context,
				_ Request,
			) (PreprocessResult, error) {
				<-ctx.Done()
				return PreprocessResult{}, cause
			},
		}
		executor, recorder, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			preprocessor,
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			20*time.Millisecond,
		)
		_, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		if !errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, cause) {
			t.Fatalf("timeout 後の前処理 error = %v", err)
		}
		if recorder.callCount() != 0 {
			t.Fatalf("timeout 後の capability call count = %d", recorder.callCount())
		}
	})

	t.Run("service timeoutがexecutorエラーより優先される", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("期限後の executor エラー")
		executor, recorder, _ := mustServiceExecutor(t)
		recorder.hook = func(
			ctx context.Context,
			_ LogicalInputKind,
			_ LegalQueryStepBudget,
		) error {
			<-ctx.Done()
			return cause
		}
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			20*time.Millisecond,
		)
		_, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("timeout 後の executor error = %v", err)
		}
	})

	t.Run("caller deadline", func(t *testing.T) {
		t.Parallel()
		var observed time.Time
		preprocessor := &servicePreprocessorFake{
			hook: func(
				ctx context.Context,
				request Request,
			) (PreprocessResult, error) {
				var ok bool
				observed, ok = ctx.Deadline()
				if !ok {
					t.Error("前処理 context に deadline がありません")
				}
				return servicePreprocessResultForRequest(request)
			},
		}
		executor, _, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			preprocessor,
			mustServiceProfileSet(t, selectorTestProfile{
				selectionMode: QuerySelectionModeAutomatic,
			}),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			5*time.Second,
		)
		root, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		expected, _ := root.Deadline()
		if _, err := service.Query(root, mustServiceRequest(t, nil)); err != nil {
			t.Fatalf("caller deadline query error = %v", err)
		}
		if !observed.Equal(expected) {
			t.Fatalf("deadline = %v, want %v", observed, expected)
		}
	})

	t.Run("preprocess error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("試験用前処理エラー")
		preprocessor := &servicePreprocessorFake{
			hook: func(
				context.Context,
				Request,
			) (PreprocessResult, error) {
				return PreprocessResult{}, cause
			},
		}
		executor, _, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			preprocessor,
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		if _, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		); !errors.Is(err, cause) {
			t.Fatalf("preprocess cause を保持しませんでした: %v", err)
		}
	})

	t.Run("profile error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("試験用 profile エラー")
		profiles := mustServiceProfileSet(t, selectorTestProfile{
			generationError: cause,
			selectionMode:   QuerySelectionModeAutomatic,
		})
		executor, _, _ := mustServiceExecutor(t)
		service := mustService(
			t,
			&servicePreprocessorFake{},
			profiles,
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		if _, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		); !errors.Is(err, cause) {
			t.Fatalf("profile cause を保持しませんでした: %v", err)
		}
	})

	t.Run("executor fatal error", func(t *testing.T) {
		t.Parallel()
		cause := errors.New("試験用実行前エラー")
		executor, recorder, _ := mustServiceExecutor(t)
		recorder.errs[InputKindLawSearch] = cause
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		if _, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		); !errors.Is(err, cause) {
			t.Fatalf("executor cause を保持しませんでした: %v", err)
		}
	})

	t.Run("all failed error", func(t *testing.T) {
		t.Parallel()
		executor, recorder, _ := mustServiceExecutor(t)
		recorder.errs[InputKindLawSearch] = mustExecutorExecutedError(
			t,
			errors.New("試験用実行済みエラー"),
		)
		service := mustService(
			t,
			&servicePreprocessorFake{},
			mustServiceSingleProfileSet(t),
			mustSelectorTestPackState(t, nil, nil),
			executor,
			time.Second,
		)
		_, err := service.Query(
			context.Background(),
			mustServiceRequest(t, nil),
		)
		var allFailed LegalQueryAllFailedError
		if !errors.As(err, &allFailed) {
			t.Fatalf("LegalQueryAllFailedError を保持しませんでした: %v", err)
		}
	})
}

func TestServiceQueryは前処理結果を要求と再照合する(t *testing.T) {
	t.Parallel()

	preprocessor := &servicePreprocessorFake{
		hook: func(
			context.Context,
			Request,
		) (PreprocessResult, error) {
			request, err := NewRequest(RequestValues{
				Query: "別の法情報を検索してください",
			})
			if err != nil {
				return PreprocessResult{}, err
			}
			return servicePreprocessResultForRequest(request)
		},
	}
	executor, recorder, _ := mustServiceExecutor(t)
	service := mustService(
		t,
		preprocessor,
		mustServiceSingleProfileSet(t),
		mustSelectorTestPackState(t, nil, nil),
		executor,
		time.Second,
	)
	if _, err := service.Query(
		context.Background(),
		mustServiceRequest(t, nil),
	); err == nil {
		t.Fatal("request と異なる前処理結果を受理しました")
	}
	if recorder.callCount() != 0 {
		t.Fatalf("不一致後の capability call count = %d", recorder.callCount())
	}
}

func TestServiceQueryは同じServiceから並行実行できる(t *testing.T) {
	t.Parallel()

	preprocessor := &servicePreprocessorFake{}
	executor, recorder, _ := mustServiceExecutor(t)
	service := mustService(
		t,
		preprocessor,
		mustServiceSingleProfileSet(t),
		mustSelectorTestPackState(t, nil, nil),
		executor,
		time.Second,
	)
	const workers = 16
	var waitGroup sync.WaitGroup
	errorsByWorker := make(chan error, workers)
	waitGroup.Add(workers)
	for index := 0; index < workers; index++ {
		go func() {
			defer waitGroup.Done()
			result, err := service.Query(
				context.Background(),
				mustServiceRequest(t, nil),
			)
			if err != nil {
				errorsByWorker <- err
				return
			}
			if result.Status() != LegalQueryResultStatusCompleted {
				errorsByWorker <- errors.New("完了状態ではありません")
			}
		}()
	}
	waitGroup.Wait()
	close(errorsByWorker)
	for err := range errorsByWorker {
		t.Errorf("並行 Query error = %v", err)
	}
	if preprocessor.callCount() != workers ||
		recorder.callCount() != workers {
		t.Fatalf(
			"並行 call count = preprocess:%d, capability:%d",
			preprocessor.callCount(),
			recorder.callCount(),
		)
	}
}

type serviceContextKey struct{}

type serviceRecordingProfile struct {
	delegate   QueryProfile
	onGenerate func()
}

func (p serviceRecordingProfile) Metadata() QueryProfileMetadata {
	return p.delegate.Metadata()
}

func (p serviceRecordingProfile) CueVocabulary() []CueVocabularyEntry {
	return p.delegate.CueVocabulary()
}

func (p serviceRecordingProfile) Generate(
	input CandidateGenerationInput,
	scope CandidateIDScope,
) (CandidateGeneration, error) {
	p.onGenerate()
	return p.delegate.Generate(input, scope)
}
