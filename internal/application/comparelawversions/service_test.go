package comparelawversions_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/comparelawversions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceProjectsValidatedSourcedComparison(t *testing.T) {
	t.Parallel()

	result := newSourcedComparison(t, "revision-after")
	fake := &compareProviderFake{result: result}
	service, err := comparelawversions.NewService(
		fake,
		newCompareDescriptor(t),
		time.Second,
	)
	if err != nil {
		t.Fatalf("service を構築できません: %v", err)
	}
	request := newPublicCompareRequest(t)
	comparison, err := service.Compare(context.Background(), request)
	if err != nil {
		t.Fatalf("比較できません: %v", err)
	}
	if comparison.After().Law().RevisionID() != "revision-after" {
		t.Fatalf("after revisionId = %q", comparison.After().Law().RevisionID())
	}
	if fake.calls != 1 {
		t.Fatalf("provider calls = %d", fake.calls)
	}
	resource := fake.request.Resource()
	if resource.ProviderID() != "e-gov-law-api-v2" || resource.Key().ResourceID() != "law-1" {
		t.Fatalf("resource = %#v", resource)
	}
	if revisionID, exists := fake.request.Before().RevisionID(); !exists || revisionID != "revision-before" {
		t.Fatalf("before selector = %#v", fake.request.Before())
	}
}

func TestServiceRejectsCapabilityResultWhoseRefDoesNotPointAfterVersion(t *testing.T) {
	t.Parallel()

	fake := &compareProviderFake{result: newSourcedComparison(t, "revision-before")}
	service, err := comparelawversions.NewService(fake, newCompareDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("service を構築できません: %v", err)
	}
	_, err = service.Compare(context.Background(), newPublicCompareRequest(t))
	if !errors.Is(err, comparelawversions.ErrInvalidSourceResponse) {
		t.Fatalf("error = %v", err)
	}
}

func TestServiceRejectsCapabilityResultForAnotherLaw(t *testing.T) {
	t.Parallel()

	fake := &compareProviderFake{
		result: newSourcedComparisonForLaw(t, "law-2", "revision-after"),
	}
	service, err := comparelawversions.NewService(fake, newCompareDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("service を構築できません: %v", err)
	}
	_, err = service.Compare(context.Background(), newPublicCompareRequest(t))
	if !errors.Is(err, comparelawversions.ErrInvalidSourceResponse) {
		t.Fatalf("error = %v", err)
	}
}

func TestServiceRejectsTypedNilProvider(t *testing.T) {
	t.Parallel()

	var fake *compareProviderFake
	if _, err := comparelawversions.NewService(fake, newCompareDescriptor(t), time.Second); err == nil {
		t.Fatal("typed nil provider を受理しました")
	}
}

func TestNewServiceRejectsMissingCapabilityAndInvalidTimeout(t *testing.T) {
	t.Parallel()

	fake := &compareProviderFake{}
	if _, err := comparelawversions.NewService(
		fake,
		newCompareDescriptorWithoutCapabilities(t),
		time.Second,
	); err == nil {
		t.Fatal("law.version.compare@1 のない descriptor を受理しました")
	}
	if _, err := comparelawversions.NewService(
		fake,
		newCompareDescriptor(t),
		0,
	); err == nil {
		t.Fatal("0 秒の requestTimeout を受理しました")
	}
}

func TestServiceRejectsNilContextAndPropagatesProviderError(t *testing.T) {
	t.Parallel()

	providerError := errors.New("provider failure")
	fake := &compareProviderFake{err: providerError}
	service, err := comparelawversions.NewService(fake, newCompareDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("service を構築できません: %v", err)
	}
	if _, err := service.Compare(nilCompareContext(), newPublicCompareRequest(t)); err == nil {
		t.Fatal("nil context を受理しました")
	}
	_, err = service.Compare(context.Background(), newPublicCompareRequest(t))
	if !errors.Is(err, providerError) {
		t.Fatalf("provider error = %v", err)
	}
}

type compareProviderFake struct {
	request lawversioncompare.Request
	result  model.SourcedResource[model.LawVersionComparison]
	err     error
	calls   int
}

func (f *compareProviderFake) Compare(
	_ context.Context,
	request lawversioncompare.Request,
) (model.SourcedResource[model.LawVersionComparison], error) {
	f.calls++
	f.request = request
	return f.result, f.err
}

