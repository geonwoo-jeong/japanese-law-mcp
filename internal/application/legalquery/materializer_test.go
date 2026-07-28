package legalquery

import (
	"encoding/json"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
)

var (
	_ CoreRequestMaterializer          = CoreMaterializer{}
	_ JudicialCasesRequestMaterializer = JudicialCasesMaterializer{}
	_ SelectedCapabilityBinding        = materializerTestBinding{}
)

func TestMaterializerCreatesSevenTypedCapabilityRequests(t *testing.T) {
	t.Parallel()

	core := NewCoreMaterializer()
	judicial := NewJudicialCasesMaterializer()
	asOf := mustDate(t, "2025-04-01")
	location := mustLawArticleLocation(t)

	lawSearchInput, err := NewLawSearchIntentV1(LawSearchIntentV1Values{
		Query: "行政手続法",
		AsOf:  &asOf,
	})
	if err != nil {
		t.Fatal(err)
	}
	var lawSearchRequest lawsearch.Request
	lawSearchRequest, err = core.MaterializeLawSearch(
		lawSearchInput,
		materializerCoreBinding(lawsearch.CapabilityID, lawsearch.MajorVersion),
		materializerCollectionBudget(7),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.search request を作成できません: %v", err)
	}
	if lawSearchRequest.Query() != "行政手続法" || lawSearchRequest.Limit() != 7 {
		t.Fatalf("SOT-ARCH-026: law.search request = %#v", lawSearchRequest)
	}
	assertMaterializerDate(t, lawSearchRequest.AsOf, "2025-04-01")
	assertMaterializerNoContinuation(t, lawSearchRequest.ContinuationToken)

	contentInput, err := NewLawContentSearchIntentV1(
		LawContentSearchIntentV1Values{
			AllTerms:     []string{"永住許可"},
			AnyTerms:     []string{"在留資格"},
			ExcludeTerms: []string{"行政書士"},
			AsOf:         &asOf,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var contentRequest lawcontentsearch.Request
	contentRequest, err = core.MaterializeLawContentSearch(
		contentInput,
		materializerCoreBinding(
			lawcontentsearch.CapabilityID,
			lawcontentsearch.MajorVersion,
		),
		materializerCollectionBudget(6),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.content.search request を作成できません: %v", err)
	}
	assertMaterializerTerms(t, contentRequest.AllTerms(), []string{"永住許可"})
	assertMaterializerTerms(t, contentRequest.AnyTerms(), []string{"在留資格"})
	assertMaterializerTerms(t, contentRequest.ExcludeTerms(), []string{"行政書士"})
	assertMaterializerDate(t, contentRequest.AsOf, "2025-04-01")
	if contentRequest.Limit() != 6 {
		t.Fatalf("SOT-ARCH-026: law.content.search limit = %d", contentRequest.Limit())
	}
	assertMaterializerNoContinuation(t, contentRequest.ContinuationToken)

	lawReadInput, err := NewLawReadIntentV1(LawReadIntentV1Values{
		LawID:      "503AC0000000037",
		RevisionID: "503AC0000000037_20250601",
	})
	if err != nil {
		t.Fatal(err)
	}
	var lawReadRequest lawdocumentread.Request
	lawReadRequest, err = core.MaterializeLawDocumentRead(
		lawReadInput,
		materializerCoreBinding(
			lawdocumentread.CapabilityID,
			lawdocumentread.MajorVersion,
		),
		materializerReadBudget(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.document.read request を作成できません: %v", err)
	}
	assertMaterializerLawRef(
		t,
		lawReadRequest.Resource(),
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"503AC0000000037",
		"503AC0000000037_20250601",
	)
	if _, exists := lawReadRequest.AsOf(); exists {
		t.Fatal("SOT-ARCH-026: revisionId 形の law.document.read が asOf を持っています")
	}

	articleInput, err := NewLawArticleReadIntentV1(
		LawArticleReadIntentV1Values{
			LawID:    "503AC0000000037",
			Location: location,
			AsOf:     &asOf,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	var articleRequest lawarticleread.Request
	articleRequest, err = core.MaterializeLawArticleRead(
		articleInput,
		materializerCoreBinding(
			lawarticleread.CapabilityID,
			lawarticleread.MajorVersion,
		),
		materializerReadBudget(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.article.read request を作成できません: %v", err)
	}
	assertMaterializerLawRef(
		t,
		articleRequest.Resource(),
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"503AC0000000037",
		"",
	)
	assertMaterializerDate(t, articleRequest.AsOf, "2025-04-01")
	assertMaterializerLocation(t, articleRequest.Location(), location)

	updateInput, err := NewLawUpdateListIntentV1(
		LawUpdateListIntentV1Values{Date: mustDate(t, "2026-07-27")},
	)
	if err != nil {
		t.Fatal(err)
	}
	var updateRequest lawupdatelist.Request
	updateRequest, err = core.MaterializeLawUpdateList(
		updateInput,
		materializerCoreBinding(
			lawupdatelist.CapabilityID,
			lawupdatelist.MajorVersion,
		),
		materializerCollectionBudget(4),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.update.list request を作成できません: %v", err)
	}
	if updateRequest.Date().String() != "2026-07-27" {
		t.Fatalf("SOT-ARCH-026: law.update.list date = %q", updateRequest.Date().String())
	}

	judicialSearchInput, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "永住許可"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var judicialSearchRequest judicialdecisionsearch.Request
	judicialSearchRequest, err = judicial.MaterializeJudicialDecisionSearch(
		judicialSearchInput,
		materializerJudicialBinding(
			judicialdecisionsearch.CapabilityID,
			judicialdecisionsearch.MajorVersion,
		),
		materializerCollectionBudget(5),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: judicial-decision.search request を作成できません: %v", err)
	}
	if judicialSearchRequest.Query() != "永住許可" ||
		judicialSearchRequest.Limit() != 5 {
		t.Fatalf(
			"SOT-ARCH-026: judicial-decision.search request = %#v",
			judicialSearchRequest,
		)
	}
	assertMaterializerNoContinuation(t, judicialSearchRequest.ContinuationToken)

	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"95570/detail2",
	)
	judicialReadInput, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: judicialRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	var judicialReadRequest judicialdecisionread.Request
	judicialReadRequest, err = judicial.MaterializeJudicialDecisionRead(
		judicialReadInput,
		materializerJudicialBinding(
			judicialdecisionread.CapabilityID,
			judicialdecisionread.MajorVersion,
		),
		materializerReadBudget(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: judicial-decision.read request を作成できません: %v", err)
	}
	if judicialReadRequest.Ref() != judicialRef {
		t.Fatalf(
			"SOT-ARCH-026: judicial-decision.read ref = %#v",
			judicialReadRequest.Ref(),
		)
	}
}

func TestMaterializerBuildsLawDocumentRefFromRevisionOrAsOf(t *testing.T) {
	t.Parallel()

	materializer := NewCoreMaterializer()
	binding := materializerCoreBinding(
		lawdocumentread.CapabilityID,
		lawdocumentread.MajorVersion,
	)
	asOf := mustDate(t, "2024-12-31")
	tests := []struct {
		name            string
		values          LawReadIntentV1Values
		wantVersionID   string
		wantAsOf        string
		wantAsOfPresent bool
	}{
		{
			name: "revisionId は versionId にだけ使う",
			values: LawReadIntentV1Values{
				LawID:      "opaque-law-id",
				RevisionID: "opaque-revision-id",
			},
			wantVersionID: "opaque-revision-id",
		},
		{
			name: "asOf は versionId に変換しない",
			values: LawReadIntentV1Values{
				LawID: "opaque-law-id",
				AsOf:  &asOf,
			},
			wantAsOf:        "2024-12-31",
			wantAsOfPresent: true,
		},
	}
	for _, testCase := range tests {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			input, err := NewLawReadIntentV1(testCase.values)
			if err != nil {
				t.Fatal(err)
			}
			request, err := materializer.MaterializeLawDocumentRead(
				input,
				binding,
				materializerReadBudget(),
			)
			if err != nil {
				t.Fatalf("SOT-ARCH-026: law.document.read request を作成できません: %v", err)
			}
			assertMaterializerLawRef(
				t,
				request.Resource(),
				"e-gov-law-api-v2",
				"e-gov-law-api-v2",
				"opaque-law-id",
				testCase.wantVersionID,
			)
			gotAsOf, exists := request.AsOf()
			if exists != testCase.wantAsOfPresent ||
				exists && gotAsOf.String() != testCase.wantAsOf {
				t.Fatalf(
					"SOT-ARCH-026: asOf = %q, %t",
					gotAsOf.String(),
					exists,
				)
			}
		})
	}
}

func TestMaterializerPreservesExplicitReferences(t *testing.T) {
	t.Parallel()

	materializer := NewCoreMaterializer()
	asOf := mustDate(t, "2025-03-31")
	location := mustLawArticleLocation(t)
	documentRef := newVersionedSourceResourceRef(
		t,
		"independent-law-provider",
		"independent-law-source",
		"law",
		"opaque-law-id",
		"opaque-revision-id",
	)
	documentInput, err := NewLawReadIntentV1(
		LawReadIntentV1Values{Ref: &documentRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	documentRequest, err := materializer.MaterializeLawDocumentRead(
		documentInput,
		materializerTestBinding{
			providerID:             "independent-law-provider",
			sourceID:               "independent-law-source",
			capabilityID:           lawdocumentread.CapabilityID,
			capabilityMajorVersion: lawdocumentread.MajorVersion,
		},
		materializerReadBudget(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: explicit law ref を変換できません: %v", err)
	}
	if documentRequest.Resource() != documentRef {
		t.Fatalf("SOT-ARCH-026: law.document.read ref = %#v", documentRequest.Resource())
	}

	articleRef := newLegalQuerySourceResourceRef(
		t,
		"independent-law-provider",
		"independent-law-source",
		"law",
		"opaque-law-id",
	)
	articleInput, err := NewLawArticleReadIntentV1(
		LawArticleReadIntentV1Values{
			Ref:      &articleRef,
			Location: location,
			AsOf:     &asOf,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	articleRequest, err := materializer.MaterializeLawArticleRead(
		articleInput,
		materializerTestBinding{
			providerID:             "independent-law-provider",
			sourceID:               "independent-law-source",
			capabilityID:           lawarticleread.CapabilityID,
			capabilityMajorVersion: lawarticleread.MajorVersion,
		},
		materializerReadBudget(),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: explicit article ref を変換できません: %v", err)
	}
	if articleRequest.Resource() != articleRef {
		t.Fatalf("SOT-ARCH-026: law.article.read ref = %#v", articleRequest.Resource())
	}
	assertMaterializerDate(t, articleRequest.AsOf, "2025-03-31")
	assertMaterializerLocation(t, articleRequest.Location(), location)
}

func TestMaterializerUsesEffectiveLimitWithoutContinuation(t *testing.T) {
	t.Parallel()

	core := NewCoreMaterializer()
	judicial := NewJudicialCasesMaterializer()
	lawInput, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "帰化"},
	)
	if err != nil {
		t.Fatal(err)
	}
	contentInput, err := NewLawContentSearchIntentV1(
		LawContentSearchIntentV1Values{AnyTerms: []string{"永住許可"}},
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialInput, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "在留資格"},
	)
	if err != nil {
		t.Fatal(err)
	}
	budget := materializerCollectionBudget(3)

	lawRequest, err := core.MaterializeLawSearch(
		lawInput,
		materializerCoreBinding(lawsearch.CapabilityID, lawsearch.MajorVersion),
		budget,
	)
	if err != nil {
		t.Fatal(err)
	}
	contentRequest, err := core.MaterializeLawContentSearch(
		contentInput,
		materializerCoreBinding(
			lawcontentsearch.CapabilityID,
			lawcontentsearch.MajorVersion,
		),
		budget,
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialRequest, err := judicial.MaterializeJudicialDecisionSearch(
		judicialInput,
		materializerJudicialBinding(
			judicialdecisionsearch.CapabilityID,
			judicialdecisionsearch.MajorVersion,
		),
		budget,
	)
	if err != nil {
		t.Fatal(err)
	}
	if lawRequest.Limit() != 3 ||
		contentRequest.Limit() != 3 ||
		judicialRequest.Limit() != 3 {
		t.Fatalf(
			"SOT-ARCH-026: limits = %d, %d, %d",
			lawRequest.Limit(),
			contentRequest.Limit(),
			judicialRequest.Limit(),
		)
	}
	assertMaterializerNoContinuation(t, lawRequest.ContinuationToken)
	assertMaterializerNoContinuation(t, contentRequest.ContinuationToken)
	assertMaterializerNoContinuation(t, judicialRequest.ContinuationToken)
}

func TestMaterializerKeepsLawUpdateListAsFullListRequest(t *testing.T) {
	t.Parallel()

	input, err := NewLawUpdateListIntentV1(
		LawUpdateListIntentV1Values{Date: mustDate(t, "2026-07-27")},
	)
	if err != nil {
		t.Fatal(err)
	}
	request, err := NewCoreMaterializer().MaterializeLawUpdateList(
		input,
		materializerCoreBinding(
			lawupdatelist.CapabilityID,
			lawupdatelist.MajorVersion,
		),
		materializerCollectionBudget(1),
	)
	if err != nil {
		t.Fatalf("SOT-ARCH-026: law.update.list request を作成できません: %v", err)
	}
	encoded, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if string(encoded) != `{"date":"2026-07-27"}` {
		t.Fatalf(
			"SOT-ARCH-026: law.update.list に上限または continuation が混入しました: %s",
			encoded,
		)
	}
}

func TestMaterializerRejectsBindingMismatch(t *testing.T) {
	t.Parallel()

	core := NewCoreMaterializer()
	judicial := NewJudicialCasesMaterializer()
	lawRef := newLegalQuerySourceResourceRef(
		t,
		"independent-law-provider",
		"independent-law-source",
		"law",
		"law-1",
	)
	lawReadInput, err := NewLawReadIntentV1(
		LawReadIntentV1Values{Ref: &lawRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	lawSearchInput, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "行政手続法"},
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"decision-1",
	)
	judicialReadInput, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: judicialRef},
	)
	if err != nil {
		t.Fatal(err)
	}
	tests := map[string]func() error{
		"capabilityId": func() error {
			_, err := core.MaterializeLawSearch(
				lawSearchInput,
				materializerCoreBinding(
					lawcontentsearch.CapabilityID,
					lawcontentsearch.MajorVersion,
				),
				materializerCollectionBudget(5),
			)
			return err
		},
		"capabilityMajorVersion": func() error {
			binding := materializerCoreBinding(
				lawsearch.CapabilityID,
				lawsearch.MajorVersion,
			)
			binding.capabilityMajorVersion++
			_, err := core.MaterializeLawSearch(
				lawSearchInput,
				binding,
				materializerCollectionBudget(5),
			)
			return err
		},
		"law ref providerId": func() error {
			binding := materializerTestBinding{
				providerID:             "different-provider",
				sourceID:               "independent-law-source",
				capabilityID:           lawdocumentread.CapabilityID,
				capabilityMajorVersion: lawdocumentread.MajorVersion,
			}
			_, err := core.MaterializeLawDocumentRead(
				lawReadInput,
				binding,
				materializerReadBudget(),
			)
			return err
		},
		"law ref sourceId": func() error {
			binding := materializerTestBinding{
				providerID:             "independent-law-provider",
				sourceID:               "different-source",
				capabilityID:           lawdocumentread.CapabilityID,
				capabilityMajorVersion: lawdocumentread.MajorVersion,
			}
			_, err := core.MaterializeLawDocumentRead(
				lawReadInput,
				binding,
				materializerReadBudget(),
			)
			return err
		},
		"judicial ref providerId": func() error {
			binding := materializerJudicialBinding(
				judicialdecisionread.CapabilityID,
				judicialdecisionread.MajorVersion,
			)
			binding.providerID = "different-provider"
			_, err := judicial.MaterializeJudicialDecisionRead(
				judicialReadInput,
				binding,
				materializerReadBudget(),
			)
			return err
		},
		"judicial ref sourceId": func() error {
			binding := materializerJudicialBinding(
				judicialdecisionread.CapabilityID,
				judicialdecisionread.MajorVersion,
			)
			binding.sourceID = "different-source"
			_, err := judicial.MaterializeJudicialDecisionRead(
				judicialReadInput,
				binding,
				materializerReadBudget(),
			)
			return err
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := test(); err == nil {
				t.Fatalf("SOT-ARCH-026: %s の不一致を受理しました", name)
			}
		})
	}
}

func TestMaterializerRejectsWrongBudgetShape(t *testing.T) {
	t.Parallel()

	core := NewCoreMaterializer()
	judicial := NewJudicialCasesMaterializer()
	lawSearchInput, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "行政手続法"},
	)
	if err != nil {
		t.Fatal(err)
	}
	lawReadInput, err := NewLawReadIntentV1(
		LawReadIntentV1Values{LawID: "law-1"},
	)
	if err != nil {
		t.Fatal(err)
	}
	updateInput, err := NewLawUpdateListIntentV1(
		LawUpdateListIntentV1Values{Date: mustDate(t, "2026-07-27")},
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialSearchInput, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "永住許可"},
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialRef := newLegalQuerySourceResourceRef(
		t,
		"courts-hanrei-html",
		"courts-hanrei",
		"judicial-decision",
		"decision-1",
	)
	judicialReadInput, err := NewJudicialDecisionReadIntentV1(
		JudicialDecisionReadIntentV1Values{Ref: judicialRef},
	)
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]func() error{
		"collection に read 予算": func() error {
			_, err := core.MaterializeLawSearch(
				lawSearchInput,
				materializerCoreBinding(
					lawsearch.CapabilityID,
					lawsearch.MajorVersion,
				),
				materializerReadBudget(),
			)
			return err
		},
		"read に collection 予算": func() error {
			_, err := core.MaterializeLawDocumentRead(
				lawReadInput,
				materializerCoreBinding(
					lawdocumentread.CapabilityID,
					lawdocumentread.MajorVersion,
				),
				materializerCollectionBudget(5),
			)
			return err
		},
		"更新一覧に read 予算": func() error {
			_, err := core.MaterializeLawUpdateList(
				updateInput,
				materializerCoreBinding(
					lawupdatelist.CapabilityID,
					lawupdatelist.MajorVersion,
				),
				materializerReadBudget(),
			)
			return err
		},
		"裁判例検索に read 予算": func() error {
			_, err := judicial.MaterializeJudicialDecisionSearch(
				judicialSearchInput,
				materializerJudicialBinding(
					judicialdecisionsearch.CapabilityID,
					judicialdecisionsearch.MajorVersion,
				),
				materializerReadBudget(),
			)
			return err
		},
		"裁判例読取りに collection 予算": func() error {
			_, err := judicial.MaterializeJudicialDecisionRead(
				judicialReadInput,
				materializerJudicialBinding(
					judicialdecisionread.CapabilityID,
					judicialdecisionread.MajorVersion,
				),
				materializerCollectionBudget(5),
			)
			return err
		},
		"検証されていない zero-value 予算": func() error {
			_, err := core.MaterializeLawSearch(
				lawSearchInput,
				materializerCoreBinding(
					lawsearch.CapabilityID,
					lawsearch.MajorVersion,
				),
				LegalQueryStepBudget{},
			)
			return err
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := test(); err == nil {
				t.Fatalf("SOT-ARCH-026: %sを受理しました", name)
			}
		})
	}
}

