// Package metadataartifact は、query profile metadata の共通成果物契約を扱う。
package metadataartifact

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const maximumProfileBytes = 64 << 10

// Artifact は、schema version ごとに検証した不変の profile metadata である。
type Artifact struct {
	metadata                    legalquery.QueryProfileMetadata
	conditionalTieBreaksPresent bool
	canonicalBytes              []byte
}

// Load は、閉じた schema version 1 または 2 の成果物を読み込む。
func Load(data []byte) (*Artifact, error) {
	if len(data) == 0 || len(data) > maximumProfileBytes {
		return nil, fmt.Errorf(
			"profile.json は 1 byte 以上 %d byte 以下でなければなりません",
			maximumProfileBytes,
		)
	}
	if err := inspectJSON(data); err != nil {
		return nil, err
	}
	schemaVersion, err := decodeSchemaVersion(data)
	if err != nil {
		return nil, err
	}
	switch schemaVersion {
	case 1:
		document, decodeErr := decodeStrict[documentV1](data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		return buildCanonicalArtifact(document, document.values())
	case 2:
		document, decodeErr := decodeStrict[documentV2](data)
		if decodeErr != nil {
			return nil, decodeErr
		}
		return buildCanonicalArtifact(document, document.values())
	default:
		return nil, fmt.Errorf(
			"profile data の schemaVersion が未対応です",
		)
	}
}

// CanonicalBytes は、version 別 field 順と省略状態を保つ JSON byte を複製する。
func (a *Artifact) CanonicalBytes() []byte {
	if a == nil {
		return nil
	}
	return append([]byte(nil), a.canonicalBytes...)
}

func buildCanonicalArtifact(document any, values documentValues) (*Artifact, error) {
	artifact, err := buildArtifact(values)
	if err != nil {
		return nil, err
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return nil, fmt.Errorf("profile metadata を canonical JSON にできません: %w", err)
	}
	artifact.canonicalBytes = append(canonical, '\n')
	return artifact, nil
}

// Metadata は、検証済みの不変 metadata を返す。
func (a *Artifact) Metadata() legalquery.QueryProfileMetadata {
	if a == nil {
		return legalquery.QueryProfileMetadata{}
	}
	return a.metadata
}

// ConditionalTieBreaksPresent は、条件付き順位 field の存在有無を返す。
func (a *Artifact) ConditionalTieBreaksPresent() bool {
	return a != nil && a.conditionalTieBreaksPresent
}

func buildArtifact(
	values documentValues,
) (*Artifact, error) {
	if values.lexicons == nil {
		return nil, fmt.Errorf("lexicons は必須です")
	}
	scoreValues, err := values.score.values()
	if err != nil {
		return nil, err
	}
	weights := make(
		[]legalquery.QueryEvidenceWeight,
		0,
		len(scoreValues.weights),
	)
	for index, raw := range scoreValues.weights {
		weight, weightErr := legalquery.NewQueryEvidenceWeight(
			legalquery.QueryEvidenceWeightValues{
				Code:   legalquery.EvidenceCode(raw.code),
				Weight: raw.weight,
			},
		)
		if weightErr != nil {
			return nil, fmt.Errorf(
				"evidenceWeights[%d]: %w",
				index,
				weightErr,
			)
		}
		weights = append(weights, weight)
	}
	score, err := legalquery.NewQueryScorePolicy(
		legalquery.QueryScorePolicyValues{
			Minimum:            scoreValues.minimum,
			Maximum:            scoreValues.maximum,
			EvidenceWeights:    weights,
			HighConfidenceAt:   scoreValues.highConfidenceAt,
			MediumConfidenceAt: scoreValues.mediumConfidenceAt,
		},
	)
	if err != nil {
		return nil, err
	}
	selectionValues, err := values.selection.values()
	if err != nil {
		return nil, err
	}
	selection, err := legalquery.NewQuerySelectionPolicy(
		legalquery.QuerySelectionPolicyValues{
			SingleThreshold:           selectionValues.singleThreshold,
			MinimumExecutionThreshold: selectionValues.minimumExecutionThreshold,
			SingleMargin:              selectionValues.singleMargin,
			HedgeMargin:               selectionValues.hedgeMargin,
			BranchRetentionMargin:     selectionValues.branchRetentionMargin,
			BranchRetentionPresent:    selectionValues.branchRetentionPresent,
			ScoreMinimum:              score.Minimum(),
			ScoreMaximum:              score.Maximum(),
		},
	)
	if err != nil {
		return nil, err
	}
	targets, err := buildTargets(values.targets)
	if err != nil {
		return nil, err
	}
	tieBreaks := buildTieBreaks(values.tieBreak)
	conditional := buildConditionalTieBreaks(values.conditionalTieBreaks)
	metadata, err := legalquery.NewQueryProfileMetadata(
		legalquery.QueryProfileMetadataValues{
			SchemaVersion:              values.schemaVersion,
			ProfileID:                  values.profileID,
			ProfileVersion:             values.profileVersion,
			RankingVersion:             values.rankingVersion,
			CueSetVersion:              values.cueSetVersion,
			LawNameLexiconVersion:      values.lexicons.LawNames,
			LegalConceptLexiconVersion: values.lexicons.LegalConcepts,
			Targets:                    targets,
			Score:                      score,
			Selection:                  selection,
			TieBreak:                   tieBreaks,
			ConditionalTieBreaks:       conditional,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Artifact{
		metadata:                    metadata,
		conditionalTieBreaksPresent: values.conditionalTieBreaksPresent,
	}, nil
}

func buildTargets(
	values []targetDocument,
) ([]legalquery.QueryProfileTarget, error) {
	targets := make([]legalquery.QueryProfileTarget, 0, len(values))
	for index, raw := range values {
		target, err := legalquery.NewQueryProfileTarget(
			legalquery.QueryProfileTargetValues{
				Task:      legalquery.Task(raw.Task),
				Resource:  legalquery.Resource(raw.Resource),
				InputKind: legalquery.LogicalInputKind(raw.InputKind),
			},
		)
		if err != nil {
			return nil, fmt.Errorf("targets[%d]: %w", index, err)
		}
		targets = append(targets, target)
	}
	return targets, nil
}

func buildTieBreaks(values []string) []legalquery.QueryTieBreak {
	result := make([]legalquery.QueryTieBreak, 0, len(values))
	for _, value := range values {
		result = append(result, legalquery.QueryTieBreak(value))
	}
	return result
}

func buildConditionalTieBreaks(
	values map[string][]string,
) map[legalquery.ConditionalTieBreakName][]legalquery.QueryTieBreak {
	if len(values) == 0 {
		return nil
	}
	result := make(
		map[legalquery.ConditionalTieBreakName][]legalquery.QueryTieBreak,
		len(values),
	)
	for name, order := range values {
		result[legalquery.ConditionalTieBreakName(name)] =
			buildTieBreaks(order)
	}
	return result
}