func newPublicCompareRequest(t *testing.T) comparelawversions.Request {
	t.Helper()
	before, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: "revision-before",
	})
	if err != nil {
		t.Fatalf("before selector を構築できません: %v", err)
	}
	after, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: "revision-after",
	})
	if err != nil {
		t.Fatalf("after selector を構築できません: %v", err)
	}
	request, err := comparelawversions.NewRequest(comparelawversions.RequestValues{
		LawID:  "law-1",
		Before: before,
		After:  after,
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	return request
}

func nilCompareContext() context.Context {
	var ctx context.Context
	return ctx
}

func newSourcedComparison(
	t *testing.T,
	refRevisionID string,
) model.SourcedResource[model.LawVersionComparison] {
	t.Helper()
	return newSourcedComparisonForLaw(t, "law-1", refRevisionID)
}

func newSourcedComparisonForLaw(
	t *testing.T,
	lawID string,
	refRevisionID string,
) model.SourcedResource[model.LawVersionComparison] {
	t.Helper()
	source := newCompareInformationSource(t)
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("legal source を構築できません: %v", err)
	}
	before := newCompareSnapshot(t, legalSource, lawID, "revision-before")
	after := newCompareSnapshot(t, legalSource, lawID, "revision-after")
	comparison, err := model.NewLawVersionComparison(model.LawVersionComparisonValues{
		LawID:              lawID,
		Scope:              model.LawVersionComparisonScopeMainAndOriginalSupplementaryArticles,
		Before:             before,
		After:              after,
		BeforeArticleCount: 1,
		AfterArticleCount:  1,
		UnchangedCount:     1,
	})
	if err != nil {
		t.Fatalf("comparison を構築できません: %v", err)
	}
	beforeKey := newCompareResourceKey(t, lawID, "revision-before")
	afterKey := newCompareResourceKey(t, lawID, "revision-after")
	refKey := newCompareResourceKey(t, lawID, refRevisionID)
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        refKey,
	})
	if err != nil {
		t.Fatalf("ref を構築できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    refKey,
		URL:            "https://laws.e-gov.go.jp/law/" + refRevisionID,
		RetrievedAt:    time.Date(2026, 8, 23, 9, 0, 0, 0, time.UTC),
		MediaType:      "application/xml",
		Transformation: model.ProvenanceTransformationDerived,
		MethodID:       "SOT-IF-060",
		InputKeys:      []model.SourceResourceKey{beforeKey, afterKey},
	})
	if err != nil {
		t.Fatalf("provenance を構築できません: %v", err)
	}
	result, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawVersionComparison]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       comparison,
	})
	if err != nil {
		t.Fatalf("sourced result を構築できません: %v", err)
	}
	return result
}

func newCompareSnapshot(
	t *testing.T,
	source model.LegalSource,
	lawID string,
	revisionID string,
) model.LawVersionSnapshot {
	t.Helper()
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "行政手続法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("law summary を構築できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      lawID,
		RevisionID: revisionID,
		URL:        "https://laws.e-gov.go.jp/law/" + revisionID,
	})
	if err != nil {
		t.Fatalf("citation を構築できません: %v", err)
	}
	snapshot, err := model.NewLawVersionSnapshot(model.LawVersionSnapshotValues{
		Law:      law,
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("snapshot を構築できません: %v", err)
	}
	return snapshot
}

func newCompareDescriptor(t *testing.T) model.ProviderDescriptor {
	t.Helper()
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           lawversioncompare.CapabilityID,
		MajorVersion: lawversioncompare.MajorVersion,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("capability を構築できません: %v", err)
	}
	verifiedAt, err := model.NewDate("2026-08-23")
	if err != nil {
		t.Fatalf("verifiedAt を構築できません: %v", err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 newCompareInformationSource(t),
		AdapterContractVersion: "1.2.0",
		UpstreamSpecVersion:    "2.1.139",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		t.Fatalf("descriptor を構築できません: %v", err)
	}
	return descriptor
}

func newCompareDescriptorWithoutCapabilities(t *testing.T) model.ProviderDescriptor {
	t.Helper()
	verifiedAt, err := model.NewDate("2026-08-23")
	if err != nil {
		t.Fatalf("verifiedAt を構築できません: %v", err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v2",
		Source:                 newCompareInformationSource(t),
		AdapterContractVersion: "1.2.0",
		UpstreamSpecVersion:    "2.1.139",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{},
	})
	if err != nil {
		t.Fatalf("descriptor を構築できません: %v", err)
	}
	return descriptor
}

func newCompareInformationSource(t *testing.T) model.InformationSource {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v2",
		Name:       "e-Gov 法令 API Version 2",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/api/2/",
	})
	if err != nil {
		t.Fatalf("source を構築できません: %v", err)
	}
	return source
}

func newCompareResourceKey(
	t *testing.T,
	lawID string,
	revisionID string,
) model.SourceResourceKey {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   lawID,
		VersionID:    revisionID,
	})
	if err != nil {
		t.Fatalf("resource key を構築できません: %v", err)
	}
	return key
}
