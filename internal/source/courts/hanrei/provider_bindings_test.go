package hanrei

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

func TestNewProviderBindingsActivatesJudicialDecisionInventory(t *testing.T) {
	t.Parallel()

	bindings, err := NewProviderBindings()
	if err != nil {
		t.Fatalf("SOT-IF-074: NewProviderBindings() のエラー = %v", err)
	}
	registry, err := application.NewProviderBindingRegistry(
		[]application.ProviderBindings{bindings},
	)
	if err != nil {
		t.Fatalf("SOT-IF-074: registry のエラー = %v", err)
	}
	descriptor, exists := registry.Descriptor(providerID)
	if !exists || len(descriptor.Capabilities()) != 3 {
		t.Fatalf("SOT-IF-072/SOT-IF-074: descriptor = %#v, %t", descriptor, exists)
	}
	candidateSearch, candidateSearchExists := registry.JudicialCitingCandidateSearch(providerID)
	search, searchExists := registry.JudicialDecisionSearch(providerID)
	read, readExists := registry.JudicialDecisionRead(providerID)
	if !candidateSearchExists || candidateSearch == nil ||
		!searchExists || search == nil || !readExists || read == nil {
		t.Fatal("SOT-IF-074: 裁判例の三つの binding がありません")
	}
	if _, exists := registry.LawSearch(providerID); exists {
		t.Fatal("SOT-IF-074: 裁判例 provider が law.search を宣言しました")
	}

	candidateAdapter, candidateOK := candidateSearch.(*JudicialCitingCandidateSearchAdapter)
	searchAdapter, searchOK := search.(*JudicialDecisionSearchAdapter)
	readAdapter, readOK := read.(*JudicialDecisionReadAdapter)
	if !candidateOK || !searchOK || !readOK ||
		candidateAdapter.dependencies.gate != searchAdapter.dependencies.gate ||
		searchAdapter.dependencies.gate != readAdapter.dependencies.gate {
		t.Fatal("SOT-IF-072: 三つの adapter が同じ同時実行枠を共有していません")
	}
}
