package lawv2

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestEmbeddedLawRevisionContractMatchesRecordedOpenAPI(t *testing.T) {
	t.Parallel()

	if err := verifyEmbeddedLawRevisionContract(); err != nil {
		t.Fatalf("記録済み契約を拒否しました: %v", err)
	}
}

func TestLawRevisionContractRejectsUnknownChange(t *testing.T) {
	t.Parallel()

	err := verifyRecordedLawRevisionContract([]byte(`{"unknown":true}`))
	assertSourceErrorCode(t, err, model.SourceErrorCodeSourceContractChanged)
}
