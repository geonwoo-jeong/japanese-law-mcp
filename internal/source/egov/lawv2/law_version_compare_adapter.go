package lawv2

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type lawVersionCompareAdapterDependencies struct {
	client lawClient
	gate   chan struct{}
	limits lawVersionCompareLimits
}

// LawVersionCompareAdapter は、e-Gov API Version 2 の law.version.compare@1 binding である。
type LawVersionCompareAdapter struct {
	dependencies lawVersionCompareAdapterDependencies
}

var _ lawversioncompare.Port = (*LawVersionCompareAdapter)(nil)

// NewLawVersionCompareAdapter は、固定接続先と共有同時実行枠を使う adapter を組み立てる。
func NewLawVersionCompareAdapter() (*LawVersionCompareAdapter, error) {
	return newLawVersionCompareAdapter(lawVersionCompareAdapterDependencies{
		client: newProductionClient(),
		gate:   sharedEGovHTTPGate,
		limits: defaultLawVersionCompareLimits(),
	})
}

func newLawVersionCompareAdapter(
	dependencies lawVersionCompareAdapterDependencies,
) (*LawVersionCompareAdapter, error) {
	if err := verifyEmbeddedLawDataContract(newLawVersionCompareSourceError); err != nil {
		return nil, err
	}
	if dependencies.gate == nil ||
		dependencies.client.dependencies.doer == nil ||
		dependencies.client.dependencies.now == nil ||
		dependencies.client.dependencies.sleep == nil {
		return nil, fmt.Errorf("e-Gov law.version.compare adapter の依存関係が不足しています")
	}
	if cap(dependencies.gate) != 1 {
		return nil, fmt.Errorf("e-Gov の同時実行枠は一件でなければなりません")
	}
	if err := dependencies.limits.validate(); err != nil {
		return nil, err
	}
	return &LawVersionCompareAdapter{dependencies: dependencies}, nil
}

// Compare は、一枠を保持したまま二版を順に取得し、条単位の比較を返す。
func (a *LawVersionCompareAdapter) Compare(
	ctx context.Context,
	request lawversioncompare.Request,
) (model.SourcedResource[model.LawVersionComparison], error) {
	if ctx == nil {
		return emptyLawVersionComparisonResult(), fmt.Errorf("context は必須です")
	}
	if err := request.Validate(); err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	resource := request.Resource()
	key := resource.Key()
	if resource.ProviderID() != providerID || key.SourceID() != providerID {
		return emptyLawVersionComparisonResult(), fmt.Errorf(
			"resource の providerId と sourceId は e-Gov と一致しなければなりません",
		)
	}
	if err := validateLawVersionCompareSelectorRange(request.Before()); err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	if err := validateLawVersionCompareSelectorRange(request.After()); err != nil {
		return emptyLawVersionComparisonResult(), err
	}

	if err := a.acquire(ctx); err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	defer a.release()

	beforeResult, err := a.fetchComparedLawDocument(ctx, key.ResourceID(), request.Before())
	if err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	afterResult, err := a.fetchComparedLawDocument(ctx, key.ResourceID(), request.After())
	if err != nil {
		return emptyLawVersionComparisonResult(), err
	}

	processingContext, cancel := context.WithTimeout(
		ctx,
		a.dependencies.limits.processingTimeout,
	)
	defer cancel()
	beforeDocument, err := parseLawVersionDocument(
		processingContext,
		beforeResult,
		a.dependencies.limits,
	)
	if err != nil {
		return emptyLawVersionComparisonResult(), normalizeLawVersionProcessingError(
			ctx,
			processingContext,
			err,
		)
	}
	afterDocument, err := parseLawVersionDocument(
		processingContext,
		afterResult,
		a.dependencies.limits,
	)
	if err != nil {
		return emptyLawVersionComparisonResult(), normalizeLawVersionProcessingError(
			ctx,
			processingContext,
			err,
		)
	}
	if err := validateLawVersionCompareTextBudget(
		beforeDocument,
		afterDocument,
		a.dependencies.limits,
	); err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	comparison, err := compareLawVersionDocuments(
		processingContext,
		beforeDocument,
		afterDocument,
		a.dependencies.limits,
	)
	if err != nil {
		return emptyLawVersionComparisonResult(), normalizeLawVersionProcessingError(
			ctx,
			processingContext,
			err,
		)
	}
	if err := validateLawVersionCompareResultBudget(comparison, a.dependencies.limits); err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	if err := processingContext.Err(); err != nil {
		return emptyLawVersionComparisonResult(), normalizeLawVersionProcessingError(
			ctx,
			processingContext,
			err,
		)
	}
	result, err := buildLawVersionComparisonResource(beforeDocument, afterDocument, comparison)
	if err != nil {
		return emptyLawVersionComparisonResult(), err
	}
	if err := processingContext.Err(); err != nil {
		return emptyLawVersionComparisonResult(), normalizeLawVersionProcessingError(
			ctx,
			processingContext,
			err,
		)
	}
	return result, nil
}

func validateLawVersionCompareSelectorRange(selector lawversioncompare.Selector) error {
	if asOf, exists := selector.AsOf(); exists && asOf.String() < "2017-04-01" {
		return newLawVersionCompareSourceError(model.SourceErrorCodeUnsupportedQuery, "")
	}
	return nil
}

