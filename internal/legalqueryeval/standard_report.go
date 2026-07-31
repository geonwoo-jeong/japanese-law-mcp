package legalqueryeval

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	standardReportArtifactKind  = "legal_query_evaluation"
	standardReportSchemaVersion = 1
)

var (
	standardCorpusVersionPattern = regexp.MustCompile(`^corpus-v[1-9][0-9]*$`)
	standardDigestPattern        = regexp.MustCompile(`^[0-9a-f]{64}$`)
	standardBaselinePattern      = regexp.MustCompile(
		`^default-[1-9][0-9]*$`,
	)
)

// ProfileVersionReportValues は、一 profile の版を保持する。
type ProfileVersionReportValues struct {
	ProfileID      string
	ProfileVersion string
	RankingVersion string
}

// ProfileVersionReport は、標準評価に使用した一 profile の版を表す。
type ProfileVersionReport struct {
	profileID      string
	profileVersion string
	rankingVersion string
}

// NewProfileVersionReport は、profile ID と版を検証して返す。
func NewProfileVersionReport(
	values ProfileVersionReportValues,
) (ProfileVersionReport, error) {
	report := ProfileVersionReport{
		profileID:      values.ProfileID,
		profileVersion: values.ProfileVersion,
		rankingVersion: values.RankingVersion,
	}
	if err := report.Validate(); err != nil {
		return ProfileVersionReport{}, err
	}
	return report, nil
}

// ProfileID は、profile ID を返す。
func (r ProfileVersionReport) ProfileID() string { return r.profileID }

// ProfileVersion は、profile 固有の版を返す。
func (r ProfileVersionReport) ProfileVersion() string {
	return r.profileVersion
}

// RankingVersion は、共有順位校正版を返す。
func (r ProfileVersionReport) RankingVersion() string {
	return r.rankingVersion
}

// Validate は、識別子と版を確認する。
func (r ProfileVersionReport) Validate() error {
	if !evaluationMetricIDPattern.MatchString(r.profileID) {
		return fmt.Errorf("profileId が定義済みの形式ではありません")
	}
	if err := validateEvaluationVersion("profileVersion", r.profileVersion); err != nil {
		return err
	}
	return validateEvaluationVersion("rankingVersion", r.rankingVersion)
}

// MarshalJSON は、個別 profile の ID と固有版だけを公開する。
func (r ProfileVersionReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ProfileID      string `json:"profileId"`
		ProfileVersion string `json:"profileVersion"`
	}{
		ProfileID:      r.ProfileID(),
		ProfileVersion: r.ProfileVersion(),
	})
}

// UnmarshalJSON は、baseline loader を介さない直接復元を拒否する。
func (*ProfileVersionReport) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ProfileVersionReport は JSON から直接復元できません。baseline loader を使用してください",
	)
}

// ProfileSetReport は、評価した profile set と構成 profile の版を保持する。
type ProfileSetReport struct {
	profileSetID      string
	profileSetVersion string
	rankingVersion    string
	profiles          []ProfileVersionReport
}

// ProfileSetID は、固定 profile set ID を返す。
func (r ProfileSetReport) ProfileSetID() string { return r.profileSetID }

// ProfileSetVersion は、合成規則を含む不透明な set 版を返す。
func (r ProfileSetReport) ProfileSetVersion() string {
	return r.profileSetVersion
}

// RankingVersion は、共有順位校正版を返す。
func (r ProfileSetReport) RankingVersion() string {
	return r.rankingVersion
}

// Profiles は、composition root の固定順で個別版を返す。
func (r ProfileSetReport) Profiles() []ProfileVersionReport {
	return append([]ProfileVersionReport{}, r.profiles...)
}

