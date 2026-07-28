package legalconceptlexicon

import (
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

const (
	maxDatasetBytes   = 1 << 20
	maxEntryCount     = 256
	maxStringBytes    = 2048
	maxCandidateCount = 4
)

var identifierPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

func validateDataset(value dataset) error {
	if value.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf(
			"schemaVersion = %d, want %d",
			value.SchemaVersion,
			supportedSchemaVersion,
		)
	}
	if err := validateIdentifier("lexiconVersion", value.LexiconVersion); err != nil {
		return err
	}
	generatedAt, err := time.Parse(time.RFC3339, value.GeneratedAt)
	if err != nil {
		return fmt.Errorf("generatedAt は RFC3339 でなければなりません")
	}
	if _, offset := generatedAt.Zone(); offset != 0 {
		return fmt.Errorf("generatedAt は UTC でなければなりません")
	}
	if len(value.Entries) == 0 || len(value.Entries) > maxEntryCount {
		return fmt.Errorf(
			"entries は 1 件以上 %d 件以下でなければなりません",
			maxEntryCount,
		)
	}

	seenConceptIDs := make(map[string]struct{}, len(value.Entries))
	termOwners := make(map[string]string)
	normalizedOwners := make(map[string]string)
	previousConceptID := ""
	for index, entry := range value.Entries {
		if err := validateEntry(entry); err != nil {
			return fmt.Errorf("entries[%d]: %w", index, err)
		}
		if _, exists := seenConceptIDs[entry.ConceptID]; exists {
			return fmt.Errorf("conceptId %q が重複しています", entry.ConceptID)
		}
		seenConceptIDs[entry.ConceptID] = struct{}{}
		if previousConceptID != "" && previousConceptID >= entry.ConceptID {
			return fmt.Errorf("entries は conceptId の昇順でなければなりません")
		}
		previousConceptID = entry.ConceptID
		for _, term := range entry.Terms {
			if err := registerCollision(
				termOwners,
				term,
				entry.ConceptID,
				entry.ConflictGroupID,
			); err != nil {
				return fmt.Errorf("entries[%d]: %w", index, err)
			}
		}
		for _, term := range entry.ComparisonTerms {
			if err := registerCollision(
				normalizedOwners,
				term,
				entry.ConceptID,
				entry.ConflictGroupID,
			); err != nil {
				return fmt.Errorf("entries[%d]: %w", index, err)
			}
		}
	}
	return nil
}

func validateEntry(value datasetEntry) error {
	if err := validateIdentifier("conceptId", value.ConceptID); err != nil {
		return err
	}
	if err := validateText("canonical", value.Canonical); err != nil {
		return err
	}
	if err := validateStringList("terms", value.Terms); err != nil {
		return err
	}
	if err := validateStringList("comparisonTerms", value.ComparisonTerms); err != nil {
		return err
	}
	if len(value.Terms) != len(value.ComparisonTerms) {
		return fmt.Errorf("terms と comparisonTerms の件数は一致しなければなりません")
	}
	derivedComparisonTerms := make([]string, 0, len(value.Terms))
	for _, term := range value.Terms {
		key := querynormalization.ComparisonKey(term)
		if key == "" {
			return fmt.Errorf("terms から空の comparison key は作成できません")
		}
		derivedComparisonTerms = append(derivedComparisonTerms, key)
	}
	slices.Sort(derivedComparisonTerms)
	derivedComparisonTerms = slices.Compact(derivedComparisonTerms)
	if !slices.Equal(derivedComparisonTerms, value.ComparisonTerms) {
		return fmt.Errorf("comparisonTerms が terms の正規化結果と一致しません")
	}
	if err := validateText("sourceName", value.SourceName); err != nil {
		return err
	}
	if err := validateHTTPSURL("sourceUrl", value.SourceURL); err != nil {
		return err
	}
	confirmedAt, err := time.Parse("2006-01-02", value.ConfirmedAt)
	if err != nil || confirmedAt.Format("2006-01-02") != value.ConfirmedAt {
		return fmt.Errorf("confirmedAt は YYYY-MM-DD でなければなりません")
	}
	if err := validateText("mappingNote", value.MappingNote); err != nil {
		return err
	}
	if value.ConflictGroupID != "" {
		if err := validateIdentifier("conflictGroupId", value.ConflictGroupID); err != nil {
			return err
		}
	}
	switch value.SelectionPolicy {
	case SelectionPolicySingleCandidate:
		if len(value.Candidates) != 1 {
			return fmt.Errorf("single_candidate には候補が一件だけ必要です")
		}
	case SelectionPolicyAmbiguousNoAutoExecute:
		if len(value.Candidates) < 2 {
			return fmt.Errorf("ambiguous_no_auto_execute には候補が二件以上必要です")
		}
	default:
		return fmt.Errorf("selectionPolicy %q は使用できません", value.SelectionPolicy)
	}
	if len(value.Candidates) == 0 || len(value.Candidates) > maxCandidateCount {
		return fmt.Errorf(
			"candidates は 1 件以上 %d 件以下でなければなりません",
			maxCandidateCount,
		)
	}
	seenCandidates := make(map[string]struct{}, len(value.Candidates))
	for index, candidate := range value.Candidates {
		if err := validateCandidate(candidate); err != nil {
			return fmt.Errorf("candidates[%d]: %w", index, err)
		}
		key := candidateKey(candidate)
		if _, exists := seenCandidates[key]; exists {
			return fmt.Errorf("candidates[%d] が重複しています", index)
		}
		seenCandidates[key] = struct{}{}
	}
	return nil
}

