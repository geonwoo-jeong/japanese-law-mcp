package legalqueryplanning

import (
	"context"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

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
	if got := dependencies.Profiles().ProfileVersion(); got !=
		"profile-set-sha256-be9ce1499a7b6708a162c4ae2f4da9a340ed2883d3bd3480b2ec21989d11bf8f" {
		t.Fatalf(
			"SOT-ARCH-033: schemaVersion 1 の現行 profile set version = %q",
			got,
		)
	}

	metadata[0] = metadata[1]
	again := dependencies.ProfileMetadata()
	if again[0].ProfileID() != "core" {
		t.Fatal("SOT-ENG-025: ProfileMetadata() が共有 slice を返しました")
	}
}

func TestCoreEvidenceProductionNeutralは製品CompositionRootで維持する(
	t *testing.T,
) {
	const verificationID = "core-evidence-production-neutral"
	dependencies, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: planning 依存を読み込めません: %v", verificationID, err)
	}
	metadata := dependencies.ProfileMetadata()
	if len(metadata) == 0 || metadata[0].ProfileID() != "core" ||
		metadata[0].SchemaVersion() != 1 {
		t.Fatalf("%s: active core metadata = %#v", verificationID, metadata)
	}
	if margin, present := metadata[0].Selection().BranchRetentionMargin(); present || margin != 0 {
		t.Fatalf(
			"%s: active core metadata がtest専用分岐を持っています",
			verificationID,
		)
	}

	request, err := legalquery.NewRequest(legalquery.RequestValues{
		Query: "永住許可、帰化を教えてください",
	})
	if err != nil {
		t.Fatalf("%s: request を構築できません: %v", verificationID, err)
	}
	preprocessed, err := dependencies.Preprocessor().Preprocess(
		context.Background(),
		request,
	)
	if err != nil {
		t.Fatalf("%s: query を前処理できません: %v", verificationID, err)
	}
	result, err := dependencies.Profiles().Collect(preprocessed)
	if err != nil {
		t.Fatalf("%s: profile set を実行できません: %v", verificationID, err)
	}
	for _, candidate := range result.RankedCandidates() {
		if len(candidate.Steps()) > 1 {
			t.Fatalf(
				"%s: 製品 composition root が test 専用の複数 step core 候補を返しました",
				verificationID,
			)
		}
	}
}
