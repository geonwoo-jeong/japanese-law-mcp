package getarticle

import (
	"context"
	"errors"
	"strconv"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestServiceBuildsPrimaryResourceAndProjectsXML(t *testing.T) {
	t.Parallel()

	paragraph := 2
	asOf := mustGetArticleDate(t, "2026-07-26")
	request, err := NewRequest(RequestValues{
		LawID:     " law-1 ",
		Article:   "38_3",
		Paragraph: &paragraph,
		AsOf:      &asOf,
	})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}
	reader := &recordingArticleReader{
		result: mustArticleResource(t, request.Location(), model.LawArticleFormatXML),
	}
	service, err := NewService(reader, mustArticleDescriptor(t), time.Second)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	fragment, err := service.Get(context.Background(), request)
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
	if _, exists := key.VersionID(); exists {
		t.Fatal("公開 facade が versionId を設定しました")
	}
	readAsOf, exists := reader.request.AsOf()
	if !exists || readAsOf.String() != "2026-07-26" {
		t.Fatalf("provider asOf = %q, %t", readAsOf.String(), exists)
	}
	if !articleLocationsEqual(reader.request.Location(), request.Location()) {
		t.Fatalf("provider location = %#v", reader.request.Location())
	}
	if fragment.Format() != model.LawArticleFormatXML ||
		fragment.Content() != "<Paragraph Num=\"2\">本文</Paragraph>" {
		t.Fatalf("fragment = %#v", fragment)
	}
}

func TestServicePreservesDomainErrors(t *testing.T) {
	t.Parallel()

	tests := map[string]error{
		"not found": lawarticleread.ErrNotFound,
		"ambiguous": lawarticleread.ErrAmbiguousLocation,
	}
	for name, providerError := range tests {
		name := name
		providerError := providerError
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service, err := NewService(
				&recordingArticleReader{err: providerError},
				mustArticleDescriptor(t),
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewService() のエラー = %v", err)
			}
			request := mustArticleRequest(t, "1", nil)

			_, err = service.Get(context.Background(), request)
			if !errors.Is(err, providerError) {
				t.Fatalf("Get() のエラー = %v, want %v", err, providerError)
			}
		})
	}
}

func TestServiceAppliesRequestTimeout(t *testing.T) {
	t.Parallel()

	reader := &recordingArticleReader{waitForContext: true}
	service, err := NewService(reader, mustArticleDescriptor(t), 10*time.Millisecond)
	if err != nil {
		t.Fatalf("NewService() のエラー = %v", err)
	}

	_, err = service.Get(context.Background(), mustArticleRequest(t, "1", nil))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Get() のエラー = %v, want deadline exceeded", err)
	}
}

func TestServiceRejectsNonXMLAndMismatchedLocation(t *testing.T) {
	t.Parallel()

	request := mustArticleRequest(t, "1", nil)
	otherLocation, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "2",
	})
	if err != nil {
		t.Fatalf("別の条文位置を作成できません: %v", err)
	}
	tests := map[string]model.SourcedResource[model.LawArticleFragment]{
		"非 XML":  mustArticleResource(t, request.Location(), model.LawArticleFormatText),
		"位置の不一致": mustArticleResource(t, otherLocation, model.LawArticleFormatXML),
	}
	for name, result := range tests {
		name := name
		result := result
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			service, err := NewService(
				&recordingArticleReader{result: result},
				mustArticleDescriptor(t),
				time.Second,
			)
			if err != nil {
				t.Fatalf("NewService() のエラー = %v", err)
			}
			if _, err := service.Get(context.Background(), request); err == nil {
				t.Fatalf("%s の provider 結果を公開しました", name)
			}
		})
	}
}

func TestNewServiceRejectsInvalidDependencies(t *testing.T) {
	t.Parallel()

	if _, err := NewService(nil, mustArticleDescriptor(t), time.Second); err == nil {
		t.Fatal("nil reader を受理しました")
	}
	if _, err := NewService(
		&recordingArticleReader{},
		mustDescriptorWithoutArticleCapability(t),
		time.Second,
	); err == nil {
		t.Fatal("能力を宣言しない provider を受理しました")
	}
	if _, err := NewService(
		&recordingArticleReader{},
		mustArticleDescriptor(t),
		0,
	); err == nil {
		t.Fatal("0 の requestTimeout を受理しました")
	}
}

