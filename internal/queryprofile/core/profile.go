// Package core は、法令コア五能力の query profile を提供する。
package core

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"slices"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	supportedSchemaVersion = 1
	maxProfileBytes        = 64 << 10
	maxCuesBytes           = 256 << 10
	maxCueCount            = 128
)

var (
	//go:embed data/profile.json
	embeddedProfile []byte

	//go:embed data/cues.json
	embeddedCues []byte

	coreIDPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
)

type profileDocument struct {
	SchemaVersion  int                     `json:"schemaVersion"`
	ProfileID      string                  `json:"profileId"`
	ProfileVersion string                  `json:"profileVersion"`
	RankingVersion string                  `json:"rankingVersion"`
	CueSetVersion  string                  `json:"cueSetVersion"`
	Targets        []targetDocument        `json:"targets"`
	Score          scoreDocument           `json:"score"`
	Selection      selectionDocument       `json:"selection"`
	TieBreak       []string                `json:"tieBreak"`
	Lexicons       lexiconVersionsDocument `json:"lexicons"`
}

type targetDocument struct {
	Task      string `json:"task"`
	Resource  string `json:"resource"`
	InputKind string `json:"inputKind"`
}

type scoreDocument struct {
	Minimum            int                      `json:"minimum"`
	Maximum            int                      `json:"maximum"`
	EvidenceWeights    []evidenceWeightDocument `json:"evidenceWeights"`
	HighConfidenceAt   int                      `json:"highConfidenceAt"`
	MediumConfidenceAt int                      `json:"mediumConfidenceAt"`
}

type evidenceWeightDocument struct {
	Code   string `json:"code"`
	Weight int    `json:"weight"`
}

type selectionDocument struct {
	SingleThreshold           int `json:"singleThreshold"`
	MinimumExecutionThreshold int `json:"minimumExecutionThreshold"`
	SingleMargin              int `json:"singleMargin"`
	HedgeMargin               int `json:"hedgeMargin"`
}

type lexiconVersionsDocument struct {
	LawNames      string `json:"lawNames"`
	LegalConcepts string `json:"legalConcepts"`
}

type cuesDocument struct {
	SchemaVersion int           `json:"schemaVersion"`
	ProfileID     string        `json:"profileId"`
	CueSetVersion string        `json:"cueSetVersion"`
	Cues          []cueDocument `json:"cues"`
}

type cueDocument struct {
	CueID    string   `json:"cueId"`
	Category string   `json:"category"`
	Value    string   `json:"value"`
	Terms    []string `json:"terms"`
}

type cueDefinition struct {
	category string
	value    string
}

type conceptDefinition struct {
	entry  legalconceptlexicon.Entry
	source legalquery.LegalConceptSource
}

// Profile は、起動時に検証した不変の法令コア profile である。
type Profile struct {
	metadata legalquery.QueryProfileMetadata
	cues     []legalquery.CueVocabularyEntry
	cueByID  map[string]cueDefinition
	concepts map[string]conceptDefinition
}

// LoadEmbedded は、組込み profile、cue および辞書を検証して構築する。
func LoadEmbedded() (*Profile, error) {
	lawNames, err := lawnamelexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法令名辞書を読み込めません: %w", err)
	}
	concepts, err := legalconceptlexicon.LoadEmbedded()
	if err != nil {
		return nil, fmt.Errorf("法概念辞書を読み込めません: %w", err)
	}
	return Load(embeddedProfile, embeddedCues, lawNames, concepts)
}

// Load は、閉じた JSON と参照辞書の版を fail closed で検証する。
func Load(
	profileJSON []byte,
	cuesJSON []byte,
	lawNames *lawnamelexicon.Lexicon,
	concepts *legalconceptlexicon.Lexicon,
) (*Profile, error) {
	if lawNames == nil || lawNames.Version() == "" {
		return nil, fmt.Errorf("法令名辞書は必須です")
	}
	if concepts == nil || concepts.Version() == "" {
		return nil, fmt.Errorf("法概念辞書は必須です")
	}
	profileData, err := decodeStrict[profileDocument](
		"profile.json",
		profileJSON,
		maxProfileBytes,
	)
	if err != nil {
		return nil, err
	}
	cueData, err := decodeStrict[cuesDocument](
		"cues.json",
		cuesJSON,
		maxCuesBytes,
	)
	if err != nil {
		return nil, err
	}
	if profileData.SchemaVersion != supportedSchemaVersion ||
		cueData.SchemaVersion != supportedSchemaVersion {
		return nil, fmt.Errorf("profile data の schemaVersion が未対応です")
	}
	if profileData.ProfileID != cueData.ProfileID ||
		profileData.CueSetVersion != cueData.CueSetVersion {
		return nil, fmt.Errorf("profile.json と cues.json の版が一致しません")
	}
	if profileData.Lexicons.LawNames != lawNames.Version() ||
		profileData.Lexicons.LegalConcepts != concepts.Version() {
		return nil, fmt.Errorf("profile が参照する辞書 version と実体が一致しません")
	}

	metadata, err := buildMetadata(profileData)
	if err != nil {
		return nil, err
	}
	cues, cueByID, err := buildCues(cueData)
	if err != nil {
		return nil, err
	}
	conceptByID, err := buildConceptDefinitions(concepts.Entries())
	if err != nil {
		return nil, err
	}
	return &Profile{
		metadata: metadata,
		cues:     cues,
		cueByID:  cueByID,
		concepts: conceptByID,
	}, nil
}

