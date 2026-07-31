// Package legalquerycandidateprofile は、採用前の固定 profile set を直接構成する。
package legalquerycandidateprofile

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateartifact"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	coreprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/core"
	judicialcasesprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/judicialcases"
)

// Set は、候補専用の前処理器と二 profile を固定順で保持する。
type Set struct {
	preprocessor legalquery.QueryPreprocessor
	profiles     legalquery.QueryProfileSet
	metadata     []legalquery.QueryProfileMetadata
	core         *coreprofile.Profile
	judicial     *judicialcasesprofile.Profile
}

// Preprocessor は候補 cue から構成した前処理器を返す。
func (s Set) Preprocessor() legalquery.QueryPreprocessor { return s.preprocessor }

// Profiles は候補 profile set を返す。
func (s Set) Profiles() legalquery.QueryProfileSet { return s.profiles }

// ProfileMetadata は固定順 metadata の複製を返す。
func (s Set) ProfileMetadata() []legalquery.QueryProfileMetadata {
	return append([]legalquery.QueryProfileMetadata(nil), s.metadata...)
}

// Core は候補 set が所有する core profile を返す。
func (s Set) Core() *coreprofile.Profile { return s.core }

// JudicialCases は候補 set が所有する judicial-cases profile を返す。
func (s Set) JudicialCases() *judicialcasesprofile.Profile { return s.judicial }

// Load は候補成果物と現行辞書から profile set を直接構成する。
func Load() (Set, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return Set{}, fmt.Errorf("法令名辞書を読み込めません: %w", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return Set{}, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	core, judicial, err := loadProfiles(lawNames, concepts)
	if err != nil {
		return Set{}, err
	}
	return buildSet(core, judicial)
}

func loadProfiles(
	lawNames *lawnamelexicon.Lexicon,
	concepts *legalconceptlexicon.Lexicon,
) (*coreprofile.Profile, *judicialcasesprofile.Profile, error) {
	coreBytes := legalquerycandidateartifact.Core()
	core, err := coreprofile.Load(
		coreBytes.Metadata(), coreBytes.Cues(), lawNames, concepts,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("候補 core profile を読み込めません: %w", err)
	}
	judicialBytes := legalquerycandidateartifact.JudicialCases()
	judicial, err := judicialcasesprofile.Load(
		judicialBytes.Metadata(), judicialBytes.Cues(), lawNames, concepts,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("候補 judicial-cases profile を読み込めません: %w", err)
	}
	return core, judicial, nil
}

func buildSet(
	core *coreprofile.Profile,
	judicial *judicialcasesprofile.Profile,
) (Set, error) {
	profiles := []legalquery.QueryProfile{core, judicial}
	profileSet, err := legalquery.NewQueryProfileSet(profiles)
	if err != nil {
		return Set{}, fmt.Errorf("候補 profile set を構成できません: %w", err)
	}
	cues := append(core.CueVocabulary(), judicial.CueVocabulary()...)
	preprocessor, err := querypreprocess.NewEmbedded(cues)
	if err != nil {
		return Set{}, fmt.Errorf("候補前処理器を構成できません: %w", err)
	}
	return Set{
		preprocessor: preprocessor,
		profiles:     profileSet,
		core:         core,
		judicial:     judicial,
		metadata: []legalquery.QueryProfileMetadata{
			core.Metadata(), judicial.Metadata(),
		},
	}, nil
}
