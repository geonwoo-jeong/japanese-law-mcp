package core

import (
	_ "embed"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

var (
	//go:embed testdata/development-calibration/profile.json
	verificationNextProfileJSON []byte

	//go:embed testdata/development-calibration/cues.json
	verificationNextCuesJSON []byte
)

// LoadNextForVerification は、検証専用の次版 core evidence profile を返す。
func LoadNextForVerification() (*Profile, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法令名辞書を読み込めません: %w", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	base, err := Load(
		verificationNextProfileJSON,
		verificationNextCuesJSON,
		lawNames,
		concepts,
	)
	if err != nil {
		return nil, fmt.Errorf("検証用次版 profile を読み込めません: %w", err)
	}
	profile, err := newCoreEvidenceProfile(base)
	if err != nil {
		return nil, fmt.Errorf("検証用 core evidence profile を準備できません: %w", err)
	}
	return profile, nil
}
