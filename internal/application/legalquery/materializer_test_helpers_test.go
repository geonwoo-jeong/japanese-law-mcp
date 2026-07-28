package legalquery

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type materializerTestBinding struct {
	providerID             string
	sourceID               string
	capabilityID           string
	capabilityMajorVersion int
}

func (b materializerTestBinding) ProviderID() string {
	return b.providerID
}

func (b materializerTestBinding) SourceID() string {
	return b.sourceID
}

func (b materializerTestBinding) CapabilityID() string {
	return b.capabilityID
}

func (b materializerTestBinding) CapabilityMajorVersion() int {
	return b.capabilityMajorVersion
}

type materializerTypedNilBinding struct{}

func (*materializerTypedNilBinding) ProviderID() string {
	panic("typed nil binding を呼び出してはなりません")
}

func (*materializerTypedNilBinding) SourceID() string {
	panic("typed nil binding を呼び出してはなりません")
}

func (*materializerTypedNilBinding) CapabilityID() string {
	panic("typed nil binding を呼び出してはなりません")
}

func (*materializerTypedNilBinding) CapabilityMajorVersion() int {
	panic("typed nil binding を呼び出してはなりません")
}

func materializerCoreBinding(
	capabilityID string,
	majorVersion int,
) materializerTestBinding {
	return materializerTestBinding{
		providerID:             "e-gov-law-api-v2",
		sourceID:               "e-gov-law-api-v2",
		capabilityID:           capabilityID,
		capabilityMajorVersion: majorVersion,
	}
}

func materializerJudicialBinding(
	capabilityID string,
	majorVersion int,
) materializerTestBinding {
	return materializerTestBinding{
		providerID:             "courts-hanrei-html",
		sourceID:               "courts-hanrei",
		capabilityID:           capabilityID,
		capabilityMajorVersion: majorVersion,
	}
}

func materializerCollectionBudget(limit int) LegalQueryStepBudget {
	effectiveLimit := limit
	return LegalQueryStepBudget{
		candidateID:    "candidate-materializer",
		stepID:         "step-materializer",
		reservedItems:  0,
		effectiveLimit: &effectiveLimit,
	}
}

func materializerReadBudget() LegalQueryStepBudget {
	return LegalQueryStepBudget{
		candidateID:   "candidate-materializer",
		stepID:        "step-materializer",
		reservedItems: 1,
	}
}

func assertMaterializerDate(
	t *testing.T,
	getter func() (model.Date, bool),
	want string,
) {
	t.Helper()
	got, exists := getter()
	if !exists || got.String() != want {
		t.Fatalf("SOT-ARCH-026: date = %q, %t", got.String(), exists)
	}
}

func assertMaterializerNoContinuation(
	t *testing.T,
	getter func() (string, bool),
) {
	t.Helper()
	if token, exists := getter(); exists || token != "" {
		t.Fatalf("SOT-ARCH-026: continuationToken = %q, %t", token, exists)
	}
}

func assertMaterializerTerms(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("SOT-ARCH-026: terms = %#v", got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("SOT-ARCH-026: terms = %#v", got)
		}
	}
}

func assertMaterializerLawRef(
	t *testing.T,
	ref model.SourceResourceRef,
	providerID string,
	sourceID string,
	resourceID string,
	versionID string,
) {
	t.Helper()
	key := ref.Key()
	gotVersionID, versionExists := key.VersionID()
	wantVersionExists := versionID != ""
	if ref.ProviderID() != providerID ||
		key.SourceID() != sourceID ||
		key.ResourceType() != "law" ||
		key.ResourceID() != resourceID ||
		versionExists != wantVersionExists ||
		versionExists && gotVersionID != versionID {
		t.Fatalf("SOT-ARCH-026: ref = %#v", ref)
	}
}

func assertMaterializerLocation(
	t *testing.T,
	got model.LawArticleLocation,
	want model.LawArticleLocation,
) {
	t.Helper()
	gotParagraph, gotParagraphExists := got.ParagraphNumber()
	wantParagraph, wantParagraphExists := want.ParagraphNumber()
	if got.Provision() != want.Provision() ||
		got.ArticleNumber() != want.ArticleNumber() ||
		gotParagraphExists != wantParagraphExists ||
		gotParagraphExists && gotParagraph != wantParagraph {
		t.Fatalf("SOT-ARCH-026: location = %#v", got)
	}
}
