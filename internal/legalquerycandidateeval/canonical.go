package legalquerycandidateeval

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"
)

const (
	candidateContentIDPrefix  = "candidate-content-sha256-"
	reviewAttestationIDPrefix = "review-sha256-"
	evaluationIDPrefix        = "evaluation-sha256-"
)

// MarshalCanonicalJSON は schema field 順の一行 JSON と末尾 LF を返す。
func MarshalCanonicalJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("canonical JSON を直列化できません: %w", err)
	}
	return append(raw, '\n'), nil
}

// RawSHA256 は原 byte の小文字 SHA-256 を返す。
func RawSHA256(raw []byte) string {
	digest := sha256.Sum256(raw)
	return hex.EncodeToString(digest[:])
}

// CanonicalCandidateContentID は ID 自身を除く manifest tuple の ID を返す。
func CanonicalCandidateContentID(document CandidateContentManifest) (string, error) {
	digest, err := canonicalStructDigest(document, "candidateContentId")
	if err != nil {
		return "", err
	}
	return candidateContentIDPrefix + digest, nil
}

// CanonicalReviewAttestationID は ID 自身を除く review tuple の ID を返す。
func CanonicalReviewAttestationID(document ReviewAttestation) (string, error) {
	digest, err := canonicalStructDigest(document, "attestationId")
	if err != nil {
		return "", err
	}
	return reviewAttestationIDPrefix + digest, nil
}

// CanonicalEvaluationID は ID 自身を除く evaluation tuple の ID を返す。
func CanonicalEvaluationID(document EvaluationRequest) (string, error) {
	digest, err := canonicalStructDigest(document, "evaluationId")
	if err != nil {
		return "", err
	}
	return evaluationIDPrefix + digest, nil
}

// CanonicalCompositionSHA256 は digest 自身を除く composition digest を返す。
func CanonicalCompositionSHA256(document CompositionDescriptor) (string, error) {
	return canonicalStructDigest(document, "descriptorSha256")
}

// CanonicalSourceSetSHA256 は digest 自身を除く source set digest を返す。
func CanonicalSourceSetSHA256(document SemanticSourceSet) (string, error) {
	return canonicalStructDigest(document, "sourceSetSha256")
}

// LexiconAggregateSHA256 は path、space、digest、LF の連結 digest を返す。
func LexiconAggregateSHA256(files []FileDigest) string {
	hash := sha256.New()
	for _, file := range files {
		_, _ = hash.Write([]byte(file.Path))
		_, _ = hash.Write([]byte{' '})
		_, _ = hash.Write([]byte(file.RawSHA256))
		_, _ = hash.Write([]byte{'\n'})
	}
	return hex.EncodeToString(hash.Sum(nil))
}

// SOTSetSHA256 は規定順 SOT reference 配列の canonical digest を返す。
func SOTSetSHA256(references []SOTReference) string {
	encoded, err := canonicalBytes(references, "")
	if err != nil {
		return ""
	}
	return RawSHA256(encoded)
}

// ReviewRubricSHA256 は固定 rubric v1 の canonical digest を返す。
func ReviewRubricSHA256() string {
	encoded, err := canonicalBytes(reviewRubricV1(), "")
	if err != nil {
		return ""
	}
	return RawSHA256(encoded)
}

func canonicalStructDigest(value any, excludedField string) (string, error) {
	encoded, err := canonicalBytes(value, excludedField)
	if err != nil {
		return "", err
	}
	return RawSHA256(encoded), nil
}

func canonicalBytes(value any, excludedRootField string) ([]byte, error) {
	var target bytes.Buffer
	if err := writeCanonicalValue(&target, reflect.ValueOf(value), excludedRootField, true); err != nil {
		return nil, fmt.Errorf("canonical tuple を直列化できません: %w", err)
	}
	return target.Bytes(), nil
}

func writeCanonicalValue(
	target *bytes.Buffer,
	value reflect.Value,
	excludedRootField string,
	isRoot bool,
) error {
	if !value.IsValid() {
		return fmt.Errorf("無効な値は使用できません")
	}
	switch value.Kind() {
	case reflect.String:
		writeCanonicalString(target, value.String())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		writeCanonicalInteger(target, value.Int())
	case reflect.Slice, reflect.Array:
		return writeCanonicalArray(target, value)
	case reflect.Struct:
		return writeCanonicalObject(target, value, excludedRootField, isRoot)
	default:
		return fmt.Errorf("canonical 入力に %s は使用できません", value.Kind())
	}
	return nil
}

func writeCanonicalArray(target *bytes.Buffer, value reflect.Value) error {
	if value.Kind() == reflect.Slice && value.IsNil() {
		return fmt.Errorf("nil 配列は使用できません")
	}
	target.WriteByte('a')
	target.WriteString(strconv.Itoa(value.Len()))
	target.WriteByte('[')
	for index := range value.Len() {
		if err := writeCanonicalValue(target, value.Index(index), "", false); err != nil {
			return err
		}
	}
	target.WriteByte(']')
	return nil
}

func writeCanonicalObject(
	target *bytes.Buffer,
	value reflect.Value,
	excludedRootField string,
	isRoot bool,
) error {
	fields, err := canonicalFields(value, excludedRootField, isRoot)
	if err != nil {
		return err
	}
	target.WriteByte('o')
	target.WriteString(strconv.Itoa(len(fields)))
	target.WriteByte('{')
	for _, field := range fields {
		writeCanonicalString(target, field.name)
		if err := writeCanonicalValue(target, field.value, "", false); err != nil {
			return err
		}
	}
	target.WriteByte('}')
	return nil
}

