package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"unicode/utf8"
)

const (
	maximumResultValues = 8192
	maximumReportValues = 65536
	maximumResultDepth  = 16
	maximumReportDepth  = 12
)

type candidateReportDocument struct {
	ArtifactKind    string                    `json:"artifactKind"`
	SchemaVersion   *int                      `json:"schemaVersion"`
	CorpusVersion   string                    `json:"corpusVersion"`
	HoldoutDigest   string                    `json:"holdoutDigest"`
	ProfileSet      candidateReportProfileSet `json:"profileSet"`
	BaselineVersion string                    `json:"baselineVersion"`
	Sets            candidateReportSets       `json:"sets"`
}

type candidateReportProfileSet struct {
	ProfileSetID      string                   `json:"profileSetId"`
	ProfileSetVersion string                   `json:"profileSetVersion"`
	RankingVersion    string                   `json:"rankingVersion"`
	Profiles          []candidateReportProfile `json:"profiles"`
}

type candidateReportProfile struct {
	ProfileID      string `json:"profileId"`
	ProfileVersion string `json:"profileVersion"`
}

type candidateReportSets struct {
	Development candidateReportDevelopment `json:"development"`
	Holdout     candidateReportHoldout     `json:"holdout"`
	Execution   candidateReportExecution   `json:"execution"`
}

type candidateReportDevelopment struct {
	CaseCount *int `json:"caseCount"`
}

type candidateReportHoldout struct {
	CaseCount           *int                      `json:"caseCount"`
	Metrics             []candidateReportMetric   `json:"metrics"`
	Categories          []candidateReportCategory `json:"categories"`
	DerivedObservations []candidateReportMetric   `json:"derivedObservations"`
	FailedCaseIDs       []string                  `json:"failedCaseIds"`
}

type candidateReportCategory struct {
	CategoryID string                  `json:"categoryId"`
	CaseCount  *int                    `json:"caseCount"`
	Metrics    []candidateReportMetric `json:"metrics"`
}

type candidateReportMetric struct {
	MetricID      string   `json:"metricId"`
	Numerator     *int     `json:"numerator"`
	Denominator   *int     `json:"denominator"`
	Ratio         *float64 `json:"ratio"`
	FailedCaseIDs []string `json:"failedCaseIds"`
}

type candidateReportExecution struct {
	CaseCount                  *int                    `json:"caseCount"`
	Metrics                    []candidateReportMetric `json:"metrics"`
	WrongResourceCallCount     *int                    `json:"wrongResourceCallCount"`
	BudgetViolationCount       *int                    `json:"budgetViolationCount"`
	AttemptOrderViolationCount *int                    `json:"attemptOrderViolationCount"`
	ImplicitFirstReadCount     *int                    `json:"implicitFirstReadCount"`
	EmptyReclassificationCount *int                    `json:"emptyReclassificationCount"`
	FailedCaseIDs              []string                `json:"failedCaseIds"`
}

func decodeCanonicalWorkerResult(raw []byte) (workerResultDocument, error) {
	var document workerResultDocument
	if err := decodeClosedCanonicalJSON(
		raw,
		maximumResultDepth,
		maximumResultValues,
		&document,
	); err != nil {
		return workerResultDocument{}, fmt.Errorf("candidate result が不正です")
	}
	if document.ArtifactKind != "legal_query_candidate_evaluation_result" ||
		document.SchemaVersion != 2 ||
		!validEvaluationID(document.EvaluationID) ||
		len(document.RequestSHA256) != 64 || !lowerHex(document.RequestSHA256) ||
		(document.Outcome != "passed" && document.Outcome != "failed") ||
		len(document.ReportSHA256) != 64 || !lowerHex(document.ReportSHA256) {
		return workerResultDocument{}, fmt.Errorf("candidate result が不正です")
	}
	return document, nil
}

func validateCanonicalCandidateReport(raw []byte) error {
	var document candidateReportDocument
	if err := decodeClosedCanonicalJSON(
		raw,
		maximumReportDepth,
		maximumReportValues,
		&document,
	); err != nil {
		return fmt.Errorf("candidate report が不正です")
	}
	if err := validateCandidateReportDocument(document); err != nil {
		return fmt.Errorf("candidate report が不正です")
	}
	return nil
}

