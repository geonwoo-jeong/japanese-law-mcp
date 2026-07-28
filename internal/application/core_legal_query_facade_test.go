package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

var _ legalquery.CoreCapabilityFacade = application.CoreLegalQueryFacade{}

func TestCoreLegalQueryFacadeCallsFiveCapabilitiesLosslessly(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	inputs := newCoreFacadeInputs(t, asOf, location)
	budgets := newCoreFacadeBudgets(t, inputs, 7)
	ctx := context.WithValue(context.Background(), coreFacadeContextKey{}, "同一 context")
	ports := newCoreFacadePorts(t, "core-provider", "core-source", asOf, location)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "core-provider", "core-source", ports),
		},
		completeProviderRouteValues("core-provider"),
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())

	searchResult, err := facade.SearchLaws(ctx, inputs.lawSearch, budgets.lawSearch)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: law.search の実行エラー = %v", err)
	}
	contentResult, err := facade.SearchLawContent(
		ctx,
		inputs.lawContentSearch,
		budgets.lawContentSearch,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: law.content.search の実行エラー = %v", err)
	}
	documentResult, err := facade.ReadLawDocument(
		ctx,
		inputs.lawDocumentRead,
		budgets.lawDocumentRead,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: law.document.read の実行エラー = %v", err)
	}
	articleResult, err := facade.ReadLawArticle(
		ctx,
		inputs.lawArticleRead,
		budgets.lawArticleRead,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: law.article.read の実行エラー = %v", err)
	}
	updateResult, err := facade.ListLawUpdates(
		ctx,
		inputs.lawUpdateList,
		budgets.lawUpdateList,
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-022/026: law.update.list の実行エラー = %v", err)
	}

	assertCoreFacadeCalls(t, ports, ctx)
	assertCoreFacadeRequests(t, ports, asOf, location, 7)
	if !reflect.DeepEqual(searchResult, ports.lawSearch.result) ||
		!reflect.DeepEqual(contentResult, ports.lawContentSearch.result) ||
		!reflect.DeepEqual(documentResult, ports.lawDocumentRead.result) ||
		!reflect.DeepEqual(articleResult, ports.lawArticleRead.result) ||
		!reflect.DeepEqual(updateResult, ports.lawUpdateList.result) {
		t.Fatal("SOT-ARCH-022: 型付き capability result を同一値で返しませんでした")
	}
}

func TestCoreLegalQueryFacadeUsesEffectiveRollbackPrimary(t *testing.T) {
	input := mustCoreFacadeLawSearchInput(t, "行政手続法", nil)
	budget := coreFacadeBudgetForInput(t, "step-search", input, 5)
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	primary := newCoreFacadePorts(t, "primary-provider", "primary-source", asOf, location)
	rollback := newCoreFacadePorts(t, "rollback-provider", "rollback-source", asOf, location)
	values := completeProviderRouteValues("primary-provider")
	for index := range values {
		if values[index].CapabilityID == lawsearch.CapabilityID {
			values[index].RollbackProviderID = "rollback-provider"
		}
	}
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "primary-provider", "primary-source", primary),
			newCoreFacadeBindings(t, "rollback-provider", "rollback-source", rollback),
		},
		values,
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())

	if _, err := facade.SearchLaws(context.Background(), input, budget); err != nil {
		t.Fatalf("SOT-ARCH-026: rollback primary の実行エラー = %v", err)
	}
	if primary.lawSearch.calls != 0 || rollback.lawSearch.calls != 1 {
		t.Fatalf(
			"SOT-ARCH-026: primary 呼出し = %d、rollback 呼出し = %d",
			primary.lawSearch.calls,
			rollback.lawSearch.calls,
		)
	}
}

