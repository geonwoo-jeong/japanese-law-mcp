package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

var _ legalquery.JudicialCasesCapabilityFacade = application.JudicialCasesLegalQueryFacade{}

func TestJudicialCasesLegalQueryFacadeCallsTwoCapabilitiesLosslessly(
	t *testing.T,
) {
	materializer := &judicialFacadeMaterializerStub{}
	ports, facade := newReadyJudicialFacadeWithMaterializer(t, materializer)
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 7)
	ctx := context.WithValue(
		context.Background(),
		judicialFacadeContextKey{},
		"同一 context",
	)

	searchResult, err := facade.SearchJudicialDecisions(
		ctx,
		inputs.search,
		budgets.search,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: 裁判例検索の実行エラー = %v", err)
	}
	readResult, err := facade.ReadJudicialDecision(
		ctx,
		inputs.read,
		budgets.read,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: 裁判例読取りの実行エラー = %v", err)
	}

	assertJudicialFacadeCalls(t, ports, ctx)
	if materializer.searchCalls != 1 || materializer.readCalls != 1 {
		t.Fatalf(
			"SOT-ARCH-026: materializer 呼出し = (%d, %d)",
			materializer.searchCalls,
			materializer.readCalls,
		)
	}
	searchRequest := ports.search.requests[0]
	if searchRequest.Query() != "永住許可" || searchRequest.Limit() != 7 {
		t.Fatalf(
			"SOT-ARCH-026: 裁判例検索 request = (%q, %d)",
			searchRequest.Query(),
			searchRequest.Limit(),
		)
	}
	if _, exists := searchRequest.ContinuationToken(); exists {
		t.Fatal("SOT-ARCH-026: 裁判例検索 request に continuationToken を設定しました")
	}
	if ports.read.requests[0].Ref() != ref {
		t.Fatal("SOT-ARCH-026/SOT-IF-042: 裁判例 read の ref を変更しました")
	}
	assertJudicialFacadeResultIdentity(
		t,
		searchResult,
		ports.search.result,
		readResult,
		ports.read.result,
	)
}

func TestJudicialCasesLegalQueryFacadeUsesEffectiveRollbackForSearch(
	t *testing.T,
) {
	primary := newJudicialFacadePorts(
		t,
		"primary-provider",
		"primary-source",
		"primary/detail2",
	)
	rollback := newJudicialFacadePorts(
		t,
		"rollback-provider",
		"rollback-source",
		"rollback/detail2",
	)
	values := judicialFacadeRouteValues("primary-provider")
	for index := range values {
		if values[index].CapabilityID == judicialdecisionsearch.CapabilityID {
			values[index].RollbackProviderID = "rollback-provider"
		}
	}
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"primary-provider",
				"primary-source",
				primary,
			),
			newJudicialFacadeBindings(
				t,
				"rollback-provider",
				"rollback-source",
				rollback,
			),
		},
		values,
	)
	facade := mustJudicialCasesLegalQueryFacade(
		t,
		routes,
		legalquery.NewJudicialCasesMaterializer(),
	)
	input := mustJudicialFacadeSearchInput(t, "永住許可")
	budget := judicialFacadeBudgetForInput(t, "step-search", input, 5)

	if _, err := facade.SearchJudicialDecisions(
		context.Background(),
		input,
		budget,
	); err != nil {
		t.Fatalf("SOT-ARCH-026: rollback 検索の実行エラー = %v", err)
	}
	if primary.search.calls != 0 || rollback.search.calls != 1 {
		t.Fatalf(
			"SOT-ARCH-026: primary 検索 = %d、rollback 検索 = %d",
			primary.search.calls,
			rollback.search.calls,
		)
	}
}

