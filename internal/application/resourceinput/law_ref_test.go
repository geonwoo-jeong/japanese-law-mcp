package resourceinput_test

import (
	"strings"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/resourceinput"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestValidateLawRefAcceptsCapabilityCompatibleOpaqueIdentifiers(t *testing.T) {
	t.Parallel()

	ref := newLawRef(t, "provider-a", "source-a", "001-AbC/%2F:章", "Rev-001/Ａ")
	if err := resourceinput.ValidateLawRef("resource", ref); err != nil {
		t.Fatalf("SOT-IF-024/025: 有効な law ref を拒否しました: %v", err)
	}
}

func TestValidateLawRefRejectsStructuralAndIdentifierViolations(t *testing.T) {
	t.Parallel()

	tests := map[string]model.SourceResourceRef{
		"ゼロ値": {},
		"異なる resource type": newRef(
			t,
			"provider-a",
			"source-a",
			"judicial-decision",
			"decision-1",
			"",
		),
		"長すぎる resource ID": newLawRef(
			t,
			"provider-a",
			"source-a",
			strings.Repeat("a", 257),
			"",
		),
		"長すぎる version ID": newLawRef(
			t,
			"provider-a",
			"source-a",
			"law-1",
			strings.Repeat("a", 513),
		),
		"先頭空白": newLawRef(t, "provider-a", "source-a", " law-1", ""),
		"制御文字": newLawRef(t, "provider-a", "source-a", "law\n1", ""),
	}
	for name, ref := range tests {
		name := name
		ref := ref
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := resourceinput.ValidateLawRef("resource", ref); err == nil {
				t.Fatal("SOT-IF-024/025: 不正な law ref を受理しました")
			}
		})
	}
}

func TestValidateLawRefDefersProviderSourceAdoptionToRegistry(t *testing.T) {
	t.Parallel()

	ref := newLawRef(t, "provider-a", "independent-source", "law-1", "")
	if err := resourceinput.ValidateLawRef("ref", ref); err != nil {
		t.Fatalf("SOT-ARCH-022: 構造的に有効な ref を拒否しました: %v", err)
	}
}

func newLawRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()
	return newRef(t, providerID, sourceID, "law", resourceID, versionID)
}

func newRef(
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
		t.Fatalf("試験用 SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("試験用 SourceResourceRef を作成できません: %v", err)
	}
	return ref
}