type recordingArticleReader struct {
	result         model.SourcedResource[model.LawArticleFragment]
	err            error
	request        lawarticleread.Request
	calls          int
	waitForContext bool
}

func (r *recordingArticleReader) Read(
	ctx context.Context,
	request lawarticleread.Request,
) (model.SourcedResource[model.LawArticleFragment], error) {
	r.calls++
	r.request = request
	if r.waitForContext {
		<-ctx.Done()
		return model.SourcedResource[model.LawArticleFragment]{}, ctx.Err()
	}
	return r.result, r.err
}

func mustArticleRequest(
	t *testing.T,
	article string,
	paragraph *int,
) Request {
	t.Helper()

	request, err := NewRequest(RequestValues{
		LawID:     "law-1",
		Article:   article,
		Paragraph: paragraph,
	})
	if err != nil {
		t.Fatalf("Request を作成できません: %v", err)
	}
	return request
}

func mustArticleDescriptor(t *testing.T) model.ProviderDescriptor {
	t.Helper()

	return mustArticleDescriptorWithCapabilities(t, []model.ProviderCapability{
		mustArticleCapability(t, lawarticleread.CapabilityID),
	})
}

func mustDescriptorWithoutArticleCapability(t *testing.T) model.ProviderDescriptor {
	t.Helper()

	return mustArticleDescriptorWithCapabilities(t, []model.ProviderCapability{
		mustArticleCapability(t, "law.document.read"),
	})
}

func mustArticleDescriptorWithCapabilities(
	t *testing.T,
	capabilities []model.ProviderCapability,
) model.ProviderDescriptor {
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
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "primary-provider",
		Source:                 source,
		AdapterContractVersion: "1.0.0",
		VerifiedAt:             mustGetArticleDate(t, "2026-07-26"),
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           capabilities,
	})
	if err != nil {
		t.Fatalf("記述子を作成できません: %v", err)
	}
	return descriptor
}

func mustArticleCapability(
	t *testing.T,
	id string,
) model.ProviderCapability {
	t.Helper()

	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           id,
		MajorVersion: 1,
		Level:        model.CapabilityLevelCore,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("能力を作成できません: %v", err)
	}
	return capability
}

func mustArticleResource(
	t *testing.T,
	location model.LawArticleLocation,
	format string,
) model.SourcedResource[model.LawArticleFragment] {
	t.Helper()

	descriptor := mustArticleDescriptor(t)
	informationSource := descriptor.Source()
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("法令情報源を作成できません: %v", err)
	}
	summary, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		Title:      "テスト法",
		Source:     source,
	})
	if err != nil {
		t.Fatalf("法令概要を作成できません: %v", err)
	}
	citationLocation := articleCitationLocation(location)
	citation, err := model.NewCitation(model.CitationValues{
		Source:     source,
		LawID:      "law-1",
		RevisionID: "law-1_rev-1",
		Location:   citationLocation,
		URL:        "https://example.com/law/law-1/rev-1",
	})
	if err != nil {
		t.Fatalf("出典を作成できません: %v", err)
	}
	content := "<Paragraph Num=\"2\">本文</Paragraph>"
	mediaType := "application/xml"
	if format == model.LawArticleFormatText {
		content = "本文"
		mediaType = "text/plain"
	}
	fragment, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      summary,
		Location: location,
		Format:   format,
		Content:  content,
		Citation: citation,
	})
	if err != nil {
		t.Fatalf("条文 fragment を作成できません: %v", err)
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     informationSource.ID(),
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
		Source:         informationSource,
		ResourceKey:    key,
		URL:            citation.URL(),
		Location:       citationLocation,
		RetrievedAt:    time.Date(2026, 7, 26, 0, 0, 0, 0, time.UTC),
		MediaType:      mediaType,
		Transformation: model.ProvenanceTransformationExtracted,
		MethodID:       "SOT-IF-012",
	})
	if err != nil {
		t.Fatalf("出典経路を作成できません: %v", err)
	}
	result, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawArticleFragment]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       fragment,
		},
	)
	if err != nil {
		t.Fatalf("情報源結果を作成できません: %v", err)
	}
	return result
}

func articleCitationLocation(location model.LawArticleLocation) string {
	value := string(location.Provision()) + ":article=" + location.ArticleNumber()
	if paragraph, exists := location.ParagraphNumber(); exists {
		value += ";paragraph=" + strconv.Itoa(paragraph)
	}
	return value
}

func mustGetArticleDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("日付を作成できません: %v", err)
	}
	return date
}
