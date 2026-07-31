package legalqueryplanning

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycandidateprofile"
	coreprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/core"
	judicialcasesprofile "github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/judicialcases"
)

const (
	profileMetadataRankingConsistencyID = "profile-metadata-ranking-consistency"
	nextProfileSetFixedCompositionID    = "next-profile-set-fixed-composition"
	nextRankingVersion                  = "legal-query-ranking-2026-07-31-2"
)

var loadStage36Next = sync.OnceValues(buildStage36NextDependencies)

func TestNextProfileSetは実Profileを固定順で一回だけ構成する(t *testing.T) {
	t.Parallel()

	dependencies, err := loadNextForVerification()
	if err != nil {
		t.Fatalf("%s: next profile set を構成できません: %v", nextProfileSetFixedCompositionID, err)
	}
	metadata := dependencies.ProfileMetadata()
	if len(metadata) != 2 ||
		metadata[0].ProfileID() != "core" ||
		metadata[1].ProfileID() != "judicial-cases" {
		t.Fatalf("%s: profile 順 = %#v", nextProfileSetFixedCompositionID, metadata)
	}
	for index, value := range metadata {
		margin, present := value.Selection().BranchRetentionMargin()
		if value.SchemaVersion() != 2 || !present || margin != 12 {
			t.Fatalf(
				"%s: metadata[%d] の schema/margin = %d/(%d,%t)",
				nextProfileSetFixedCompositionID,
				index,
				value.SchemaVersion(),
				margin,
				present,
			)
		}
	}
	assertStage36SharedCalibration(t, metadata[0], metadata[1])
	if dependencies.Profiles().RankingVersion() != nextRankingVersion {
		t.Fatalf(
			"%s: rankingVersion = %q",
			nextProfileSetFixedCompositionID,
			dependencies.Profiles().RankingVersion(),
		)
	}
	if err := dependencies.Profiles().Validate(); err != nil {
		t.Fatalf("%s: fixed set が無効です: %v", nextProfileSetFixedCompositionID, err)
	}

	again, err := loadNextForVerification()
	if err != nil {
		t.Fatalf("%s: fixed set を再取得できません: %v", nextProfileSetFixedCompositionID, err)
	}
	if again.Profiles().ProfileVersion() != dependencies.Profiles().ProfileVersion() {
		t.Fatalf("%s: 一回構成した set version が変わりました", nextProfileSetFixedCompositionID)
	}
	metadata[0] = metadata[1]
	if again.ProfileMetadata()[0].ProfileID() != "core" {
		t.Fatalf("%s: metadata getter が共有 slice を返しました", nextProfileSetFixedCompositionID)
	}
}

func TestNextProfileSetは欠落重複逆順を拒否する(t *testing.T) {
	t.Parallel()

	core, judicial := mustStage36Profiles(t)
	tests := []struct {
		name     string
		profiles []legalquery.QueryProfile
	}{
		{name: "coreだけ", profiles: []legalquery.QueryProfile{core}},
		{name: "judicialだけ", profiles: []legalquery.QueryProfile{judicial}},
		{name: "core重複", profiles: []legalquery.QueryProfile{core, core}},
		{name: "judicial重複", profiles: []legalquery.QueryProfile{judicial, judicial}},
		{name: "逆順", profiles: []legalquery.QueryProfile{judicial, core}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if _, err := newNextProfileSet(test.profiles); err == nil {
				t.Fatalf("%s: 不正な固定構成を受理しました", nextProfileSetFixedCompositionID)
			}
		})
	}
}

