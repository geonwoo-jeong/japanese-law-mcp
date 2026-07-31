package legalquerycandidateartifact

import "testing"

func TestProfileBytesは候補成果物の複製だけを返す(t *testing.T) {
	t.Parallel()

	core := Core()
	judicial := JudicialCases()
	if len(core.Metadata()) == 0 || len(core.Cues()) == 0 ||
		len(judicial.Metadata()) == 0 || len(judicial.Cues()) == 0 {
		t.Fatal("candidate-evaluation-candidate-content-identity: 候補成果物が空です")
	}
	metadata := core.Metadata()
	original := metadata[0]
	metadata[0] ^= 0xff
	if Core().Metadata()[0] != original {
		t.Fatal("candidate-evaluation-candidate-content-identity: caller の変更が埋込み成果物へ到達しました")
	}
}
