// Package judicialcases は、裁判例検索と裁判例読取りの query profile を提供する。
package judicialcases

import (
	_ "embed"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/cueartifact"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/metadataartifact"
)

const (
	profileID      = "judicial-cases"
	requiredPackID = "judicial-cases"
)

var (
	//go:embed data/profile.json
	embeddedProfile []byte

	//go:embed data/cues.json
	embeddedCues []byte
)

// Profile は、起動時に検証した不変の裁判例 profile である。
type Profile struct {
	metadata           legalquery.QueryProfileMetadata
	cues               []legalquery.CueVocabularyEntry
	cueByID            map[string]cueDefinition
	concepts           map[string]conceptDefinition
	intentEvidenceMode cueIntentEvidenceMode
}

// LoadEmbedded は、組込み profile、cue および共有辞書を検証して構築する。
func LoadEmbedded() (*Profile, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法令名辞書を読み込めません: %w", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	return Load(embeddedProfile, embeddedCues, lawNames, concepts)
}

// Load は、閉じた JSON と共有辞書の版を fail closed で検証する。
func Load(
	profileJSON []byte,
	cuesJSON []byte,
	lawNames *lawnamelexicon.Lexicon,
	concepts *legalconceptlexicon.Lexicon,
) (*Profile, error) {
	if lawNames == nil || lawNames.Version() == "" {
		return nil, fmt.Errorf("法令名辞書は必須です")
	}
	if concepts == nil || concepts.Version() == "" {
		return nil, fmt.Errorf("法概念辞書は必須です")
	}
	profileArtifact, err := metadataartifact.Load(profileJSON)
	if err != nil {
		return nil, err
	}
	metadata := profileArtifact.Metadata()
	cueData, err := cueartifact.Load(cuesJSON)
	if err != nil {
		return nil, err
	}
	if metadata.ProfileID() != profileID ||
		cueData.ProfileID() != profileID {
		return nil, fmt.Errorf("profileId は %q でなければなりません", profileID)
	}
	if err := cueData.MatchProfile(
		metadata.ProfileID(),
		metadata.CueSetVersion(),
	); err != nil {
		return nil, err
	}
	if metadata.LawNameLexiconVersion() != lawNames.Version() ||
		metadata.LegalConceptLexiconVersion() != concepts.Version() {
		return nil, fmt.Errorf("profile が参照する辞書 version と実体が一致しません")
	}
	if err := validateJudicialMetadata(
		metadata,
		profileArtifact.ConditionalTieBreaksPresent(),
	); err != nil {
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
	profile := &Profile{
		metadata: metadata,
		cues:     cues,
		cueByID:  cueByID,
		concepts: conceptByID,
	}
	if metadata.SchemaVersion() == 2 {
		return newJudicialEvidenceProfile(profile)
	}
	return profile, nil
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
			ProfileID:  cue.ProfileID,
			CueID:      cue.CueID,
			MatchGroup: cue.MatchGroup,
			SyntaxRole: cue.SyntaxRole,
			Terms:      append([]string(nil), cue.Terms...),
		})
	}
	return result
}
