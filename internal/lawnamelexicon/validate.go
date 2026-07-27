package lawnamelexicon

import (
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/unicode/norm"
)

const maxLexiconStringBytes = 2048

func validateOfficialDataset(dataset officialDataset) error {
	if dataset.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf(
			"schemaVersion = %d, want %d",
			dataset.SchemaVersion,
			supportedSchemaVersion,
		)
	}
	if err := validateString("datasetId", dataset.DatasetID); err != nil {
		return err
	}
	if dataset.Source.ProviderID != "e-gov-law-api-v2" ||
		dataset.Source.Operation != "GET /laws" {
		return fmt.Errorf("公式辞書の情報源が e-Gov GET /laws ではありません")
	}
	if err := validateHTTPSURL("source.url", dataset.Source.URL); err != nil {
		return err
	}
	retrievedAt, err := time.Parse(time.RFC3339, dataset.Source.RetrievedAt)
	if err != nil {
		return fmt.Errorf("source.retrievedAt は RFC 3339 ではありません")
	}
	_, offset := retrievedAt.Zone()
	if offset != 0 {
		return fmt.Errorf("source.retrievedAt は UTC でなければなりません")
	}
	if len(dataset.Entries) == 0 || len(dataset.Entries) > maxLawCount {
		return fmt.Errorf(
			"entries は 1 件以上 %d 件以下でなければなりません",
			maxLawCount,
		)
	}

	aliasCount := 0
	seenLawIDs := make(map[string]struct{}, len(dataset.Entries))
	previousLawID := ""
	for index, entry := range dataset.Entries {
		if err := validateOfficialEntry(entry); err != nil {
			return fmt.Errorf("entries[%d]: %w", index, err)
		}
		if _, duplicated := seenLawIDs[entry.LawID]; duplicated {
			return fmt.Errorf("lawId %q が重複しています", entry.LawID)
		}
		if previousLawID != "" && previousLawID >= entry.LawID {
			return fmt.Errorf("entries は lawId の昇順ではありません")
		}
		seenLawIDs[entry.LawID] = struct{}{}
		previousLawID = entry.LawID
		aliasCount += len(*entry.Aliases)
	}
	if dataset.Statistics.LawCount != len(dataset.Entries) ||
		dataset.Statistics.AliasCount != aliasCount ||
		aliasCount > maxAliasCount {
		return fmt.Errorf(
			"statistics が実データと一致しません: laws=%d, aliases=%d",
			len(dataset.Entries),
			aliasCount,
		)
	}
	return nil
}

func validateOfficialEntry(entry officialEntry) error {
	for name, value := range map[string]string{
		"lawId":      entry.LawID,
		"revisionId": entry.RevisionID,
		"lawNumber":  entry.LawNumber,
		"title":      entry.Title,
	} {
		if err := validateString(name, value); err != nil {
			return err
		}
	}
	if entry.TitleKana != "" {
		if err := validateSourceString("titleKana", entry.TitleKana); err != nil {
			return err
		}
	}
	if entry.Aliases == nil {
		return fmt.Errorf("aliases は空配列または値を持つ配列でなければなりません")
	}
	if !slices.IsSorted(*entry.Aliases) {
		return fmt.Errorf("aliases は昇順ではありません")
	}
	seenAliases := make(map[string]struct{}, len(*entry.Aliases))
	for index, alias := range *entry.Aliases {
		if err := validateString(fmt.Sprintf("aliases[%d]", index), alias); err != nil {
			return err
		}
		if _, duplicated := seenAliases[alias]; duplicated {
			return fmt.Errorf("alias %q が重複しています", alias)
		}
		seenAliases[alias] = struct{}{}
	}
	return nil
}