// Validate は、profile set の版と構成順を確認する。
func (r ProfileSetReport) Validate() error {
	if !evaluationMetricIDPattern.MatchString(r.profileSetID) {
		return fmt.Errorf("profileSetId が定義済みの形式ではありません")
	}
	if err := validateEvaluationVersion(
		"profileSetVersion",
		r.profileSetVersion,
	); err != nil {
		return err
	}
	if err := validateEvaluationVersion(
		"rankingVersion",
		r.rankingVersion,
	); err != nil {
		return err
	}
	if len(r.profiles) == 0 {
		return fmt.Errorf("profiles は一件以上必要です")
	}
	previous := ""
	for _, profile := range r.profiles {
		if err := profile.Validate(); err != nil {
			return fmt.Errorf("profile version が有効ではありません: %w", err)
		}
		if profile.RankingVersion() != r.rankingVersion {
			return fmt.Errorf("profiles は同じ rankingVersion を必要とします")
		}
		if previous != "" && profile.ProfileID() <= previous {
			return fmt.Errorf("profiles は profileId 昇順かつ重複なしでなければなりません")
		}
		previous = profile.ProfileID()
	}
	return nil
}

// MarshalJSON は、set 版と個別 profile 版を固定 object にする。
func (r ProfileSetReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ProfileSetID      string                 `json:"profileSetId"`
		ProfileSetVersion string                 `json:"profileSetVersion"`
		RankingVersion    string                 `json:"rankingVersion"`
		Profiles          []ProfileVersionReport `json:"profiles"`
	}{
		ProfileSetID:      r.ProfileSetID(),
		ProfileSetVersion: r.ProfileSetVersion(),
		RankingVersion:    r.RankingVersion(),
		Profiles:          r.Profiles(),
	})
}

// UnmarshalJSON は、baseline loader を介さない直接復元を拒否する。
func (*ProfileSetReport) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ProfileSetReport は JSON から直接復元できません。baseline loader を使用してください",
	)
}

// EvaluationCategoryReport は、category 件数と同じ標準指標を保持する。
type EvaluationCategoryReport struct {
	categoryID string
	caseCount  int
	metrics    []EvaluationMetricReport
}

// CategoryID は、corpus category ID を返す。
func (r EvaluationCategoryReport) CategoryID() string { return r.categoryID }

// CaseCount は、category の case 件数を返す。
func (r EvaluationCategoryReport) CaseCount() int { return r.caseCount }

// Metrics は、固定順の category 指標を返す。
func (r EvaluationCategoryReport) Metrics() []EvaluationMetricReport {
	return append([]EvaluationMetricReport{}, r.metrics...)
}

// MarshalJSON は、category 件数と指標を返す。
func (r EvaluationCategoryReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CategoryID string                   `json:"categoryId"`
		CaseCount  int                      `json:"caseCount"`
		Metrics    []EvaluationMetricReport `json:"metrics"`
	}{
		CategoryID: r.CategoryID(),
		CaseCount:  r.CaseCount(),
		Metrics:    r.Metrics(),
	})
}

// EvaluationHoldoutReport は、holdout 件数、指標、category および失敗 ID を保持する。
type EvaluationHoldoutReport struct {
	caseCount           int
	metrics             []EvaluationMetricReport
	categories          []EvaluationCategoryReport
	derivedObservations []EvaluationMetricReport
	failedCaseIDs       []string
}

// CaseCount は、holdout 件数を返す。
func (r EvaluationHoldoutReport) CaseCount() int { return r.caseCount }

// Metrics は、再現性を先頭にした固定順の指標を返す。
func (r EvaluationHoldoutReport) Metrics() []EvaluationMetricReport {
	return append([]EvaluationMetricReport{}, r.metrics...)
}

// Categories は、category ID 順の report を返す。
func (r EvaluationHoldoutReport) Categories() []EvaluationCategoryReport {
	return append([]EvaluationCategoryReport{}, r.categories...)
}

// DerivedObservations は、期待 meaning から導出した合成条件の固定指標を返す。
func (r EvaluationHoldoutReport) DerivedObservations() []EvaluationMetricReport {
	return append([]EvaluationMetricReport{}, r.derivedObservations...)
}

// FailedCaseIDs は、一つ以上の holdout 指標に失敗した case ID を manifest 順で返す。
func (r EvaluationHoldoutReport) FailedCaseIDs() []string {
	return append([]string{}, r.failedCaseIDs...)
}

