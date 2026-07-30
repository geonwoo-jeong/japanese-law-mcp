// Package core は、法令コア五能力の query profile を提供する。
package core

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/lawnamelexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalconceptlexicon"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/cueartifact"
)

const (
	supportedProfileSchemaVersion = 1
	maxProfileBytes               = 64 << 10
)

var (
	//go:embed data/profile.json
	embeddedProfile []byte

	//go:embed data/cues.json
	embeddedCues []byte
)

type profileDocument struct {
	SchemaVersion        int                          `json:"schemaVersion"`
	ProfileID            string                       `json:"profileId"`
	ProfileVersion       string                       `json:"profileVersion"`
	RankingVersion       string                       `json:"rankingVersion"`
	CueSetVersion        string                       `json:"cueSetVersion"`
	Targets              []targetDocument             `json:"targets"`
	Score                scoreDocument                `json:"score"`
	Selection            selectionDocument            `json:"selection"`
	TieBreak             []string                     `json:"tieBreak"`
	ConditionalTieBreaks conditionalTieBreaksDocument `json:"conditionalTieBreaks"`
	Lexicons             lexiconVersionsDocument      `json:"lexicons"`
}

type conditionalTieBreaksDocument struct {
	LawAliasCollisionGroupsOverCandidateLimit []string `json:"lawAliasCollisionGroupsOverCandidateLimit"`
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

type cueDefinition struct {
	category    string
	value       string
	intentGroup string
	signal      legalquery.CandidateGenerationSignal
	syntaxRole  legalquery.CueSyntaxRole
}

type conceptDefinition struct {
	entry  legalconceptlexicon.Entry
	source legalquery.LegalConceptSource
}

// Profile は、起動時に検証した不変の法令コア profile である。
type Profile struct {
	metadata                         legalquery.QueryProfileMetadata
	cues                             []legalquery.CueVocabularyEntry
	cueByID                          map[string]cueDefinition
	concepts                         map[string]conceptDefinition
	rankAliasCollisionGroupsBySource bool
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
	cueData, err := cueartifact.Load(cuesJSON)
	if err != nil {
		return nil, err
	}
	if profileData.SchemaVersion != supportedProfileSchemaVersion {
		return nil, fmt.Errorf("profile data の schemaVersion が未対応です")
	}
	if err := cueData.MatchProfile(
		profileData.ProfileID,
		profileData.CueSetVersion,
	); err != nil {
		return nil, err
	}
	if profileData.Lexicons.LawNames != lawNames.Version() ||
		profileData.Lexicons.LegalConcepts != concepts.Version() {
		return nil, fmt.Errorf("profile が参照する辞書 version と実体が一致しません")
	}
	if err := validateConditionalTieBreaks(
		profileData.ConditionalTieBreaks,
	); err != nil {
		return nil, err
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
		metadata:                         metadata,
		cues:                             cues,
		cueByID:                          cueByID,
		concepts:                         conceptByID,
		rankAliasCollisionGroupsBySource: true,
	}, nil
}

func validateConditionalTieBreaks(
	document conditionalTieBreaksDocument,
) error {
	expected := []string{
		"evidence_set",
		"step_count",
		"source_position",
		"meaning_signature",
	}
	if !slices.Equal(
		document.LawAliasCollisionGroupsOverCandidateLimit,
		expected,
	) {
		return fmt.Errorf(
			"lawAliasCollisionGroupsOverCandidateLimit の tie-break が定義済みの完全順と一致しません",
		)
	}
	return nil
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
			ProfileID:  cue.ProfileID,
			CueID:      cue.CueID,
			MatchGroup: cue.MatchGroup,
			SyntaxRole: cue.SyntaxRole,
			Terms:      append([]string(nil), cue.Terms...),
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
	document *cueartifact.Artifact,
) (
	[]legalquery.CueVocabularyEntry,
	map[string]cueDefinition,
	error,
) {
	if err := document.ValidateEntries(validateCueEntry); err != nil {
		return nil, nil, err
	}
	entries := document.Entries()
	if err := validateRequiredUnsupportedIntentGroups(entries); err != nil {
		return nil, nil, err
	}
	definitions := make(map[string]cueDefinition, len(entries))
	for _, entry := range entries {
		intentGroup, _ := entry.IntentGroup()
		signal, err := validateCueSignal(entry)
		if err != nil {
			return nil, nil, err
		}
		definitions[entry.CueID()] = cueDefinition{
			category:    entry.Category(),
			value:       entry.Value(),
			intentGroup: intentGroup,
			signal:      signal,
			syntaxRole:  entry.SyntaxRole(),
		}
	}
	return document.Vocabulary(), definitions, nil
}

func validateCueEntry(document cueartifact.Entry) error {
	if !validCueMeaning(document.Category(), document.Value()) {
		return fmt.Errorf("category/value が未対応です")
	}
	if err := validateCueSyntaxRole(document); err != nil {
		return err
	}
	_, err := validateCueSignal(document)
	return err
}

func validateCueSyntaxRole(document cueartifact.Entry) error {
	role := document.SyntaxRole()
	switch document.Category() {
	case "unsupported":
		if role != legalquery.CueSyntaxRoleTaskExpression &&
			role != legalquery.CueSyntaxRoleTaskObject {
			return fmt.Errorf(
				"対象外 cue の syntaxRole は task_expression または task_object でなければなりません",
			)
		}
	case "task":
		if role != legalquery.CueSyntaxRoleTaskExpression &&
			role != legalquery.CueSyntaxRoleTaskPredicate {
			return fmt.Errorf(
				"task cue の syntaxRole は task_expression または task_predicate でなければなりません",
			)
		}
	case "syntax":
		if document.Value() == "task_predicate" {
			if role != legalquery.CueSyntaxRoleTaskPredicate {
				return fmt.Errorf(
					"task_predicate syntax cue の syntaxRole は task_predicate でなければなりません",
				)
			}
			break
		}
		if role != legalquery.CueSyntaxRoleNone {
			return fmt.Errorf(
				"task relation に使わない syntax cue の syntaxRole は none でなければなりません",
			)
		}
	default:
		if role != legalquery.CueSyntaxRoleNone {
			return fmt.Errorf(
				"task relation に使わない cue の syntaxRole は none でなければなりません",
			)
		}
	}
	return nil
}

func validateRequiredUnsupportedIntentGroups(
	cues []cueartifact.Entry,
) error {
	required := []string{
		"explicit_out_of_scope_task",
		"external_information_source",
		"legal_advice",
		"relationship_analysis",
		"translation",
		"unadopted_information_or_extension",
		"version_comparison",
	}
	seen := make(map[string]struct{}, len(required))
	for _, cue := range cues {
		if cue.Category() == "unsupported" {
			intentGroup, exists := cue.IntentGroup()
			if exists {
				seen[intentGroup] = struct{}{}
			}
		}
	}
	for _, intentGroup := range required {
		if _, exists := seen[intentGroup]; !exists {
			return fmt.Errorf(
				"対象外 cue に必須の intentGroup %q がありません",
				intentGroup,
			)
		}
	}
	return nil
}

func validateCueSignal(
	document cueartifact.Entry,
) (legalquery.CandidateGenerationSignal, error) {
	intentGroup, hasIntentGroup := document.IntentGroup()
	signalValue, hasSignal := document.Signal()
	if document.Category() != "unsupported" {
		if hasIntentGroup || hasSignal {
			return "", fmt.Errorf(
				"対象外以外の cue に intentGroup または signal は指定できません",
			)
		}
		return "", nil
	}
	if !hasIntentGroup || !validUnsupportedIntentGroup(intentGroup) {
		return "", fmt.Errorf("intentGroup が未対応です")
	}
	if !hasSignal {
		return "", fmt.Errorf("signal は必須です")
	}
	signal := legalquery.CandidateGenerationSignal(signalValue)
	expectedValue, expectedSignal, exists := unsupportedCueMapping(
		intentGroup,
	)
	if !exists ||
		document.Value() != expectedValue ||
		signal != expectedSignal {
		return "", fmt.Errorf(
			"intentGroup、value および signal の対応が一致しません",
		)
	}
	return signal, nil
}

func validUnsupportedIntentGroup(value string) bool {
	switch value {
	case "external_information_source",
		"explicit_out_of_scope_task",
		"legal_advice",
		"relationship_analysis",
		"translation",
		"unadopted_information_or_extension",
		"version_comparison":
		return true
	default:
		return false
	}
}

func unsupportedCueMapping(
	intentGroup string,
) (
	string,
	legalquery.CandidateGenerationSignal,
	bool,
) {
	switch intentGroup {
	case "legal_advice":
		return "legal_advice",
			legalquery.CandidateSignalUnsupportedLegalAdvice,
			true
	case "translation":
		return "translation",
			legalquery.CandidateSignalUnsupportedTranslation,
			true
	case "external_information_source",
		"explicit_out_of_scope_task",
		"relationship_analysis",
		"unadopted_information_or_extension",
		"version_comparison":
		return "task_or_resource",
			legalquery.CandidateSignalUnsupportedTaskOrResource,
			true
	default:
		return "", "", false
	}
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
			value == "individual" ||
			value == "resource_choice" ||
			value == "single_choice"
	case "syntax":
		return value == "content_result_unit" ||
			value == "related_law_scope" ||
			value == "task_predicate"
	case "safety":
		return value == "implicit_first_read"
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