func TestJudicialCasesLegalQueryFacadeReadsExplicitRefWithoutFallback(
	t *testing.T,
) {
	primary := newJudicialFacadePorts(
		t,
		"primary-provider",
		"primary-source",
		"primary/detail2",
	)
	explicit := newJudicialFacadePorts(
		t,
		"explicit-provider",
		"explicit-source",
		"explicit/detail2",
	)
	cause := errors.New("明示 provider の裁判例読取りに失敗しました")
	explicit.read.err = cause
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"primary-provider",
				"primary-source",
				primary,
			),
			newJudicialFacadeBindings(
				t,
				"explicit-provider",
				"explicit-source",
				explicit,
			),
		},
		judicialFacadeRouteValues("primary-provider"),
	)
	facade := mustJudicialCasesLegalQueryFacade(
		t,
		routes,
		legalquery.NewJudicialCasesMaterializer(),
	)
	ref := mustJudicialFacadeRef(
		t,
		"explicit-provider",
		"explicit-source",
		"explicit/detail2",
	)
	input := mustJudicialFacadeReadInput(t, ref)
	budget := judicialFacadeBudgetForInput(t, "step-read", input, 5)

	_, err := facade.ReadJudicialDecision(context.Background(), input, budget)
	assertJudicialFacadeExecutedError(t, err, cause)
	if explicit.read.calls != 1 || primary.read.calls != 0 {
		t.Fatalf(
			"SOT-ARCH-026/SOT-IF-042: explicit read = %d、primary read = %d",
			explicit.read.calls,
			primary.read.calls,
		)
	}
	if explicit.read.requests[0].Ref() != ref {
		t.Fatal("SOT-IF-042: 明示 ref を変更しました")
	}
}

func TestJudicialCasesLegalQueryFacadeRejectsRefMismatchBeforePort(
	t *testing.T,
) {
	primary := newJudicialFacadePorts(
		t,
		"primary-provider",
		"primary-source",
		"primary/detail2",
	)
	explicit := newJudicialFacadePorts(
		t,
		"explicit-provider",
		"explicit-source",
		"explicit/detail2",
	)
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"primary-provider",
				"primary-source",
				primary,
			),
			newJudicialFacadeBindings(
				t,
				"explicit-provider",
				"explicit-source",
				explicit,
			),
		},
		judicialFacadeRouteValues("primary-provider"),
	)
	facade := mustJudicialCasesLegalQueryFacade(
		t,
		routes,
		legalquery.NewJudicialCasesMaterializer(),
	)
	tests := []struct {
		name       string
		providerID string
		sourceID   string
	}{
		{
			name:       "未知 provider",
			providerID: "unknown-provider",
			sourceID:   "explicit-source",
		},
		{
			name:       "provider metadata と異なる source",
			providerID: "explicit-provider",
			sourceID:   "other-source",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ref := mustJudicialFacadeRef(
				t,
				test.providerID,
				test.sourceID,
				"explicit/detail2",
			)
			input := mustJudicialFacadeReadInput(t, ref)
			_, err := facade.ReadJudicialDecision(
				context.Background(),
				input,
				judicialFacadeBudgetForInput(t, "step-read", input, 5),
			)
			assertJudicialFacadeFatalError(t, err)
			var argument legalquery.ArgumentError
			if !errors.As(err, &argument) {
				t.Fatalf("SOT-IF-042: ref 不一致を入力エラーとして分類しませんでした: %v", err)
			}
		})
	}
	if primary.totalCalls() != 0 || explicit.totalCalls() != 0 {
		t.Fatalf(
			"SOT-IF-042: ref 不一致で port を呼びました: primary=%d explicit=%d",
			primary.totalCalls(),
			explicit.totalCalls(),
		)
	}
}

func TestJudicialCasesLegalQueryFacadeWrapsOnlyPortErrors(
	t *testing.T,
) {
	ports, facade := newReadyJudicialFacade(t)
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 5)
	searchCause := errors.New("裁判例検索 port の失敗")
	readCause := errors.New("裁判例読取り port の失敗")
	ports.search.err = searchCause
	ports.read.err = readCause

	_, err := facade.SearchJudicialDecisions(
		context.Background(),
		inputs.search,
		budgets.search,
	)
	assertJudicialFacadeExecutedError(t, err, searchCause)
	_, err = facade.ReadJudicialDecision(
		context.Background(),
		inputs.read,
		budgets.read,
	)
	assertJudicialFacadeExecutedError(t, err, readCause)
}

