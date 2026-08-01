package core

import (
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateartifact"
)

func TestLoadNextForVerificationはDevelopment校正版をActiveから分離する(
	t *testing.T,
) {
	t.Parallel()

	next, err := loadCandidateForTest()
	if err != nil {
		t.Fatalf("SOT-ENG-024/SOT-ENG-039: 校正版を読み込めません: %v", err)
	}
	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("SOT-ARCH-033: active 版を読み込めません: %v", err)
	}
	if next.Metadata().ProfileVersion() != "core-2026-07-31-38" ||
		next.Metadata().CueSetVersion() != "core-cues-2026-07-31-17" ||
		active.Metadata().ProfileVersion() != "core-2026-07-30-33" ||
		active.Metadata().CueSetVersion() != "core-cues-2026-07-30-15" {
		t.Fatal("SOT-ARCH-033: 校正版と active 版の identity が分離されていません")
	}
}

func loadCandidateForTest() (*Profile, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, err
	}
	artifact := legalquerycandidateartifact.Core()
	return Load(artifact.Metadata(), artifact.Cues(), lawNames, concepts)
}