func decodeClosedCanonicalJSON(
	raw []byte,
	maximumDepth int,
	maximumValues int,
	target any,
) error {
	if err := inspectClosedJSONObject(raw, maximumDepth, maximumValues); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		return fmt.Errorf("JSON の終端が不正です")
	}
	canonical, err := json.Marshal(target)
	if err != nil {
		return err
	}
	canonical = append(canonical, '\n')
	if !bytes.Equal(raw, canonical) {
		return fmt.Errorf("JSON が canonical byte ではありません")
	}
	return nil
}

func inspectClosedJSONObject(raw []byte, maximumDepth, maximumValues int) error {
	if len(raw) == 0 || !utf8.Valid(raw) || maximumDepth < 1 || maximumValues < 1 {
		return fmt.Errorf("JSON の入力境界が不正です")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return fmt.Errorf("JSON の最上位は object でなければなりません")
	}
	valueCount := 0
	if err := inspectJSONValue(decoder, first, 1, maximumDepth, maximumValues, &valueCount); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("JSON の後方 token が不正です")
	}
	return nil
}

func inspectJSONValue(
	decoder *json.Decoder,
	value json.Token,
	depth int,
	maximumDepth int,
	maximumValues int,
	valueCount *int,
) error {
	*valueCount++
	if *valueCount > maximumValues {
		return fmt.Errorf("JSON value 数が上限を超えています")
	}
	if value == nil {
		return fmt.Errorf("JSON に null は使用できません")
	}
	delimiter, collection := value.(json.Delim)
	if !collection {
		return nil
	}
	if depth > maximumDepth {
		return fmt.Errorf("JSON depth が上限を超えています")
	}
	switch delimiter {
	case '{':
		return inspectJSONObjectEntries(
			decoder, depth, maximumDepth, maximumValues, valueCount,
		)
	case '[':
		return inspectJSONArrayEntries(
			decoder, depth, maximumDepth, maximumValues, valueCount,
		)
	default:
		return fmt.Errorf("JSON collection が不正です")
	}
}

func inspectJSONObjectEntries(
	decoder *json.Decoder,
	depth int,
	maximumDepth int,
	maximumValues int,
	valueCount *int,
) error {
	keys := make(map[string]struct{})
	for decoder.More() {
		keyToken, err := decoder.Token()
		key, ok := keyToken.(string)
		if err != nil || !ok {
			return fmt.Errorf("JSON object key が不正です")
		}
		if _, duplicate := keys[key]; duplicate {
			return fmt.Errorf("JSON object key が重複しています")
		}
		keys[key] = struct{}{}
		child, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("JSON object value が不正です")
		}
		if err := inspectJSONValue(
			decoder, child, depth+1, maximumDepth, maximumValues, valueCount,
		); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim('}') {
		return fmt.Errorf("JSON object の終端が不正です")
	}
	return nil
}

func inspectJSONArrayEntries(
	decoder *json.Decoder,
	depth int,
	maximumDepth int,
	maximumValues int,
	valueCount *int,
) error {
	for decoder.More() {
		child, err := decoder.Token()
		if err != nil {
			return fmt.Errorf("JSON array value が不正です")
		}
		if err := inspectJSONValue(
			decoder, child, depth+1, maximumDepth, maximumValues, valueCount,
		); err != nil {
			return err
		}
	}
	end, err := decoder.Token()
	if err != nil || end != json.Delim(']') {
		return fmt.Errorf("JSON array の終端が不正です")
	}
	return nil
}

func validateCandidateReportDocument(document candidateReportDocument) error {
	if document.ArtifactKind != "legal_query_evaluation" ||
		document.SchemaVersion == nil || *document.SchemaVersion != 1 ||
		!validPrefixedPositiveNumber(document.CorpusVersion, "corpus-v", 64) ||
		len(document.HoldoutDigest) != 64 || !lowerHex(document.HoldoutDigest) ||
		!validPrefixedPositiveNumber(document.BaselineVersion, "default-", 64) {
		return fmt.Errorf("report identity が不正です")
	}
	if err := validateCandidateReportProfiles(document.ProfileSet); err != nil {
		return err
	}
	if document.Sets.Development.CaseCount == nil ||
		*document.Sets.Development.CaseCount < 1 ||
		*document.Sets.Development.CaseCount > 400 {
		return fmt.Errorf("development caseCount が不正です")
	}
	if err := validateCandidateReportHoldout(document.Sets.Holdout); err != nil {
		return err
	}
	return validateCandidateReportExecution(document.Sets.Execution)
}

