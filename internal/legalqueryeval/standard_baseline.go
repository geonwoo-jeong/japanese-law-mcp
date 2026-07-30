package legalqueryeval

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
)

const maximumStandardBaselineBytes = 4 << 20

type standardReportDTO struct {
	ArtifactKind    string              `json:"artifactKind"`
	SchemaVersion   *int                `json:"schemaVersion"`
	CorpusVersion   string              `json:"corpusVersion"`
	HoldoutDigest   string              `json:"holdoutDigest"`
	ProfileSet      profileSetReportDTO `json:"profileSet"`
	BaselineVersion string              `json:"baselineVersion"`
	Sets            evaluationSetsDTO   `json:"sets"`
}

type profileSetReportDTO struct {
	ProfileSetID      string                    `json:"profileSetId"`
	ProfileSetVersion string                    `json:"profileSetVersion"`
	RankingVersion    string                    `json:"rankingVersion"`
	Profiles          []profileVersionReportDTO `json:"profiles"`
}

type profileVersionReportDTO struct {
	ProfileID      string `json:"profileId"`
	ProfileVersion string `json:"profileVersion"`
}

type evaluationSetsDTO struct {
	Development evaluationCaseSetDTO `json:"development"`
	Holdout     evaluationHoldoutDTO `json:"holdout"`
	Execution   executionReportDTO   `json:"execution"`
}

type evaluationCaseSetDTO struct {
	CaseCount *int `json:"caseCount"`
}

type evaluationHoldoutDTO struct {
	CaseCount           *int                        `json:"caseCount"`
	Metrics             []evaluationMetricReportDTO `json:"metrics"`
	Categories          []evaluationCategoryDTO     `json:"categories"`
	DerivedObservations []evaluationMetricReportDTO `json:"derivedObservations"`
	FailedCaseIDs       []string                    `json:"failedCaseIds"`
}

type evaluationCategoryDTO struct {
	CategoryID string                      `json:"categoryId"`
	CaseCount  *int                        `json:"caseCount"`
	Metrics    []evaluationMetricReportDTO `json:"metrics"`
}

type evaluationMetricReportDTO struct {
	MetricID      string   `json:"metricId"`
	Numerator     *int     `json:"numerator"`
	Denominator   *int     `json:"denominator"`
	Ratio         *float64 `json:"ratio"`
	FailedCaseIDs []string `json:"failedCaseIds"`
}

type executionReportDTO struct {
	CaseCount                  *int                        `json:"caseCount"`
	Metrics                    []evaluationMetricReportDTO `json:"metrics"`
	WrongResourceCallCount     *int                        `json:"wrongResourceCallCount"`
	BudgetViolationCount       *int                        `json:"budgetViolationCount"`
	AttemptOrderViolationCount *int                        `json:"attemptOrderViolationCount"`
	ImplicitFirstReadCount     *int                        `json:"implicitFirstReadCount"`
	EmptyReclassificationCount *int                        `json:"emptyReclassificationCount"`
	FailedCaseIDs              []string                    `json:"failedCaseIds"`
}

// LoadStandardBaseline は、サイズ制限と閉じた schema で review 済み baseline を読む。
func LoadStandardBaseline(path string) (StandardReport, error) {
	if path == "" {
		return StandardReport{}, fmt.Errorf("baseline path は必須です")
	}
	file, err := os.Open(path) //nolint:gosec // SOT-ENG-024: 利用者が明示した読取り専用 baseline path だけを開く。
	if err != nil {
		return StandardReport{}, fmt.Errorf("baseline を開けません: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return StandardReport{}, fmt.Errorf("baseline を確認できません: %w", err)
	}
	if !info.Mode().IsRegular() {
		return StandardReport{}, fmt.Errorf("baseline は通常ファイルでなければなりません")
	}
	if info.Size() > maximumStandardBaselineBytes {
		return StandardReport{}, fmt.Errorf("baseline が安全上の上限を超えています")
	}

	decoder := json.NewDecoder(
		io.LimitReader(file, maximumStandardBaselineBytes+1),
	)
	decoder.DisallowUnknownFields()
	var document standardReportDTO
	if err := decoder.Decode(&document); err != nil {
		return StandardReport{}, fmt.Errorf("baseline JSON が不正です: %w", err)
	}
	var trailing json.RawMessage
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		if err == nil {
			return StandardReport{}, fmt.Errorf("baseline JSON の後に別の値があります")
		}
		return StandardReport{}, fmt.Errorf("baseline JSON の末尾が不正です: %w", err)
	}
	return standardReportFromDTO(document)
}

