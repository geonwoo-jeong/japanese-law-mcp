// Package querypreprocess は、統合法情報照会の provider 非依存前処理を提供する。
package querypreprocess

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/searchquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/querynormalization"
)

const (
	maxCueEntries          = 512
	maxCueTermsPerEntry    = 64
	maxCueMatchGroupBytes  = 128
	maxPreprocessTermBytes = 2048
)

var internalIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Analyzer は、登録語と全 token の原文位置を返す。
type Analyzer interface {
	RegisteredTerms(context.Context, string) ([]string, error)
	AnalyzeTokenOccurrences(context.Context, string) ([]kagome.TokenOccurrence, error)
}

// Values は、起動時に構築する不変な前処理依存を保持する。
type Values struct {
	Analyzer      Analyzer
	LawNames      []lawnamelexicon.Entry
	LegalConcepts []legalconceptlexicon.Entry
	Cues          []legalquery.CueVocabularyEntry
}

type dictionaryKind uint8

const (
	dictionaryLawName dictionaryKind = iota + 1
	dictionaryLegalConcept
	dictionaryCue
)

type dictionaryTarget struct {
	kind       dictionaryKind
	id         string
	secondary  string
	matchGroup string
	term       string
}

type identifierTarget struct {
	kind       legalquery.IdentifierMentionKind
	lawID      string
	revisionID string
	lawNumber  string
}

type cueTarget struct {
	profileID string
	cueID     string
}

// Preprocessor は、起動時の辞書と tokenizer を共有する不変な前処理器である。
type Preprocessor struct {
	analyzer        Analyzer
	lawResolver     *searchquery.Resolver
	conceptResolver *searchquery.Resolver
	normalizedTerms *runeTrie[dictionaryTarget]
	identifiers     *runeTrie[identifierTarget]
	lawsByID        map[string]lawnamelexicon.Entry
	conceptsByID    map[string]legalconceptlexicon.Entry
	cuesByKey       map[string]cueTarget
}

// NewEmbedded は、組込み辞書と注入された cue から前処理器を構築する。
func NewEmbedded(cues []legalquery.CueVocabularyEntry) (*Preprocessor, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法令名辞書を構築できません: %w", err)
	}
	legalConcepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を構築できません: %w", err)
	}
	terms := append([]string(nil), lawNames.Terms()...)
	terms = append(terms, legalConcepts.Terms()...)
	for _, cue := range cues {
		terms = append(terms, cue.Terms...)
	}
	slices.Sort(terms)
	terms = slices.Compact(terms)
	analyzer, err := kagome.NewAnalyzer(terms)
	if err != nil {
		return nil, fmt.Errorf("形態素解析器を構築できません: %w", err)
	}
	return New(Values{
		Analyzer:      analyzer,
		LawNames:      lawNames.Entries(),
		LegalConcepts: legalConcepts.Entries(),
		Cues:          cues,
	})
}

