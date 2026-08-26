package judicialcitationtrace_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationtrace"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestNewRequestは省略値をbothと5にする(t *testing.T) {
	t.Parallel()

	request, err := judicialcitationtrace.NewRequest(judicialcitationtrace.RequestValues{
		Ref: mustRef(t, "95570/detail2"),
	})
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	if request.Direction() != model.JudicialCitationRequestedDirectionBoth {
		t.Fatalf("direction = %q", request.Direction())
	}
	if request.IncomingLimit() != 5 {
		t.Fatalf("incomingLimit = %d", request.IncomingLimit())
	}
}

func TestRequestは不正な参照と直接JSON復元を拒否する(t *testing.T) {
	t.Parallel()

	zero := judicialcitationtrace.Request{}
	var argument judicialcitationtrace.ArgumentError
	if err := zero.Validate(); !errors.As(err, &argument) || argument.Code() != model.ErrorCodeInvalidArgument ||
		argument.Reason() == "" || argument.Error() == "" || argument.Validate() != nil {
		t.Fatalf("zero request error = %#v", err)
	}
	if err := json.Unmarshal([]byte(`{}`), &zero); err == nil {
		t.Fatal("Request の直接 JSON 復元を受理しました")
	}

	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     "courts-hanrei",
		ResourceType: "law",
		ResourceID:   "root",
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: "courts-hanrei-html",
		Key:        key,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := judicialcitationtrace.NewRequest(judicialcitationtrace.RequestValues{Ref: ref}); err == nil {
		t.Fatal("law ref を受理しました")
	}
}

func TestNewRequestはproviderSourceCanonicalResourceIDを外部呼出し前に拒否する(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		ref  model.SourceResourceRef
	}{
		{
			name: "providerId",
			ref:  mustCustomRef(t, "other-provider", "courts-hanrei", "judicial-decision", "12345/detail2", ""),
		},
		{
			name: "sourceId",
			ref:  mustCustomRef(t, "courts-hanrei-html", "other-source", "judicial-decision", "12345/detail2", ""),
		},
		{
			name: "resourceId",
			ref:  mustCustomRef(t, "courts-hanrei-html", "courts-hanrei", "judicial-decision", "12345:detail2", ""),
		},
		{
			name: "versionId",
			ref:  mustCustomRef(t, "courts-hanrei-html", "courts-hanrei", "judicial-decision", "12345/detail2", "v1"),
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := judicialcitationtrace.NewRequest(judicialcitationtrace.RequestValues{
				Ref: test.ref,
			}); err == nil {
				t.Fatalf("%s を受理しました", test.name)
			}
		})
	}
}

func TestNewRequestは方向と候補上限を外部呼出し前に拒否する(t *testing.T) {
	t.Parallel()

	for _, test := range []struct {
		name   string
		values judicialcitationtrace.RequestValues
		field  string
	}{
		{
			name: "direction",
			values: judicialcitationtrace.RequestValues{
				Ref:       mustRef(t, "95570/detail2"),
				Direction: model.JudicialCitationRequestedDirection("sideways"),
			},
			field: "direction",
		},
		{
			name: "incomingLimit lower",
			values: judicialcitationtrace.RequestValues{
				Ref:           mustRef(t, "95570/detail2"),
				IncomingLimit: intPointer(0),
			},
			field: "incomingLimit",
		},
		{
			name: "incomingLimit upper",
			values: judicialcitationtrace.RequestValues{
				Ref:           mustRef(t, "95570/detail2"),
				IncomingLimit: intPointer(11),
			},
			field: "incomingLimit",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			_, err := judicialcitationtrace.NewRequest(test.values)
			var argument judicialcitationtrace.ArgumentError
			if !errors.As(err, &argument) || argument.Field() != test.field {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func intPointer(value int) *int { return &value }

func mustCustomRef(
	t *testing.T,
	providerID, sourceID, resourceType, resourceID, versionID string,
) model.SourceResourceRef {
	t.Helper()

	keyValues := model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
	}
	if versionID != "" {
		keyValues.VersionID = versionID
	}
	key, err := model.NewSourceResourceKey(keyValues)
	if err != nil {
		t.Fatalf("key = %v", err)
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		t.Fatalf("ref = %v", err)
	}
	return ref
}
