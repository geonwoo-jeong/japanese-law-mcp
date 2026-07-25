package githook

import (
	"bufio"
	"strings"
	"testing"
)

func TestMaterializeBatchBlobRejectsMalformedProtocol(t *testing.T) {
	oid := strings.Repeat("a", 40)
	tests := []struct {
		name     string
		response string
	}{
		{name: "header", response: "missing\n"},
		{name: "size", response: oid + " blob invalid\n"},
		{name: "truncated blob", response: oid + " blob 2\nx"},
		{name: "missing separator", response: oid + " blob 1\nx"},
		{name: "invalid separator", response: oid + " blob 1\nx!"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := materializeBatchBlob(
				bufio.NewReader(strings.NewReader(test.response)),
				t.TempDir(),
				"対象.txt",
				expectedFile{oid: oid},
			)
			if err == nil {
				t.Fatal("不正な Git cat-file 応答が受理されました")
			}
		})
	}
}
