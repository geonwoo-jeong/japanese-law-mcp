package metadataartifact

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

type schemaVersionDocument struct {
	SchemaVersion *int `json:"schemaVersion"`
}

type documentV1 struct {
	SchemaVersion        int                      `json:"schemaVersion"`
	ProfileID            string                   `json:"profileId"`
	ProfileVersion       string                   `json:"profileVersion"`
	RankingVersion       string                   `json:"rankingVersion"`
	CueSetVersion        string                   `json:"cueSetVersion"`
	Targets              []targetDocument         `json:"targets"`
	Score                *scoreDocument           `json:"score"`
	Selection            *selectionDocumentV1     `json:"selection"`
	TieBreak             []string                 `json:"tieBreak"`
	ConditionalTieBreaks *map[string][]string     `json:"conditionalTieBreaks"`
	Lexicons             *lexiconVersionsDocument `json:"lexicons"`
}

type documentV2 struct {
	SchemaVersion        int                      `json:"schemaVersion"`
	ProfileID            string                   `json:"profileId"`
	ProfileVersion       string                   `json:"profileVersion"`
	RankingVersion       string                   `json:"rankingVersion"`
	CueSetVersion        string                   `json:"cueSetVersion"`
	Targets              []targetDocument         `json:"targets"`
	Score                *scoreDocument           `json:"score"`
	Selection            *selectionDocumentV2     `json:"selection"`
	TieBreak             []string                 `json:"tieBreak"`
	ConditionalTieBreaks *map[string][]string     `json:"conditionalTieBreaks"`
	Lexicons             *lexiconVersionsDocument `json:"lexicons"`
}

type targetDocument struct {
	Task      string `json:"task"`
	Resource  string `json:"resource"`
	InputKind string `json:"inputKind"`
}

type scoreDocument struct {
	Minimum            *int                     `json:"minimum"`
	Maximum            *int                     `json:"maximum"`
	EvidenceWeights    []evidenceWeightDocument `json:"evidenceWeights"`
	HighConfidenceAt   *int                     `json:"highConfidenceAt"`
	MediumConfidenceAt *int                     `json:"mediumConfidenceAt"`
}

type evidenceWeightDocument struct {
	Code   string `json:"code"`
	Weight *int   `json:"weight"`
}

type selectionDocumentV1 struct {
	SingleThreshold           *int `json:"singleThreshold"`
	MinimumExecutionThreshold *int `json:"minimumExecutionThreshold"`
	SingleMargin              *int `json:"singleMargin"`
	HedgeMargin               *int `json:"hedgeMargin"`
}

type selectionDocumentV2 struct {
	SingleThreshold           *int `json:"singleThreshold"`
	MinimumExecutionThreshold *int `json:"minimumExecutionThreshold"`
	SingleMargin              *int `json:"singleMargin"`
	HedgeMargin               *int `json:"hedgeMargin"`
	BranchRetentionMargin     *int `json:"branchRetentionMargin"`
}

type lexiconVersionsDocument struct {
	LawNames      string `json:"lawNames"`
	LegalConcepts string `json:"legalConcepts"`
}

type documentValues struct {
	schemaVersion               int
	profileID                   string
	profileVersion              string
	rankingVersion              string
	cueSetVersion               string
	targets                     []targetDocument
	score                       *scoreDocument
	selection                   selectionDocumentValues
	tieBreak                    []string
	conditionalTieBreaks        map[string][]string
	conditionalTieBreaksPresent bool
	lexicons                    *lexiconVersionsDocument
}

type scoreDocumentValues struct {
	minimum            int
	maximum            int
	weights            []evidenceWeightValues
	highConfidenceAt   int
	mediumConfidenceAt int
}

type evidenceWeightValues struct {
	code   string
	weight int
}

type selectionDocumentValues struct {
	singleThreshold           *int
	minimumExecutionThreshold *int
	singleMargin              *int
	hedgeMargin               *int
	branchRetentionMargin     *int
	branchRetentionPresent    bool
}

type selectionValues struct {
	singleThreshold           int
	minimumExecutionThreshold int
	singleMargin              int
	hedgeMargin               int
	branchRetentionMargin     int
	branchRetentionPresent    bool
}

func decodeSchemaVersion(data []byte) (int, error) {
	var header schemaVersionDocument
	if err := json.Unmarshal(data, &header); err != nil {
		return 0, fmt.Errorf("profile.json を読み込めません: %w", err)
	}
	if header.SchemaVersion == nil {
		return 0, fmt.Errorf("schemaVersion は必須です")
	}
	return *header.SchemaVersion, nil
}

func decodeStrict[T any](data []byte) (T, error) {
	var decoded T
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return decoded, fmt.Errorf("profile.json を読み込めません: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return decoded, fmt.Errorf("profile.json の後に値があります")
		}
		return decoded, fmt.Errorf(
			"profile.json の終端が不正です: %w",
			err,
		)
	}
	return decoded, nil
}

func (d documentV1) values() documentValues {
	selection := selectionDocumentValues{}
	if d.Selection != nil {
		selection = selectionDocumentValues{
			singleThreshold:           d.Selection.SingleThreshold,
			minimumExecutionThreshold: d.Selection.MinimumExecutionThreshold,
			singleMargin:              d.Selection.SingleMargin,
			hedgeMargin:               d.Selection.HedgeMargin,
		}
	}
	return commonDocumentValues(
		d.SchemaVersion,
		d.ProfileID,
		d.ProfileVersion,
		d.RankingVersion,
		d.CueSetVersion,
		d.Targets,
		d.Score,
		selection,
		d.TieBreak,
		d.ConditionalTieBreaks,
		d.Lexicons,
	)
}