func validateSupplementalDataset(
	official officialDataset,
	supplemental supplementalDataset,
) error {
	if supplemental.SchemaVersion != supportedSchemaVersion {
		return fmt.Errorf(
			"schemaVersion = %d, want %d",
			supplemental.SchemaVersion,
			supportedSchemaVersion,
		)
	}
	if err := validateString("datasetId", supplemental.DatasetID); err != nil {
		return err
	}
	officialByLawID := make(map[string]officialEntry, len(official.Entries))
	termOwners := make(map[string]string)
	normalizedOwners := make(map[string]string)
	for _, entry := range official.Entries {
		officialByLawID[entry.LawID] = entry
		termOwners[entry.Title] = entry.LawID
		registerNormalizedOwner(normalizedOwners, entry.Title, entry.LawID)
		if entry.TitleKana != "" {
			termOwners[entry.TitleKana] = entry.LawID
			registerNormalizedOwner(
				normalizedOwners,
				entry.TitleKana,
				entry.LawID,
			)
		}
		for _, alias := range *entry.Aliases {
			if existing, exists := termOwners[alias]; exists &&
				existing != entry.LawID {
				registerNormalizedOwner(normalizedOwners, alias, entry.LawID)
				continue
			}
			termOwners[alias] = entry.LawID
			registerNormalizedOwner(normalizedOwners, alias, entry.LawID)
		}
	}

	seenEntries := make(map[string]struct{}, len(supplemental.Entries))
	for index, entry := range supplemental.Entries {
		if err := validateSupplementalEntry(entry); err != nil {
			return fmt.Errorf("entries[%d]: %w", index, err)
		}
		officialEntry, exists := officialByLawID[entry.LawID]
		if !exists {
			return fmt.Errorf("entries[%d] は未知の lawId を参照しています", index)
		}
		if officialEntry.LawNumber != entry.LawNumber ||
			officialEntry.Title != entry.Title {
			return fmt.Errorf("entries[%d] の法令番号または正式名称が一致しません", index)
		}
		entryKey := entry.LawID + "\x00" + entry.Alias
		if _, duplicated := seenEntries[entryKey]; duplicated {
			return fmt.Errorf("entries[%d] が重複しています", index)
		}
		seenEntries[entryKey] = struct{}{}
		if existing, exists := termOwners[entry.Alias]; exists {
			if existing == entry.LawID {
				return fmt.Errorf(
					"entries[%d] は公式辞書の語を重複しています",
					index,
				)
			}
			return fmt.Errorf(
				"entries[%d] は別の法令に属する語と衝突しています",
				index,
			)
		}
		if existing, exists := normalizedOwners[comparisonKey(entry.Alias)]; exists &&
			existing != entry.LawID {
			return fmt.Errorf(
				"entries[%d] は正規化後に別の法令の語と衝突しています",
				index,
			)
		}
		termOwners[entry.Alias] = entry.LawID
		registerNormalizedOwner(normalizedOwners, entry.Alias, entry.LawID)
	}
	return nil
}

func validateSupplementalEntry(entry supplementalEntry) error {
	for name, value := range map[string]string{
		"lawId":      entry.LawID,
		"lawNumber":  entry.LawNumber,
		"title":      entry.Title,
		"alias":      entry.Alias,
		"sourceName": entry.SourceName,
	} {
		if err := validateString(name, value); err != nil {
			return err
		}
	}
	if entry.Kind != "common-abbreviation" {
		return fmt.Errorf("kind %q は使用できません", entry.Kind)
	}
	if err := validateHTTPSURL("sourceUrl", entry.SourceURL); err != nil {
		return err
	}
	parsedDate, err := time.Parse("2006-01-02", entry.ConfirmedAt)
	if err != nil || parsedDate.Format("2006-01-02") != entry.ConfirmedAt {
		return fmt.Errorf("confirmedAt は YYYY-MM-DD ではありません")
	}
	return nil
}

func validateString(name string, value string) error {
	if !utf8.ValidString(value) ||
		value == "" ||
		strings.TrimSpace(value) != value ||
		len(value) > maxLexiconStringBytes {
		return fmt.Errorf(
			"%s は有効な UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maxLexiconStringBytes,
		)
	}
	for _, current := range value {
		if current <= '\u001f' || current == '\u007f' {
			return fmt.Errorf("%s に ASCII 制御文字は使用できません", name)
		}
	}
	return nil
}

func validateSourceString(name string, value string) error {
	if !utf8.ValidString(value) ||
		strings.TrimSpace(value) == "" ||
		len(value) > maxLexiconStringBytes {
		return fmt.Errorf(
			"%s は有効な UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maxLexiconStringBytes,
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

func registerNormalizedOwner(
	owners map[string]string,
	value string,
	lawID string,
) {
	key := comparisonKey(value)
	if key == "" {
		return
	}
	if _, exists := owners[key]; !exists {
		owners[key] = lawID
	}
}

func comparisonKey(value string) string {
	normalized := norm.NFKC.String(value)
	var builder strings.Builder
	builder.Grow(len(normalized))
	for _, current := range normalized {
		if unicode.IsSpace(current) || unicode.IsPunct(current) {
			continue
		}
		switch {
		case current >= 'A' && current <= 'Z':
			current += 'a' - 'A'
		case current >= '\u30a1' && current <= '\u30f6':
			current -= '\u0060'
		case current == '\u30fd':
			current = '\u309d'
		case current == '\u30fe':
			current = '\u309e'
		}
		builder.WriteRune(current)
	}
	return builder.String()
}
