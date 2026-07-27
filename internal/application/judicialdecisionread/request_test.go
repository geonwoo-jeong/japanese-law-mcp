package judicialdecisionread_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestRequestPreservesSearchResourceReference(t *testing.T) {
	t.Parallel()

	ref := newJudicialDecisionRef(t, "courts-hanrei-html", "courts-hanrei", "95570", "")
	request, err := judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{Ref: ref},
	)
	if err != nil {
		t.Fatalf("SOT-IF-042: NewRequest() のエラー = %v", err)
	}
	if request.Ref() != ref {
		t.Fatalf("SOT-IF-042: Ref() = %#v、期待値 = %#v", request.Ref(), ref)
	}
	if err := request.Validate(); err != nil {
		t.Fatalf("SOT-IF-042: Validate() のエラー = %v", err)
	}
}

func TestRequestRejectsInvalidJudicialDecisionReference(t *testing.T) {
	t.Parallel()

	tests := map[string]model.SourceResourceRef{
		"参照の欠落": {},
		"異なる resourceType": newResourceRef(
			t,
			"courts-hanrei-html",
			"courts-hanrei",
			"law",
			"95570",
			"",
		),
		"version の指定": newJudicialDecisionRef(
			t,
			"courts-hanrei-html",
			"courts-hanrei",
			"95570",
			"published",
		),
	}
	for name, ref := range tests {
		name, ref := name, ref
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			_, err := judicialdecisionread.NewRequest(
				judicialdecisionread.RequestValues{Ref: ref},
			)
			assertInvalidRefArgument(t, err)
		})
	}
}

func TestRequestRejectsZeroValueAndDirectJSONRestore(t *testing.T) {
	t.Parallel()

	var request judicialdecisionread.Request
	assertInvalidRefArgument(t, request.Validate())
	if err := json.Unmarshal([]byte(`{"ref":{}}`), &request); err == nil {
		t.Fatal("SOT-IF-042: Request を JSON から直接復元できました")
	}
}

func assertInvalidRefArgument(t *testing.T, err error) {
	t.Helper()

	var argumentError judicialdecisionread.ArgumentError
	if !errors.As(err, &argumentError) {
		t.Fatalf("SOT-IF-042: ArgumentError ではありません: %T %v", err, err)
	}
	if argumentError.Code() != model.ErrorCodeInvalidArgument {
		t.Fatalf("SOT-IF-042: Code() = %q", argumentError.Code())
	}
	if argumentError.Field() != "ref" {
		t.Fatalf("SOT-IF-042: Field() = %q", argumentError.Field())
	}
}

func newJudicialDecisionRef(
	t *testing.T,
	providerID string,
	sourceID string,
	resourceID string,
	versionID string,
) model.SourceResourceRef {
	t.Helper()

	return newResourceRef(
		t,
		providerID,
		sourceID,
		"judicial-decision",
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
		t.Fatalf("SourceResourceKey を作成できません: %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("SourceResourceRef を作成できません: %v", err)
	}
	return ref
}
