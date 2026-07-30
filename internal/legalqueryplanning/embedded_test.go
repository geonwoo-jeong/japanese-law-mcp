package legalqueryplanning

import "testing"

func TestLoadEmbeddedは評価用のProfile版を不変に公開する(t *testing.T) {
	dependencies, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("組込み planning 依存を読み込めません: %v", err)
	}

	metadata := dependencies.ProfileMetadata()
	if len(metadata) != 2 ||
		metadata[0].ProfileID() != "core" ||
		metadata[1].ProfileID() != "judicial-cases" {
		t.Fatalf("SOT-ENG-024: profile metadata = %#v", metadata)
	}
	if dependencies.Profiles().ProfileVersion() == "" ||
		dependencies.Profiles().RankingVersion() == "" {
		t.Fatal("SOT-ENG-024: profile set の版が空です")
	}

	metadata[0] = metadata[1]
	again := dependencies.ProfileMetadata()
	if again[0].ProfileID() != "core" {
		t.Fatal("SOT-ENG-025: ProfileMetadata() が共有 slice を返しました")
	}
}
