package hanrei

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

func TestNewProviderBindingsActivatesJudicialDecisionInventory(t *testing.T) {
	t.Parallel()

	bindings, err := NewProviderBindings()
	if err != nil {
		t.Fatalf("SOT-IF-046: NewProviderBindings() のエラー = %v", err)
	}
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-IF-046: registry のエラー = %v", err)
	}
	descriptor, exists := registry.Descriptor(providerID)
	if !exists || len(descriptor.Capabilities()) != 2 {
		t.Fatalf("SOT-IF-043/SOT-IF-046: descriptor = %#v, %t", descriptor, exists)
	}
	search, searchExists := registry.JudicialDecisionSearch(providerID)
	read, readExists := registry.JudicialDecisionRead(providerID)
	if !searchExists || search == nil || !readExists || read == nil {
		t.Fatal("SOT-IF-046: 裁判例の二つの binding がありません")
	}
	if _, exists := registry.LawSearch(providerID); exists {
		t.Fatal("SOT-IF-046: 裁判例 provider が law.search を宣言しました")
	}

	searchAdapter, searchOK := search.(*JudicialDecisionSearchAdapter)
	readAdapter, readOK := read.(*JudicialDecisionReadAdapter)
	if !searchOK || !readOK ||
		searchAdapter.dependencies.gate != readAdapter.dependencies.gate {
		t.Fatal("SOT-IF-043: search と read が同じ同時実行枠を共有していません")
	}
}
