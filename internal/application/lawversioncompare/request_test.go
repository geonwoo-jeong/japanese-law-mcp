package lawversioncompare_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawversioncompare"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestSelectorRequiresExactlyOneBoundedSelector(t *testing.T) {
	t.Parallel()

	date := mustCompareDate(t, "2024-04-01")
	valid := []lawversioncompare.SelectorValues{
		{RevisionID: "revision-1"},
		{AsOf: &date},
	}
	for _, values := range valid {
		if _, err := lawversioncompare.NewSelector(values); err != nil {
			t.Fatalf("有効な selector を拒否しました: %v", err)
		}
	}

	invalid := []lawversioncompare.SelectorValues{
		{},
		{RevisionID: "revision-1", AsOf: &date},
		{RevisionID: " revision-1"},
		{RevisionID: "revision-1\n"},
		{RevisionID: strings.Repeat("a", 513)},
	}
	for _, values := range invalid {
		if _, err := lawversioncompare.NewSelector(values); err == nil {
			t.Fatalf("不正な selector を受理しました: %#v", values)
		}
	}
}

func TestRequestKeepsNestedBeforeAndAfterAndRejectsVersionedResource(t *testing.T) {
	t.Parallel()

	before, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		RevisionID: "revision-before",
	})
	if err != nil {
		t.Fatalf("before selector を構築できません: %v", err)
	}
	afterDate := mustCompareDate(t, "2025-01-01")
	after, err := lawversioncompare.NewSelector(lawversioncompare.SelectorValues{
		AsOf: &afterDate,
	})
	if err != nil {
		t.Fatalf("after selector を構築できません: %v", err)
	}
	request, err := lawversioncompare.NewRequest(lawversioncompare.RequestValues{
		Resource: newCompareResourceRef(t, ""),
		Before:   before,
		After:    after,
	})
	if err != nil {
		t.Fatalf("request を構築できません: %v", err)
	}
	if revisionID, exists := request.Before().RevisionID(); !exists || revisionID != "revision-before" {
		t.Fatalf("before = %#v", request.Before())
	}
	if asOf, exists := request.After().AsOf(); !exists || asOf.String() != "2025-01-01" {
		t.Fatalf("after = %#v", request.After())
	}

	if _, err := lawversioncompare.NewRequest(lawversioncompare.RequestValues{
		Resource: newCompareResourceRef(t, "already-selected"),
		Before:   before,
		After:    after,
	}); err == nil {
		t.Fatal("resource.key.versionId を受理しました")
	}
}

func newCompareResourceRef(t *testing.T, versionID string) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "e-gov-law-api-v2",
		ResourceType: "law",
		ResourceID:   "law-1",
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("resource key を構築できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "e-gov-law-api-v2",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("resource ref を構築できません: %v", err)
	}
	return ref
}

func mustCompareDate(t *testing.T, value string) model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("日付を構築できません: %v", err)
	}
	return date
}
