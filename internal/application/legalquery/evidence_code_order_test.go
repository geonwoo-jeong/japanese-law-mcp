package legalquery

import "testing"

func TestEvidenceCodeOrderは候補根拠の固定順を返す(t *testing.T) {
	t.Parallel()

	values := []EvidenceCode{
		EvidenceOfficialIdentifier,
		EvidenceStructuredReference,
		EvidenceExplicitTask,
		EvidenceExplicitResource,
		EvidenceOfficialAlias,
		EvidenceLegalConcept,
		EvidenceMorphologicalContext,
		EvidenceUniqueTypoCorrection,
		EvidenceGeneralTerm,
	}
	for expected, value := range values {
		actual, exists := value.Order()
		if !exists || actual != expected {
			t.Fatalf(
				"根拠順が一致しません: code=%q actual=%d exists=%t expected=%d",
				value,
				actual,
				exists,
				expected,
			)
		}
	}

	if _, exists := EvidenceCode("unknown").Order(); exists {
		t.Fatal("未定義の根拠コードに順序を返しました")
	}
}