func TestMaterializerRejectsNilBindingAndZeroValueMaterializers(t *testing.T) {
	t.Parallel()

	lawInput, err := NewLawSearchIntentV1(
		LawSearchIntentV1Values{Query: "行政手続法"},
	)
	if err != nil {
		t.Fatal(err)
	}
	judicialInput, err := NewJudicialDecisionSearchIntentV1(
		JudicialDecisionSearchIntentV1Values{Query: "永住許可"},
	)
	if err != nil {
		t.Fatal(err)
	}
	var typedNil *materializerTypedNilBinding
	var zeroCore CoreMaterializer
	var zeroJudicial JudicialCasesMaterializer
	tests := map[string]func() error{
		"nil binding": func() error {
			_, err := NewCoreMaterializer().MaterializeLawSearch(
				lawInput,
				nil,
				materializerCollectionBudget(5),
			)
			return err
		},
		"typed nil binding": func() error {
			_, err := NewCoreMaterializer().MaterializeLawSearch(
				lawInput,
				typedNil,
				materializerCollectionBudget(5),
			)
			return err
		},
		"core zero value": func() error {
			_, err := zeroCore.MaterializeLawSearch(
				lawInput,
				materializerCoreBinding(
					lawsearch.CapabilityID,
					lawsearch.MajorVersion,
				),
				materializerCollectionBudget(5),
			)
			return err
		},
		"judicial-cases zero value": func() error {
			_, err := zeroJudicial.MaterializeJudicialDecisionSearch(
				judicialInput,
				materializerJudicialBinding(
					judicialdecisionsearch.CapabilityID,
					judicialdecisionsearch.MajorVersion,
				),
				materializerCollectionBudget(5),
			)
			return err
		},
	}
	for name, test := range tests {
		name, test := name, test
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := test(); err == nil {
				t.Fatalf("SOT-ARCH-026: %sを受理しました", name)
			}
		})
	}
}