// Metadata は、selector と評価が参照する不変 metadata を返す。
func (p *Profile) Metadata() legalquery.QueryProfileMetadata {
	if p == nil {
		return legalquery.QueryProfileMetadata{}
	}
	return p.metadata
}

// CueVocabulary は、共通前処理へ注入する cue の深い複製を返す。
func (p *Profile) CueVocabulary() []legalquery.CueVocabularyEntry {
	if p == nil {
		return nil
	}
	result := make([]legalquery.CueVocabularyEntry, 0, len(p.cues))
	for _, cue := range p.cues {
		result = append(result, legalquery.CueVocabularyEntry{
			ProfileID: cue.ProfileID,
			CueID:     cue.CueID,
			Terms:     append([]string(nil), cue.Terms...),
		})
	}
	return result
}

func decodeStrict[T any](
	name string,
	value []byte,
	maxBytes int,
) (T, error) {
	var decoded T
	if len(value) == 0 || len(value) > maxBytes {
		return decoded, fmt.Errorf(
			"%s は 1 byte 以上 %d byte 以下でなければなりません",
			name,
			maxBytes,
		)
	}
	decoder := json.NewDecoder(bytes.NewReader(value))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&decoded); err != nil {
		return decoded, fmt.Errorf("%s を読み込めません: %w", name, err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return decoded, fmt.Errorf("%s の後に値があります", name)
		}
		return decoded, fmt.Errorf("%s の終端が不正です: %w", name, err)
	}
	return decoded, nil
}

func buildMetadata(
	document profileDocument,
) (legalquery.QueryProfileMetadata, error) {
	targets := make([]legalquery.QueryProfileTarget, 0, len(document.Targets))
	for index, raw := range document.Targets {
		target, err := legalquery.NewQueryProfileTarget(
			legalquery.QueryProfileTargetValues{
				Task:      legalquery.Task(raw.Task),
				Resource:  legalquery.Resource(raw.Resource),
				InputKind: legalquery.LogicalInputKind(raw.InputKind),
			},
		)
		if err != nil {
			return legalquery.QueryProfileMetadata{}, fmt.Errorf(
				"targets[%d]: %w",
				index,
				err,
			)
		}
		targets = append(targets, target)
	}
	if !isExactCoreTargets(targets) {
		return legalquery.QueryProfileMetadata{}, fmt.Errorf(
			"core profile targets は法令コア五能力の固定順でなければなりません",
		)
	}
	weights := make([]legalquery.QueryEvidenceWeight, 0, len(document.Score.EvidenceWeights))
	for index, raw := range document.Score.EvidenceWeights {
		weight, err := legalquery.NewQueryEvidenceWeight(
			legalquery.QueryEvidenceWeightValues{
				Code:   legalquery.EvidenceCode(raw.Code),
				Weight: raw.Weight,
			},
		)
		if err != nil {
			return legalquery.QueryProfileMetadata{}, fmt.Errorf(
				"evidenceWeights[%d]: %w",
				index,
				err,
			)
		}
		weights = append(weights, weight)
	}
	score, err := legalquery.NewQueryScorePolicy(legalquery.QueryScorePolicyValues{
		Minimum:            document.Score.Minimum,
		Maximum:            document.Score.Maximum,
		EvidenceWeights:    weights,
		HighConfidenceAt:   document.Score.HighConfidenceAt,
		MediumConfidenceAt: document.Score.MediumConfidenceAt,
	})
	if err != nil {
		return legalquery.QueryProfileMetadata{}, err
	}
	selection, err := legalquery.NewQuerySelectionPolicy(
		legalquery.QuerySelectionPolicyValues{
			SingleThreshold:           document.Selection.SingleThreshold,
			MinimumExecutionThreshold: document.Selection.MinimumExecutionThreshold,
			SingleMargin:              document.Selection.SingleMargin,
			HedgeMargin:               document.Selection.HedgeMargin,
			ScoreMinimum:              score.Minimum(),
			ScoreMaximum:              score.Maximum(),
		},
	)
	if err != nil {
		return legalquery.QueryProfileMetadata{}, err
	}
	tieBreak := make([]legalquery.QueryTieBreak, 0, len(document.TieBreak))
	for _, value := range document.TieBreak {
		tieBreak = append(tieBreak, legalquery.QueryTieBreak(value))
	}
	return legalquery.NewQueryProfileMetadata(
		legalquery.QueryProfileMetadataValues{
			SchemaVersion:              document.SchemaVersion,
			ProfileID:                  document.ProfileID,
			ProfileVersion:             document.ProfileVersion,
			RankingVersion:             document.RankingVersion,
			CueSetVersion:              document.CueSetVersion,
			LawNameLexiconVersion:      document.Lexicons.LawNames,
			LegalConceptLexiconVersion: document.Lexicons.LegalConcepts,
			Targets:                    targets,
			Score:                      score,
			Selection:                  selection,
			TieBreak:                   tieBreak,
		},
	)
}