func TestProfileMetadataRankingは実Fixtureの校正差を拒否する(t *testing.T) {
	t.Parallel()

	core, judicial := mustStage36Profiles(t)
	different := stage36ProfileWithMetadata{
		QueryProfile: judicial,
		metadata:     mustStage36WeightChangedMetadata(t, judicial.Metadata()),
	}
	if _, err := newNextProfileSet([]legalquery.QueryProfile{core, different}); err == nil {
		t.Fatalf("%s: 同じ rankingVersion の weight 差を受理しました", profileMetadataRankingConsistencyID)
	}

	weights := core.Metadata().Score().EvidenceWeights()
	weights[0], weights[1] = weights[1], weights[0]
	if _, err := legalquery.NewQueryScorePolicy(legalquery.QueryScorePolicyValues{
		Minimum:            core.Metadata().Score().Minimum(),
		Maximum:            core.Metadata().Score().Maximum(),
		EvidenceWeights:    weights,
		HighConfidenceAt:   core.Metadata().Score().HighConfidenceAt(),
		MediumConfidenceAt: core.Metadata().Score().MediumConfidenceAt(),
	}); err == nil {
		t.Fatalf("%s: evidence weight の順序差を受理しました", profileMetadataRankingConsistencyID)
	}
}

func TestNextProfileBridgeはProductionから到達しない(t *testing.T) {
	t.Parallel()

	active, err := LoadEmbedded()
	if err != nil {
		t.Fatalf("%s: active set を読み込めません: %v", nextProfileSetFixedCompositionID, err)
	}
	metadata := active.ProfileMetadata()
	if len(metadata) != 2 || active.Profiles().ProfileVersion() !=
		"profile-set-sha256-be9ce1499a7b6708a162c4ae2f4da9a340ed2883d3bd3480b2ec21989d11bf8f" {
		t.Fatalf("%s: active set が変わりました", nextProfileSetFixedCompositionID)
	}
	for index, value := range metadata {
		margin, present := value.Selection().BranchRetentionMargin()
		if value.SchemaVersion() != 1 || present || margin != 0 {
			t.Fatalf("%s: active metadata[%d] が next 化しました", nextProfileSetFixedCompositionID, index)
		}
	}
	assertNoProductionNextProfileCalls(t)
}

func loadNextForVerification() (Dependencies, error) {
	return loadStage36Next()
}

func buildStage36NextDependencies() (Dependencies, error) {
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		return Dependencies{}, fmt.Errorf("next profile set を読み込めません: %w", err)
	}
	return Dependencies{
		preprocessor:    candidate.Preprocessor(),
		profiles:        candidate.Profiles(),
		profileMetadata: candidate.ProfileMetadata(),
	}, nil
}

func newNextProfileSet(
	profiles []legalquery.QueryProfile,
) (legalquery.QueryProfileSet, error) {
	expected := []string{"core", "judicial-cases"}
	if len(profiles) != len(expected) {
		return legalquery.QueryProfileSet{}, fmt.Errorf("next profiles は二件必要です")
	}
	for index, profile := range profiles {
		if profile == nil || profile.Metadata().ProfileID() != expected[index] {
			return legalquery.QueryProfileSet{}, fmt.Errorf("next profiles の固定順が一致しません")
		}
		if profile.Metadata().SchemaVersion() != 2 {
			return legalquery.QueryProfileSet{}, fmt.Errorf("next profiles は schema version 2 が必要です")
		}
	}
	return legalquery.NewQueryProfileSet(profiles)
}

func mustStage36Profiles(
	t *testing.T,
) (*coreprofile.Profile, *judicialcasesprofile.Profile) {
	t.Helper()
	candidate, err := legalquerycandidateprofile.Load()
	if err != nil {
		t.Fatalf("%s: next profile set を読み込めません: %v", nextProfileSetFixedCompositionID, err)
	}
	return candidate.Core(), candidate.JudicialCases()
}

func assertStage36SharedCalibration(
	t *testing.T,
	left legalquery.QueryProfileMetadata,
	right legalquery.QueryProfileMetadata,
) {
	t.Helper()
	leftMargin, leftPresent := left.Selection().BranchRetentionMargin()
	rightMargin, rightPresent := right.Selection().BranchRetentionMargin()
	if left.RankingVersion() != right.RankingVersion() ||
		!reflect.DeepEqual(left.Score(), right.Score()) ||
		left.Selection() != right.Selection() ||
		!slices.Equal(left.TieBreak(), right.TieBreak()) ||
		leftPresent != rightPresent || leftMargin != rightMargin {
		t.Fatalf("%s: next profile の共有校正が一致しません", profileMetadataRankingConsistencyID)
	}
}

type stage36ProfileWithMetadata struct {
	legalquery.QueryProfile
	metadata legalquery.QueryProfileMetadata
}