func TestCoreLegalQueryFacadeUsesExplicitNonPrimaryRefWithoutFallback(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	primary := newCoreFacadePorts(t, "primary-provider", "primary-source", asOf, location)
	explicit := newCoreFacadePorts(t, "explicit-provider", "explicit-source", asOf, location)
	cause := errors.New("明示 provider の読取りエラー")
	explicit.lawDocumentRead.err = cause
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "primary-provider", "primary-source", primary),
			newCoreFacadeBindings(t, "explicit-provider", "explicit-source", explicit),
		},
		completeProviderRouteValues("primary-provider"),
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())
	ref := mustCoreFacadeLawRef(
		t,
		"explicit-provider",
		"explicit-source",
		"law-explicit",
		"revision-explicit",
	)
	input := mustCoreFacadeLawReadRefInput(t, ref)
	budget := coreFacadeBudgetForInput(t, "step-read", input, 5)

	_, err := facade.ReadLawDocument(context.Background(), input, budget)
	assertCoreFacadeExecutedError(t, err, cause)
	if explicit.lawDocumentRead.calls != 1 || primary.lawDocumentRead.calls != 0 {
		t.Fatalf(
			"SOT-ARCH-026: explicit 呼出し = %d、primary 呼出し = %d",
			explicit.lawDocumentRead.calls,
			primary.lawDocumentRead.calls,
		)
	}
	if explicit.lawDocumentRead.requests[0].Resource() != ref {
		t.Fatal("SOT-ARCH-026: 入力 ref を変更しました")
	}
}

func TestCoreLegalQueryFacadeUsesExplicitRefForArticleRead(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	primary := newCoreFacadePorts(t, "primary-provider", "primary-source", asOf, location)
	explicit := newCoreFacadePorts(t, "explicit-provider", "explicit-source", asOf, location)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "primary-provider", "primary-source", primary),
			newCoreFacadeBindings(t, "explicit-provider", "explicit-source", explicit),
		},
		completeProviderRouteValues("primary-provider"),
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())
	ref := mustCoreFacadeLawRef(
		t,
		"explicit-provider",
		"explicit-source",
		"law-1",
		"",
	)
	input, err := legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			Ref:      &ref,
			Location: location,
			AsOf:     &asOf,
		},
	)
	if err != nil {
		t.Fatalf("試験用 law article read input を作成できません: %v", err)
	}
	budget := coreFacadeBudgetForInput(t, "step-article", input, 5)

	if _, err := facade.ReadLawArticle(context.Background(), input, budget); err != nil {
		t.Fatalf("SOT-ARCH-026: explicit article read の実行エラー = %v", err)
	}
	if explicit.lawArticleRead.calls != 1 || primary.lawArticleRead.calls != 0 {
		t.Fatalf(
			"SOT-ARCH-026: explicit article 呼出し = %d、primary 呼出し = %d",
			explicit.lawArticleRead.calls,
			primary.lawArticleRead.calls,
		)
	}
	request := explicit.lawArticleRead.requests[0]
	if request.Resource() != ref {
		t.Fatal("SOT-ARCH-026: article read の入力 ref を変更しました")
	}
	assertCoreFacadeDateOption(t, request.AsOf, asOf, true)
}

func TestCoreLegalQueryFacadeRejectsRefProviderAndSourceMismatchBeforePort(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	primary := newCoreFacadePorts(t, "primary-provider", "primary-source", asOf, location)
	explicit := newCoreFacadePorts(t, "explicit-provider", "explicit-source", asOf, location)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "primary-provider", "primary-source", primary),
			newCoreFacadeBindings(t, "explicit-provider", "explicit-source", explicit),
		},
		completeProviderRouteValues("primary-provider"),
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())

	tests := []struct {
		name string
		ref  coreFacadeRefValues
	}{
		{
			name: "未知 provider",
			ref: coreFacadeRefValues{
				providerID: "unknown-provider",
				sourceID:   "explicit-source",
			},
		},
		{
			name: "採用済み provider と異なる source",
			ref: coreFacadeRefValues{
				providerID: "explicit-provider",
				sourceID:   "other-source",
			},
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			ref := mustCoreFacadeLawRef(
				t,
				test.ref.providerID,
				test.ref.sourceID,
				"law-explicit",
				"revision-explicit",
			)
			input := mustCoreFacadeLawReadRefInput(t, ref)
			budget := coreFacadeBudgetForInput(t, "step-read", input, 5)
			_, err := facade.ReadLawDocument(context.Background(), input, budget)
			assertCoreFacadeFatalError(t, err)
		})
	}
	if primary.totalCalls() != 0 || explicit.totalCalls() != 0 {
		t.Fatalf(
			"SOT-ARCH-026: 不一致 ref で port を呼びました: primary=%d explicit=%d",
			primary.totalCalls(),
			explicit.totalCalls(),
		)
	}
}

