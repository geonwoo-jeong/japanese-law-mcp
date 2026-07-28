// Package judicialcases は、裁判例検索と裁判例読取りの query profile を提供する。
package judicialcases

import (
	_ "embed"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
)

const (
	profileID              = "judicial-cases"
	requiredPackID         = "judicial-cases"
	sharedRankingVersion   = "legal-query-ranking-2026-07-28-1"
	supportedSchemaVersion = 1
	maximumProfileBytes    = 64 << 10
	maximumCuesBytes       = 256 << 10
	maximumCueCount        = 128
)

var (
	//go:embed data/profile.json
	embeddedProfile []byte

	//go:embed data/cues.json
	embeddedCues []byte
)

// Profile は、起動時に検証した不変の裁判例 profile である。
type Profile struct {
	metadata legalquery.QueryProfileMetadata
	cues     []legalquery.CueVocabularyEntry
	cueByID  map[string]cueDefinition
	concepts map[string]conceptDefinition
}

// LoadEmbedded は、組込み profile、cue および共有辞書を検証して構築する。
func LoadEmbedded() (*Profile, error) {
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	return Load(embeddedProfile, embeddedCues, concepts)
}

// Load は、閉じた JSON と共有辞書の版を fail closed で検証する。
func Load(
	profileJSON []byte,
	cuesJSON []byte,
	concepts *legalconceptlexicon.Lexicon,
) (*Profile, error) {
	if concepts == nil || concepts.Version() == "" {
		return nil, fmt.Errorf("法概念辞書は必須です")
	}
	profileData, err := decodeStrict[profileDocument](
		"profile.json",
		profileJSON,
		maximumProfileBytes,
	)
	if err != nil {
		return nil, err
	}
	cueData, err := decodeStrict[cuesDocument](
		"cues.json",
		cuesJSON,
		maximumCuesBytes,
	)
	if err != nil {
		return nil, err
	}
	if profileData.SchemaVersion != supportedSchemaVersion ||
		cueData.SchemaVersion != supportedSchemaVersion {
		return nil, fmt.Errorf("profile data の schemaVersion が未対応です")
	}
	if profileData.ProfileID != profileID ||
		cueData.ProfileID != profileID {
		return nil, fmt.Errorf("profileId は %q でなければなりません", profileID)
	}
	if profileData.CueSetVersion != cueData.CueSetVersion {
		return nil, fmt.Errorf("profile.json と cues.json の版が一致しません")
	}
	if profileData.Lexicons.LegalConcepts != concepts.Version() {
		return nil, fmt.Errorf("profile が参照する辞書 version と実体が一致しません")
	}

	metadata, err := buildMetadata(profileData)
	if err != nil {
		return nil, err
	}
	cues, cueByID, err := buildCues(cueData)
	if err != nil {
		return nil, err
	}
	conceptByID, err := buildConceptDefinitions(concepts.Entries())
	if err != nil {
		return nil, err
	}
	return &Profile{
		metadata: metadata,
		cues:     cues,
		cueByID:  cueByID,
		concepts: conceptByID,
	}, nil
}

// Metadata は、selector と評価が参照する不変 metadata を返す。
func (p *Profile) Metadata() legalquery.QueryProfileMetadata {
	if p == nil {
		return legalquery.QueryProfileMetadata{}
	}
	return p.metadata
}

// CueVocabulary は、共通前処理へ注入する cue の深い複製を返す。
func (p *Profile) CueVocabulary() []legalquery.CueVocabularyEntry {
	if p == nil {
		return nil
	}
	result := make([]legalquery.CueVocabularyEntry, 0, len(p.cues))
	for _, cue := range p.cues {
		result = append(result, legalquery.CueVocabularyEntry{
			ProfileID: cue.ProfileID,
			CueID:     cue.CueID,
			Terms:     append([]string(nil), cue.Terms...),
		})
	}
	return result
}
