package lawv2

import (
	"context"
	"fmt"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

type lawDocumentAdapterDependencies struct {
	client lawClient
	gate   chan struct{}
}

// LawDocumentAdapter は、e-Gov API Version 2 の law.document.read@1 planned binding である。
//
// runtime registry と MCP route への登録は、四能力を揃える後続変更まで行わない。
type LawDocumentAdapter struct {
	dependencies lawDocumentAdapterDependencies
}

var _ lawdocumentread.Port = (*LawDocumentAdapter)(nil)

// NewLawDocumentAdapter は、固定接続先を使う law.document.read@1 adapter を組み立てる。
func NewLawDocumentAdapter() (*LawDocumentAdapter, error) {
	return newLawDocumentAdapter(lawDocumentAdapterDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
	})
}

func newLawDocumentAdapter(
	dependencies lawDocumentAdapterDependencies,
) (*LawDocumentAdapter, error) {
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.document.read adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	return &LawDocumentAdapter{dependencies: dependencies}, nil
}

// Read は、一つの法令本文を固定された e-Gov GET /law_data へ対応させる。
func (a *LawDocumentAdapter) Read(
	ctx context.Context,
	request lawdocumentread.Request,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	if ctx == nil {
		return emptyLawDocumentResult(), fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return emptyLawDocumentResult(), err
	}
	resource := request.Resource()
	key := resource.Key()
	if resource.ProviderID() != providerID || key.SourceID() != providerID {
		return emptyLawDocumentResult(),
			fmt.Errorf("resource の providerId と sourceId は e-Gov と一致しなければなりません")
	}

	asOf, hasAsOf := request.AsOf()
	if hasAsOf && asOf.String() < "2017-04-01" {
		return emptyLawDocumentResult(), newLawDocumentSourceError(
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
		return emptyLawDocumentResult(), err
	}
	defer a.release()

	fetched, err := a.dependencies.client.fetchLawDocument(
		ctx,
		lawDocumentRequest{
			identifier: identifier,
			asOf:       cloneLawDocumentDate(asOfPointer),
		},
	)
	if err != nil {
		return emptyLawDocumentResult(), err
	}
	response, err := parseLawDocumentResponse(ctx, fetched.body)
	if err != nil {
		return emptyLawDocumentResult(), err
	}
	if err := validateLawDocumentResponse(key, response); err != nil {
		return emptyLawDocumentResult(), err
	}
	return mapLawDocument(response, asOfPointer, fetched.retrievedAt)
}

func validateLawDocumentResponse(
	input model.SourceResourceKey,
	response lawDocumentResponse,
) error {
	if input.ResourceID() != response.law.lawID {
		return newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	if versionID, exists := input.VersionID(); exists &&
		versionID != response.law.revisionID {
		return newLawDocumentSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return nil
}

func (a *LawDocumentAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextErrorWithFactory(
			err,
			newLawDocumentSourceError,
		)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawDocumentSourceError(
			model.SourceErrorCodeSourceBusy,
			"",
		)
	}
}

func (a *LawDocumentAdapter) release() {
	<-a.dependencies.gate
}

func emptyLawDocumentResult() model.SourcedResource[model.LawDocumentRepresentation] {
	return model.SourcedResource[model.LawDocumentRepresentation]{}
}