func TestCoreLegalQueryFacadeWrapsOnlyPortErrorsAsExecutedStepError(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	inputs := newCoreFacadeInputs(t, asOf, location)
	budgets := newCoreFacadeBudgets(t, inputs, 5)
	ports := newCoreFacadePorts(t, "core-provider", "core-source", asOf, location)
	causes := []error{
		errors.New("law.search port error"),
		errors.New("law.content.search port error"),
		errors.New("law.document.read port error"),
		errors.New("law.article.read port error"),
		errors.New("law.update.list port error"),
	}
	ports.lawSearch.err = causes[0]
	ports.lawContentSearch.err = causes[1]
	ports.lawDocumentRead.err = causes[2]
	ports.lawArticleRead.err = causes[3]
	ports.lawUpdateList.err = causes[4]
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "core-provider", "core-source", ports),
		},
		completeProviderRouteValues("core-provider"),
	)
	facade := mustCoreLegalQueryFacade(t, routes, legalquery.NewCoreMaterializer())
	ctx := context.Background()

	_, err := facade.SearchLaws(ctx, inputs.lawSearch, budgets.lawSearch)
	assertCoreFacadeExecutedError(t, err, causes[0])
	_, err = facade.SearchLawContent(ctx, inputs.lawContentSearch, budgets.lawContentSearch)
	assertCoreFacadeExecutedError(t, err, causes[1])
	_, err = facade.ReadLawDocument(ctx, inputs.lawDocumentRead, budgets.lawDocumentRead)
	assertCoreFacadeExecutedError(t, err, causes[2])
	_, err = facade.ReadLawArticle(ctx, inputs.lawArticleRead, budgets.lawArticleRead)
	assertCoreFacadeExecutedError(t, err, causes[3])
	_, err = facade.ListLawUpdates(ctx, inputs.lawUpdateList, budgets.lawUpdateList)
	assertCoreFacadeExecutedError(t, err, causes[4])
}

func TestCoreLegalQueryFacadeTreatsPreAndPostPortFailuresAsFatal(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	input := mustCoreFacadeLawSearchInput(t, "行政手続法", nil)
	budget := coreFacadeBudgetForInput(t, "step-search", input, 1)

	t.Run("materializer error", func(t *testing.T) {
		ports := newCoreFacadePorts(t, "core-provider", "core-source", asOf, location)
		routes := newCoreFacadeRoutes(
			t,
			[]application.ProviderBindings{
				newCoreFacadeBindings(t, "core-provider", "core-source", ports),
			},
			completeProviderRouteValues("core-provider"),
		)
		cause := errors.New("request materialization error")
		materializer := &coreFacadeMaterializerStub{lawSearchErr: cause}
		facade := mustCoreLegalQueryFacade(t, routes, materializer)
		_, err := facade.SearchLaws(context.Background(), input, budget)
		assertCoreFacadeFatalError(t, err)
		if !errors.Is(err, cause) {
			t.Fatal("SOT-ARCH-026: materializer error の原因を保持しませんでした")
		}
		if ports.totalCalls() != 0 {
			t.Fatal("SOT-ARCH-026: materializer error 後に port を呼びました")
		}
	})

	t.Run("invalid logical input", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		_, err := facade.SearchLaws(
			context.Background(),
			legalquery.LawSearchIntentV1{},
			budget,
		)
		assertCoreFacadeFatalError(t, err)
		if ports.totalCalls() != 0 {
			t.Fatal("SOT-ARCH-026: invalid logical input で port を呼びました")
		}
	})

	t.Run("invalid budget", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		_, err := facade.SearchLaws(
			context.Background(),
			input,
			legalquery.LegalQueryStepBudget{},
		)
		assertCoreFacadeFatalError(t, err)
		if ports.totalCalls() != 0 {
			t.Fatal("SOT-MODEL-023: invalid budget で port を呼びました")
		}
	})
}