func validateCandidateReportProfiles(profileSet candidateReportProfileSet) error {
	if !validSegmentedID(profileSet.ProfileSetID, 64) ||
		!validASCIIReportVersion(profileSet.ProfileSetVersion) ||
		!validASCIIReportVersion(profileSet.RankingVersion) ||
		len(profileSet.Profiles) < 1 || len(profileSet.Profiles) > 16 {
		return fmt.Errorf("profileSet が不正です")
	}
	previous := ""
	for _, profile := range profileSet.Profiles {
		if !validSegmentedID(profile.ProfileID, 64) ||
			!validASCIIReportVersion(profile.ProfileVersion) ||
			(previous != "" && profile.ProfileID <= previous) {
			return fmt.Errorf("profile が不正です")
		}
		previous = profile.ProfileID
	}
	return nil
}

func validateCandidateReportHoldout(holdout candidateReportHoldout) error {
	if holdout.CaseCount == nil || *holdout.CaseCount < 1 || *holdout.CaseCount > 400 {
		return fmt.Errorf("holdout caseCount が不正です")
	}
	caseCount := *holdout.CaseCount
	holdoutMetricIDs := append([]string{"plan-reproducibility"}, semanticReportMetricIDs()...)
	if err := validateCandidateReportMetrics(holdout.Metrics, holdoutMetricIDs, caseCount); err != nil {
		return err
	}
	if *holdout.Metrics[0].Denominator != *holdout.Metrics[1].Denominator {
		return fmt.Errorf("plan metric の母集団が不正です")
	}
	if err := validateCandidateReportCategories(holdout.Categories, caseCount); err != nil {
		return err
	}
	if err := validateCandidateReportMetrics(
		holdout.DerivedObservations,
		derivedReportMetricIDs(),
		caseCount,
	); err != nil {
		return err
	}
	for _, metric := range holdout.DerivedObservations {
		if *metric.Denominator == 0 {
			return fmt.Errorf("derived observation の分母が不正です")
		}
	}
	return validateCandidateReportCaseIDs(holdout.FailedCaseIDs, caseCount)
}

func validateCandidateReportCategories(categories []candidateReportCategory, maximum int) error {
	if len(categories) < 1 || len(categories) > 12 {
		return fmt.Errorf("categories 件数が不正です")
	}
	previous := ""
	for _, category := range categories {
		if !validSegmentedID(category.CategoryID, 64) ||
			(previous != "" && category.CategoryID <= previous) ||
			category.CaseCount == nil || *category.CaseCount < 1 ||
			*category.CaseCount > maximum {
			return fmt.Errorf("category が不正です")
		}
		if err := validateCandidateReportMetrics(
			category.Metrics,
			semanticReportMetricIDs(),
			*category.CaseCount,
		); err != nil {
			return err
		}
		previous = category.CategoryID
	}
	return nil
}

func validateCandidateReportExecution(execution candidateReportExecution) error {
	if execution.CaseCount == nil || *execution.CaseCount < 1 || *execution.CaseCount > 400 {
		return fmt.Errorf("execution caseCount が不正です")
	}
	if err := validateCandidateReportMetrics(
		execution.Metrics,
		executionReportMetricIDs(),
		*execution.CaseCount,
	); err != nil {
		return err
	}
	for _, metric := range execution.Metrics {
		if *metric.Denominator != *execution.CaseCount {
			return fmt.Errorf("execution metric の母集団が不正です")
		}
	}
	counts := []*int{
		execution.WrongResourceCallCount,
		execution.BudgetViolationCount,
		execution.AttemptOrderViolationCount,
		execution.ImplicitFirstReadCount,
		execution.EmptyReclassificationCount,
	}
	for _, count := range counts {
		if count == nil || *count < 0 {
			return fmt.Errorf("execution 違反件数が不正です")
		}
	}
	return validateCandidateReportCaseIDs(execution.FailedCaseIDs, *execution.CaseCount)
}

