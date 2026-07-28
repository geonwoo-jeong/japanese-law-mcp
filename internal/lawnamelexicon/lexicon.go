// Package lawnamelexicon は、出典付き法令名検索辞書を読み込む。
package lawnamelexicon

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
)

const (
	supportedSchemaVersion = 1
	maxDatasetBytes        = 8 << 20
	maxLawCount            = 20000
	maxAliasCount          = 100000
)

var (
	//go:embed data/egov-current.json
	embeddedOfficial []byte

	//go:embed data/supplemental.json
	embeddedSupplemental []byte
)

// Entry は、一つの法令の識別子、法令番号、正式名称および登録語の複製である。
type Entry struct {
	ResourceID string
	RevisionID string
	LawNumber  string
	Canonical  string
	Terms      []string
}

// Lexicon は、検証済みの不変な法令名辞書である。
type Lexicon struct {
	entries []Entry
	terms   []string
}

// LoadEmbedded は、バイナリへ組み込んだ公式・補足辞書を読み込む。
func LoadEmbedded() (*Lexicon, error) {
	return Load(embeddedOfficial, embeddedSupplemental)
}

// Load は、公式・補足 JSON を厳格に検証して一つの辞書を構築する。
func Load(officialJSON []byte, supplementalJSON []byte) (*Lexicon, error) {
	if len(officialJSON) == 0 || len(officialJSON) > maxDatasetBytes {
		return nil, fmt.Errorf(
			"公式辞書は 1 byte 以上 %d byte 以下でなければなりません",
			maxDatasetBytes,
		)
	}
	if len(supplementalJSON) == 0 || len(supplementalJSON) > maxDatasetBytes {
		return nil, fmt.Errorf(
			"補足辞書は 1 byte 以上 %d byte 以下でなければなりません",
			maxDatasetBytes,
		)
	}

	official, err := decodeStrict[officialDataset](officialJSON)
	if err != nil {
		return nil, fmt.Errorf("公式辞書を読み込めません: %w", err)
	}
	supplemental, err := decodeStrict[supplementalDataset](
		supplementalJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("補足辞書を読み込めません: %w", err)
	}
	if err := validateOfficialDataset(official); err != nil {
		return nil, fmt.Errorf("公式辞書が有効ではありません: %w", err)
	}
	if err := validateSupplementalDataset(official, supplemental); err != nil {
		return nil, fmt.Errorf("補足辞書が有効ではありません: %w", err)
	}

	return buildLexicon(official, supplemental), nil
}

// Entries は、辞書 entry と内部 slice を深く複製して返す。
func (l *Lexicon) Entries() []Entry {
	if l == nil {
		return nil
	}
	entries := make([]Entry, len(l.entries))
	for index, entry := range l.entries {
		entries[index] = Entry{
			ResourceID: entry.ResourceID,
			RevisionID: entry.RevisionID,
			LawNumber:  entry.LawNumber,
			Canonical:  entry.Canonical,
			Terms:      append([]string(nil), entry.Terms...),
		}
	}
	return entries
}

// Terms は、Kagome user dictionary に登録する重複のない語を返す。
func (l *Lexicon) Terms() []string {
	if l == nil {
		return nil
	}
	return append([]string(nil), l.terms...)
}

func decodeStrict[T any](value []byte) (T, error) {
	var decoded T
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return decoded, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return decoded, fmt.Errorf("JSON object の後に値があります")
		}
		return decoded, err
	}
	return decoded, nil
}

func buildLexicon(
	official officialDataset,
	supplemental supplementalDataset,
) *Lexicon {
	supplementalByLawID := make(map[string][]string)
	for _, entry := range supplemental.Entries {
		supplementalByLawID[entry.LawID] = append(
			supplementalByLawID[entry.LawID],
			entry.Alias,
		)
	}

	entries := make([]Entry, 0, len(official.Entries))
	allTerms := make(map[string]struct{})
	for _, sourceEntry := range official.Entries {
		terms := make([]string, 0, len(*sourceEntry.Aliases)+2)
		if sourceEntry.TitleKana != "" {
			terms = append(terms, strings.TrimSpace(sourceEntry.TitleKana))
		}
		terms = append(terms, (*sourceEntry.Aliases)...)
		terms = append(terms, supplementalByLawID[sourceEntry.LawID]...)
		slices.Sort(terms)
		terms = slices.Compact(terms)
		entries = append(entries, Entry{
			ResourceID: sourceEntry.LawID,
			RevisionID: sourceEntry.RevisionID,
			LawNumber:  sourceEntry.LawNumber,
			Canonical:  sourceEntry.Title,
			Terms:      append([]string(nil), terms...),
		})
		allTerms[sourceEntry.Title] = struct{}{}
		for _, term := range terms {
			allTerms[term] = struct{}{}
		}
	}
	terms := make([]string, 0, len(allTerms))
	for term := range allTerms {
		terms = append(terms, term)
	}
	slices.Sort(terms)
	return &Lexicon{
		entries: entries,
		terms:   terms,
	}
}