// CompareStandardBaseline は、同じ schema の測定文書を完全一致で比較する。
func CompareStandardBaseline(
	actual StandardReport,
	baseline StandardReport,
) error {
	if err := actual.Validate(); err != nil {
		return fmt.Errorf("actual report が有効ではありません: %w", err)
	}
	if err := baseline.Validate(); err != nil {
		return fmt.Errorf("baseline report が有効ではありません: %w", err)
	}
	actualJSON, err := json.Marshal(actual)
	if err != nil {
		return fmt.Errorf("actual report を JSON 化できません: %w", err)
	}
	baselineJSON, err := json.Marshal(baseline)
	if err != nil {
		return fmt.Errorf("baseline report を JSON 化できません: %w", err)
	}
	if !bytes.Equal(actualJSON, baselineJSON) {
		return fmt.Errorf("評価結果が review 済み baseline と一致しません")
	}
	return nil
}

func standardReportFromDTO(
	document standardReportDTO,
) (StandardReport, error) {
	if document.ArtifactKind != standardReportArtifactKind {
		return StandardReport{}, fmt.Errorf("baseline の artifactKind が不正です")
	}
	if document.SchemaVersion == nil ||
		*document.SchemaVersion != standardReportSchemaVersion {
		return StandardReport{}, fmt.Errorf("baseline の schemaVersion が未対応です")
	}
	profileSet, err := profileSetFromDTO(document.ProfileSet)
	if err != nil {
		return StandardReport{}, err
	}
	holdout, err := holdoutReportFromDTO(document.Sets.Holdout)
	if err != nil {
		return StandardReport{}, err
	}
	execution, err := executionReportFromDTO(document.Sets.Execution)
	if err != nil {
		return StandardReport{}, err
	}
	if document.Sets.Development.CaseCount == nil {
		return StandardReport{}, fmt.Errorf("development.caseCount は必須です")
	}
	report := StandardReport{
		corpusVersion:   document.CorpusVersion,
		holdoutDigest:   document.HoldoutDigest,
		profileSet:      profileSet,
		baselineVersion: document.BaselineVersion,
		sets: EvaluationSetsReport{
			development: EvaluationCaseSetReport{
				caseCount: *document.Sets.Development.CaseCount,
			},
			holdout:   holdout,
			execution: execution,
		},
		initialized: true,
	}
	if err := report.Validate(); err != nil {
		return StandardReport{}, fmt.Errorf("baseline report が有効ではありません: %w", err)
	}
	return report, nil
}

func profileSetFromDTO(
	document profileSetReportDTO,
) (ProfileSetReport, error) {
	if document.Profiles == nil {
		return ProfileSetReport{}, fmt.Errorf("profileSet.profiles は必須です")
	}
	profiles := make(
		[]ProfileVersionReport,
		0,
		len(document.Profiles),
	)
	for _, profile := range document.Profiles {
		value, err := NewProfileVersionReport(ProfileVersionReportValues{
			ProfileID:      profile.ProfileID,
			ProfileVersion: profile.ProfileVersion,
			RankingVersion: document.RankingVersion,
		})
		if err != nil {
			return ProfileSetReport{}, err
		}
		profiles = append(profiles, value)
	}
	report := ProfileSetReport{
		profileSetID:      document.ProfileSetID,
		profileSetVersion: document.ProfileSetVersion,
		rankingVersion:    document.RankingVersion,
		profiles:          profiles,
	}
	return report, report.Validate()
}