// MarshalJSON は、holdout の機械可読な測定結果を返す。
func (r EvaluationHoldoutReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		CaseCount           int                        `json:"caseCount"`
		Metrics             []EvaluationMetricReport   `json:"metrics"`
		Categories          []EvaluationCategoryReport `json:"categories"`
		DerivedObservations []EvaluationMetricReport   `json:"derivedObservations"`
		FailedCaseIDs       []string                   `json:"failedCaseIds"`
	}{
		CaseCount:           r.CaseCount(),
		Metrics:             r.Metrics(),
		Categories:          r.Categories(),
		DerivedObservations: r.DerivedObservations(),
		FailedCaseIDs:       r.FailedCaseIDs(),
	})
}

// EvaluationCaseSetReport は、development 集合の件数を保持する。
type EvaluationCaseSetReport struct {
	caseCount int
}

// CaseCount は、集合の case 件数を返す。
func (r EvaluationCaseSetReport) CaseCount() int { return r.caseCount }

// MarshalJSON は、集合件数だけを返す。
func (r EvaluationCaseSetReport) MarshalJSON() ([]byte, error) {
	if r.caseCount < 0 {
		return nil, fmt.Errorf("caseCount は零以上でなければなりません")
	}
	return json.Marshal(struct {
		CaseCount int `json:"caseCount"`
	}{CaseCount: r.CaseCount()})
}

// EvaluationSetsReport は、三つの corpus 集合の測定結果を保持する。
type EvaluationSetsReport struct {
	development EvaluationCaseSetReport
	holdout     EvaluationHoldoutReport
	execution   ExecutionReport
}

// Development は、development 件数を返す。
func (r EvaluationSetsReport) Development() EvaluationCaseSetReport {
	return r.development
}

// Holdout は、holdout 評価を返す。
func (r EvaluationSetsReport) Holdout() EvaluationHoldoutReport {
	return r.holdout
}

// Execution は、execution 評価を返す。
func (r EvaluationSetsReport) Execution() ExecutionReport {
	return r.execution
}

// MarshalJSON は、三集合を固定順の object にする。
func (r EvaluationSetsReport) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Development EvaluationCaseSetReport `json:"development"`
		Holdout     EvaluationHoldoutReport `json:"holdout"`
		Execution   ExecutionReport         `json:"execution"`
	}{
		Development: r.Development(),
		Holdout:     r.Holdout(),
		Execution:   r.Execution(),
	})
}

// StandardReportValues は、標準評価 report の構築値を保持する。
type StandardReportValues struct {
	CorpusVersion        string
	HoldoutDigest        string
	ProfileSetID         string
	ProfileSetVersion    string
	RankingVersion       string
	ProfileVersions      []ProfileVersionReport
	BaselineVersion      string
	DevelopmentCaseCount int
	HoldoutCaseIDs       []string
	Semantic             SemanticReport
	Execution            ExecutionReport
	Reproducibility      EvaluationMetricReport
	DerivedObservations  []EvaluationMetricReport
}

// StandardReport は、標準 command と baseline が共有する純粋な測定文書である。
type StandardReport struct {
	corpusVersion   string
	holdoutDigest   string
	profileSet      ProfileSetReport
	baselineVersion string
	sets            EvaluationSetsReport
	initialized     bool
}

// NewStandardReport は、意味・実行・再現性の測定値を一つの固定 report にする。
func NewStandardReport(values StandardReportValues) (StandardReport, error) {
	profileSet := ProfileSetReport{
		profileSetID:      values.ProfileSetID,
		profileSetVersion: values.ProfileSetVersion,
		rankingVersion:    values.RankingVersion,
		profiles:          append([]ProfileVersionReport{}, values.ProfileVersions...),
	}
	holdout, err := newEvaluationHoldoutReport(
		values.HoldoutCaseIDs,
		values.Semantic,
		values.Reproducibility,
		values.DerivedObservations,
	)
	if err != nil {
		return StandardReport{}, err
	}
	report := StandardReport{
		corpusVersion:   values.CorpusVersion,
		holdoutDigest:   values.HoldoutDigest,
		profileSet:      profileSet,
		baselineVersion: values.BaselineVersion,
		sets: EvaluationSetsReport{
			development: EvaluationCaseSetReport{
				caseCount: values.DevelopmentCaseCount,
			},
			holdout:   holdout,
			execution: values.Execution,
		},
		initialized: true,
	}
	if err := report.Validate(); err != nil {
		return StandardReport{}, err
	}
	return report, nil
}

