package legalquery

import (
	"errors"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestMaterializerClassifiesRefMismatchAsInvalidArgument(t *testing.T) {
	t.Parallel()

	ref := newLegalQuerySourceResourceRef(
		t,
		"explicit-provider",
		"explicit-source",
		"law",
		"opaque-law-id",
	)
	input, err := NewLawReadIntentV1(LawReadIntentV1Values{Ref: &ref})
	if err != nil {
		t.Fatal(err)
	}
	_, err = NewCoreMaterializer().MaterializeLawDocumentRead(
		input,
		materializerTestBinding{
			providerID:             "different-provider",
			sourceID:               "explicit-source",
			capabilityID:           lawdocumentread.CapabilityID,
			capabilityMajorVersion: lawdocumentread.MajorVersion,
		},
		materializerReadBudget(),
	)
	if err == nil {
		t.Fatal("SOT-ARCH-026: ref provider の不一致を受理しました")
	}
	var argumentError ArgumentError
	if !errors.As(err, &argumentError) ||
		argumentError.Code() != model.ErrorCodeInvalidArgument ||
		argumentError.Field() != "ref" {
		t.Fatalf(
			"SOT-ARCH-026: ref の不一致 error = %T %v",
			err,
			err,
		)
	}
}