func (a *LawVersionCompareAdapter) fetchComparedLawDocument(
	ctx context.Context,
	expectedLawID string,
	selector lawversioncompare.Selector,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	request := lawDocumentRequest{identifier: expectedLawID}
	requestedRevisionID := ""
	if revisionID, exists := selector.RevisionID(); exists {
		request.identifier = revisionID
		requestedRevisionID = revisionID
	} else if asOf, exists := selector.AsOf(); exists {
		request.asOf = &asOf
	}
	fetched, err := a.dependencies.client.fetchLawDocument(ctx, request)
	if err != nil {
		return emptyLawDocumentResult(), mapLawVersionCompareError(err)
	}
	response, err := parseLawDocumentResponse(ctx, fetched.body)
	if err != nil {
		return emptyLawDocumentResult(), mapLawVersionCompareError(err)
	}
	if response.law.lawID != expectedLawID {
		if requestedRevisionID != "" {
			return emptyLawDocumentResult(), lawversioncompare.ErrNotFound
		}
		return emptyLawDocumentResult(), invalidLawVersionCompareResponse()
	}
	if requestedRevisionID != "" && response.law.revisionID != requestedRevisionID {
		return emptyLawDocumentResult(), invalidLawVersionCompareResponse()
	}

	var asOf *model.Date
	if value, exists := selector.AsOf(); exists {
		asOf = &value
	}
	document, err := mapLawDocument(response, asOf, fetched.retrievedAt)
	if err != nil {
		return emptyLawDocumentResult(), mapLawVersionCompareError(err)
	}
	if err := document.Validate(); err != nil {
		return emptyLawDocumentResult(), invalidLawVersionCompareResponse()
	}
	return document, nil
}

func (a *LawVersionCompareAdapter) acquire(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return normalizeContextErrorWithFactory(err, newLawVersionCompareSourceError)
	}
	select {
	case a.dependencies.gate <- struct{}{}:
		return nil
	default:
		return newLawVersionCompareSourceError(model.SourceErrorCodeSourceBusy, "")
	}
}

func (a *LawVersionCompareAdapter) release() {
	<-a.dependencies.gate
}

func mapLawVersionCompareError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, lawdocumentread.ErrNotFound) {
		return lawversioncompare.ErrNotFound
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return normalizeContextErrorWithFactory(err, newLawVersionCompareSourceError)
	}
	var sourceError model.SourceError
	if errors.As(err, &sourceError) {
		retryAfter, _ := sourceError.RetryAfter()
		return newLawVersionCompareSourceError(sourceError.Code(), retryAfter)
	}
	return err
}

func normalizeLawVersionProcessingError(
	parent context.Context,
	processing context.Context,
	err error,
) error {
	if parentErr := parent.Err(); parentErr != nil {
		return normalizeContextErrorWithFactory(parentErr, newLawVersionCompareSourceError)
	}
	if errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(processing.Err(), context.DeadlineExceeded) {
		return newLawVersionCompareSourceError(
			model.SourceErrorCodeSourceProcessingLimit,
			"",
		)
	}
	if errors.Is(err, context.Canceled) {
		return normalizeContextErrorWithFactory(err, newLawVersionCompareSourceError)
	}
	return err
}

func buildLawVersionComparisonResource(
	before parsedLawVersionDocument,
	after parsedLawVersionDocument,
	comparison model.LawVersionComparison,
) (model.SourcedResource[model.LawVersionComparison], error) {
	afterRef := after.ref
	afterKey := afterRef.Key()
	derived, err := model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource(),
		ResourceKey:    afterKey,
		URL:            after.snapshot.Citation().URL(),
		RetrievedAt:    latestCompareRetrievedAt(before.provenance, after.provenance),
		MediaType:      lawDocumentMediaType,
		Transformation: model.ProvenanceTransformationDerived,
		MethodID:       "SOT-IF-060",
		InputKeys:      []model.SourceResourceKey{before.ref.Key(), afterKey},
	})
	if err != nil {
		return emptyLawVersionComparisonResult(), invalidLawVersionCompareResponse()
	}
	provenance := make([]model.Provenance, 0, len(before.provenance)+len(after.provenance)+1)
	provenance = append(provenance, before.provenance...)
	provenance = append(provenance, after.provenance...)
	provenance = append(provenance, derived)
	result, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawVersionComparison]{
			Ref:        afterRef,
			Provenance: provenance,
			Data:       comparison,
		},
	)
	if err != nil {
		return emptyLawVersionComparisonResult(), invalidLawVersionCompareResponse()
	}
	return result, nil
}

func latestCompareRetrievedAt(before, after []model.Provenance) time.Time {
	latest := time.Time{}
	for _, provenance := range before {
		if provenance.RetrievedAt().After(latest) {
			latest = provenance.RetrievedAt()
		}
	}
	for _, provenance := range after {
		if provenance.RetrievedAt().After(latest) {
			latest = provenance.RetrievedAt()
		}
	}
	return latest
}

func emptyLawVersionComparisonResult() model.SourcedResource[model.LawVersionComparison] {
	return model.SourcedResource[model.LawVersionComparison]{}
}