// CorpusVersion は、評価した corpus 版を返す。
func (r StandardReport) CorpusVersion() string { return r.corpusVersion }

// HoldoutDigest は、manifest の holdout digest を返す。
func (r StandardReport) HoldoutDigest() string { return r.holdoutDigest }

// ProfileSet は、評価した profile set の版を返す。
func (r StandardReport) ProfileSet() ProfileSetReport { return r.profileSet }

// BaselineVersion は、人手 review 済み baseline の版を返す。
func (r StandardReport) BaselineVersion() string { return r.baselineVersion }

// Sets は、三集合の測定結果を返す。
func (r StandardReport) Sets() EvaluationSetsReport { return r.sets }

// Validate は、標準 report の識別情報、件数および指標を確認する。
func (r StandardReport) Validate() error {
	if !r.initialized {
		return fmt.Errorf("StandardReport は NewStandardReport で作成しなければなりません")
	}
	if !standardCorpusVersionPattern.MatchString(r.corpusVersion) {
		return fmt.Errorf("corpusVersion が定義済みの形式ではありません")
	}
	if !standardDigestPattern.MatchString(r.holdoutDigest) {
		return fmt.Errorf("holdoutDigest は小文字十六進六十四桁でなければなりません")
	}
	if !standardBaselinePattern.MatchString(r.baselineVersion) {
		return fmt.Errorf("baselineVersion が定義済みの形式ではありません")
	}
	if err := r.profileSet.Validate(); err != nil {
		return fmt.Errorf("profileSet が有効ではありません: %w", err)
	}
	if r.sets.development.CaseCount() < 1 {
		return fmt.Errorf("development case は一件以上必要です")
	}
	if r.sets.holdout.CaseCount() < 1 {
		return fmt.Errorf("holdout case は一件以上必要です")
	}
	if err := validateEvaluationHoldoutReport(r.sets.holdout); err != nil {
		return fmt.Errorf("holdout report が有効ではありません: %w", err)
	}
	if err := r.sets.execution.Validate(); err != nil {
		return fmt.Errorf("execution report が有効ではありません: %w", err)
	}
	return nil
}

// MarshalJSON は、時刻、path、照会文および比較状態を含まない測定文書を返す。
func (r StandardReport) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		ArtifactKind    string               `json:"artifactKind"`
		SchemaVersion   int                  `json:"schemaVersion"`
		CorpusVersion   string               `json:"corpusVersion"`
		HoldoutDigest   string               `json:"holdoutDigest"`
		ProfileSet      ProfileSetReport     `json:"profileSet"`
		BaselineVersion string               `json:"baselineVersion"`
		Sets            EvaluationSetsReport `json:"sets"`
	}{
		ArtifactKind:    standardReportArtifactKind,
		SchemaVersion:   standardReportSchemaVersion,
		CorpusVersion:   r.CorpusVersion(),
		HoldoutDigest:   r.HoldoutDigest(),
		ProfileSet:      r.ProfileSet(),
		BaselineVersion: r.BaselineVersion(),
		Sets:            r.Sets(),
	})
}

// UnmarshalJSON は、baseline loader を介さない直接復元を拒否する。
func (*StandardReport) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"StandardReport は JSON から直接復元できません。baseline loader を使用してください",
	)
}

