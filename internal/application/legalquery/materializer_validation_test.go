package legalquery

import "testing"

func TestRequestMaterializersExposeStartupValidation(t *testing.T) {
	t.Parallel()

	if err := NewCoreMaterializer().Validate(); err != nil {
		t.Fatalf("SOT-ARCH-026: CoreMaterializer が有効ではありません: %v", err)
	}
	if err := NewJudicialCasesMaterializer().Validate(); err != nil {
		t.Fatalf("SOT-ARCH-026: JudicialCasesMaterializer が有効ではありません: %v", err)
	}
	if err := (CoreMaterializer{}).Validate(); err == nil {
		t.Fatal("SOT-ARCH-026: zero-value CoreMaterializer を受理しました")
	}
	if err := (JudicialCasesMaterializer{}).Validate(); err == nil {
		t.Fatal("SOT-ARCH-026: zero-value JudicialCasesMaterializer を受理しました")
	}
}
