package lawv2

import (
	"context"
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type lawArticleAdapterDependencies struct {
	client lawClient
	gate   chan struct{}
}

// LawArticleAdapter は、e-Gov API Version 2 の law.article.read@1 planned binding である。
//
// runtime registry と MCP route への登録は、四能力を揃える後続変更まで行わない。
type LawArticleAdapter struct {
	dependencies lawArticleAdapterDependencies
}

var _ lawarticleread.Port = (*LawArticleAdapter)(nil)

// NewLawArticleAdapter は、固定接続先を使う law.article.read@1 adapter を組み立てる。
func NewLawArticleAdapter() (*LawArticleAdapter, error) {
	return newLawArticleAdapter(lawArticleAdapterDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
	})
}

func newLawArticleAdapter(
	dependencies lawArticleAdapterDependencies,
) (*LawArticleAdapter, error) {
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.article.read adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &LawArticleAdapter{dependencies: dependencies}, nil
}

// Read は、一つの条または項を固定された e-Gov GET /law_data へ対応させる。
func (a *LawArticleAdapter) Read(
	ctx context.Context,
	request lawarticleread.Request,
) (model.SourcedResource[model.LawArticleFragment], error) {
	if ctx == nil {
		return emptyLawArticleResult(), fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return emptyLawArticleResult(), err
	}
	resource := request.Resource()
	key := resource.Key()
	if resource.ProviderID() != providerID || key.SourceID() != providerID {
		return emptyLawArticleResult(),
			fmt.Errorf("resource の providerId と sourceId は e-Gov と一致しなければなりません")
	}

	asOf, hasAsOf := request.AsOf()
	if hasAsOf && asOf.String() < "2017-04-01" {
		return emptyLawArticleResult(), newLawArticleSourceError(
			model.SourceErrorCodeUnsupportedQuery,
			"",
		)
	}
	identifier := key.ResourceID()
	if versionID, exists := key.VersionID(); exists {
		identifier = versionID
	}
	var asOfPointer *model.Date
	if hasAsOf {
		asOfPointer = &asOf
	}

	if err := a.acquire(ctx); err != nil {
		return emptyLawArticleResult(), err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetchLawArticle(
		ctx,
		lawDocumentRequest{
			identifier: identifier,
			asOf:       cloneLawDocumentDate(asOfPointer),
		},
	)
	if err != nil {
		return emptyLawArticleResult(), err
	}
	response, err := parseLawArticleResponse(
		ctx,
		fetched.body,
		request.Location(),
	)
	if err != nil {
		return emptyLawArticleResult(), err
	}
	if err := validateLawArticleResponse(key, response); err != nil {
		return emptyLawArticleResult(), err
	}
	return mapLawArticle(response, fetched.retrievedAt)
}

func validateLawArticleResponse(
	input model.SourceResourceKey,
	response lawArticleResponse,
) error {
	if input.ResourceID() != response.law.lawID {
		return newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if versionID, exists := input.VersionID(); exists &&
		versionID != response.law.revisionID {
		return newLawArticleSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return nil
}

func (a *LawArticleAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextErrorWithFactory(
			err,
			newLawArticleSourceError,
		)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawArticleSourceError(
			model.SourceErrorCodeSourceBusy,
			"",
		)
	}
}

func (a *LawArticleAdapter) release() {
	<-a.dependencies.gate
}

func emptyLawArticleResult() model.SourcedResource[model.LawArticleFragment] {
	return model.SourcedResource[model.LawArticleFragment]{}
}