func newEvaluationHoldoutReport(
	caseIDs []string,
	semantic SemanticReport,
	reproducibility EvaluationMetricReport,
	derivedObservations []EvaluationMetricReport,
) (EvaluationHoldoutReport, error) {
	if !semantic.initialized || semantic.CaseCount() < 1 {
		return EvaluationHoldoutReport{}, fmt.Errorf("semantic report が初期化されていません")
	}
	if len(caseIDs) != semantic.CaseCount() {
		return EvaluationHoldoutReport{}, fmt.Errorf(
			"holdout case ID 件数と semantic report が一致しません",
		)
	}
	if err := validateOrderedCaseIDs(caseIDs); err != nil {
		return EvaluationHoldoutReport{}, err
	}
	if err := reproducibility.Validate(); err != nil {
		return EvaluationHoldoutReport{}, fmt.Errorf(
			"plan reproducibility が有効ではありません: %w",
			err,
		)
	}
	planCaseCount, err := semanticMetricDenominator(
		semantic.Metrics(),
		SemanticMetricPlanOutcome,
	)
	if err != nil {
		return EvaluationHoldoutReport{}, err
	}
	if reproducibility.MetricID() != "plan-reproducibility" ||
		reproducibility.Denominator() != planCaseCount {
		return EvaluationHoldoutReport{}, fmt.Errorf(
			"plan reproducibility の指標 ID または母集団が不正です",
		)
	}
	metrics := []EvaluationMetricReport{reproducibility}
	for _, metric := range semantic.Metrics() {
		converted, err := evaluationMetricFromSemantic(metric)
		if err != nil {
			return EvaluationHoldoutReport{}, err
		}
		metrics = append(metrics, converted)
	}
	categories := make([]EvaluationCategoryReport, 0, len(semantic.Categories()))
	for _, category := range semantic.Categories() {
		converted, err := evaluationCategoryFromSemantic(category)
		if err != nil {
			return EvaluationHoldoutReport{}, err
		}
		categories = append(categories, converted)
	}
	failed := orderedUnionCaseIDs(
		caseIDs,
		reproducibility.FailedCaseIDs(),
		semantic.FailedCaseIDs(),
		derivedObservationFailedCaseIDs(derivedObservations),
	)
	return EvaluationHoldoutReport{
		caseCount:  semantic.CaseCount(),
		metrics:    metrics,
		categories: categories,
		derivedObservations: append(
			[]EvaluationMetricReport{},
			derivedObservations...,
		),
		failedCaseIDs: failed,
	}, nil
}

func evaluationMetricFromSemantic(
	metric SemanticMetricReport,
) (EvaluationMetricReport, error) {
	return NewEvaluationMetricReport(EvaluationMetricReportValues{
		MetricID:      string(metric.MetricID()),
		Numerator:     metric.Matched(),
		Denominator:   metric.Total(),
		FailedCaseIDs: metric.FailedCaseIDs(),
	})
}

func evaluationCategoryFromSemantic(
	category SemanticCategoryReport,
) (EvaluationCategoryReport, error) {
	metrics := make([]EvaluationMetricReport, 0, len(category.Metrics()))
	for _, metric := range category.Metrics() {
		converted, err := evaluationMetricFromSemantic(metric)
		if err != nil {
			return EvaluationCategoryReport{}, err
		}
		metrics = append(metrics, converted)
	}
	return EvaluationCategoryReport{
		categoryID: category.CategoryID(),
		caseCount:  category.CaseCount(),
		metrics:    metrics,
	}, nil
}

func validateEvaluationHoldoutReport(
	report EvaluationHoldoutReport,
) error {
	if report.caseCount < 1 {
		return fmt.Errorf("holdout caseCount は一件以上でなければなりません")
	}
	expectedMetricIDs := append(
		[]string{"plan-reproducibility"},
		semanticMetricIDStrings()...,
	)
	if err := validateEvaluationMetrics(
		report.metrics,
		expectedMetricIDs,
		report.caseCount,
	); err != nil {
		return err
	}
	if report.metrics[0].Denominator() !=
		report.metrics[1].Denominator() {
		return fmt.Errorf(
			"plan reproducibility の母集団は plan-outcome と一致しなければなりません",
		)
	}
	if err := validateEvaluationMetrics(
		report.derivedObservations,
		derivedObservationIDs(),
		report.caseCount,
	); err != nil {
		return fmt.Errorf("derived observations が有効ではありません: %w", err)
	}
	for _, observation := range report.derivedObservations {
		if observation.Denominator() == 0 {
			return fmt.Errorf("derived observation の分母は一件以上必要です")
		}
	}
	previousCategory := ""
	for _, category := range report.categories {
		if !evaluationMetricIDPattern.MatchString(category.categoryID) {
			return fmt.Errorf("categoryId が定義済みの形式ではありません")
		}
		if previousCategory != "" &&
			category.categoryID <= previousCategory {
			return fmt.Errorf("categories は categoryId 昇順でなければなりません")
		}
		if category.caseCount < 1 ||
			category.caseCount > report.caseCount {
			return fmt.Errorf("category caseCount が holdout 件数を超えています")
		}
		if err := validateEvaluationMetrics(
			category.metrics,
			semanticMetricIDStrings(),
			category.caseCount,
		); err != nil {
			return err
		}
		previousCategory = category.categoryID
	}
	return validateFailedCaseIDs(
		report.failedCaseIDs,
		report.caseCount,
	)
}