func (p stage36ProfileWithMetadata) Metadata() legalquery.QueryProfileMetadata {
	return p.metadata
}

func mustStage36WeightChangedMetadata(
	t *testing.T,
	base legalquery.QueryProfileMetadata,
) legalquery.QueryProfileMetadata {
	t.Helper()
	weights := base.Score().EvidenceWeights()
	changed, err := legalquery.NewQueryEvidenceWeight(
		legalquery.QueryEvidenceWeightValues{
			Code:   weights[0].Code(),
			Weight: weights[0].Weight() + 1,
		},
	)
	if err != nil {
		t.Fatalf("%s: 差分 weight を作成できません: %v", profileMetadataRankingConsistencyID, err)
	}
	weights[0] = changed
	score, err := legalquery.NewQueryScorePolicy(legalquery.QueryScorePolicyValues{
		Minimum:            base.Score().Minimum(),
		Maximum:            base.Score().Maximum() + 1,
		EvidenceWeights:    weights,
		HighConfidenceAt:   base.Score().HighConfidenceAt(),
		MediumConfidenceAt: base.Score().MediumConfidenceAt(),
	})
	if err != nil {
		t.Fatalf("%s: 差分 score を作成できません: %v", profileMetadataRankingConsistencyID, err)
	}
	margin, present := base.Selection().BranchRetentionMargin()
	selection, err := legalquery.NewQuerySelectionPolicy(legalquery.QuerySelectionPolicyValues{
		SingleThreshold:           base.Selection().SingleThreshold(),
		MinimumExecutionThreshold: base.Selection().MinimumExecutionThreshold(),
		SingleMargin:              base.Selection().SingleMargin(),
		HedgeMargin:               base.Selection().HedgeMargin(),
		BranchRetentionMargin:     margin,
		BranchRetentionPresent:    present,
		ScoreMinimum:              score.Minimum(),
		ScoreMaximum:              score.Maximum(),
	})
	if err != nil {
		t.Fatalf("%s: 差分 selection を作成できません: %v", profileMetadataRankingConsistencyID, err)
	}
	metadata, err := legalquery.NewQueryProfileMetadata(legalquery.QueryProfileMetadataValues{
		SchemaVersion:              base.SchemaVersion(),
		ProfileID:                  base.ProfileID(),
		ProfileVersion:             base.ProfileVersion(),
		RankingVersion:             base.RankingVersion(),
		CueSetVersion:              base.CueSetVersion(),
		LawNameLexiconVersion:      base.LawNameLexiconVersion(),
		LegalConceptLexiconVersion: base.LegalConceptLexiconVersion(),
		Targets:                    base.Targets(),
		Score:                      score,
		Selection:                  selection,
		TieBreak:                   base.TieBreak(),
		ConditionalTieBreaks:       base.ConditionalTieBreaks(),
	})
	if err != nil {
		t.Fatalf("%s: 差分 metadata を作成できません: %v", profileMetadataRankingConsistencyID, err)
	}
	return metadata
}

func assertNoProductionNextProfileCalls(t *testing.T) {
	t.Helper()
	repository, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("%s: repository path を解決できません: %v", nextProfileSetFixedCompositionID, err)
	}
	for _, root := range []string{"cmd", "internal"} {
		err = filepath.WalkDir(filepath.Join(repository, root), func(
			path string,
			entry fs.DirEntry,
			walkErr error,
		) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") ||
				strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, parseErr := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if parseErr != nil {
				return parseErr
			}
			identifiers := 0
			declarations := 0
			ast.Inspect(file, func(node ast.Node) bool {
				switch value := node.(type) {
				case *ast.Ident:
					if value.Name == "LoadNextForVerification" {
						identifiers++
					}
				case *ast.FuncDecl:
					if value.Name.Name == "LoadNextForVerification" {
						declarations++
					}
				}
				return true
			})
			if identifiers != declarations {
				t.Errorf(
					"%s: production source が next bridge を参照しています: %s",
					nextProfileSetFixedCompositionID,
					path,
				)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("%s: production call site を検査できません: %v", nextProfileSetFixedCompositionID, err)
		}
	}
}
