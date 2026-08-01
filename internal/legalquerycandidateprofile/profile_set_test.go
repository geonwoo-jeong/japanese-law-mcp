package legalquerycandidateprofile

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalqueryplanning"
)

func TestLoadは校正済み二Profileを固定順で構成する(t *testing.T) {
	t.Parallel()

	set, err := Load()
	if err != nil {
		t.Fatalf("SOT-ENG-038: 候補 profile set を構成できません: %v", err)
	}
	metadata := set.ProfileMetadata()
	if len(metadata) != 2 ||
		metadata[0].ProfileID() != "core" ||
		metadata[0].ProfileVersion() != "core-2026-07-31-37" ||
		metadata[1].ProfileID() != "judicial-cases" ||
		metadata[1].ProfileVersion() != "judicial-cases-2026-07-31-12" ||
		set.Profiles().RankingVersion() != "legal-query-ranking-2026-07-31-2" ||
		set.Profiles().ProfileVersion() != "profile-set-sha256-1c8c43fa032148bf0cadd2548afef04b3b0927a9d1a621ef175f5bee781c41a3" {
		t.Fatalf(
			"SOT-ENG-038: 候補 identity が固定値と一致しません: metadata=%#v profileSetVersion=%q",
			metadata,
			set.Profiles().ProfileVersion(),
		)
	}
}

func TestLoadはActiveProfileSetを変更しない(t *testing.T) {
	t.Parallel()

	candidate, err := Load()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: 候補 profile set を構成できません: %v", err)
	}
	active, err := legalqueryplanning.LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: active profile set を構成できません: %v", err)
	}
	if candidate.Profiles().ProfileVersion() == active.Profiles().ProfileVersion() ||
		active.Profiles().ProfileVersion() != "profile-set-sha256-be9ce1499a7b6708a162c4ae2f4da9a340ed2883d3bd3480b2ec21989d11bf8f" {
		t.Fatal("SOT-ARCH-033: 候補と active の profile set が分離されていません")
	}
}
