package legalquery

import "fmt"

const (
	minimumSharedTerminalSequenceTopics = 2
	maximumSharedTerminalSequenceTopics = 256
	maximumSharedTerminalSequences      = 128
)

// SharedTerminalSequence は、複数の主題が一つの節末 task cue を共有できることを
// 構文だけで確認した、一 request 限定の不変 sidecar である。
type SharedTerminalSequence struct {
	topicSpans           []QuerySpan
	terminalTaskRelation CueTaskRelation
}

func newSharedTerminalSequence(
	topicSpans []QuerySpan,
	terminalTaskRelation CueTaskRelation,
) (SharedTerminalSequence, error) {
	sequence := SharedTerminalSequence{
		topicSpans: append([]QuerySpan(nil), topicSpans...),
		terminalTaskRelation: cloneCueTaskRelation(
			terminalTaskRelation,
		),
	}
	if err := sequence.Validate(); err != nil {
		return SharedTerminalSequence{}, err
	}
	return sequence, nil
}

// TopicSpans は、原文順に並ぶ構造上の主題 span の複製を返す。
func (s SharedTerminalSequence) TopicSpans() []QuerySpan {
	return append([]QuerySpan(nil), s.topicSpans...)
}

// TerminalTaskRelation は、共有する節末 direct task relation の複製を返す。
func (s SharedTerminalSequence) TerminalTaskRelation() CueTaskRelation {
	return cloneCueTaskRelation(s.terminalTaskRelation)
}

// Validate は、sidecar 自体が保持する順序、包含および上限を確認する。
func (s SharedTerminalSequence) Validate() error {
	if len(s.topicSpans) < minimumSharedTerminalSequenceTopics ||
		len(s.topicSpans) > maximumSharedTerminalSequenceTopics {
		return fmt.Errorf(
			"topicSpans は %d 件以上 %d 件以下でなければなりません",
			minimumSharedTerminalSequenceTopics,
			maximumSharedTerminalSequenceTopics,
		)
	}
	if err := s.terminalTaskRelation.Validate(); err != nil {
		return fmt.Errorf(
			"terminalTaskRelation が有効ではありません: %w",
			err,
		)
	}
	if s.terminalTaskRelation.Kind() != CueTaskRelationDirectTask ||
		s.terminalTaskRelation.Subject() !=
			s.terminalTaskRelation.Predicate() {
		return fmt.Errorf(
			"terminalTaskRelation は同じ cue を参照する direct_task でなければなりません",
		)
	}

	clauseSpan := s.terminalTaskRelation.ClauseSpan()
	terminalSpan := s.terminalTaskRelation.Subject().Span()
	for index, span := range s.topicSpans {
		if err := span.Validate(); err != nil {
			return fmt.Errorf("topicSpans[%d] が有効ではありません: %w", index, err)
		}
		if !querySpanContains(clauseSpan, span) {
			return fmt.Errorf(
				"topicSpans[%d] は terminal relation の clauseSpan に含まれなければなりません",
				index,
			)
		}
		if span.EndByte() > terminalSpan.StartByte() {
			return fmt.Errorf(
				"topicSpans[%d] は terminal cue より前でなければなりません",
				index,
			)
		}
		if index > 0 &&
			s.topicSpans[index-1].EndByte() > span.StartByte() {
			return fmt.Errorf(
				"topicSpans は原文順で重ならず保持しなければなりません",
			)
		}
	}
	return nil
}

func cloneSharedTerminalSequence(
	value SharedTerminalSequence,
) SharedTerminalSequence {
	return SharedTerminalSequence{
		topicSpans: append([]QuerySpan(nil), value.topicSpans...),
		terminalTaskRelation: cloneCueTaskRelation(
			value.terminalTaskRelation,
		),
	}
}

func cloneSharedTerminalSequences(
	values []SharedTerminalSequence,
) []SharedTerminalSequence {
	if len(values) == 0 {
		return nil
	}
	result := make([]SharedTerminalSequence, len(values))
	for index, value := range values {
		result[index] = cloneSharedTerminalSequence(value)
	}
	return result
}

func compareSharedTerminalSequence(
	left SharedTerminalSequence,
	right SharedTerminalSequence,
) int {
	if relationOrder := compareCueTaskRelation(
		left.terminalTaskRelation,
		right.terminalTaskRelation,
	); relationOrder != 0 {
		return relationOrder
	}
	return compareQuerySpanSequences(left.topicSpans, right.topicSpans)
}

func compareQuerySpanSequences(left []QuerySpan, right []QuerySpan) int {
	limit := min(len(left), len(right))
	for index := 0; index < limit; index++ {
		switch {
		case left[index].StartByte() < right[index].StartByte():
			return -1
		case left[index].StartByte() > right[index].StartByte():
			return 1
		case left[index].EndByte() < right[index].EndByte():
			return -1
		case left[index].EndByte() > right[index].EndByte():
			return 1
		}
	}
	switch {
	case len(left) < len(right):
		return -1
	case len(left) > len(right):
		return 1
	default:
		return 0
	}
}