func TestCoreLegalQueryFacadeValidatesStepBeforeCustomMaterializer(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	input := mustCoreFacadeLawSearchInput(t, "行政手続法", nil)
	budget := coreFacadeBudgetForInput(t, "step-search", input, 5)
	ports := newCoreFacadePorts(t, "core-provider", "core-source", asOf, location)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{
			newCoreFacadeBindings(t, "core-provider", "core-source", ports),
		},
		completeProviderRouteValues("core-provider"),
	)
	limit := 5
	request, err := lawsearch.NewRequest(lawsearch.RequestValues{
		Query: "materializer が返す有効な request",
		Limit: &limit,
	})
	if err != nil {
		t.Fatalf("試験用 law.search request を作成できません: %v", err)
	}
	facade := mustCoreLegalQueryFacade(
		t,
		routes,
		&coreFacadeMaterializerStub{lawSearchRequest: &request},
	)

	_, err = facade.SearchLaws(
		context.Background(),
		legalquery.LawSearchIntentV1{},
		budget,
	)
	assertCoreFacadeFatalError(t, err)
	_, err = facade.SearchLaws(
		context.Background(),
		input,
		legalquery.LegalQueryStepBudget{},
	)
	assertCoreFacadeFatalError(t, err)
	readInput := mustCoreFacadeLawReadIDInput(t)
	_, err = facade.SearchLaws(
		context.Background(),
		input,
		coreFacadeBudgetForInput(t, "step-read", readInput, 5),
	)
	assertCoreFacadeFatalError(t, err)
	if ports.totalCalls() != 0 {
		t.Fatalf(
			"SOT-ARCH-022/026: facade の事前検証失敗で port を %d 回呼びました",
			ports.totalCalls(),
		)
	}
}

