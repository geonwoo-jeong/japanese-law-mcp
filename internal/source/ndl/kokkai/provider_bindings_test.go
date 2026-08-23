package kokkai

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

func TestNewProviderBindingsActivatesParliamentSpeechInventory(t *testing.T) {
	t.Parallel()

	bindings, err := NewProviderBindings()
	if err != nil {
		t.Fatalf("SOT-IF-065: NewProviderBindings() のエラー = %v", err)
	}
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-IF-065: registry のエラー = %v", err)
	}
	descriptor, exists := registry.Descriptor(providerID)
	if !exists || len(descriptor.Capabilities()) != 1 {
		t.Fatalf("SOT-IF-063/065: descriptor = %#v, %t", descriptor, exists)
	}
	search, searchExists := registry.ParliamentSpeechSearch(providerID)
	if !searchExists || search == nil {
		t.Fatal("SOT-IF-065: 国会発言検索 binding がありません")
	}
	if _, exists := registry.LawSearch(providerID); exists {
		t.Fatal("SOT-IF-065: 国会発言 provider が law.search を宣言しました")
	}
}