// New は、入力を検証して provider 非依存の不変索引を構築する。
func New(values Values) (*Preprocessor, error) {
	if isNilAnalyzer(values.Analyzer) {
		return nil, fmt.Errorf("前処理 analyzer は必須です")
	}
	if len(values.Cues) > maxCueEntries {
		return nil, fmt.Errorf("cue 語彙は %d 件以下でなければなりません", maxCueEntries)
	}

	lawsByID := make(map[string]lawnamelexicon.Entry, len(values.LawNames))
	lawResolverEntries := make([]searchquery.EntryValues, 0, len(values.LawNames))
	normalizedTerms := newRuneTrie[dictionaryTarget]()
	identifiers := newRuneTrie[identifierTarget]()
	for index, entry := range values.LawNames {
		if err := validateLawEntry(entry); err != nil {
			return nil, fmt.Errorf("lawNames[%d]: %w", index, err)
		}
		if _, exists := lawsByID[entry.ResourceID]; exists {
			return nil, fmt.Errorf("lawId %q が重複しています", entry.ResourceID)
		}
		copied := cloneLawEntry(entry)
		lawsByID[copied.ResourceID] = copied
		lawResolverEntries = append(lawResolverEntries, searchquery.EntryValues{
			ResourceID: copied.ResourceID,
			Canonical:  copied.Canonical,
			Terms:      append([]string(nil), copied.Terms...),
		})
		for _, term := range append([]string{copied.Canonical}, copied.Terms...) {
			if err := normalizedTerms.add(
				querynormalization.ComparisonKey(term),
				dictionaryTarget{
					kind: dictionaryLawName,
					id:   copied.ResourceID,
					term: term,
				},
			); err != nil {
				return nil, fmt.Errorf("lawNames[%d] の語を登録できません: %w", index, err)
			}
		}
		for pattern, target := range map[string]identifierTarget{
			copied.ResourceID: {
				kind:  legalquery.IdentifierMentionLawID,
				lawID: copied.ResourceID,
			},
			copied.RevisionID: {
				kind:       legalquery.IdentifierMentionLawRevisionID,
				lawID:      copied.ResourceID,
				revisionID: copied.RevisionID,
			},
			copied.LawNumber: {
				kind:      legalquery.IdentifierMentionLawNumber,
				lawID:     copied.ResourceID,
				lawNumber: copied.LawNumber,
			},
		} {
			if err := identifiers.add(pattern, target); err != nil {
				return nil, fmt.Errorf("lawNames[%d] の識別子を登録できません: %w", index, err)
			}
		}
	}
	var lawResolver *searchquery.Resolver
	var err error
	if len(lawResolverEntries) > 0 {
		lawResolver, err = searchquery.NewResolver(
			lawResolverEntries,
			values.Analyzer,
		)
		if err != nil {
			return nil, fmt.Errorf("法令名 resolver を構築できません: %w", err)
		}
	}

	conceptsByID := make(map[string]legalconceptlexicon.Entry, len(values.LegalConcepts))
	conceptResolverEntries := make(
		[]searchquery.EntryValues,
		0,
		len(values.LegalConcepts),
	)
	for index, entry := range values.LegalConcepts {
		if err := validateConceptEntry(entry); err != nil {
			return nil, fmt.Errorf("legalConcepts[%d]: %w", index, err)
		}
		if _, exists := conceptsByID[entry.ConceptID]; exists {
			return nil, fmt.Errorf("conceptId %q が重複しています", entry.ConceptID)
		}
		copied := cloneConceptEntry(entry)
		conceptsByID[copied.ConceptID] = copied
		conceptResolverEntries = append(
			conceptResolverEntries,
			searchquery.EntryValues{
				ResourceID: copied.ConceptID,
				Canonical:  copied.Canonical,
				Terms:      append([]string(nil), copied.Terms...),
			},
		)
		for _, term := range copied.Terms {
			if err := normalizedTerms.add(
				querynormalization.ComparisonKey(term),
				dictionaryTarget{
					kind: dictionaryLegalConcept,
					id:   copied.ConceptID,
					term: term,
				},
			); err != nil {
				return nil, fmt.Errorf(
					"legalConcepts[%d] の語を登録できません: %w",
					index,
					err,
				)
			}
		}
	}
	var conceptResolver *searchquery.Resolver
	if len(conceptResolverEntries) > 0 {
		conceptResolver, err = searchquery.NewResolver(
			conceptResolverEntries,
			values.Analyzer,
		)
		if err != nil {
			return nil, fmt.Errorf("法概念 resolver を構築できません: %w", err)
		}
	}

	cuesByKey := make(map[string]cueTarget, len(values.Cues))
	for index, entry := range values.Cues {
		if err := validateCueEntry(entry); err != nil {
			return nil, fmt.Errorf("cues[%d]: %w", index, err)
		}
		key := cueKey(entry.ProfileID, entry.CueID)
		if _, exists := cuesByKey[key]; exists {
			return nil, fmt.Errorf(
				"profileId=%q, cueId=%q が重複しています",
				entry.ProfileID,
				entry.CueID,
			)
		}
		matchGroup := entry.MatchGroup
		if matchGroup == "" {
			matchGroup = entry.CueID
		}
		cuesByKey[key] = cueTarget{
			profileID: entry.ProfileID,
			cueID:     entry.CueID,
		}
		for _, term := range entry.Terms {
			if err := normalizedTerms.add(
				querynormalization.ComparisonKey(term),
				dictionaryTarget{
					kind:       dictionaryCue,
					id:         entry.ProfileID,
					secondary:  entry.CueID,
					matchGroup: matchGroup,
					term:       term,
				},
			); err != nil {
				return nil, fmt.Errorf("cues[%d] の語を登録できません: %w", index, err)
			}
		}
	}

	return &Preprocessor{
		analyzer:        values.Analyzer,
		lawResolver:     lawResolver,
		conceptResolver: conceptResolver,
		normalizedTerms: normalizedTerms,
		identifiers:     identifiers,
		lawsByID:        lawsByID,
		conceptsByID:    conceptsByID,
		cuesByKey:       cuesByKey,
	}, nil
}

func validateLawEntry(entry lawnamelexicon.Entry) error {
	for field, value := range map[string]string{
		"lawId":      entry.ResourceID,
		"revisionId": entry.RevisionID,
		"lawNumber":  entry.LawNumber,
		"canonical":  entry.Canonical,
	} {
		if err := validateTerm(field, value); err != nil {
			return err
		}
	}
	for index, term := range entry.Terms {
		if err := validateTerm(fmt.Sprintf("terms[%d]", index), term); err != nil {
			return err
		}
	}
	return nil
}

