package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestLogicalInputSignatureSeparatesPipeDelimitedQueryAndAsOf(t *testing.T) {
	t.Parallel()

	asOf, err := model.NewDate("2026-07-31")
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 日付を作成できません: %v", err)
	}
	left, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: "永住|2026-07-31",
		},
	)
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 左辺の logical input を作成できません: %v", err)
	}
	right, err := legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{
			Query: "永住",
			AsOf:  &asOf,
		},
	)
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 右辺の logical input を作成できません: %v", err)
	}

	assertDifferentLogicalInputSignatures(t, left, right)
}

func TestLogicalInputSignatureSeparatesPipeDelimitedResourceAndVersion(t *testing.T) {
	t.Parallel()

	left := mustSignatureLawReadIntent(t, "law|revision", "v1")
	right := mustSignatureLawReadIntent(t, "law", "revision|v1")

	assertDifferentLogicalInputSignatures(t, left, right)
}

func assertDifferentLogicalInputSignatures(
	t *testing.T,
	left legalquery.LogicalInput,
	right legalquery.LogicalInput,
) {
	t.Helper()

	leftSignature, err := logicalInputSignature(left)
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 左辺の意味署名を作成できません: %v", err)
	}
	rightSignature, err := logicalInputSignature(right)
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 右辺の意味署名を作成できません: %v", err)
	}
	if leftSignature == rightSignature {
		t.Fatalf(
			"core-evidence-mapping-fail-closed: 異なる logical input が同じ意味署名 %q になりました",
			leftSignature,
		)
	}
}

func mustSignatureLawReadIntent(
	t *testing.T,
	resourceID string,
	versionID string,
) legalquery.LawReadIntentV1 {
	t.Helper()

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "egov-laws",
		ResourceType: "law",
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 資源 key を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "egov",
		Key:        key,
	})
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: 資源 ref を作成できません: %v", err)
	}
	intent, err := legalquery.NewLawReadIntentV1(legalquery.LawReadIntentV1Values{
		Ref: &ref,
	})
	if err != nil {
		t.Fatalf("core-evidence-mapping-fail-closed: law read input を作成できません: %v", err)
	}
	return intent
}