func derivedObservationFailedCaseIDs(
	observations []EvaluationMetricReport,
) []string {
	caseIDs := make([]string, 0)
	for _, observation := range observations {
		caseIDs = append(caseIDs, observation.FailedCaseIDs()...)
	}
	return caseIDs
}

func semanticMetricDenominator(
	metrics []SemanticMetricReport,
	metricID SemanticMetricID,
) (int, error) {
	for _, metric := range metrics {
		if metric.MetricID() == metricID {
			return metric.Total(), nil
		}
	}
	return 0, fmt.Errorf("semantic report に指標 %s がありません", metricID)
}

func validateEvaluationMetrics(
	metrics []EvaluationMetricReport,
	expectedIDs []string,
	maximumDenominator int,
) error {
	if len(metrics) != len(expectedIDs) {
		return fmt.Errorf("metrics が固定指標数と一致しません")
	}
	for index, expectedID := range expectedIDs {
		metric := metrics[index]
		if err := metric.Validate(); err != nil {
			return err
		}
		if metric.MetricID() != expectedID {
			return fmt.Errorf("metrics が固定順と一致しません")
		}
		if metric.Denominator() > maximumDenominator {
			return fmt.Errorf("metric の分母が caseCount を超えています")
		}
	}
	return nil
}

func semanticMetricIDStrings() []string {
	ids := semanticMetricIDs()
	values := make([]string, 0, len(ids))
	for _, metricID := range ids {
		values = append(values, string(metricID))
	}
	return values
}

func validateFailedCaseIDs(
	caseIDs []string,
	maximum int,
) error {
	if caseIDs == nil {
		return fmt.Errorf("failedCaseIds は必須です")
	}
	if len(caseIDs) > maximum {
		return fmt.Errorf("failedCaseIds が caseCount を超えています")
	}
	return validateOrderedCaseIDs(caseIDs)
}

func validateOrderedCaseIDs(caseIDs []string) error {
	seen := make(map[string]struct{}, len(caseIDs))
	for _, caseID := range caseIDs {
		if caseID == "" {
			return fmt.Errorf("holdout case ID は空にできません")
		}
		if _, exists := seen[caseID]; exists {
			return fmt.Errorf("holdout case ID を重複させられません")
		}
		seen[caseID] = struct{}{}
	}
	return nil
}

func orderedUnionCaseIDs(
	order []string,
	groups ...[]string,
) []string {
	included := make(map[string]struct{})
	for _, group := range groups {
		for _, caseID := range group {
			included[caseID] = struct{}{}
		}
	}
	values := make([]string, 0, len(included))
	for _, caseID := range order {
		if _, exists := included[caseID]; exists {
			values = append(values, caseID)
			delete(included, caseID)
		}
	}
	if len(included) > 0 {
		extra := make([]string, 0, len(included))
		for caseID := range included {
			extra = append(extra, caseID)
		}
		slices.Sort(extra)
		values = append(values, extra...)
	}
	return values
}

func validateEvaluationVersion(field, value string) error {
	if !utf8.ValidString(value) ||
		len(value) < 1 ||
		len(value) > 128 ||
		strings.TrimSpace(value) != value {
		return fmt.Errorf("%s が有効な版文字列ではありません", field)
	}
	for _, current := range value {
		if unicode.IsControl(current) {
			return fmt.Errorf("%s に制御文字を含められません", field)
		}
	}
	return nil
}
