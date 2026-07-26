package lawdocumentread_test

import (
	"testing"

	"github.com/japanese-law-mcp/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/japanese-law-mcp/japanese-law-mcp/internal/model"
)

func TestRequest(t *testing.T) {
	got, err := lawdocumentread.NewRequest(lawdocumentread.RequestValues{
		Resource: newLawResourceRef(t, "325AC0000000105", ""),
		AsOf:     newDatePointer(t, "2026-01-01"),
	})
	if err != nil {
		t.Fatalf("SOT-IF-024: NewRequest() のエラー = %v", err)
	}
	asOf, exists := got.AsOf()
	if !exists || asOf.String() != "2026-01-01" {
		t.Fatalf("SOT-IF-024: asOf = %q, %t", asOf.String(), exists)
	}
}

func TestRequestRejectsInvalidValues(t *testing.T) {
	date := newDatePointer(t, "2026-01-01")
	testCases := map[string]lawdocumentread.RequestValues{
		"wrong_resource_type": {
			Resource: newResourceRef(t, "law-api-adapter-v2", "official-law-source", "article", "id", ""),
		},
		"version_and_asof": {
			Resource: newLawResourceRef(t, "325AC0000000105", "325AC0000000105_rev1"),
			AsOf:     date,
		},
		"leading_space_resource_id": {
			Resource: newLawResourceRef(t, " law", ""),
		},
	}
	for name, values := range testCases {
		t.Run(name, func(t *testing.T) {
			if _, err := lawdocumentread.NewRequest(values); err == nil {
				t.Fatalf("SOT-IF-024: NewRequest(%#v) が成功した", values)
			}
		})
	}
}

func newLawResourceRef(
	t *testing.T,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	return newResourceRef(
		t,
		"e-gov-law-api-v2",
		"e-gov-law-api-v2",
		"law",
		resourceID,
		versionID,
	)
}

func newResourceRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("SourceResourceKey を作成できない: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できない: %v", err)
	}
	return ref
}

func newDatePointer(t *testing.T, value string) *model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("date を作成できない: %v", err)
	}
	return &date
}
