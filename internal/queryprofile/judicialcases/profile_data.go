package judicialcases

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type profileDocument struct {
	SchemaVersion  int                     `json:"schemaVersion"`
	ProfileID      string                  `json:"profileId"`
	ProfileVersion string                  `json:"profileVersion"`
	RankingVersion string                  `json:"rankingVersion"`
	CueSetVersion  string                  `json:"cueSetVersion"`
	Targets        []targetDocument        `json:"targets"`
	Score          scoreDocument           `json:"score"`
	Selection      selectionDocument       `json:"selection"`
	TieBreak       []string                `json:"tieBreak"`
	Lexicons       lexiconVersionsDocument `json:"lexicons"`
}

type targetDocument struct {
	Task      string `json:"task"`
	Resource  string `json:"resource"`
	InputKind string `json:"inputKind"`
}

type scoreDocument struct {
	Minimum            int                      `json:"minimum"`
	Maximum            int                      `json:"maximum"`
	EvidenceWeights    []evidenceWeightDocument `json:"evidenceWeights"`
	HighConfidenceAt   int                      `json:"highConfidenceAt"`
	MediumConfidenceAt int                      `json:"mediumConfidenceAt"`
}

type evidenceWeightDocument struct {
	Code   string `json:"code"`
	Weight int    `json:"weight"`
}

type selectionDocument struct {
	SingleThreshold           int `json:"singleThreshold"`
	MinimumExecutionThreshold int `json:"minimumExecutionThreshold"`
	SingleMargin              int `json:"singleMargin"`
	HedgeMargin               int `json:"hedgeMargin"`
}

type lexiconVersionsDocument struct {
	LawNames      string `json:"lawNames"`
	LegalConcepts string `json:"legalConcepts"`
}

func decodeStrict[T any](
	name string,
	value []byte,
	maximumBytes int,
) (T, error) {
	var decoded T
	if len(value) == 0 || len(value) > maximumBytes {
		return decoded, fmt.Errorf(
			"%s は 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maximumBytes,
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return decoded, fmt.Errorf("%s を読み込めません: %w", name, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return decoded, fmt.Errorf("%s の後に値があります", name)
		}
		return decoded, fmt.Errorf("%s の終端が不正です: %w", name, err)
	}
	return decoded, nil
}

func buildMetadata(
	document profileDocument,
) (legalquery.QueryProfileMetadata, error) {
	if document.RankingVersion != sharedRankingVersion {
		return legalquery.QueryProfileMetadata{}, fmt.Errorf(
			"rankingVersion は共有校正版 %q と一致しなければなりません",
			sharedRankingVersion,
		)
	}
	targets := make([]legalquery.QueryProfileTarget, 0, len(document.Targets))
	for index, raw := range document.Targets {
		target, err := legalquery.NewQueryProfileTarget(
			legalquery.QueryProfileTargetValues{
				Task:      legalquery.Task(raw.Task),
				Resource:  legalquery.Resource(raw.Resource),
				InputKind: legalquery.LogicalInputKind(raw.InputKind),
			},
		)
		if err != nil {
			return legalquery.QueryProfileMetadata{}, fmt.Errorf(
				"targets[%d]: %w",
				index,
				err,
			)
		}
		targets = append(targets, target)
	}
	if !isExactJudicialTargets(targets) {
		return legalquery.QueryProfileMetadata{}, fmt.Errorf(
			"judicial-cases profile targets は裁判例二能力の固定順でなければなりません",
		)
	}

	weights := make([]legalquery.QueryEvidenceWeight, 0, len(document.Score.EvidenceWeights))
	for index, raw := range document.Score.EvidenceWeights {
		weight, err := legalquery.NewQueryEvidenceWeight(
			legalquery.QueryEvidenceWeightValues{
				Code:   legalquery.EvidenceCode(raw.Code),
				Weight: raw.Weight,
			},
		)
		if err != nil {
			return legalquery.QueryProfileMetadata{}, fmt.Errorf(
				"evidenceWeights[%d]: %w",
				index,
				err,
			)
		}
		weights = append(weights, weight)
	}
	score, err := legalquery.NewQueryScorePolicy(legalquery.QueryScorePolicyValues{
		Minimum:            document.Score.Minimum,
		Maximum:            document.Score.Maximum,
		EvidenceWeights:    weights,
		HighConfidenceAt:   document.Score.HighConfidenceAt,
		MediumConfidenceAt: document.Score.MediumConfidenceAt,
	})
	if err != nil {
		return legalquery.QueryProfileMetadata{}, err
	}
	selection, err := legalquery.NewQuerySelectionPolicy(
		legalquery.QuerySelectionPolicyValues{
			SingleThreshold:           document.Selection.SingleThreshold,
			MinimumExecutionThreshold: document.Selection.MinimumExecutionThreshold,
			SingleMargin:              document.Selection.SingleMargin,
			HedgeMargin:               document.Selection.HedgeMargin,
			ScoreMinimum:              score.Minimum(),
			ScoreMaximum:              score.Maximum(),
		},
	)
	if err != nil {
		return legalquery.QueryProfileMetadata{}, err
	}
	if !usesSharedCalibration(score, selection) {
		return legalquery.QueryProfileMetadata{}, fmt.Errorf(
			"score と selection は共有 rankingVersion の校正値と一致しなければなりません",
		)
	}
	tieBreak := make([]legalquery.QueryTieBreak, 0, len(document.TieBreak))
	for _, value := range document.TieBreak {
		tieBreak = append(tieBreak, legalquery.QueryTieBreak(value))
	}
	return legalquery.NewQueryProfileMetadata(
		legalquery.QueryProfileMetadataValues{
			SchemaVersion:              document.SchemaVersion,
			ProfileID:                  document.ProfileID,
			ProfileVersion:             document.ProfileVersion,
			RankingVersion:             document.RankingVersion,
			CueSetVersion:              document.CueSetVersion,
			LawNameLexiconVersion:      document.Lexicons.LawNames,
			LegalConceptLexiconVersion: document.Lexicons.LegalConcepts,
			Targets:                    targets,
			Score:                      score,
			Selection:                  selection,
			TieBreak:                   tieBreak,
		},
	)
}

func isExactJudicialTargets(values []legalquery.QueryProfileTarget) bool {
	expected := []legalquery.LogicalInputKind{
		legalquery.InputKindJudicialDecisionSearch,
		legalquery.InputKindJudicialDecisionRead,
	}
	if len(values) != len(expected) {
		return false
	}
	for index, target := range values {
		if target.InputKind() != expected[index] {
			return false
		}
	}
	return true
}

func usesSharedCalibration(
	score legalquery.QueryScorePolicy,
	selection legalquery.QuerySelectionPolicy,
) bool {
	expectedWeights := []int{90, 80, 60, 50, 40, 35, 25, 15, 10}
	weights := score.EvidenceWeights()
	if len(weights) != len(expectedWeights) {
		return false
	}
	for index, expected := range expectedWeights {
		if weights[index].Weight() != expected {
			return false
		}
	}
	return score.Minimum() == 0 &&
		score.Maximum() == 405 &&
		score.HighConfidenceAt() == 130 &&
		score.MediumConfidenceAt() == 80 &&
		selection.SingleThreshold() == 85 &&
		selection.MinimumExecutionThreshold() == 80 &&
		selection.SingleMargin() == 25 &&
		selection.HedgeMargin() == 10
}
