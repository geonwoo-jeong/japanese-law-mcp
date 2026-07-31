package judicialcases

import (
	_ "embed"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

var (
	//go:embed testdata/relation-v2/profile.json
	verificationNextJudicialProfileJSON []byte
)

// LoadNextForVerification は、検証専用の次版 judicial evidence profile を返す。
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
		verificationNextJudicialProfileJSON,
		embeddedCues,
		lawNames,
		concepts,
	)
	if err != nil {
		return nil, fmt.Errorf("検証用次版 profile を読み込めません: %w", err)
	}
	profile, err := newJudicialEvidenceProfile(base)
	if err != nil {
		return nil, fmt.Errorf("検証用 judicial evidence profile を準備できません: %w", err)
	}
	return profile, nil
}
