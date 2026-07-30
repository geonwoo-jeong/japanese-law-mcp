package legalquery

import (
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

func buildSharedTerminalSequences(
	result PreprocessResult,
) ([]SharedTerminalSequence, error) {
	topicSpans := sharedTerminalTopicSpans(result)
	sequences := make([]SharedTerminalSequence, 0)
	for _, relation := range result.CueTaskRelations() {
		if relation.Kind() != CueTaskRelationDirectTask {
			continue
		}
		topics, unique := uniqueMaximalSharedTerminalTopics(
			result.Query(),
			topicSpans,
			result.CueMentions(),
			relation,
		)
		if !unique {
			continue
		}
		sequence, err := newSharedTerminalSequence(topics, relation)
		if err != nil {
			return nil, fmt.Errorf(
				"共有末尾列を構築できません: %w",
				err,
			)
		}
		sequences = append(sequences, sequence)
	}
	slices.SortFunc(sequences, compareSharedTerminalSequence)
	if len(sequences) > maximumSharedTerminalSequences {
		return nil, fmt.Errorf(
			"sharedTerminalSequences は %d 件以下でなければなりません",
			maximumSharedTerminalSequences,
		)
	}
	return sequences, nil
}

func sharedTerminalTopicSpans(result PreprocessResult) []QuerySpan {
	seen := make(map[QuerySpan]struct{})
	add := func(span QuerySpan) {
		seen[span] = struct{}{}
	}
	for _, mention := range result.LawNameMentions() {
		add(mention.Span())
	}
	for _, mention := range result.LegalConceptMentions() {
		add(mention.Span())
	}
	for _, mention := range result.QueryTermMentions() {
		add(mention.Span())
	}
	resultSpans := make([]QuerySpan, 0, len(seen))
	for span := range seen {
		resultSpans = append(resultSpans, span)
	}
	slices.SortFunc(resultSpans, compareSharedTerminalTopicSpan)
	return resultSpans
}

func compareSharedTerminalTopicSpan(left QuerySpan, right QuerySpan) int {
	switch {
	case left.StartByte() < right.StartByte():
		return -1
	case left.StartByte() > right.StartByte():
		return 1
	case left.EndByte() > right.EndByte():
		return -1
	case left.EndByte() < right.EndByte():
		return 1
	default:
		return 0
	}
}

func uniqueMaximalSharedTerminalTopics(
	query string,
	allTopics []QuerySpan,
	cues []CueMention,
	relation CueTaskRelation,
) ([]QuerySpan, bool) {
	if relation.Subject() != relation.Predicate() {
		return nil, false
	}
	clause := relation.ClauseSpan()
	terminal := relation.Subject()
	if !querySpanContains(clause, terminal.Span()) ||
		!onlyUnicodeWhitespace(
			query[terminal.Span().EndByte():clause.EndByte()],
		) {
		return nil, false
	}

	topics := make([]QuerySpan, 0, len(allTopics))
	for _, span := range allTopics {
		if querySpanContains(clause, span) &&
			span.EndByte() <= terminal.Span().StartByte() {
			topics = append(topics, span)
		}
	}
	if len(topics) < minimumSharedTerminalSequenceTopics {
		return nil, false
	}

	predecessors := make([][]int, len(topics))
	for current := range topics {
		for previous := 0; previous < current; previous++ {
			if topics[previous].EndByte() > topics[current].StartByte() {
				continue
			}
			if matchesSharedTerminalSeparator(
				query[topics[previous].EndByte():topics[current].StartByte()],
			) {
				predecessors[current] = append(
					predecessors[current],
					previous,
				)
			}
		}
	}
	connectorCues := sharedTerminalConnectorCues(
		query,
		topics,
		predecessors,
		cues,
		terminal,
	)

	pathStates := sharedTerminalPathStates(
		topics,
		predecessors,
		cues,
		terminal,
		connectorCues,
	)
	var found []QuerySpan
	for index, span := range topics {
		if !matchesSharedTerminalTail(
			query[span.EndByte():terminal.Span().StartByte()],
		) {
			continue
		}
		state := pathStates[index]
		if state.count == 0 ||
			len(state.representative) < minimumSharedTerminalSequenceTopics {
			continue
		}
		if state.count > 1 {
			return nil, false
		}
		if found != nil &&
			!slices.Equal(found, state.representative) {
			return nil, false
		}
		found = append([]QuerySpan(nil), state.representative...)
	}
	if found == nil {
		return nil, false
	}
	return found, true
}

type sharedTerminalPathState struct {
	count          int
	representative []QuerySpan
}

func sharedTerminalPathStates(
	topics []QuerySpan,
	predecessors [][]int,
	cues []CueMention,
	terminal CueTaskRelationRef,
	connectorCues map[CueTaskRelationRef]struct{},
) []sharedTerminalPathState {
	states := make([]sharedTerminalPathState, len(topics))
	for current := range topics {
		if len(predecessors[current]) == 0 {
			if !hasIntermediateSharedTerminalCue(
				cues,
				terminal,
				topics[current].StartByte(),
				connectorCues,
			) {
				states[current] = sharedTerminalPathState{
					count:          1,
					representative: []QuerySpan{topics[current]},
				}
			}
			continue
		}
		for _, previous := range predecessors[current] {
			previousState := states[previous]
			if previousState.count == 0 {
				continue
			}
			if states[current].count == 0 {
				states[current].representative = append(
					append(
						[]QuerySpan(nil),
						previousState.representative...,
					),
					topics[current],
				)
			}
			states[current].count += previousState.count
			if states[current].count >= 2 {
				states[current].count = 2
				break
			}
		}
	}
	return states
}

func matchesSharedTerminalSeparator(value string) bool {
	switch strings.TrimFunc(value, unicode.IsSpace) {
	case "、", ",", "，", "と", "及び", "および", "並びに", "ならびに":
		return true
	default:
		return false
	}
}

func matchesSharedTerminalTail(value string) bool {
	switch strings.TrimFunc(value, unicode.IsSpace) {
	case "", "を", "について":
		return true
	default:
		return false
	}
}

func hasIntermediateSharedTerminalCue(
	cues []CueMention,
	terminal CueTaskRelationRef,
	startByte int,
	connectorCues map[CueTaskRelationRef]struct{},
) bool {
	for _, cue := range cues {
		ref, err := newCueTaskRelationRefFromMention(cue)
		if err != nil {
			return true
		}
		if ref == terminal {
			continue
		}
		if _, connector := connectorCues[ref]; connector {
			continue
		}
		span := cue.Span()
		if span.EndByte() > startByte &&
			span.StartByte() < terminal.Span().StartByte() {
			return true
		}
	}
	return false
}

func sharedTerminalConnectorCues(
	query string,
	topics []QuerySpan,
	predecessors [][]int,
	cues []CueMention,
	terminal CueTaskRelationRef,
) map[CueTaskRelationRef]struct{} {
	result := make(map[CueTaskRelationRef]struct{})
	cuesBySpan := make(map[QuerySpan][]CueTaskRelationRef)
	for _, cue := range cues {
		ref, err := newCueTaskRelationRefFromMention(cue)
		if err != nil || ref == terminal {
			continue
		}
		cuesBySpan[cue.Span()] = append(cuesBySpan[cue.Span()], ref)
	}
	addGap := func(startByte int, endByte int) {
		connectorSpan, exists := trimmedUnicodeContentSpan(
			query,
			startByte,
			endByte,
		)
		if !exists {
			return
		}
		for _, ref := range cuesBySpan[connectorSpan] {
			result[ref] = struct{}{}
		}
	}
	for current, previousIndexes := range predecessors {
		for _, previous := range previousIndexes {
			addGap(
				topics[previous].EndByte(),
				topics[current].StartByte(),
			)
		}
	}
	for _, topic := range topics {
		if matchesSharedTerminalTail(
			query[topic.EndByte():terminal.Span().StartByte()],
		) {
			addGap(
				topic.EndByte(),
				terminal.Span().StartByte(),
			)
		}
	}
	return result
}

func trimmedUnicodeContentSpan(
	query string,
	startByte int,
	endByte int,
) (QuerySpan, bool) {
	firstOffset := -1
	lastOffset := 0
	for offset, character := range query[startByte:endByte] {
		if unicode.IsSpace(character) {
			continue
		}
		if firstOffset < 0 {
			firstOffset = offset
		}
		lastOffset = offset + utf8.RuneLen(character)
	}
	if firstOffset < 0 {
		return QuerySpan{}, false
	}
	return QuerySpan{
		startByte: startByte + firstOffset,
		endByte:   startByte + lastOffset,
	}, true
}

func validateSharedTerminalSequenceReferencesAndOrder(
	sequences []SharedTerminalSequence,
	lawNames []LawNameMention,
	concepts []LegalConceptMention,
	queryTerms []QueryTermMention,
	relations []CueTaskRelation,
) error {
	if len(sequences) > maximumSharedTerminalSequences {
		return fmt.Errorf(
			"sharedTerminalSequences は %d 件以下でなければなりません",
			maximumSharedTerminalSequences,
		)
	}
	allowedTopics := make(map[QuerySpan]struct{})
	for _, mention := range lawNames {
		allowedTopics[mention.Span()] = struct{}{}
	}
	for _, mention := range concepts {
		allowedTopics[mention.Span()] = struct{}{}
	}
	for _, mention := range queryTerms {
		allowedTopics[mention.Span()] = struct{}{}
	}
	allowedRelations := make(map[CueTaskRelation]struct{}, len(relations))
	for _, relation := range relations {
		allowedRelations[relation] = struct{}{}
	}
	seenRelations := make(map[CueTaskRelation]struct{}, len(sequences))
	for index, sequence := range sequences {
		if err := sequence.Validate(); err != nil {
			return fmt.Errorf(
				"sharedTerminalSequences[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		for topicIndex, span := range sequence.topicSpans {
			if _, exists := allowedTopics[span]; !exists {
				return fmt.Errorf(
					"sharedTerminalSequences[%d].topicSpans[%d] と完全一致する出現がありません",
					index,
					topicIndex,
				)
			}
		}
		relation := sequence.terminalTaskRelation
		if _, exists := allowedRelations[relation]; !exists {
			return fmt.Errorf(
				"sharedTerminalSequences[%d].terminalTaskRelation が既存 relation と一致しません",
				index,
			)
		}
		if _, duplicate := seenRelations[relation]; duplicate {
			return fmt.Errorf(
				"一つの terminalTaskRelation に複数の sidecar を保持できません",
			)
		}
		seenRelations[relation] = struct{}{}
		if index > 0 &&
			compareSharedTerminalSequence(sequences[index-1], sequence) >= 0 {
			return fmt.Errorf(
				"sharedTerminalSequences は正規順で重複なく保持しなければなりません",
			)
		}
	}
	return nil
}