func TestJudicialCasesLegalQueryFacadeRejectsPrePortContractFailures(
	t *testing.T,
) {
	ports := newJudicialFacadePorts(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"judicial-provider",
				"judicial-source",
				ports,
			),
		},
		judicialFacadeRouteValues("judicial-provider"),
	)
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 5)
	materializer := &judicialFacadeMaterializerStub{}
	facade := mustJudicialCasesLegalQueryFacade(t, routes, materializer)

	_, err := facade.SearchJudicialDecisions(
		context.Background(),
		legalquery.JudicialDecisionSearchIntentV1{},
		budgets.search,
	)
	assertJudicialFacadeFatalError(t, err)
	_, err = facade.ReadJudicialDecision(
		context.Background(),
		legalquery.JudicialDecisionReadIntentV1{},
		budgets.read,
	)
	assertJudicialFacadeFatalError(t, err)
	_, err = facade.SearchJudicialDecisions(
		context.Background(),
		inputs.search,
		budgets.read,
	)
	assertJudicialFacadeFatalError(t, err)
	_, err = facade.ReadJudicialDecision(
		context.Background(),
		inputs.read,
		budgets.search,
	)
	assertJudicialFacadeFatalError(t, err)

	if materializer.searchCalls != 0 || materializer.readCalls != 0 {
		t.Fatalf(
			"SOT-ARCH-022/026: 事前検証失敗後の materializer 呼出し = (%d, %d)",
			materializer.searchCalls,
			materializer.readCalls,
		)
	}
	if ports.totalCalls() != 0 {
		t.Fatalf("SOT-ARCH-026: 事前検証失敗後の port 呼出し = %d", ports.totalCalls())
	}
}

func TestJudicialCasesLegalQueryFacadeKeepsMaterializerFailuresFatal(
	t *testing.T,
) {
	ports := newJudicialFacadePorts(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	routes := newJudicialFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newJudicialFacadeBindings(
				t,
				"judicial-provider",
				"judicial-source",
				ports,
			),
		},
		judicialFacadeRouteValues("judicial-provider"),
	)
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 5)
	searchCause := errors.New("裁判例検索 request の組立てに失敗しました")
	readCause := errors.New("裁判例読取り request の組立てに失敗しました")
	materializer := &judicialFacadeMaterializerStub{
		searchErr: searchCause,
		readErr:   readCause,
	}
	facade := mustJudicialCasesLegalQueryFacade(t, routes, materializer)

	_, err := facade.SearchJudicialDecisions(
		context.Background(),
		inputs.search,
		budgets.search,
	)
	assertJudicialFacadeFatalError(t, err)
	if !errors.Is(err, searchCause) {
		t.Fatal("SOT-ARCH-026: 裁判例検索 materializer の原因を保持しませんでした")
	}
	_, err = facade.ReadJudicialDecision(
		context.Background(),
		inputs.read,
		budgets.read,
	)
	assertJudicialFacadeFatalError(t, err)
	if !errors.Is(err, readCause) {
		t.Fatal("SOT-ARCH-026: 裁判例読取り materializer の原因を保持しませんでした")
	}
	if ports.totalCalls() != 0 {
		t.Fatalf("SOT-ARCH-026: materializer 失敗後の port 呼出し = %d", ports.totalCalls())
	}
}

func TestJudicialCasesLegalQueryFacadeRejectsInvalidMaterializedRequests(
	t *testing.T,
) {
	materializer := &judicialFacadeMaterializerStub{}
	ports, facade := newReadyJudicialFacadeWithMaterializer(t, materializer)
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 5)
	invalidSearch := judicialdecisionsearch.Request{}
	invalidRead := judicialdecisionread.Request{}
	materializer.searchRequest = &invalidSearch
	materializer.readRequest = &invalidRead

	_, err := facade.SearchJudicialDecisions(
		context.Background(),
		inputs.search,
		budgets.search,
	)
	assertJudicialFacadeFatalError(t, err)
	if materializer.searchCalls != 1 ||
		materializer.readCalls != 0 ||
		ports.totalCalls() != 0 {
		t.Fatalf(
			"SOT-ARCH-026: 無効な検索 request 後の呼出し = materializer(%d, %d), port(%d)",
			materializer.searchCalls,
			materializer.readCalls,
			ports.totalCalls(),
		)
	}
	_, err = facade.ReadJudicialDecision(
		context.Background(),
		inputs.read,
		budgets.read,
	)
	assertJudicialFacadeFatalError(t, err)
	if materializer.searchCalls != 1 ||
		materializer.readCalls != 1 ||
		ports.totalCalls() != 0 {
		t.Fatalf(
			"SOT-ARCH-026: 無効な読取り request 後の呼出し = materializer(%d, %d), port(%d)",
			materializer.searchCalls,
			materializer.readCalls,
			ports.totalCalls(),
		)
	}
}