func validateCandidateReportMetrics(
	metrics []candidateReportMetric,
	expectedIDs []string,
	maximumDenominator int,
) error {
	if len(metrics) != len(expectedIDs) {
		return fmt.Errorf("metric 件数が不正です")
	}
	for index, metric := range metrics {
		if err := validateCandidateReportMetric(
			metric,
			expectedIDs[index],
			maximumDenominator,
		); err != nil {
			return err
		}
	}
	return nil
}

func validateCandidateReportMetric(
	metric candidateReportMetric,
	expectedID string,
	maximumDenominator int,
) error {
	if metric.MetricID != expectedID || metric.Numerator == nil ||
		metric.Denominator == nil || metric.Ratio == nil ||
		*metric.Numerator < 0 || *metric.Denominator < 0 ||
		*metric.Numerator > *metric.Denominator ||
		*metric.Denominator > maximumDenominator ||
		*metric.Denominator > 400 || math.Signbit(*metric.Ratio) {
		return fmt.Errorf("metric が不正です")
	}
	wantRatio := float64(0)
	if *metric.Denominator > 0 {
		wantRatio = float64(*metric.Numerator) / float64(*metric.Denominator)
	}
	if *metric.Ratio != wantRatio ||
		len(metric.FailedCaseIDs) != *metric.Denominator-*metric.Numerator {
		return fmt.Errorf("metric の計算値が不正です")
	}
	return validateCandidateReportCaseIDs(metric.FailedCaseIDs, *metric.Denominator)
}

func validateCandidateReportCaseIDs(caseIDs []string, maximum int) error {
	if caseIDs == nil || len(caseIDs) > maximum || len(caseIDs) > 46800 {
		return fmt.Errorf("failedCaseIds が不正です")
	}
	seen := make(map[string]struct{}, len(caseIDs))
	for _, caseID := range caseIDs {
		if !validLooseReportID(caseID, 64) {
			return fmt.Errorf("caseId が不正です")
		}
		if _, duplicate := seen[caseID]; duplicate {
			return fmt.Errorf("caseId が重複しています")
		}
		seen[caseID] = struct{}{}
	}
	return nil
}

func semanticReportMetricIDs() []string {
	return []string{
		"plan-outcome",
		"request-error",
		"meaning-signature",
		"top-1",
		"top-2",
		"high-confidence-precision",
		"evidence-assertion",
		"concept-assertion",
	}
}

func derivedReportMetricIDs() []string {
	return []string{
		"composition-core-pack",
		"composition-pack-disabled",
		"composition-ref-read-search",
		"composition-four-step-budget",
	}
}

func executionReportMetricIDs() []string {
	return []string{
		"expected-execution",
		"no-wrong-resource-call",
		"budget-adherence",
		"attempt-order-determinism",
		"no-implicit-first-read",
		"no-empty-reclassification",
	}
}

func validEvaluationID(value string) bool {
	const prefix = "evaluation-sha256-"
	return len(value) == len(prefix)+64 &&
		bytes.HasPrefix([]byte(value), []byte(prefix)) &&
		lowerHex(value[len(prefix):])
}

func validPrefixedPositiveNumber(value, prefix string, maximum int) bool {
	if len(value) <= len(prefix) || len(value) > maximum ||
		!bytes.HasPrefix([]byte(value), []byte(prefix)) {
		return false
	}
	digits := value[len(prefix):]
	if digits[0] < '1' || digits[0] > '9' {
		return false
	}
	for index := 1; index < len(digits); index++ {
		if digits[index] < '0' || digits[index] > '9' {
			return false
		}
	}
	return true
}

func validASCIIReportVersion(value string) bool {
	if len(value) < 1 || len(value) > 128 || !asciiLowerDigit(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !asciiLowerDigit(value[index]) && value[index] != '-' && value[index] != '.' {
			return false
		}
	}
	return true
}

func validSegmentedID(value string, maximum int) bool {
	if !validLooseReportID(value, maximum) || value[len(value)-1] == '-' {
		return false
	}
	for index := 1; index < len(value); index++ {
		if value[index] == '-' && value[index-1] == '-' {
			return false
		}
	}
	return true
}

func validLooseReportID(value string, maximum int) bool {
	if len(value) < 1 || len(value) > maximum || !asciiLowerDigit(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !asciiLowerDigit(value[index]) && value[index] != '-' {
			return false
		}
	}
	return true
}

func asciiLowerDigit(value byte) bool {
	return (value >= 'a' && value <= 'z') || (value >= '0' && value <= '9')
}
