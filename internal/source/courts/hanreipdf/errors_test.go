package hanreipdf

import (
	"errors"
	"net/http"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcasecitationextract"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestOperationAndHTTPErrorContracts(t *testing.T) {
	t.Parallel()

	for _, operation := range []operation{operationFetch, operationParse} {
		if operation.SourceOperationProviderID() != providerID ||
			operation.SourceOperationName() == "" || operation.ValidateSourceOperation() != nil {
			t.Fatalf("operation=%q", operation)
		}
	}
	if err := operation("unknown").ValidateSourceOperation(); err == nil {
		t.Fatal("未定義 operation を受理しました")
	}
	if err := errorForHTTPStatus(http.StatusNotFound); !errors.Is(err, judicialcasecitationextract.ErrNotFound) {
		t.Fatalf("error=%v", err)
	}
	errorValue := newSourceError(model.SourceErrorCodeSourceAuthFailed, operationFetch, "10")
	if errorValue.Code() != model.SourceErrorCodeInvalidSourceResponse {
		t.Fatalf("code=%s", errorValue.Code())
	}
}