func holdoutReportFromDTO(
	document evaluationHoldoutDTO,
) (EvaluationHoldoutReport, error) {
	if document.CaseCount == nil ||
		document.Metrics == nil ||
		document.Categories == nil ||
		document.DerivedObservations == nil ||
		document.FailedCaseIDs == nil {
		return EvaluationHoldoutReport{}, fmt.Errorf(
			"holdout の必須項目がありません",
		)
	}
	metrics, err := metricsFromDTO(document.Metrics)
	if err != nil {
		return EvaluationHoldoutReport{}, err
	}
	categories := make(
		[]EvaluationCategoryReport,
		0,
		len(document.Categories),
	)
	for _, category := range document.Categories {
		if category.CaseCount == nil || category.Metrics == nil {
			return EvaluationHoldoutReport{}, fmt.Errorf(
				"category の必須項目がありません",
			)
		}
		categoryMetrics, metricErr := metricsFromDTO(category.Metrics)
		if metricErr != nil {
			return EvaluationHoldoutReport{}, metricErr
		}
		categories = append(categories, EvaluationCategoryReport{
			categoryID: category.CategoryID,
			caseCount:  *category.CaseCount,
			metrics:    categoryMetrics,
		})
	}
	derivedObservations, err := metricsFromDTO(
		document.DerivedObservations,
	)
	if err != nil {
		return EvaluationHoldoutReport{}, err
	}
	report := EvaluationHoldoutReport{
		caseCount:           *document.CaseCount,
		metrics:             metrics,
		categories:          categories,
		derivedObservations: derivedObservations,
		failedCaseIDs:       append([]string{}, document.FailedCaseIDs...),
	}
	return report, validateEvaluationHoldoutReport(report)
}

func executionReportFromDTO(
	document executionReportDTO,
) (ExecutionReport, error) {
	if document.CaseCount == nil ||
		document.Metrics == nil ||
		document.WrongResourceCallCount == nil ||
		document.BudgetViolationCount == nil ||
		document.AttemptOrderViolationCount == nil ||
		document.ImplicitFirstReadCount == nil ||
		document.EmptyReclassificationCount == nil ||
		document.FailedCaseIDs == nil {
		return ExecutionReport{}, fmt.Errorf(
			"execution の必須項目がありません",
		)
	}
	metrics, err := metricsFromDTO(document.Metrics)
	if err != nil {
		return ExecutionReport{}, err
	}
	report := ExecutionReport{
		caseCount:                  *document.CaseCount,
		metrics:                    metrics,
		wrongResourceCallCount:     *document.WrongResourceCallCount,
		budgetViolationCount:       *document.BudgetViolationCount,
		attemptOrderViolationCount: *document.AttemptOrderViolationCount,
		implicitFirstReadCount:     *document.ImplicitFirstReadCount,
		emptyReclassificationCount: *document.EmptyReclassificationCount,
		failedCaseIDs:              append([]string{}, document.FailedCaseIDs...),
		initialized:                true,
	}
	return report, report.Validate()
}

func metricsFromDTO(
	documents []evaluationMetricReportDTO,
) ([]EvaluationMetricReport, error) {
	metrics := make([]EvaluationMetricReport, 0, len(documents))
	for _, document := range documents {
		if document.Numerator == nil ||
			document.Denominator == nil ||
			document.Ratio == nil ||
			document.FailedCaseIDs == nil {
			return nil, fmt.Errorf("metric の必須項目がありません")
		}
		metric, err := NewEvaluationMetricReport(
			EvaluationMetricReportValues{
				MetricID:      document.MetricID,
				Numerator:     *document.Numerator,
				Denominator:   *document.Denominator,
				FailedCaseIDs: document.FailedCaseIDs,
			},
		)
		if err != nil {
			return nil, err
		}
		if metric.Ratio() != *document.Ratio {
			return nil, fmt.Errorf("metric.ratio が分子と分母からの値に一致しません")
		}
		metrics = append(metrics, metric)
	}
	return metrics, nil
}
