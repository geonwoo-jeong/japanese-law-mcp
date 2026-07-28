// Package legalconceptlexicon は、出典付き法概念辞書を読み込む。
package legalconceptlexicon

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

const supportedSchemaVersion = 1

var (
	//go:embed data/current.json
	embeddedCurrent []byte
)

// SelectionPolicy は、同一概念から得た候補の扱いを表す。
type SelectionPolicy string

const (
	// SelectionPolicySingleCandidate は、一件の候補だけを返す概念を表す。
	SelectionPolicySingleCandidate SelectionPolicy = "single_candidate"
	// SelectionPolicyAmbiguousNoAutoExecute は、複数候補を返し自動実行しない概念を表す。
	SelectionPolicyAmbiguousNoAutoExecute SelectionPolicy = "ambiguous_no_auto_execute"
)

// Candidate は、法概念一致から作る provider 非依存候補である。
type Candidate struct {
	Task          legalquery.Task
	Resource      legalquery.Resource
	InputKind     legalquery.LogicalInputKind
	OfficialTerm  string
	RequiredPacks []string
}

// Entry は、一つの法概念 entry の複製である。
type Entry struct {
	ConceptID       string
	Canonical       string
	Terms           []string
	ComparisonTerms []string
	SourceName      string
	SourceURL       string
	ConfirmedAt     string
	MappingNote     string
	ConflictGroupID string
	SelectionPolicy SelectionPolicy
	Candidates      []Candidate
}

// Lexicon は、検証済みで不変の法概念辞書である。
type Lexicon struct {
	version         string
	entries         []Entry
	terms           []string
	comparisonTerms []string
}

// LoadEmbedded は、バイナリへ組み込んだ snapshot を読み込む。
func LoadEmbedded() (*Lexicon, error) {
	return Load(embeddedCurrent)
}

// Load は、snapshot JSON を厳格に検証して辞書を構築する。
func Load(datasetJSON []byte) (*Lexicon, error) {
	if len(datasetJSON) == 0 || len(datasetJSON) > maxDatasetBytes {
		return nil, fmt.Errorf(
			"法概念辞書は 1 byte 以上 %d byte 以下でなければなりません",
			maxDatasetBytes,
		)
	}
	value, err := decodeStrict[dataset](datasetJSON)
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	if err := validateDataset(value); err != nil {
		return nil, fmt.Errorf("法概念辞書が有効ではありません: %w", err)
	}
	return buildLexicon(value), nil
}

// Version は、読み込んだ snapshot version を返す。
func (l *Lexicon) Version() string {
	if l == nil {
		return ""
	}
	return l.version
}

// Entries は、entry 配列を深く複製して返す。
func (l *Lexicon) Entries() []Entry {
	if l == nil {
		return nil
	}
	entries := make([]Entry, len(l.entries))
	for index, entry := range l.entries {
		candidates := make([]Candidate, len(entry.Candidates))
		for candidateIndex, candidate := range entry.Candidates {
			candidates[candidateIndex] = Candidate{
				Task:          candidate.Task,
				Resource:      candidate.Resource,
				InputKind:     candidate.InputKind,
				OfficialTerm:  candidate.OfficialTerm,
				RequiredPacks: append([]string(nil), candidate.RequiredPacks...),
			}
		}
		entries[index] = Entry{
			ConceptID:       entry.ConceptID,
			Canonical:       entry.Canonical,
			Terms:           append([]string(nil), entry.Terms...),
			ComparisonTerms: append([]string(nil), entry.ComparisonTerms...),
			SourceName:      entry.SourceName,
			SourceURL:       entry.SourceURL,
			ConfirmedAt:     entry.ConfirmedAt,
			MappingNote:     entry.MappingNote,
			ConflictGroupID: entry.ConflictGroupID,
			SelectionPolicy: entry.SelectionPolicy,
			Candidates:      candidates,
		}
	}
	return entries
}

// Terms は、Kagome 登録に使う語を返す。
func (l *Lexicon) Terms() []string {
	if l == nil {
		return nil
	}
	return append([]string(nil), l.terms...)
}

// ComparisonTerms は、比較用の正規化語を返す。
func (l *Lexicon) ComparisonTerms() []string {
	if l == nil {
		return nil
	}
	return append([]string(nil), l.comparisonTerms...)
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

func buildLexicon(value dataset) *Lexicon {
	entries := make([]Entry, 0, len(value.Entries))
	allTerms := make(map[string]struct{})
	allComparisonTerms := make(map[string]struct{})
	for _, current := range value.Entries {
		candidates := make([]Candidate, 0, len(current.Candidates))
		for _, candidate := range current.Candidates {
			candidates = append(candidates, Candidate{
				Task:          legalquery.Task(candidate.Task),
				Resource:      legalquery.Resource(candidate.Resource),
				InputKind:     legalquery.LogicalInputKind(candidate.InputKind),
				OfficialTerm:  candidate.OfficialTerm,
				RequiredPacks: append([]string(nil), (*candidate.RequiredPacks)...),
			})
		}
		entries = append(entries, Entry{
			ConceptID:       current.ConceptID,
			Canonical:       current.Canonical,
			Terms:           append([]string(nil), current.Terms...),
			ComparisonTerms: append([]string(nil), current.ComparisonTerms...),
			SourceName:      current.SourceName,
			SourceURL:       current.SourceURL,
			ConfirmedAt:     current.ConfirmedAt,
			MappingNote:     current.MappingNote,
			ConflictGroupID: current.ConflictGroupID,
			SelectionPolicy: current.SelectionPolicy,
			Candidates:      candidates,
		})
		for _, term := range current.Terms {
			allTerms[term] = struct{}{}
		}
		for _, term := range current.ComparisonTerms {
			allComparisonTerms[term] = struct{}{}
		}
	}
	terms := make([]string, 0, len(allTerms))
	for term := range allTerms {
		terms = append(terms, term)
	}
	slices.Sort(terms)
	comparisonTerms := make([]string, 0, len(allComparisonTerms))
	for term := range allComparisonTerms {
		comparisonTerms = append(comparisonTerms, term)
	}
	slices.Sort(comparisonTerms)
	return &Lexicon{
		version:         value.LexiconVersion,
		entries:         entries,
		terms:           terms,
		comparisonTerms: comparisonTerms,
	}
}