func (d documentV2) values() documentValues {
	selection := selectionDocumentValues{}
	if d.Selection != nil {
		selection = selectionDocumentValues{
			singleThreshold:           d.Selection.SingleThreshold,
			minimumExecutionThreshold: d.Selection.MinimumExecutionThreshold,
			singleMargin:              d.Selection.SingleMargin,
			hedgeMargin:               d.Selection.HedgeMargin,
			branchRetentionMargin:     d.Selection.BranchRetentionMargin,
			branchRetentionPresent:    true,
		}
	}
	return commonDocumentValues(
		d.SchemaVersion,
		d.ProfileID,
		d.ProfileVersion,
		d.RankingVersion,
		d.CueSetVersion,
		d.Targets,
		d.Score,
		selection,
		d.TieBreak,
		d.ConditionalTieBreaks,
		d.Lexicons,
	)
}

func commonDocumentValues(
	schemaVersion int,
	profileID string,
	profileVersion string,
	rankingVersion string,
	cueSetVersion string,
	targets []targetDocument,
	score *scoreDocument,
	selection selectionDocumentValues,
	tieBreak []string,
	conditionalTieBreaks *map[string][]string,
	lexicons *lexiconVersionsDocument,
) documentValues {
	var conditionalValues map[string][]string
	if conditionalTieBreaks != nil {
		conditionalValues = cloneStringSlices(*conditionalTieBreaks)
	}
	return documentValues{
		schemaVersion:               schemaVersion,
		profileID:                   profileID,
		profileVersion:              profileVersion,
		rankingVersion:              rankingVersion,
		cueSetVersion:               cueSetVersion,
		targets:                     append([]targetDocument(nil), targets...),
		score:                       score,
		selection:                   selection,
		tieBreak:                    append([]string(nil), tieBreak...),
		conditionalTieBreaks:        conditionalValues,
		conditionalTieBreaksPresent: conditionalTieBreaks != nil,
		lexicons:                    lexicons,
	}
}

func (d *scoreDocument) values() (scoreDocumentValues, error) {
	if d == nil {
		return scoreDocumentValues{}, fmt.Errorf("score は必須です")
	}
	minimum, err := requiredInt("score.minimum", d.Minimum)
	if err != nil {
		return scoreDocumentValues{}, err
	}
	maximum, err := requiredInt("score.maximum", d.Maximum)
	if err != nil {
		return scoreDocumentValues{}, err
	}
	high, err := requiredInt(
		"score.highConfidenceAt",
		d.HighConfidenceAt,
	)
	if err != nil {
		return scoreDocumentValues{}, err
	}
	medium, err := requiredInt(
		"score.mediumConfidenceAt",
		d.MediumConfidenceAt,
	)
	if err != nil {
		return scoreDocumentValues{}, err
	}
	weights := make(
		[]evidenceWeightValues,
		0,
		len(d.EvidenceWeights),
	)
	for index, raw := range d.EvidenceWeights {
		weight, weightErr := requiredInt(
			fmt.Sprintf("score.evidenceWeights[%d].weight", index),
			raw.Weight,
		)
		if weightErr != nil {
			return scoreDocumentValues{}, weightErr
		}
		weights = append(weights, evidenceWeightValues{
			code:   raw.Code,
			weight: weight,
		})
	}
	return scoreDocumentValues{
		minimum:            minimum,
		maximum:            maximum,
		weights:            weights,
		highConfidenceAt:   high,
		mediumConfidenceAt: medium,
	}, nil
}

func (d selectionDocumentValues) values() (selectionValues, error) {
	singleThreshold, err := requiredInt(
		"selection.singleThreshold",
		d.singleThreshold,
	)
	if err != nil {
		return selectionValues{}, err
	}
	minimumExecution, err := requiredInt(
		"selection.minimumExecutionThreshold",
		d.minimumExecutionThreshold,
	)
	if err != nil {
		return selectionValues{}, err
	}
	singleMargin, err := requiredInt(
		"selection.singleMargin",
		d.singleMargin,
	)
	if err != nil {
		return selectionValues{}, err
	}
	hedgeMargin, err := requiredInt(
		"selection.hedgeMargin",
		d.hedgeMargin,
	)
	if err != nil {
		return selectionValues{}, err
	}
	branchMargin := 0
	if d.branchRetentionPresent {
		branchMargin, err = requiredInt(
			"selection.branchRetentionMargin",
			d.branchRetentionMargin,
		)
		if err != nil {
			return selectionValues{}, err
		}
	}
	return selectionValues{
		singleThreshold:           singleThreshold,
		minimumExecutionThreshold: minimumExecution,
		singleMargin:              singleMargin,
		hedgeMargin:               hedgeMargin,
		branchRetentionMargin:     branchMargin,
		branchRetentionPresent:    d.branchRetentionPresent,
	}, nil
}

func requiredInt(name string, value *int) (int, error) {
	if value == nil {
		return 0, fmt.Errorf("%s は必須です", name)
	}
	return *value, nil
}

func cloneStringSlices(
	values map[string][]string,
) map[string][]string {
	if len(values) == 0 {
		return nil
	}
	result := make(map[string][]string, len(values))
	for key, value := range values {
		result[key] = append([]string(nil), value...)
	}
	return result
}
