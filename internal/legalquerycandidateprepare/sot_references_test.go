package legalquerycandidateprepare

import (
	"path/filepath"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateeval"
)

func TestBuildRequiredSOTReferencesは廃止SOTをStaleとして隔離する(
	t *testing.T,
) {
	t.Parallel()

	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("candidate-evaluation-review-content-binding: repository を解決できません: %v", err)
	}
	for _, schemaVersion := range []int{
		legalquerycandidateeval.SchemaVersionV2,
		legalquerycandidateeval.SchemaVersionV3,
	} {
		references, err := BuildRequiredSOTReferences(t.Context(), repository, schemaVersion)
		reasons, stale := legalquerycandidateeval.StaleReasonsFromError(err)
		if len(references) != 0 || !stale || !legalquerycandidateeval.EqualStaleReasons(
			reasons,
			[]legalquerycandidateeval.StaleReason{
				legalquerycandidateeval.StaleReasonReviewSOTLifecycleDrift,
			},
		) {
			t.Fatalf("candidate-evaluation-stale-candidate-readiness-fail: schema version %d の結果=(%v,%v)", schemaVersion, references, err)
		}
	}
}

func TestValidateRequiredSOTDocumentはLifecycleDriftと破損を区別する(t *testing.T) {
	t.Parallel()

	const heading = "# SOT-IF-040: テスト\n"
	if err := validateRequiredSOTDocument(
		"SOT-IF-040",
		[]byte(heading+"\n- 状態: 有効\n\n## 規定\n"),
	); err != nil {
		t.Fatalf("有効な SOT を拒否しました: %v", err)
	}
	for _, status := range []string{"草案", "廃止"} {
		err := validateRequiredSOTDocument(
			"SOT-IF-040",
			[]byte(heading+"\n- 状態: "+status+"\n\n## 規定\n"),
		)
		if !legalquerycandidateeval.IsCurrentStale(err) {
			t.Fatalf("candidate-evaluation-stale-reason-closed-set: status=%s error=%v", status, err)
		}
	}
	for name, raw := range map[string][]byte{
		"状態欠落":       []byte(heading + "\n## 規定\n"),
		"未知状態":       []byte(heading + "\n- 状態: 保留\n\n## 規定\n"),
		"状態重複":       []byte(heading + "\n- 状態: 有効\n- 状態: 廃止\n\n## 規定\n"),
		"heading不一致": []byte("# SOT-IF-067: テスト\n\n- 状態: 有効\n"),
	} {
		name, raw := name, raw
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			err := validateRequiredSOTDocument("SOT-IF-040", raw)
			if err == nil || legalquerycandidateeval.IsCurrentStale(err) {
				t.Fatalf("candidate-evaluation-stale-does-not-mask-artifact-corruption: error=%v", err)
			}
		})
	}
}
