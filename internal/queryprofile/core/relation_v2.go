package core

import (
	"fmt"
	"maps"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

type cueIntentEvidenceMode uint8

const (
	cueIntentEvidenceLegacy cueIntentEvidenceMode = iota
	cueIntentEvidenceRelationV2
)

type cueRelationRefKey struct {
	profileID string
	cueID     string
	startByte int
	endByte   int
}

func newCueTaskRelationV2Profile(profile *Profile) (*Profile, error) {
	if profile == nil {
		return nil, fmt.Errorf("core profile は必須です")
	}
	if err := validateRelationV2PositiveCueRoles(profile.cueByID); err != nil {
		return nil, err
	}
	result := *profile
	result.cues = cloneCueVocabularyEntries(profile.cues)
	result.cueByID = maps.Clone(profile.cueByID)
	result.concepts = maps.Clone(profile.concepts)
	result.intentEvidenceMode = cueIntentEvidenceRelationV2
	return &result, nil
}

func cloneCueVocabularyEntries(
	entries []legalquery.CueVocabularyEntry,
) []legalquery.CueVocabularyEntry {
	result := slices.Clone(entries)
	for index := range result {
		result[index].Terms = slices.Clone(result[index].Terms)
	}
	return result
}

func validateRelationV2PositiveCueRoles(
	definitions map[string]cueDefinition,
) error {
	required := map[string]bool{
		cueMeaningKey("task", "search"):           false,
		cueMeaningKey("task", "read"):             false,
		cueMeaningKey("task", "list_updates"):     false,
		cueMeaningKey("syntax", "task_predicate"): false,
	}
	for cueID, definition := range definitions {
		if definition.category == "unsupported" {
			continue
		}
		want := legalquery.CueSyntaxRoleNone
		switch {
		case definition.category == "task":
			want = legalquery.CueSyntaxRoleTaskExpression
		case definition.category == "syntax" &&
			definition.value == "task_predicate":
			want = legalquery.CueSyntaxRoleTaskPredicate
		}
		if definition.syntaxRole != want {
			return fmt.Errorf(
				"positive cue %q (%s/%s) の syntaxRole は %q でなければなりません",
				cueID,
				definition.category,
				definition.value,
				want,
			)
		}
		meaning := cueMeaningKey(definition.category, definition.value)
		if _, exists := required[meaning]; exists {
			required[meaning] = true
		}
	}
	for meaning, found := range required {
		if !found {
			return fmt.Errorf(
				"relation v2 に必要な positive cue %q がありません",
				meaning,
			)
		}
	}
	return nil
}

func (p *Profile) resolveRelationV2Cues(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) (resolvedCues, error) {
	taskRefs := make(map[cueRelationRefKey]struct{})
	unsupportedRefs := make(map[cueRelationRefKey]struct{})
	for _, relation := range input.CueTaskRelations() {
		subject := relation.Subject()
		if subject.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		definition, exists := p.cueByID[subject.CueID()]
		if !exists {
			return resolvedCues{}, fmt.Errorf(
				"relation が未登録の core cueId %q を参照しています",
				subject.CueID(),
			)
		}
		key := cueRelationRefKeyFromRelationRef(subject)
		switch definition.category {
		case "task":
			if relation.Kind() == legalquery.CueTaskRelationDirectTask {
				taskRefs[key] = struct{}{}
			}
		case "unsupported":
			unsupportedRefs[key] = struct{}{}
		}
	}

	filtered := make(map[string][]legalquery.CueMention, len(cues.mentions))
	for meaning, mentions := range cues.mentions {
		for _, mention := range mentions {
			definition := p.cueByID[mention.CueID()]
			if relationV2CueOverlapsQuotedQueryTerm(
				mention,
				input.QueryTermMentions(),
			) {
				continue
			}
			key := cueRelationRefKeyFromMention(mention)
			switch definition.category {
			case "task":
				if _, exists := taskRefs[key]; !exists {
					continue
				}
			case "unsupported":
				if _, exists := unsupportedRefs[key]; !exists {
					continue
				}
			case "reserved_pack":
				if p.relationV2CueIsContentTarget(input, cues, mention) {
					continue
				}
			}
			filtered[meaning] = append(filtered[meaning], mention)
		}
	}
	return resolvedCues{mentions: filtered}, nil
}

func cueRelationRefKeyFromMention(
	mention legalquery.CueMention,
) cueRelationRefKey {
	return cueRelationRefKey{
		profileID: mention.ProfileID(),
		cueID:     mention.CueID(),
		startByte: mention.Span().StartByte(),
		endByte:   mention.Span().EndByte(),
	}
}

func cueRelationRefKeyFromRelationRef(
	ref legalquery.CueTaskRelationRef,
) cueRelationRefKey {
	return cueRelationRefKey{
		profileID: ref.ProfileID(),
		cueID:     ref.CueID(),
		startByte: ref.Span().StartByte(),
		endByte:   ref.Span().EndByte(),
	}
}