func validateCandidate(value datasetCandidate) error {
	if err := validateText("officialTerm", value.OfficialTerm); err != nil {
		return err
	}
	if value.RequiredPacks == nil {
		return fmt.Errorf("requiredPacks は空配列または値を持つ配列でなければなりません")
	}
	if err := validateIdentifierList("requiredPacks", *value.RequiredPacks); err != nil {
		return err
	}
	switch {
	case value.Task == string(legalquery.TaskSearch) &&
		value.Resource == string(legalquery.ResourceLawProvision) &&
		value.InputKind == string(legalquery.InputKindLawContentSearch):
		if len(*value.RequiredPacks) != 0 {
			return fmt.Errorf("law_content_search 候補に requiredPacks は設定できません")
		}
	case value.Task == string(legalquery.TaskSearch) &&
		value.Resource == string(legalquery.ResourceJudicialDecision) &&
		value.InputKind == string(legalquery.InputKindJudicialDecisionSearch):
		if !slices.Equal(*value.RequiredPacks, []string{"judicial-cases"}) {
			return fmt.Errorf(
				"judicial_decision_search 候補の requiredPacks は [judicial-cases] でなければなりません",
			)
		}
	default:
		return fmt.Errorf("未採用の task/resource/inputKind 候補です")
	}
	return nil
}

func candidateKey(value datasetCandidate) string {
	return strings.Join([]string{
		value.Task,
		value.Resource,
		value.InputKind,
		value.OfficialTerm,
		strings.Join(*value.RequiredPacks, "\x00"),
	}, "\x01")
}

func validateIdentifierList(name string, values []string) error {
	if !slices.IsSorted(values) {
		return fmt.Errorf("%s は昇順でなければなりません", name)
	}
	previous := ""
	for _, value := range values {
		if err := validateIdentifier(name, value); err != nil {
			return err
		}
		if previous == value {
			return fmt.Errorf("%s に重複があります", name)
		}
		previous = value
	}
	return nil
}

func validateStringList(name string, values []string) error {
	if len(values) == 0 {
		return fmt.Errorf("%s は一件以上必要です", name)
	}
	if !slices.IsSorted(values) {
		return fmt.Errorf("%s は昇順でなければなりません", name)
	}
	previous := ""
	for index, value := range values {
		if err := validateText(fmt.Sprintf("%s[%d]", name, index), value); err != nil {
			return err
		}
		if previous == value {
			return fmt.Errorf("%s に重複があります", name)
		}
		previous = value
	}
	return nil
}

func validateIdentifier(name string, value string) error {
	if value == "" ||
		len(value) > 64 ||
		!identifierPattern.MatchString(value) {
		return fmt.Errorf("%s は 1 byte 以上 64 byte 以下の正規形でなければなりません", name)
	}
	return nil
}

func validateText(name string, value string) error {
	if !utf8.ValidString(value) ||
		value == "" ||
		strings.TrimSpace(value) != value ||
		len(value) > maxStringBytes {
		return fmt.Errorf(
			"%s は有効な UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maxStringBytes,
		)
	}
	for _, current := range value {
		if current <= '\u001f' || current == '\u007f' {
			return fmt.Errorf("%s に ASCII 制御文字は使用できません", name)
		}
	}
	return nil
}

func validateHTTPSURL(name string, value string) error {
	parsed, err := url.Parse(value)
	if err != nil ||
		parsed.Scheme != "https" ||
		parsed.Host == "" ||
		parsed.User != nil {
		return fmt.Errorf("%s は userinfo のない HTTPS URL でなければなりません", name)
	}
	return nil
}

func registerCollision(
	owners map[string]string,
	term string,
	conceptID string,
	conflictGroupID string,
) error {
	if existing, exists := owners[term]; exists {
		if existing != conflictGroupID || conflictGroupID == "" {
			return fmt.Errorf(
				"term %q は conceptId %q と正規化衝突します",
				term,
				conceptID,
			)
		}
		return nil
	}
	owners[term] = conflictGroupID
	return nil
}
