package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func assertCoreFacadeCalls(
	t *testing.T,
	ports *coreFacadePorts,
	ctx context.Context,
) {
	t.Helper()

	callValues := []struct {
		name     string
		calls    int
		contexts []context.Context
	}{
		{name: lawsearch.CapabilityID, calls: ports.lawSearch.calls, contexts: ports.lawSearch.contexts},
		{name: lawcontentsearch.CapabilityID, calls: ports.lawContentSearch.calls, contexts: ports.lawContentSearch.contexts},
		{name: lawdocumentread.CapabilityID, calls: ports.lawDocumentRead.calls, contexts: ports.lawDocumentRead.contexts},
		{name: lawarticleread.CapabilityID, calls: ports.lawArticleRead.calls, contexts: ports.lawArticleRead.contexts},
		{name: lawupdatelist.CapabilityID, calls: ports.lawUpdateList.calls, contexts: ports.lawUpdateList.contexts},
	}
	for _, value := range callValues {
		if value.calls != 1 || len(value.contexts) != 1 {
			t.Fatalf("%s の呼出し回数 = %d、期待値 = 1", value.name, value.calls)
		}
		if value.contexts[0] != ctx {
			t.Fatalf("%s に同一 context を渡しませんでした", value.name)
		}
	}
}

func assertCoreFacadeRequests(
	t *testing.T,
	ports *coreFacadePorts,
	asOf model.Date,
	location model.LawArticleLocation,
	limit int,
) {
	t.Helper()

	search := ports.lawSearch.requests[0]
	if search.Query() != "行政手続法" || search.Limit() != limit {
		t.Fatal("law.search request の query または limit が一致しません")
	}
	assertCoreFacadeDateOption(t, search.AsOf, asOf, true)
	if _, exists := search.ContinuationToken(); exists {
		t.Fatal("law.search request に continuationToken を設定しました")
	}

	content := ports.lawContentSearch.requests[0]
	if !reflect.DeepEqual(content.AllTerms(), []string{"許可"}) ||
		!reflect.DeepEqual(content.AnyTerms(), []string{"申請"}) ||
		!reflect.DeepEqual(content.ExcludeTerms(), []string{"罰則"}) ||
		content.Limit() != limit {
		t.Fatal("law.content.search request の検索条件または limit が一致しません")
	}
	assertCoreFacadeDateOption(t, content.AsOf, asOf, true)
	if _, exists := content.ContinuationToken(); exists {
		t.Fatal("law.content.search request に continuationToken を設定しました")
	}

	document := ports.lawDocumentRead.requests[0]
	assertCoreFacadeLawResource(
		t,
		document.Resource(),
		"core-provider",
		"core-source",
		"law-1",
		"revision-1",
	)
	assertCoreFacadeDateOption(t, document.AsOf, model.Date{}, false)

	article := ports.lawArticleRead.requests[0]
	assertCoreFacadeLawResource(
		t,
		article.Resource(),
		"core-provider",
		"core-source",
		"law-1",
		"",
	)
	if !sameCoreFacadeLocation(article.Location(), location) {
		t.Fatal("law.article.read request の location が一致しません")
	}
	assertCoreFacadeDateOption(t, article.AsOf, asOf, true)

	if ports.lawUpdateList.requests[0].Date() != asOf {
		t.Fatal("law.update.list request の date が一致しません")
	}
}

func assertCoreFacadeLawResource(
	t *testing.T,
	ref model.SourceResourceRef,
	providerID string,
	sourceID string,
	lawID string,
	revisionID string,
) {
	t.Helper()
	key := ref.Key()
	version, versionExists := key.VersionID()
	if ref.ProviderID() != providerID ||
		key.SourceID() != sourceID ||
		key.ResourceType() != "law" ||
		key.ResourceID() != lawID ||
		versionExists != (revisionID != "") ||
		(versionExists && version != revisionID) {
		t.Fatalf(
			"法令 ref = (%q, %q, %q, %q, %q)",
			ref.ProviderID(),
			key.SourceID(),
			key.ResourceType(),
			key.ResourceID(),
			version,
		)
	}
}

func assertCoreFacadeDateOption(
	t *testing.T,
	getter func() (model.Date, bool),
	expected model.Date,
	expectedExists bool,
) {
	t.Helper()
	actual, exists := getter()
	if exists != expectedExists || (exists && actual != expected) {
		t.Fatalf(
			"日付 = (%q, %t)、期待値 = (%q, %t)",
			actual.String(),
			exists,
			expected.String(),
			expectedExists,
		)
	}
}

func sameCoreFacadeLocation(
	left model.LawArticleLocation,
	right model.LawArticleLocation,
) bool {
	leftParagraph, leftExists := left.ParagraphNumber()
	rightParagraph, rightExists := right.ParagraphNumber()
	return left.Provision() == right.Provision() &&
		left.ArticleNumber() == right.ArticleNumber() &&
		leftExists == rightExists &&
		(!leftExists || leftParagraph == rightParagraph)
}

func assertCoreFacadeExecutedError(
	t *testing.T,
	err error,
	cause error,
) {
	t.Helper()
	if err == nil {
		t.Fatal("実行済み port error が返されませんでした")
	}
	var executed legalquery.ExecutedStepError
	if !errors.As(err, &executed) {
		t.Fatalf("port error を ExecutedStepError として分類できません: %v", err)
	}
	if !errors.Is(err, cause) {
		t.Fatalf("ExecutedStepError が元の原因を保持しません: %v", err)
	}
}

func assertCoreFacadeFatalError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("外部呼出し前後の致命的 error が返されませんでした")
	}
	var executed legalquery.ExecutedStepError
	if errors.As(err, &executed) {
		t.Fatalf("致命的 error を ExecutedStepError として分類しました: %v", err)
	}
}