func TestCoreLegalQueryFacadeValidatesResultsAndCollectionLimit(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	input := mustCoreFacadeLawSearchInput(t, "行政手続法", nil)

	t.Run("選択済み binding と異なる結果", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawSearch.result = coreFacadeLawSearchPage(
			t,
			[]coreFacadeLawResourceValues{
				{lawID: "law-other", revisionID: "revision-other"},
			},
			"other-provider",
			"other-source",
		)
		_, err := facade.SearchLaws(
			context.Background(),
			input,
			coreFacadeBudgetForInput(t, "step-search", input, 5),
		)
		assertCoreFacadeFatalError(t, err)
		if ports.lawSearch.calls != 1 {
			t.Fatalf("SOT-MODEL-024: port 呼出し回数 = %d", ports.lawSearch.calls)
		}
	})

	t.Run("effectiveLimit 超過", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawSearch.result = coreFacadeLawSearchPage(
			t,
			[]coreFacadeLawResourceValues{
				{lawID: "law-1", revisionID: "revision-1"},
				{lawID: "law-2", revisionID: "revision-2"},
			},
			"core-provider",
			"core-source",
		)
		_, err := facade.SearchLaws(
			context.Background(),
			input,
			coreFacadeBudgetForInput(t, "step-search", input, 1),
		)
		assertCoreFacadeFatalError(t, err)
	})

	t.Run("invalid typed read result", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawDocumentRead.result = coreFacadeZeroDocumentResult()
		readInput := mustCoreFacadeLawReadIDInput(t)
		_, err := facade.ReadLawDocument(
			context.Background(),
			readInput,
			coreFacadeBudgetForInput(t, "step-read", readInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
	})

	t.Run("document read の対象法令不一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawDocumentRead.result = coreFacadeLawDocumentResult(
			t,
			"core-provider",
			"core-source",
			"law-other",
			"revision-other",
			nil,
		)
		readInput := mustCoreFacadeLawReadIDInput(t)
		_, err := facade.ReadLawDocument(
			context.Background(),
			readInput,
			coreFacadeBudgetForInput(t, "step-read", readInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
		if ports.lawDocumentRead.calls != 1 {
			t.Fatalf(
				"SOT-IF-024: document read の port 呼出し回数 = %d",
				ports.lawDocumentRead.calls,
			)
		}
	})

	t.Run("document read の対象版不一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawDocumentRead.result = coreFacadeLawDocumentResult(
			t,
			"core-provider",
			"core-source",
			"law-1",
			"revision-other",
			nil,
		)
		readInput := mustCoreFacadeLawReadIDInput(t)
		_, err := facade.ReadLawDocument(
			context.Background(),
			readInput,
			coreFacadeBudgetForInput(t, "step-read", readInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
		if ports.lawDocumentRead.calls != 1 {
			t.Fatalf(
				"SOT-IF-024: document read の port 呼出し回数 = %d",
				ports.lawDocumentRead.calls,
			)
		}
	})

	t.Run("article read の条文位置不一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		otherLocation := mustCoreFacadeLocationAt(t, "2")
		ports.lawArticleRead.result = coreFacadeLawArticleResult(
			t,
			"core-provider",
			"core-source",
			"law-1",
			"revision-1",
			otherLocation,
		)
		articleInput, err := legalquery.NewLawArticleReadIntentV1(
			legalquery.LawArticleReadIntentV1Values{
				LawID:    "law-1",
				Location: location,
			},
		)
		if err != nil {
			t.Fatalf("試験用 law article read input を作成できません: %v", err)
		}
		_, err = facade.ReadLawArticle(
			context.Background(),
			articleInput,
			coreFacadeBudgetForInput(t, "step-article", articleInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
		if ports.lawArticleRead.calls != 1 {
			t.Fatalf(
				"SOT-IF-025: article read の port 呼出し回数 = %d",
				ports.lawArticleRead.calls,
			)
		}
	})

	t.Run("document read の asOf 不一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		otherDate := mustCoreFacadeDate(t, "2026-07-26")
		ports.lawDocumentRead.result = coreFacadeLawDocumentResult(
			t,
			"core-provider",
			"core-source",
			"law-1",
			"revision-1",
			&otherDate,
		)
		readInput, err := legalquery.NewLawReadIntentV1(
			legalquery.LawReadIntentV1Values{
				LawID: "law-1",
				AsOf:  &asOf,
			},
		)
		if err != nil {
			t.Fatalf("試験用 law read input を作成できません: %v", err)
		}
		_, err = facade.ReadLawDocument(
			context.Background(),
			readInput,
			coreFacadeBudgetForInput(t, "step-read", readInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
	})

	t.Run("document read の asOf 一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		ports.lawDocumentRead.result = coreFacadeLawDocumentResult(
			t,
			"core-provider",
			"core-source",
			"law-1",
			"revision-1",
			&asOf,
		)
		readInput, err := legalquery.NewLawReadIntentV1(
			legalquery.LawReadIntentV1Values{
				LawID: "law-1",
				AsOf:  &asOf,
			},
		)
		if err != nil {
			t.Fatalf("試験用 law read input を作成できません: %v", err)
		}
		if _, err := facade.ReadLawDocument(
			context.Background(),
			readInput,
			coreFacadeBudgetForInput(t, "step-read", readInput, 5),
		); err != nil {
			t.Fatalf("SOT-IF-024: asOf 一致の document read エラー = %v", err)
		}
	})

	t.Run("law update list の対象日不一致", func(t *testing.T) {
		ports, facade := newReadyCoreFacade(t, asOf, location)
		otherDate := mustCoreFacadeDate(t, "2026-07-26")
		ports.lawUpdateList.result = coreFacadeLawUpdatePage(
			t,
			"core-provider",
			"core-source",
			otherDate,
			1,
		)
		updateInput := mustCoreFacadeLawUpdateInput(t, asOf)
		_, err := facade.ListLawUpdates(
			context.Background(),
			updateInput,
			coreFacadeBudgetForInput(t, "step-updates", updateInput, 5),
		)
		assertCoreFacadeFatalError(t, err)
		if ports.lawUpdateList.calls != 1 {
			t.Fatalf(
				"SOT-IF-034: law update list の port 呼出し回数 = %d",
				ports.lawUpdateList.calls,
			)
		}
	})
}

func TestCoreLegalQueryFacadeDoesNotTruncateFullLawUpdateList(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	ports, facade := newReadyCoreFacade(t, asOf, location)
	ports.lawUpdateList.result = coreFacadeLawUpdatePage(
		t,
		"core-provider",
		"core-source",
		asOf,
		3,
	)
	input := mustCoreFacadeLawUpdateInput(t, asOf)
	budget := coreFacadeBudgetForInput(t, "step-updates", input, 1)

	result, err := facade.ListLawUpdates(context.Background(), input, budget)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: 完全更新一覧の実行エラー = %v", err)
	}
	if len(result.Items()) != 3 || len(ports.lawUpdateList.result.Items()) != 3 {
		t.Fatalf(
			"SOT-ARCH-026/SOT-MODEL-023: facade が完全一覧を %d 件へ切り詰めました",
			len(result.Items()),
		)
	}
}