type canonicalField struct {
	name  string
	value reflect.Value
}

func canonicalFields(
	value reflect.Value,
	excludedRootField string,
	isRoot bool,
) ([]canonicalField, error) {
	fields := make([]canonicalField, 0, value.NumField())
	typeOfValue := value.Type()
	for index := range value.NumField() {
		fieldType := typeOfValue.Field(index)
		if !fieldType.IsExported() {
			continue
		}
		name := strings.Split(fieldType.Tag.Get("json"), ",")[0]
		if name == "" || name == "-" {
			return nil, fmt.Errorf("field %s に JSON 名がありません", fieldType.Name)
		}
		if isRoot && name == excludedRootField {
			continue
		}
		fields = append(fields, canonicalField{name: name, value: value.Field(index)})
	}
	return fields, nil
}

func writeCanonicalString(target *bytes.Buffer, value string) {
	target.WriteByte('s')
	target.WriteString(strconv.Itoa(len([]byte(value))))
	target.WriteByte(':')
	target.WriteString(value)
	target.WriteByte(';')
}

func writeCanonicalInteger(target *bytes.Buffer, value int64) {
	encoded := strconv.FormatInt(value, 10)
	target.WriteByte('i')
	target.WriteString(strconv.Itoa(len(encoded)))
	target.WriteByte(':')
	target.WriteString(encoded)
	target.WriteByte(';')
}

type rubricDocument struct {
	RubricVersion                   string              `json:"rubricVersion"`
	MinimumScore100                 int                 `json:"minimumScore100"`
	MaximumScore100                 int                 `json:"maximumScore100"`
	MinimumApprovedCriterionScore20 int                 `json:"minimumApprovedCriterionScore20"`
	BlockerMaximum                  int                 `json:"blockerMaximum"`
	RequiredDecision                string              `json:"requiredDecision"`
	AllowedCriterionScores          []int               `json:"allowedCriterionScores"`
	ScoreAnchors                    []rubricScoreAnchor `json:"scoreAnchors"`
	ArchitectureCriteria            []rubricCriterion   `json:"architectureCriteria"`
	TestabilityCriteria             []rubricCriterion   `json:"testabilityCriteria"`
}

type rubricScoreAnchor struct {
	Score20 int    `json:"score20"`
	Meaning string `json:"meaning"`
}

type rubricCriterion struct {
	CriterionID string `json:"criterionId"`
	Question    string `json:"question"`
}

func reviewRubricV1() rubricDocument {
	return rubricDocument{
		RubricVersion:                   ReviewRubricVersion,
		MinimumScore100:                 80,
		MaximumScore100:                 100,
		MinimumApprovedCriterionScore20: 16,
		BlockerMaximum:                  0,
		RequiredDecision:                ReviewDecisionApproved,
		AllowedCriterionScores:          []int{0, 10, 16, 20},
		ScoreAnchors: []rubricScoreAnchor{
			{Score20: 0, Meaning: "missing-or-blocking-contradiction"},
			{Score20: 10, Meaning: "intent-present-but-major-choice-open"},
			{Score20: 16, Meaning: "deterministic-and-testable-with-nonblocking-gaps"},
			{Score20: 20, Meaning: "closed-ownership-boundary-failure-and-verification"},
		},
		ArchitectureCriteria: rubricArchitectureCriteria(),
		TestabilityCriteria:  rubricTestabilityCriteria(),
	}
}

func rubricArchitectureCriteria() []rubricCriterion {
	return []rubricCriterion{
		{CriterionID: "single-sot-ownership", Question: "is-each-fact-owned-once"},
		{CriterionID: "dependency-direction", Question: "are-active-dependencies-current-and-acyclic"},
		{CriterionID: "lifecycle-and-successor", Question: "are-replacements-versioned-and-predecessors-preserved"},
		{CriterionID: "provider-independence", Question: "is-semantic-planning-free-of-provider-runtime-state"},
		{CriterionID: "rollout-boundary", Question: "are-preparation-evaluation-adoption-and-publication-separated"},
	}
}

func rubricTestabilityCriteria() []rubricCriterion {
	return []rubricCriterion{
		{CriterionID: "deterministic-input", Question: "are-all-semantic-inputs-content-bound-and-ordered"},
		{CriterionID: "closed-failure-unit", Question: "does-each-invalid-or-ambiguous-input-have-one-failure-unit"},
		{CriterionID: "resource-bounds", Question: "are-count-byte-expansion-path-and-cancellation-bounds-closed"},
		{CriterionID: "replay-and-identity", Question: "can-current-validation-and-historical-replay-be-distinguished"},
		{CriterionID: "fixed-verification", Question: "do-fixed-tests-cover-normal-maximum-plus-one-and-conflict-cases"},
	}
}

// ArchitectureCriterionIDs は architecture rubric の固定順 ID を複製して返す。
func ArchitectureCriterionIDs() []string {
	return criterionIDs(rubricArchitectureCriteria())
}

// TestabilityCriterionIDs は testability rubric の固定順 ID を複製して返す。
func TestabilityCriterionIDs() []string {
	return criterionIDs(rubricTestabilityCriteria())
}

func criterionIDs(criteria []rubricCriterion) []string {
	ids := make([]string, 0, len(criteria))
	for _, criterion := range criteria {
		ids = append(ids, criterion.CriterionID)
	}
	return ids
}