func TestJudicialCasesLegalQueryFacadeValidatesPortResults(
	t *testing.T,
) {
	t.Run("検索の zero-value result", func(t *testing.T) {
		ports, facade := newReadyJudicialFacade(t)
		ports.search.result = judicialdecisionsearch.Page{}
		input := mustJudicialFacadeSearchInput(t, "永住許可")
		_, err := facade.SearchJudicialDecisions(
			context.Background(),
			input,
			judicialFacadeBudgetForInput(t, "step-search", input, 5),
		)
		assertJudicialFacadeFatalError(t, err)
		if ports.search.calls != 1 {
			t.Fatalf("SOT-IF-041: 検索 port 呼出し = %d", ports.search.calls)
		}
	})

	t.Run("検索上限の超過", func(t *testing.T) {
		ports, facade := newReadyJudicialFacade(t)
		first := newJudicialFacadePayload(
			t,
			"judicial-provider",
			"judicial-source",
			"first/detail2",
		)
		second := newJudicialFacadePayload(
			t,
			"judicial-provider",
			"judicial-source",
			"second/detail2",
		)
		ports.search.result = mustJudicialFacadePage(
			t,
			[]model.SourcedResource[model.JudicialDecisionSummary]{
				first.summary,
				second.summary,
			},
		)
		input := mustJudicialFacadeSearchInput(t, "永住許可")
		_, err := facade.SearchJudicialDecisions(
			context.Background(),
			input,
			judicialFacadeBudgetForInput(t, "step-search", input, 1),
		)
		assertJudicialFacadeFatalError(t, err)
	})

	t.Run("検索結果と binding の不一致", func(t *testing.T) {
		ports, facade := newReadyJudicialFacade(t)
		ports.search.result = newJudicialFacadePayload(
			t,
			"other-provider",
			"other-source",
			"other/detail2",
		).page
		input := mustJudicialFacadeSearchInput(t, "永住許可")
		_, err := facade.SearchJudicialDecisions(
			context.Background(),
			input,
			judicialFacadeBudgetForInput(t, "step-search", input, 5),
		)
		assertJudicialFacadeFatalError(t, err)
	})

	t.Run("読取りの zero-value result", func(t *testing.T) {
		ports, facade := newReadyJudicialFacade(t)
		ports.read.result =
			model.SourcedResource[model.JudicialDecisionDetails]{}
		ref := mustJudicialFacadeRef(
			t,
			"judicial-provider",
			"judicial-source",
			"95570/detail2",
		)
		input := mustJudicialFacadeReadInput(t, ref)
		_, err := facade.ReadJudicialDecision(
			context.Background(),
			input,
			judicialFacadeBudgetForInput(t, "step-read", input, 5),
		)
		assertJudicialFacadeFatalError(t, err)
	})

	t.Run("読取り結果と request ref の不一致", func(t *testing.T) {
		ports, facade := newReadyJudicialFacade(t)
		ports.read.result = newJudicialFacadePayload(
			t,
			"judicial-provider",
			"judicial-source",
			"other/detail2",
		).details
		ref := mustJudicialFacadeRef(
			t,
			"judicial-provider",
			"judicial-source",
			"95570/detail2",
		)
		input := mustJudicialFacadeReadInput(t, ref)
		_, err := facade.ReadJudicialDecision(
			context.Background(),
			input,
			judicialFacadeBudgetForInput(t, "step-read", input, 5),
		)
		assertJudicialFacadeFatalError(t, err)
	})
}

