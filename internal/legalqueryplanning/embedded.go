// Package legalqueryplanning は、製品と評価が共有する統合照会 planning 依存を構成する。
package legalqueryplanning

import (
	"fmt"
	"sync"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querypreprocess"
	coreprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/core"
	judicialcasesprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/judicialcases"
)

// JudicialCasesPackID は、default profile set が採用する裁判例 pack を表す。
const JudicialCasesPackID = "judicial-cases"

// Dependencies は、起動時に固定した前処理器と default profile set を保持する。
type Dependencies struct {
	preprocessor    legalquery.QueryPreprocessor
	profiles        legalquery.QueryProfileSet
	profileMetadata []legalquery.QueryProfileMetadata
}

// Preprocessor は、製品と評価で共有する不変な前処理器を返す。
func (d Dependencies) Preprocessor() legalquery.QueryPreprocessor {
	return d.preprocessor
}

// Profiles は、core と judicial-cases の固定 profile set を返す。
func (d Dependencies) Profiles() legalquery.QueryProfileSet {
	return d.profiles
}

// ProfileMetadata は、profile ID 順の個別 profile 版情報を返す。
func (d Dependencies) ProfileMetadata() []legalquery.QueryProfileMetadata {
	return append(
		[]legalquery.QueryProfileMetadata{},
		d.profileMetadata...,
	)
}

var loadEmbedded = sync.OnceValues(
	func() (Dependencies, error) {
		core, err := coreprofile.LoadEmbedded()
		if err != nil {
			return Dependencies{}, fmt.Errorf(
				"法令コア query profile を初期化できません: %w",
				err,
			)
		}
		judicialCases, err := judicialcasesprofile.LoadEmbedded()
		if err != nil {
			return Dependencies{}, fmt.Errorf(
				"裁判例 query profile を初期化できません: %w",
				err,
			)
		}
		cues := append(
			core.CueVocabulary(),
			judicialCases.CueVocabulary()...,
		)
		preprocessor, err := querypreprocess.NewEmbedded(cues)
		if err != nil {
			return Dependencies{}, fmt.Errorf(
				"統合照会の前処理器を初期化できません: %w",
				err,
			)
		}
		profiles, err := legalquery.NewQueryProfileSet(
			[]legalquery.QueryProfile{core, judicialCases},
		)
		if err != nil {
			return Dependencies{}, fmt.Errorf(
				"統合照会 query profile set を初期化できません: %w",
				err,
			)
		}
		return Dependencies{
			preprocessor: preprocessor,
			profiles:     profiles,
			profileMetadata: []legalquery.QueryProfileMetadata{
				core.Metadata(),
				judicialCases.Metadata(),
			},
		}, nil
	},
)

// LoadEmbedded は、組込み辞書と profile から共有 planning 依存を一度だけ構築する。
func LoadEmbedded() (Dependencies, error) {
	return loadEmbedded()
}
