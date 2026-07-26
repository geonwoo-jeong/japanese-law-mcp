package lawv2

import (
	"slices"
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/continuation"
)

func TestNewProviderBindingsActivatesDescriptorInventory(t *testing.T) {
	t.Parallel()

	manager, err := continuation.NewManager()
	if err != nil {
		t.Fatalf("continuation manager を作成できません: %v", err)
	}
	bindings, err := NewProviderBindings(manager)
	if err != nil {
		t.Fatalf("NewProviderBindings() のエラー = %v", err)
	}
	descriptor := bindings.Descriptor
	if descriptor.ProviderID() != "e-gov-law-api-v2" {
		t.Fatalf("providerId = %q", descriptor.ProviderID())
	}
	if _, ok := bindings.LawSearch.(*LawSearchAdapter); !ok {
		t.Fatalf("law.search binding = %T", bindings.LawSearch)
	}
	if _, ok := bindings.LawContentSearch.(*LawContentSearchAdapter); !ok {
		t.Fatalf("law.content.search binding = %T", bindings.LawContentSearch)
	}
	if _, ok := bindings.LawDocumentRead.(*LawDocumentAdapter); !ok {
		t.Fatalf("law.document.read binding = %T", bindings.LawDocumentRead)
	}
	if _, ok := bindings.LawArticleRead.(*LawArticleAdapter); !ok {
		t.Fatalf("law.article.read binding = %T", bindings.LawArticleRead)
	}

	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("binding registry を作成できません: %v", err)
	}
	registered, exists := registry.Descriptor("e-gov-law-api-v2")
	if !exists || registered.ProviderID() != descriptor.ProviderID() {
		t.Fatalf("登録済み descriptor = %#v, %t", registered, exists)
	}
	if port, exists := registry.LawSearch("e-gov-law-api-v2"); !exists || port == nil {
		t.Fatal("law.search binding に到達できません")
	}
	if port, exists := registry.LawContentSearch("e-gov-law-api-v2"); !exists || port == nil {
		t.Fatal("law.content.search binding に到達できません")
	}
	if port, exists := registry.LawDocumentRead("e-gov-law-api-v2"); !exists || port == nil {
		t.Fatal("law.document.read binding に到達できません")
	}
	if port, exists := registry.LawArticleRead("e-gov-law-api-v2"); !exists || port == nil {
		t.Fatal("law.article.read binding に到達できません")
	}

	capabilities := descriptor.Capabilities()
	ids := make([]string, len(capabilities))
	for index, capability := range capabilities {
		ids[index] = capability.ID()
	}
	want := []string{
		"law.article.read",
		"law.content.search",
		"law.document.read",
		"law.search",
	}
	if !slices.Equal(ids, want) {
		t.Fatalf("descriptor capability = %v, want %v", ids, want)
	}
}

func TestNewProviderBindingsRequiresContinuationManager(t *testing.T) {
	t.Parallel()

	if _, err := NewProviderBindings(nil); err == nil {
		t.Fatal("nil continuation manager を受理しました")
	}
}