func TestJudicialCasesLegalQueryFacadeFailsClosedAtConstruction(
	t *testing.T,
) {
	ports := newJudicialFacadePorts(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	bindings := []application.ProviderBindings{
		newJudicialFacadeBindings(
			t,
			"judicial-provider",
			"judicial-source",
			ports,
		),
	}
	validRoutes := newJudicialFacadeRoutes(
		t,
		bindings,
		judicialFacadeRouteValues("judicial-provider"),
	)

	t.Run("nil materializer", func(t *testing.T) {
		var materializer legalquery.JudicialCasesRequestMaterializer
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			validRoutes,
			materializer,
		); err == nil {
			t.Fatal("SOT-ARCH-026: nil materializer を受理しました")
		}
	})
	t.Run("typed nil materializer", func(t *testing.T) {
		var materializer *judicialFacadeMaterializerStub
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			validRoutes,
			materializer,
		); err == nil {
			t.Fatal("SOT-ARCH-026: typed nil materializer を受理しました")
		}
	})
	t.Run("zero-value materializer", func(t *testing.T) {
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			validRoutes,
			legalquery.JudicialCasesMaterializer{},
		); err == nil {
			t.Fatal("SOT-ARCH-026: zero-value materializer を受理しました")
		}
	})
	t.Run("materializer の起動時検証失敗", func(t *testing.T) {
		cause := errors.New("裁判例 materializer の起動時検証に失敗しました")
		_, err := application.NewJudicialCasesLegalQueryFacade(
			validRoutes,
			&judicialFacadeMaterializerStub{validateErr: cause},
		)
		if err == nil || !errors.Is(err, cause) {
			t.Fatalf("SOT-ARCH-026: 起動時検証の原因を保持しませんでした: %v", err)
		}
	})
	t.Run("未初期化 routes", func(t *testing.T) {
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			application.ProviderRoutes{},
			legalquery.NewJudicialCasesMaterializer(),
		); err == nil {
			t.Fatal("SOT-ARCH-023/026: 未初期化 routes を受理しました")
		}
	})
	t.Run("core-only routes", func(t *testing.T) {
		coreOnly := newJudicialFacadeRoutes(
			t,
			bindings,
			completeProviderRouteValues("judicial-provider"),
		)
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			coreOnly,
			legalquery.NewJudicialCasesMaterializer(),
		); err == nil {
			t.Fatal("SOT-ARCH-023: judicial route のない構成を受理しました")
		}
	})
	t.Run("検索 route だけ", func(t *testing.T) {
		values := append(
			completeProviderRouteValues("judicial-provider"),
			application.ProviderRouteValues{
				CapabilityID:      judicialdecisionsearch.CapabilityID,
				MajorVersion:      judicialdecisionsearch.MajorVersion,
				Selection:         application.ProviderRouteSelectionPrimary,
				DefaultProviderID: "judicial-provider",
			},
		)
		routes := newJudicialFacadeRoutes(t, bindings, values)
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			routes,
			legalquery.NewJudicialCasesMaterializer(),
		); err == nil {
			t.Fatal("SOT-ARCH-023: read route の欠落を受理しました")
		}
	})
	t.Run("読取り route だけ", func(t *testing.T) {
		values := append(
			completeProviderRouteValues("judicial-provider"),
			application.ProviderRouteValues{
				CapabilityID:      judicialdecisionread.CapabilityID,
				MajorVersion:      judicialdecisionread.MajorVersion,
				Selection:         application.ProviderRouteSelectionPrimary,
				DefaultProviderID: "judicial-provider",
			},
		)
		routes := newJudicialFacadeRoutes(t, bindings, values)
		if _, err := application.NewJudicialCasesLegalQueryFacade(
			routes,
			legalquery.NewJudicialCasesMaterializer(),
		); err == nil {
			t.Fatal("SOT-ARCH-023: search route の欠落を受理しました")
		}
	})
}

func TestJudicialCasesLegalQueryFacadeFailsClosedForZeroValueAndNilContext(
	t *testing.T,
) {
	ref := mustJudicialFacadeRef(
		t,
		"judicial-provider",
		"judicial-source",
		"95570/detail2",
	)
	inputs := newJudicialFacadeInputs(t, ref)
	budgets := newJudicialFacadeBudgets(t, inputs, 5)

	var zero application.JudicialCasesLegalQueryFacade
	if err := zero.Validate(); err == nil {
		t.Fatal("SOT-ENG-025: zero-value facade を有効と判定しました")
	}
	_, err := zero.SearchJudicialDecisions(
		context.Background(),
		inputs.search,
		budgets.search,
	)
	assertJudicialFacadeFatalError(t, err)
	_, err = zero.ReadJudicialDecision(
		context.Background(),
		inputs.read,
		budgets.read,
	)
	assertJudicialFacadeFatalError(t, err)

	ports, facade := newReadyJudicialFacade(t)
	//nolint:staticcheck // SOT-ARCH-023 の nil context 拒否を境界で確認する。
	_, err = facade.SearchJudicialDecisions(nil, inputs.search, budgets.search)
	assertJudicialFacadeFatalError(t, err)
	//nolint:staticcheck // SOT-ARCH-023 の nil context 拒否を境界で確認する。
	_, err = facade.ReadJudicialDecision(nil, inputs.read, budgets.read)
	assertJudicialFacadeFatalError(t, err)
	if ports.totalCalls() != 0 {
		t.Fatalf("SOT-ARCH-023: nil context で port を %d 回呼びました", ports.totalCalls())
	}
}
