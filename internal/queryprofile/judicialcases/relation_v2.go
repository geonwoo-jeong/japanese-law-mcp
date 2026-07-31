package judicialcases

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
	cueIntentEvidenceJudicial
)

type cueRelationRefKey struct {
	profileID string
	cueID     string
	startByte int
	endByte   int
}

func newCueTaskRelationV2Profile(profile *Profile) (*Profile, error) {
	if profile == nil {
		return nil, fmt.Errorf("judicial-cases profile は必須です")
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
		cueMeaningKey("task", "search"): false,
		cueMeaningKey("task", "read"):   false,
	}
	for cueID, definition := range definitions {
		want := legalquery.CueSyntaxRoleNone
		if definition.category == "task" {
			want = legalquery.CueSyntaxRoleTaskExpression
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
	for _, relation := range input.CueTaskRelations() {
		subject := relation.Subject()
		if subject.ProfileID() != p.metadata.ProfileID() {
			continue
		}
		definition, exists := p.cueByID[subject.CueID()]
		if !exists {
			return resolvedCues{}, fmt.Errorf(
				"relation が未登録の judicial-cases cueId %q を参照しています",
				subject.CueID(),
			)
		}
		if definition.category == "task" &&
			relation.Kind() == legalquery.CueTaskRelationDirectTask {
			taskRefs[cueRelationRefKeyFromRelationRef(subject)] = struct{}{}
		}
	}

	filtered := make(map[string][]legalquery.CueMention, len(cues.mentions))
	for meaning, mentions := range cues.mentions {
		for _, mention := range mentions {
			definition := p.cueByID[mention.CueID()]
			if cueOverlapsQuotedQueryTerm(
				mention,
				input.QueryTermMentions(),
			) {
				continue
			}
			if definition.category == "task" {
				if _, exists := taskRefs[cueRelationRefKeyFromMention(mention)]; !exists {
					continue
				}
			}
			filtered[meaning] = append(filtered[meaning], mention)
		}
	}
	return resolvedCues{mentions: filtered}, nil
}

func cueOverlapsQuotedQueryTerm(
	cue legalquery.CueMention,
	terms []legalquery.QueryTermMention,
) bool {
	for _, term := range terms {
		if term.Kind() == legalquery.QueryTermMentionQuotedPhrase &&
			cue.Span().StartByte() < term.Span().EndByte() &&
			term.Span().StartByte() < cue.Span().EndByte() {
			return true
		}
	}
	return false
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
