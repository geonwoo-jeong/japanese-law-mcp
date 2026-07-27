package hanrei

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestBuildReadHTTPRequestBuildsCanonicalURL(t *testing.T) {
	t.Parallel()
	request := mustReadRequest(t, providerID, sourceID, "95878/detail3")
	httpRequest, detailURL, decisionID, categoryNumber, err := buildReadHTTPRequest(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatal(err)
	}
	if httpRequest.Method != http.MethodGet {
		t.Errorf("method = %q", httpRequest.Method)
	}
	if detailURL != "https://www.courts.go.jp/hanrei/95878/detail3/index.html" ||
		httpRequest.URL.String() != detailURL ||
		decisionID != "95878" ||
		categoryNumber != "3" {
		t.Errorf("detailURL = %q, decisionID = %q, categoryNumber = %q", detailURL, decisionID, categoryNumber)
	}
}

func TestValidateReadRefRejectsMismatches(t *testing.T) {
	t.Parallel()
	versionID := "v1"
	cases := []struct {
		name string
		ref  model.SourceResourceRef
	}{
		{
			name: "provider",
			ref:  mustSourceRef(t, "other", sourceID, "judicial-decision", "95878/detail3", nil),
		},
		{
			name: "source",
			ref:  mustSourceRef(t, providerID, "other", "judicial-decision", "95878/detail3", nil),
		},
		{
			name: "version",
			ref:  mustSourceRef(t, providerID, sourceID, "judicial-decision", "95878/detail3", &versionID),
		},
		{
			name: "resourceId",
			ref:  mustSourceRef(t, providerID, sourceID, "judicial-decision", "bad", nil),
		},
		{
			name: "resourceType",
			ref:  mustSourceRef(t, providerID, sourceID, "other", "95878/detail3", nil),
		},
	}
	for _, testCase := range cases {
		testCase := testCase
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			request, err := judicialdecisionread.NewRequest(judicialdecisionread.RequestValues{Ref: testCase.ref})
			if err != nil {
				if testCase.name == "resourceType" || testCase.name == "version" {
					assertReadInvalidArgument(t, err)
					return
				}
				t.Fatal(err)
			}
			_, _, err = validateReadRef(request)
			var argumentError judicialdecisionread.ArgumentError
			if !errors.As(err, &argumentError) || argumentError.Code() != model.ErrorCodeInvalidArgument {
				t.Fatalf("invalid_argument ではない: %T %v", err, err)
			}
		})
	}
}

func assertReadInvalidArgument(t *testing.T, err error) {
	t.Helper()

	var argumentError judicialdecisionread.ArgumentError
	if !errors.As(err, &argumentError) ||
		argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "ref" {
		t.Fatalf("invalid_argument ではない: %T %v", err, err)
	}
}

func mustSourceRef(
	t *testing.T,
	provider string,
	source string,
	resourceType string,
	resourceID string,
	versionID *string,
) model.SourceResourceRef {
	t.Helper()
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     source,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	})
	if versionID != nil {
		key, err = model.NewSourceResourceKey(model.SourceResourceKeyValues{
			SourceID:     source,
			ResourceType: resourceType,
			ResourceID:   resourceID,
			VersionID:    *versionID,
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: provider,
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	return ref
}
