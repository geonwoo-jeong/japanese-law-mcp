package defaultprofile

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
)

func TestEvaluatorは候補専用Planningを明示注入できる(t *testing.T) {
	const verificationID = "candidate-evaluation-deterministic-replay"

	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("%s: 候補 profile set を構成できません: %v", verificationID, err)
	}
	evaluator, err := NewWithPlanning(candidate)
	if err != nil {
		t.Fatalf("%s: 候補 planning を evaluator へ注入できません: %v", verificationID, err)
	}
	identity, err := evaluator.Identity()
	if err != nil {
		t.Fatalf("%s: evaluator identity を取得できません: %v", verificationID, err)
	}
	if identity.ProfileSetVersion() != candidate.Profiles().ProfileVersion() ||
		identity.RankingVersion() != candidate.Profiles().RankingVersion() {
		t.Fatalf(
			"%s: evaluator identity = %q/%q, candidate = %q/%q",
			verificationID,
			identity.ProfileSetVersion(),
			identity.RankingVersion(),
			candidate.Profiles().ProfileVersion(),
			candidate.Profiles().RankingVersion(),
		)
	}
}
