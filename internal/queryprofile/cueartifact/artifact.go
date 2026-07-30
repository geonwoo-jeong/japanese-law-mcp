// Package cueartifact は、query profile に共通する cue 成果物契約を提供する。
package cueartifact

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

// SchemaVersion は、CueTaskRelation に対応する cue 成果物 schema である。
const SchemaVersion = 3

// EntryValidator は、profile が所有する cue の意味を検証する。
type EntryValidator func(Entry) error

// Artifact は、構造と語彙衝突を検証した不変の cue 成果物である。
type Artifact struct {
	schemaVersion int
	profileID     string
	cueSetVersion string
	entries       []Entry
}

// Entry は、構造を検証した一つの cue entry である。
type Entry struct {
	cueID       string
	category    string
	value       string
	intentGroup optionalValue
	signal      optionalValue
	syntaxRole  legalquery.CueSyntaxRole
	matchGroup  string
	terms       []string
}

type optionalValue struct {
	value   string
	present bool
}

// SchemaVersion は、cue 成果物の schema version を返す。
func (a *Artifact) SchemaVersion() int {
	if a == nil {
		return 0
	}
	return a.schemaVersion
}

// ProfileID は、cue 成果物を所有する profile ID を返す。
func (a *Artifact) ProfileID() string {
	if a == nil {
		return ""
	}
	return a.profileID
}

// CueSetVersion は、cue 集合の版を返す。
func (a *Artifact) CueSetVersion() string {
	if a == nil {
		return ""
	}
	return a.cueSetVersion
}

// Entries は、cue entry の深い複製を返す。
func (a *Artifact) Entries() []Entry {
	if a == nil {
		return nil
	}
	result := make([]Entry, 0, len(a.entries))
	for _, entry := range a.entries {
		result = append(result, cloneEntry(entry))
	}
	return result
}

// Vocabulary は、共通前処理へ渡す cue 語彙の深い複製を返す。
func (a *Artifact) Vocabulary() []legalquery.CueVocabularyEntry {
	if a == nil {
		return nil
	}
	result := make([]legalquery.CueVocabularyEntry, 0, len(a.entries))
	for _, entry := range a.entries {
		result = append(result, legalquery.CueVocabularyEntry{
			ProfileID:  a.profileID,
			CueID:      entry.cueID,
			MatchGroup: entry.matchGroup,
			SyntaxRole: entry.syntaxRole,
			Terms:      append([]string(nil), entry.terms...),
		})
	}
	return result
}

// MatchProfile は、profile ID と cue 集合の版が一致することを確認する。
func (a *Artifact) MatchProfile(profileID string, cueSetVersion string) error {
	if a == nil {
		return fmt.Errorf("cue 成果物は必須です")
	}
	if a.profileID != profileID || a.cueSetVersion != cueSetVersion {
		return fmt.Errorf(
			"profile.json と cues.json の profileId または cueSetVersion が一致しません",
		)
	}
	return nil
}

// ValidateEntries は、共通構造の検証後に profile 固有の意味検証を実行する。
func (a *Artifact) ValidateEntries(validate EntryValidator) error {
	if a == nil {
		return fmt.Errorf("cue 成果物は必須です")
	}
	if validate == nil {
		return fmt.Errorf("cue の意味 validator は必須です")
	}
	for index, entry := range a.entries {
		if err := validate(cloneEntry(entry)); err != nil {
			return fmt.Errorf("cues[%d] の意味が有効ではありません: %w", index, err)
		}
	}
	return nil
}

// CueID は、成果物内で一意な cue ID を返す。
func (e Entry) CueID() string {
	return e.cueID
}

// Category は、profile が定義する cue category を返す。
func (e Entry) Category() string {
	return e.category
}

// Value は、category に対応する cue value を返す。
func (e Entry) Value() string {
	return e.value
}

// IntentGroup は、条件付き intent group と存在有無を返す。
func (e Entry) IntentGroup() (string, bool) {
	return e.intentGroup.value, e.intentGroup.present
}

// Signal は、条件付き signal と存在有無を返す。
func (e Entry) Signal() (string, bool) {
	return e.signal.value, e.signal.present
}

// SyntaxRole は、cue の構文上の役割を返す。
func (e Entry) SyntaxRole() legalquery.CueSyntaxRole {
	return e.syntaxRole
}

// Terms は、登録表現の深い複製を返す。
func (e Entry) Terms() []string {
	return append([]string(nil), e.terms...)
}

func cloneEntry(entry Entry) Entry {
	result := entry
	result.terms = append([]string(nil), entry.terms...)
	return result
}