func isExactCoreTargets(values []legalquery.QueryProfileTarget) bool {
	expected := []legalquery.LogicalInputKind{
		legalquery.InputKindLawSearch,
		legalquery.InputKindLawContentSearch,
		legalquery.InputKindLawRead,
		legalquery.InputKindLawArticleRead,
		legalquery.InputKindLawUpdates,
	}
	if len(values) != len(expected) {
		return false
	}
	for index, target := range values {
		if target.InputKind() != expected[index] {
			return false
		}
	}
	return true
}

func buildCues(
	document cuesDocument,
) (
	[]legalquery.CueVocabularyEntry,
	map[string]cueDefinition,
	error,
) {
	if len(document.Cues) == 0 || len(document.Cues) > maxCueCount {
		return nil, nil, fmt.Errorf("cues は一件以上 %d 件以下必要です", maxCueCount)
	}
	cues := make([]legalquery.CueVocabularyEntry, 0, len(document.Cues))
	definitions := make(map[string]cueDefinition, len(document.Cues))
	previousID := ""
	for index, raw := range document.Cues {
		if !coreIDPattern.MatchString(raw.CueID) ||
			(index > 0 && previousID >= raw.CueID) {
			return nil, nil, fmt.Errorf("cues は有効な cueId の昇順でなければなりません")
		}
		if !validCueMeaning(raw.Category, raw.Value) {
			return nil, nil, fmt.Errorf("cues[%d] の category/value が未対応です", index)
		}
		if len(raw.Terms) == 0 || len(raw.Terms) > 64 {
			return nil, nil, fmt.Errorf("cues[%d].terms の件数が有効ではありません", index)
		}
		terms := append([]string(nil), raw.Terms...)
		for termIndex, term := range terms {
			if strings.TrimSpace(term) == "" {
				return nil, nil, fmt.Errorf("cues[%d].terms[%d] は必須です", index, termIndex)
			}
		}
		slices.Sort(terms)
		if len(slices.Compact(terms)) != len(terms) {
			return nil, nil, fmt.Errorf("cues[%d].terms を重複させることはできません", index)
		}
		cues = append(cues, legalquery.CueVocabularyEntry{
			ProfileID: document.ProfileID,
			CueID:     raw.CueID,
			Terms:     terms,
		})
		definitions[raw.CueID] = cueDefinition{
			category: raw.Category,
			value:    raw.Value,
		}
		previousID = raw.CueID
	}
	return cues, definitions, nil
}

func validCueMeaning(category string, value string) bool {
	switch category {
	case "task":
		return value == "search" ||
			value == "read" ||
			value == "list_updates"
	case "resource":
		return value == "law" ||
			value == "law_provision" ||
			value == "updates"
	case "operator":
		return value == "all" ||
			value == "any" ||
			value == "as_of" ||
			value == "dual_candidate" ||
			value == "exclude" ||
			value == "individual"
	case "unsupported":
		return value == "legal_advice" ||
			value == "translation" ||
			value == "task_or_resource"
	case "reserved_pack":
		return value == "judicial-cases"
	default:
		return false
	}
}

func buildConceptDefinitions(
	entries []legalconceptlexicon.Entry,
) (map[string]conceptDefinition, error) {
	result := make(map[string]conceptDefinition, len(entries))
	for _, entry := range entries {
		confirmedOn, err := model.NewDate(entry.ConfirmedAt)
		if err != nil {
			return nil, fmt.Errorf("concept %q の確認日が不正です: %w", entry.ConceptID, err)
		}
		source, err := legalquery.NewLegalConceptSource(
			legalquery.LegalConceptSourceValues{
				ConceptID:   entry.ConceptID,
				Title:       entry.SourceName,
				URL:         entry.SourceURL,
				ConfirmedOn: confirmedOn,
			},
		)
		if err != nil {
			return nil, fmt.Errorf("concept %q の出典が不正です: %w", entry.ConceptID, err)
		}
		result[entry.ConceptID] = conceptDefinition{
			entry:  cloneConceptEntry(entry),
			source: source,
		}
	}
	return result, nil
}

func cloneConceptEntry(
	entry legalconceptlexicon.Entry,
) legalconceptlexicon.Entry {
	candidates := make([]legalconceptlexicon.Candidate, 0, len(entry.Candidates))
	for _, candidate := range entry.Candidates {
		candidate.RequiredPacks = append([]string(nil), candidate.RequiredPacks...)
		candidates = append(candidates, candidate)
	}
	entry.Terms = append([]string(nil), entry.Terms...)
	entry.ComparisonTerms = append([]string(nil), entry.ComparisonTerms...)
	entry.Candidates = candidates
	return entry
}