func validateConceptEntry(entry legalconceptlexicon.Entry) error {
	if !internalIDPattern.MatchString(entry.ConceptID) {
		return fmt.Errorf("conceptId は小文字 ASCII 識別子でなければなりません")
	}
	if err := validateTerm("canonical", entry.Canonical); err != nil {
		return err
	}
	if len(entry.Terms) == 0 {
		return fmt.Errorf("terms は一件以上必要です")
	}
	for index, term := range entry.Terms {
		if err := validateTerm(fmt.Sprintf("terms[%d]", index), term); err != nil {
			return err
		}
	}
	return nil
}

func validateCueEntry(entry legalquery.CueVocabularyEntry) error {
	if !internalIDPattern.MatchString(entry.ProfileID) {
		return fmt.Errorf("profileId は小文字 ASCII 識別子でなければなりません")
	}
	if !internalIDPattern.MatchString(entry.CueID) {
		return fmt.Errorf("cueId は小文字 ASCII 識別子でなければなりません")
	}
	if entry.MatchGroup != "" &&
		(len(entry.MatchGroup) > maxCueMatchGroupBytes ||
			!internalIDPattern.MatchString(entry.MatchGroup)) {
		return fmt.Errorf(
			"matchGroup は空または %d byte 以下の小文字 ASCII 識別子でなければなりません",
			maxCueMatchGroupBytes,
		)
	}
	if err := validateCueSyntaxRole(entry.SyntaxRole); err != nil {
		return err
	}
	if len(entry.Terms) == 0 || len(entry.Terms) > maxCueTermsPerEntry {
		return fmt.Errorf(
			"terms は 1 件以上 %d 件以下でなければなりません",
			maxCueTermsPerEntry,
		)
	}
	seen := make(map[string]struct{}, len(entry.Terms))
	for index, term := range entry.Terms {
		if err := validateTerm(fmt.Sprintf("terms[%d]", index), term); err != nil {
			return err
		}
		if _, exists := seen[term]; exists {
			return fmt.Errorf("terms[%d] が重複しています", index)
		}
		seen[term] = struct{}{}
	}
	return nil
}

func validateCueSyntaxRole(role legalquery.CueSyntaxRole) error {
	switch role {
	case legalquery.CueSyntaxRoleNone,
		legalquery.CueSyntaxRoleTaskExpression,
		legalquery.CueSyntaxRoleTaskObject,
		legalquery.CueSyntaxRoleTaskPredicate:
		return nil
	default:
		return fmt.Errorf("syntaxRole が定義されていません")
	}
}

func validateTerm(field string, value string) error {
	if !utf8.ValidString(value) ||
		strings.TrimSpace(value) == "" ||
		len(value) > maxPreprocessTermBytes ||
		querynormalization.ComparisonKey(value) == "" {
		return fmt.Errorf(
			"%s は比較できる UTF-8 で 1 byte 以上 %d byte 以下でなければなりません",
			field,
			maxPreprocessTermBytes,
		)
	}
	return nil
}

func cloneLawEntry(entry lawnamelexicon.Entry) lawnamelexicon.Entry {
	return lawnamelexicon.Entry{
		ResourceID: entry.ResourceID,
		RevisionID: entry.RevisionID,
		LawNumber:  entry.LawNumber,
		Canonical:  entry.Canonical,
		Terms:      append([]string(nil), entry.Terms...),
	}
}

func cloneConceptEntry(entry legalconceptlexicon.Entry) legalconceptlexicon.Entry {
	copied := entry
	copied.Terms = append([]string(nil), entry.Terms...)
	copied.ComparisonTerms = append([]string(nil), entry.ComparisonTerms...)
	copied.Candidates = append(
		[]legalconceptlexicon.Candidate(nil),
		entry.Candidates...,
	)
	for index := range copied.Candidates {
		copied.Candidates[index].TermOfficialOverrides = append(
			[]legalconceptlexicon.TermOfficialOverride(nil),
			entry.Candidates[index].TermOfficialOverrides...,
		)
		copied.Candidates[index].RequiredPacks = append(
			[]string(nil),
			entry.Candidates[index].RequiredPacks...,
		)
	}
	return copied
}

func cueKey(profileID string, cueID string) string {
	return profileID + "\x00" + cueID
}

func isNilAnalyzer(value Analyzer) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan,
		reflect.Func,
		reflect.Interface,
		reflect.Map,
		reflect.Pointer,
		reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
