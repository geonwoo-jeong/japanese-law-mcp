package legalquerycandidateeval

import (
	"bytes"
	"context"
	"testing"
)

func TestCanonicalSchemaV2は五種の閉じた成果物を区別する(t *testing.T) {
	t.Parallel()

	schema, err := ParseSchemaV2(CanonicalSchemaV2())
	if err != nil {
		t.Fatalf("canonical schema v2 を解決できません: %v", err)
	}
	manifest := manifestWithID(t)
	architecture := validReviewAttestation(t, manifest, ReviewScopeArchitecture, "authority-a")
	samples := map[string][]byte{
		"pointer": mustCanonicalJSON(t, PointerDocument{
			ArtifactKind:  ArtifactKindPointer,
			SchemaVersion: SchemaVersionV2,
			EvaluationID:  "evaluation-sha256-" + repeatHex('a'),
		}),
		"candidate-content":  mustCanonicalJSON(t, manifest),
		"review-attestation": mustCanonicalJSON(t, architecture),
		"evaluation-request": mustCanonicalJSON(t, validEvaluationRequest(t, manifest)),
		"evaluation-result": mustCanonicalJSON(t, struct {
			ArtifactKind  string `json:"artifactKind"`
			SchemaVersion int    `json:"schemaVersion"`
			EvaluationID  string `json:"evaluationId"`
			RequestSHA256 string `json:"requestSha256"`
			Outcome       string `json:"outcome"`
			ReportSHA256  string `json:"reportSha256"`
		}{
			ArtifactKind:  "legal_query_candidate_evaluation_result",
			SchemaVersion: SchemaVersionV2,
			EvaluationID:  "evaluation-sha256-" + repeatHex('a'),
			RequestSHA256: repeatHex('b'),
			Outcome:       "passed",
			ReportSHA256:  repeatHex('c'),
		}),
	}
	for name, raw := range samples {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if err := schema.Validate(context.Background(), raw); err != nil {
				t.Fatalf("有効な成果物を拒否しました: %v", err)
			}
			unknown := bytes.Replace(raw, []byte(`"schemaVersion":2`), []byte(`"schemaVersion":2,"unknown":1`), 1)
			if err := schema.Validate(context.Background(), unknown); err == nil {
				t.Fatal("未知 field を受理しました")
			}
		})
	}
}

func TestCanonicalSchemaV2は複製を返す(t *testing.T) {
	t.Parallel()

	first := CanonicalSchemaV2()
	second := CanonicalSchemaV2()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("canonical schema v2 が空です")
	}
	first[0] ^= 0xff
	if bytes.Equal(first, second) {
		t.Fatal("canonical schema v2 が呼出し元と可変 byte を共有しています")
	}
}
