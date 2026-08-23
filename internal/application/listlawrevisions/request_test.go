package listlawrevisions

import "testing"

func TestRequestUsesCommonLawIdentifierRules(t *testing.T) {
	t.Parallel()

	request, err := NewRequest(RequestValues{LawIDOrNumber: " 503AC0000000036 "})
	if err != nil {
		t.Fatalf("有効な Request を拒否しました: %v", err)
	}
	if request.LawIDOrNumber() != "503AC0000000036" {
		t.Fatalf("lawIdOrNumber = %q", request.LawIDOrNumber())
	}
	if _, err := NewRequest(RequestValues{}); err == nil {
		t.Fatal("空の lawIdOrNumber を拒否しませんでした")
	}
}