func TestCoreLegalQueryFacadeFailsClosedForInvalidConstructionAndContext(t *testing.T) {
	asOf := mustCoreFacadeDate(t, "2026-07-27")
	location := mustCoreFacadeLocation(t)
	ports, validFacade := newReadyCoreFacade(t, asOf, location)
	input := mustCoreFacadeLawSearchInput(t, "行政手続法", nil)
	budget := coreFacadeBudgetForInput(t, "step-search", input, 5)

	t.Run("nil materializer", func(t *testing.T) {
		routes := coreFacadeRoutesFromReadyFacade(t, asOf, location)
		var materializer legalquery.CoreRequestMaterializer
		if _, err := application.NewCoreLegalQueryFacade(routes, materializer); err == nil {
			t.Fatal("SOT-ARCH-026: nil materializer を受理しました")
		}
	})

	t.Run("typed nil materializer", func(t *testing.T) {
		routes := coreFacadeRoutesFromReadyFacade(t, asOf, location)
		var materializer *coreFacadeMaterializerStub
		if _, err := application.NewCoreLegalQueryFacade(routes, materializer); err == nil {
			t.Fatal("SOT-ARCH-026: typed nil materializer を受理しました")
		}
	})

	t.Run("zero materializer", func(t *testing.T) {
		routes := coreFacadeRoutesFromReadyFacade(t, asOf, location)
		if _, err := application.NewCoreLegalQueryFacade(
			routes,
			legalquery.CoreMaterializer{},
		); err == nil {
			t.Fatal("SOT-ARCH-026: zero-value materializer を受理しました")
		}
	})

	t.Run("invalid materializer", func(t *testing.T) {
		routes := coreFacadeRoutesFromReadyFacade(t, asOf, location)
		cause := errors.New("materializer startup validation error")
		_, err := application.NewCoreLegalQueryFacade(
			routes,
			&coreFacadeMaterializerStub{validateErr: cause},
		)
		if err == nil || !errors.Is(err, cause) {
			t.Fatalf(
				"SOT-ARCH-026: materializer の起動時検証失敗を保持しませんでした: %v",
				err,
			)
		}
	})

	t.Run("uninitialized routes", func(t *testing.T) {
		if _, err := application.NewCoreLegalQueryFacade(
			application.ProviderRoutes{},
			legalquery.NewCoreMaterializer(),
		); err == nil {
			t.Fatal("SOT-ARCH-023/026: 未初期化 routes を受理しました")
		}
	})

	t.Run("zero facade", func(t *testing.T) {
		var facade application.CoreLegalQueryFacade
		if err := facade.Validate(); err == nil {
			t.Fatal("SOT-ENG-025: zero-value facade を有効と判定しました")
		}
		_, err := facade.SearchLaws(context.Background(), input, budget)
		assertCoreFacadeFatalError(t, err)
	})

	t.Run("nil context", func(t *testing.T) {
		//nolint:staticcheck // SOT-ARCH-023 の nil context 拒否を境界で確認する。
		_, err := validFacade.SearchLaws(nil, input, budget)
		assertCoreFacadeFatalError(t, err)
		if ports.totalCalls() != 0 {
			t.Fatal("SOT-ARCH-023: nil context で port を呼びました")
		}
	})
}

func TestCoreLegalQueryFacadeDoesNotRequireJudicialDependencies(t *testing.T) {
	ports := newCoreFacadePorts(
		t,
		"core-only-provider",
		"core-only-source",
		mustCoreFacadeDate(t, "2026-07-27"),
		mustCoreFacadeLocation(t),
	)
	bindings := newCoreOnlyFacadeBindings(
		t,
		"core-only-provider",
		"core-only-source",
		ports,
	)
	routes := newCoreFacadeRoutes(
		t,
		[]application.ProviderBindings{bindings},
		completeProviderRouteValues("core-only-provider"),
	)
	facade, err := application.NewCoreLegalQueryFacade(
		routes,
		legalquery.NewCoreMaterializer(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026/SOT-ENG-025: core-only facade を構成できません: %v", err)
	}
	if err := facade.Validate(); err != nil {
		t.Fatalf("SOT-ENG-025: core-only facade が有効ではありません: %v", err)
	}
}
