package hanreipdf

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application"
)

func TestNewProviderBindingsActivatesJudicialCitationExtractInventory(t *testing.T) {
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
	if !exists || len(descriptor.Capabilities()) != 1 {
		t.Fatalf("SOT-IF-070/SOT-IF-074: descriptor = %#v, %t", descriptor, exists)
	}
	extract, exists := registry.JudicialCaseCitationExtract(providerID)
	if !exists || extract == nil {
		t.Fatal("SOT-IF-068/SOT-IF-074: 引用抽出 binding がありません")
	}
	if _, exists := registry.JudicialCitingCandidateSearch(providerID); exists {
		t.Fatal("SOT-IF-070: PDF provider が候補検索を宣言しました")
	}
	if _, exists := registry.JudicialDecisionRead(providerID); exists {
		t.Fatal("SOT-IF-070: PDF provider が裁判例詳細取得を宣言しました")
	}
}
