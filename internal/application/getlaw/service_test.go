package getlaw

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceBuildsPrimaryResourceAndProjectsXML(t *testing.T) {
	t.Parallel()

	asOf := mustRequestDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{LawID: " law-1 ", AsOf: &asOf})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}
	providerResult := mustServiceResource(t, model.LawDocumentFormatXML, &asOf)
	reader := &recordingDocumentReader{result: providerResult}
	service, err := NewService(reader, mustServiceDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	document, err := service.Get(context.Background(), request)
	if err != nil {
		t.Fatalf("Get() のエラー = %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("provider 呼出し回数 = %d", reader.calls)
	}
	resource := reader.request.Resource()
	if resource.ProviderID() != "primary-provider" {
		t.Fatalf("providerId = %q", resource.ProviderID())
	}
	key := resource.Key()
	if key.SourceID() != "primary-source" ||
		key.ResourceType() != "law" ||
		key.ResourceID() != "law-1" {
		t.Fatalf("resource key = %#v", key)
	}
	readAsOf, exists := reader.request.AsOf()
	if !exists || readAsOf.String() != "2026-07-26" {
		t.Fatalf("provider asOf = %q, %v", readAsOf.String(), exists)
	}
	if document.Format() != model.LawDocumentFormatXML ||
		document.Content() != "<Law>本文</Law>" {
		t.Fatalf("LawDocument = %#v", document)
	}
}

func TestServicePreservesNotFound(t *testing.T) {
	t.Parallel()

	reader := &recordingDocumentReader{err: lawdocumentread.ErrNotFound}
	service, err := NewService(reader, mustServiceDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{LawID: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}

	_, err = service.Get(context.Background(), request)
	if !errors.Is(err, lawdocumentread.ErrNotFound) {
		t.Fatalf("Get() のエラー = %v", err)
	}
}

func TestServiceAppliesRequestTimeout(t *testing.T) {
	t.Parallel()

	reader := &recordingDocumentReader{waitForContext: true}
	service, err := NewService(reader, mustServiceDescriptor(t), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{LawID: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}

	_, err = service.Get(context.Background(), request)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Get() のエラー = %v, want deadline exceeded", err)
	}
}

func TestServiceRejectsNonXMLRepresentation(t *testing.T) {
	t.Parallel()

	reader := &recordingDocumentReader{
		result: mustServiceResource(t, model.LawDocumentFormatText, nil),
	}
	service, err := NewService(reader, mustServiceDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}
	request, err := NewRequest(RequestValues{LawID: "law-1"})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}

	if _, err := service.Get(context.Background(), request); err == nil {
		t.Fatal("Get() が text 表現を LawDocument として返しました")
	}
}

type recordingDocumentReader struct {
	result         model.SourcedResource[model.LawDocumentRepresentation]
	err            error
	request        lawdocumentread.Request
	calls          int
	waitForContext bool
}

func (r *recordingDocumentReader) Read(
	ctx context.Context,
	request lawdocumentread.Request,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	r.calls++
	r.request = request
	if r.waitForContext {
		<-ctx.Done()
		return model.SourcedResource[model.LawDocumentRepresentation]{}, ctx.Err()
	}
	return r.result, r.err
}

func mustServiceDescriptor(t *testing.T) model.ProviderDescriptor {
	t.Helper()

	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "primary-source",
		Name:       "一次情報源",
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://example.com/",
	})
	if err != nil {
		t.Fatalf("情報源を作成できません: %v", err)
	}
	verifiedAt := mustRequestDate(t, "2026-07-26")
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           lawdocumentread.CapabilityID,
		MajorVersion: lawdocumentread.MajorVersion,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("能力を作成できません: %v", err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "primary-provider",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		t.Fatalf("記述子を作成できません: %v", err)
	}
	return descriptor
}

func mustServiceResource(
	t *testing.T,
	format string,
	asOf *model.Date,
) model.SourcedResource[model.LawDocumentRepresentation] {
	t.Helper()

	descriptor := mustServiceDescriptor(t)
	source := descriptor.Source()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		t.Fatalf("法令情報源を作成できません: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		Title:      "テスト法",
		Source:     legalSource,
	})
	if err != nil {
		t.Fatalf("法令概要を作成できません: %v", err)
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		URL:        "https://example.com/law/law-1/rev-1",
	})
	if err != nil {
		t.Fatalf("出典を作成できません: %v", err)
	}
	content := "<Law>本文</Law>"
	if format == model.LawDocumentFormatText {
		content = "本文"
	}
	representation, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law:      summary,
			AsOf:     asOf,
			Format:   format,
			Content:  content,
			Citation: citation,
		},
	)
	if err != nil {
		t.Fatalf("本文表現を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source.ID(),
		ResourceType: "law",
		ResourceID:   "law-1",
		VersionID:    "law-1_rev-1",
	})
	if err != nil {
		t.Fatalf("資源キーを作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: descriptor.ProviderID(),
		Key:        key,
	})
	if err != nil {
		t.Fatalf("資源参照を作成できません: %v", err)
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            citation.URL(),
		RetrievedAt:    time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
		MediaType:      "application/xml",
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-011",
	})
	if err != nil {
		t.Fatalf("出典経路を作成できません: %v", err)
	}
	resource, err := model.NewSourcedResource(model.SourcedResourceValues[model.LawDocumentRepresentation]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       representation,
	})
	if err != nil {
		t.Fatalf("情報源結果を作成できません: %v", err)
	}
	return resource
}
