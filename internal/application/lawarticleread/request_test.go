package lawarticleread_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequest(t *testing.T) {
	t.Parallel()

	location := newArticleLocation(t)
	got, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: newArticleLawResourceRef(t, "325AC0000000105", ""),
		AsOf:     newArticleDatePointer(t, "2026-01-01"),
		Location: location,
	})
	if err != nil {
		t.Fatalf("SOT-IF-025: NewRequest() のエラー = %v", err)
	}
	if got.Resource().Key().ResourceID() != "325AC0000000105" {
		t.Fatalf("SOT-IF-025: Resource() = %#v", got.Resource())
	}
	if asOf, exists := got.AsOf(); !exists || asOf.String() != "2026-01-01" {
		t.Fatalf("SOT-IF-025: AsOf() = %q, %t", asOf.String(), exists)
	}
	if got.Location() != location {
		t.Fatalf("SOT-IF-025: Location() = %#v", got.Location())
	}
	if err := got.Validate(); err != nil {
		t.Fatalf("SOT-IF-025: Validate() のエラー = %v", err)
	}

	withoutAsOf, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: newArticleLawResourceRef(t, "325AC0000000105", "revision-1"),
		Location: location,
	})
	if err != nil {
		t.Fatalf("SOT-IF-025: versionId だけの NewRequest() のエラー = %v", err)
	}
	if _, exists := withoutAsOf.AsOf(); exists {
		t.Fatal("SOT-IF-025: 省略した asOf が存在する")
	}
}

func TestRequestDefersProviderSourceAdoptionToRegistry(t *testing.T) {
	t.Parallel()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "independent-law-source",
		ResourceType: "law",
		ResourceID:   "law-1",
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceKey を作成できない: %v", err)
	}
	resource, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "law-provider",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できない: %v", err)
	}
	if _, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
		Resource: resource,
		Location: newArticleLocation(t),
	}); err != nil {
		t.Fatalf("SOT-IF-025/SOT-ARCH-022: 構造的に有効な resource を拒否した: %v", err)
	}
}

func TestRequestOpaqueIdentifierBoundary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		resourceID string
		versionID  string
		wantError  bool
	}{
		{name: "不透明な値を保持", resourceID: "001-AbC/%2F:章", versionID: "Rev-001/Ａ"},
		{name: "resourceId が 256 byte", resourceID: strings.Repeat("a", 256)},
		{name: "resourceId が 257 byte", resourceID: strings.Repeat("a", 257), wantError: true},
		{name: "versionId が 512 byte", resourceID: "law", versionID: strings.Repeat("a", 512)},
		{name: "versionId が 513 byte", resourceID: "law", versionID: strings.Repeat("a", 513), wantError: true},
		{name: "resourceId の先頭に U+0020", resourceID: " law", wantError: true},
		{name: "resourceId の末尾に U+0020", resourceID: "law ", wantError: true},
		{name: "resourceId に制御文字", resourceID: "law\nid", wantError: true},
		{name: "versionId の先頭に U+0020", resourceID: "law", versionID: " rev", wantError: true},
		{name: "versionId に DEL", resourceID: "law", versionID: "rev\x7f", wantError: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			resource := newArticleLawResourceRef(t, test.resourceID, test.versionID)
			got, err := lawarticleread.NewRequest(lawarticleread.RequestValues{
				Resource: resource,
				Location: newArticleLocation(t),
			})
			if (err != nil) != test.wantError {
				t.Fatalf("SOT-IF-025: NewRequest() のエラー = %v、期待値 = %t", err, test.wantError)
			}
			if err == nil && got.Resource().Key().ResourceID() != test.resourceID {
				t.Fatalf("SOT-IF-025: resourceId が変更された: %q", got.Resource().Key().ResourceID())
			}
		})
	}
}

func TestRequestRejectsInvalidValues(t *testing.T) {
	t.Parallel()

	date := newArticleDatePointer(t, "2026-01-01")
	zeroDate := model.Date{}
	validLocation := newArticleLocation(t)
	tests := map[string]lawarticleread.RequestValues{
		"resource の欠落": {
			Location: validLocation,
		},
		"resourceType が law ではない": {
			Resource: newArticleResourceRef(t, "article"),
			Location: validLocation,
		},
		"versionId と asOf の同時指定": {
			Resource: newArticleLawResourceRef(t, "law", "revision"),
			AsOf:     date,
			Location: validLocation,
		},
		"asOf のゼロ値": {
			Resource: newArticleLawResourceRef(t, "law", ""),
			AsOf:     &zeroDate,
			Location: validLocation,
		},
		"location の欠落": {
			Resource: newArticleLawResourceRef(t, "law", ""),
		},
	}
	for name, values := range tests {
		name, values := name, values
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, err := lawarticleread.NewRequest(values); err == nil {
				t.Fatalf("SOT-IF-025: NewRequest(%#v) が成功した", values)
			}
		})
	}
}

func newArticleLawResourceRef(
	t *testing.T,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	return ref
}

func newArticleResourceRef(t *testing.T, resourceType string) model.SourceResourceRef {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: resourceType,
		ResourceID:   "law",
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	return ref
}

func newArticleLocation(t *testing.T) model.LawArticleLocation {
	t.Helper()

	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:     model.LawArticleProvisionMain,
		ArticleNumber: "1",
	})
	if err != nil {
		t.Fatalf("LawArticleLocation を作成できない: %v", err)
	}
	return location
}

func newArticleDatePointer(t *testing.T, value string) *model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できない: %v", err)
	}
	return &date
}
