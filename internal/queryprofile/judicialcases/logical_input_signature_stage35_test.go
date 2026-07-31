package judicialcases

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func TestJudicialEvidenceRef署名は区切り文字を含む境界を保持する(
	t *testing.T,
) {
	firstRef := mustJudicialEvidenceRef(
		t,
		"provider-one",
		"source-one",
		"judicial-decision|95878/detail3",
	)
	secondRef := mustJudicialEvidenceRef(
		t,
		"provider-one",
		"source-one|judicial-decision",
		"95878/detail3",
	)
	first, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: firstRef},
	)
	if err != nil {
		t.Fatalf(
			"%s: first read input を作成できません: %v",
			judicialEvidenceMappingFailClosedID,
			err,
		)
	}
	second, err := legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: secondRef},
	)
	if err != nil {
		t.Fatalf(
			"%s: second read input を作成できません: %v",
			judicialEvidenceMappingFailClosedID,
			err,
		)
	}
	firstSignature, err := judicialLogicalInputSignature(first)
	if err != nil {
		t.Fatalf("%s: first signature: %v", judicialEvidenceMappingFailClosedID, err)
	}
	secondSignature, err := judicialLogicalInputSignature(second)
	if err != nil {
		t.Fatalf("%s: second signature: %v", judicialEvidenceMappingFailClosedID, err)
	}
	if firstSignature == secondSignature {
		t.Fatalf(
			"%s: 異なる ref の署名が衝突しました: %q",
			judicialEvidenceMappingFailClosedID,
			firstSignature,
		)
	}
}
