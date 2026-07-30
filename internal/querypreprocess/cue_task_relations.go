package querypreprocess

import (
	"context"
	"fmt"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/nlp/kagome"
)

const maximumCueTaskRelations = 128

var (
	mentionSuffixes = [...]string{
		"という語",
		"という言葉",
		"という表現",
		"という用語",
		"という文字列",
		"という文言",
	}
	taskObjectTopicSuffixes = [...]string{
		"に関する",
		"に関して",
		"について",
		"に係る",
	}
)

type cueClause struct {
	span legalquery.QuerySpan
}

func (p *Preprocessor) buildCueTaskRelations(
	ctx context.Context,
	query string,
	tokens []kagome.TokenOccurrence,
	cues []legalquery.CueMention,
	queryTerms []legalquery.QueryTermMention,
) ([]legalquery.CueTaskRelation, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	clauses, err := splitCueRelationClauses(query)
	if err != nil {
		return nil, err
	}
	relations := make([]legalquery.CueTaskRelation, 0)
	seen := make(map[legalquery.CueTaskRelation]struct{})
	appendRelation := func(relation legalquery.CueTaskRelation) error {
		if _, exists := seen[relation]; exists {
			return nil
		}
		if len(relations) == maximumCueTaskRelations {
			return fmt.Errorf(
				"cue task relation は %d 件以下でなければなりません",
				maximumCueTaskRelations,
			)
		}
		seen[relation] = struct{}{}
		relations = append(relations, relation)
		return nil
	}

	for index, subject := range cues {
		if index%16 == 0 {
			if err := ctx.Err(); err != nil {
				return nil, err
			}
		}
		targetKey := cueKey(subject.ProfileID(), subject.CueID())
		target, exists := p.cuesByKey[targetKey]
		if !exists {
			return nil, fmt.Errorf(
				"profileId=%q, cueId=%q の syntaxRole が見つかりません",
				subject.ProfileID(),
				subject.CueID(),
			)
		}
		clause, belongs := cueClauseForMention(subject, clauses)
		if !belongs ||
			cueMentionIsExcluded(
				query,
				subject,
				target.syntaxRole,
				clause,
				queryTerms,
			) {
			continue
		}

		switch target.syntaxRole {
		case legalquery.CueSyntaxRoleTaskExpression:
			if onlyCueUnicodeWhitespace(
				query[subject.Span().EndByte():clause.span.EndByte()],
			) {
				relation, relationErr := newCueTaskRelation(
					query,
					subject,
					subject,
					target.syntaxRole,
					target.syntaxRole,
					clause.span,
					legalquery.CueTaskRelationDirectTask,
				)
				if relationErr != nil {
					return nil, relationErr
				}
				if err := appendRelation(relation); err != nil {
					return nil, err
				}
			}
		case legalquery.CueSyntaxRoleTaskObject:
			if onlyCueUnicodeWhitespace(
				query[clause.span.StartByte():subject.Span().StartByte()],
			) && onlyCueUnicodeWhitespace(
				query[subject.Span().EndByte():clause.span.EndByte()],
			) {
				relation, relationErr := newCueTaskRelation(
					query,
					subject,
					subject,
					target.syntaxRole,
					target.syntaxRole,
					clause.span,
					legalquery.CueTaskRelationStandaloneTask,
				)
				if relationErr != nil {
					return nil, relationErr
				}
				if err := appendRelation(relation); err != nil {
					return nil, err
				}
			}
			for predicateIndex, predicate := range cues {
				if predicateIndex%16 == 0 {
					if err := ctx.Err(); err != nil {
						return nil, err
					}
				}
				if predicate.ProfileID() != subject.ProfileID() ||
					predicate.Span().StartByte() < subject.Span().EndByte() {
					continue
				}
				predicateKey := cueKey(
					predicate.ProfileID(),
					predicate.CueID(),
				)
				predicateTarget, predicateExists :=
					p.cuesByKey[predicateKey]
				if !predicateExists {
					return nil, fmt.Errorf(
						"profileId=%q, cueId=%q の syntaxRole が見つかりません",
						predicate.ProfileID(),
						predicate.CueID(),
					)
				}
				if predicateTarget.syntaxRole !=
					legalquery.CueSyntaxRoleTaskPredicate &&
					predicateTarget.syntaxRole !=
						legalquery.CueSyntaxRoleTaskExpression {
					continue
				}
				predicateClause, sameClause :=
					cueClauseForMention(predicate, []cueClause{clause})
				if !sameClause ||
					cueMentionIsExcluded(
						query,
						predicate,
						predicateTarget.syntaxRole,
						predicateClause,
						queryTerms,
					) ||
					!onlyCueUnicodeWhitespace(
						query[predicate.Span().EndByte():clause.span.EndByte()],
					) ||
					!hasAccusativeParticleConnection(
						query,
						tokens,
						subject.Span().EndByte(),
						predicate.Span().StartByte(),
					) {
					continue
				}
				relation, relationErr := newCueTaskRelation(
					query,
					subject,
					predicate,
					target.syntaxRole,
					predicateTarget.syntaxRole,
					clause.span,
					legalquery.CueTaskRelationObjectPredicate,
				)
				if relationErr != nil {
					return nil, relationErr
				}
				if err := appendRelation(relation); err != nil {
					return nil, err
				}
			}
		}
	}
	slices.SortFunc(relations, compareCueTaskRelations)
	return relations, nil
}

func splitCueRelationClauses(query string) ([]cueClause, error) {
	result := make([]cueClause, 0)
	startByte := 0
	appendClause := func(endByte int) error {
		if endByte <= startByte ||
			onlyCueUnicodeWhitespace(query[startByte:endByte]) {
			return nil
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: startByte,
			EndByte:   endByte,
		})
		if err != nil {
			return err
		}
		result = append(result, cueClause{span: span})
		return nil
	}
	for index, current := range query {
		if !isCueClauseBoundary(current) {
			continue
		}
		if err := appendClause(index); err != nil {
			return nil, err
		}
		startByte = index + utf8.RuneLen(current)
	}
	if err := appendClause(len(query)); err != nil {
		return nil, err
	}
	return result, nil
}

func isCueClauseBoundary(value rune) bool {
	switch value {
	case '。', '！', '？', '!', '?', ';', '；', '\r', '\n':
		return true
	default:
		return false
	}
}

func cueClauseForMention(
	mention legalquery.CueMention,
	clauses []cueClause,
) (cueClause, bool) {
	span := mention.Span()
	for _, clause := range clauses {
		if clause.span.StartByte() <= span.StartByte() &&
			span.EndByte() <= clause.span.EndByte() {
			return clause, true
		}
	}
	return cueClause{}, false
}

func cueMentionIsExcluded(
	query string,
	mention legalquery.CueMention,
	role legalquery.CueSyntaxRole,
	clause cueClause,
	queryTerms []legalquery.QueryTermMention,
) bool {
	for _, term := range queryTerms {
		if term.Kind() != legalquery.QueryTermMentionQuotedPhrase {
			continue
		}
		if term.Span().StartByte() <= mention.Span().StartByte() &&
			mention.Span().EndByte() <= term.Span().EndByte() {
			return true
		}
	}
	remaining := strings.TrimLeftFunc(
		query[mention.Span().EndByte():clause.span.EndByte()],
		unicode.IsSpace,
	)
	if hasAnyPrefix(remaining, mentionSuffixes[:]) {
		return true
	}
	return role == legalquery.CueSyntaxRoleTaskObject &&
		hasAnyPrefix(remaining, taskObjectTopicSuffixes[:])
}

func hasAnyPrefix(value string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}

func onlyCueUnicodeWhitespace(value string) bool {
	for _, current := range value {
		if !unicode.IsSpace(current) {
			return false
		}
	}
	return true
}

func hasAccusativeParticleConnection(
	query string,
	tokens []kagome.TokenOccurrence,
	startByte int,
	endByte int,
) bool {
	particleStart := startByte
	for particleStart < endByte {
		current, size := utf8.DecodeRuneInString(query[particleStart:endByte])
		if !unicode.IsSpace(current) {
			break
		}
		particleStart += size
	}
	particleEnd := endByte
	for particleStart < particleEnd {
		current, size := utf8.DecodeLastRuneInString(
			query[particleStart:particleEnd],
		)
		if !unicode.IsSpace(current) {
			break
		}
		particleEnd -= size
	}
	if query[particleStart:particleEnd] != "を" {
		return false
	}
	for _, token := range tokens {
		if token.StartByte() != particleStart ||
			token.EndByte() != particleEnd ||
			token.Surface() != "を" {
			continue
		}
		partOfSpeech := token.PartOfSpeech()
		return len(partOfSpeech) >= 2 &&
			partOfSpeech[0] == "助詞" &&
			partOfSpeech[1] == "格助詞"
	}
	return false
}

func newCueTaskRelation(
	query string,
	subject legalquery.CueMention,
	predicate legalquery.CueMention,
	subjectRole legalquery.CueSyntaxRole,
	predicateRole legalquery.CueSyntaxRole,
	clauseSpan legalquery.QuerySpan,
	kind legalquery.CueTaskRelationKind,
) (legalquery.CueTaskRelation, error) {
	relation, err := legalquery.NewCueTaskRelation(
		legalquery.CueTaskRelationValues{
			Query:         query,
			Subject:       subject,
			Predicate:     predicate,
			SubjectRole:   subjectRole,
			PredicateRole: predicateRole,
			ClauseSpan:    clauseSpan,
			Kind:          kind,
		},
	)
	if err != nil {
		return legalquery.CueTaskRelation{}, fmt.Errorf(
			"cue task relation を構築できません: %w",
			err,
		)
	}
	return relation, nil
}

func compareCueTaskRelations(
	left legalquery.CueTaskRelation,
	right legalquery.CueTaskRelation,
) int {
	leftSubject := left.Subject()
	rightSubject := right.Subject()
	leftPredicate := left.Predicate()
	rightPredicate := right.Predicate()
	switch {
	case left.ClauseSpan().StartByte() != right.ClauseSpan().StartByte():
		return compareIntegers(
			left.ClauseSpan().StartByte(),
			right.ClauseSpan().StartByte(),
		)
	case leftSubject.Span().StartByte() != rightSubject.Span().StartByte():
		return compareIntegers(
			leftSubject.Span().StartByte(),
			rightSubject.Span().StartByte(),
		)
	case leftPredicate.Span().StartByte() != rightPredicate.Span().StartByte():
		return compareIntegers(
			leftPredicate.Span().StartByte(),
			rightPredicate.Span().StartByte(),
		)
	case leftSubject.ProfileID() != rightSubject.ProfileID():
		return strings.Compare(
			leftSubject.ProfileID(),
			rightSubject.ProfileID(),
		)
	case leftSubject.CueID() != rightSubject.CueID():
		return strings.Compare(leftSubject.CueID(), rightSubject.CueID())
	case leftPredicate.CueID() != rightPredicate.CueID():
		return strings.Compare(leftPredicate.CueID(), rightPredicate.CueID())
	case left.Kind() != right.Kind():
		return strings.Compare(string(left.Kind()), string(right.Kind()))
	case left.ClauseSpan().EndByte() != right.ClauseSpan().EndByte():
		return compareIntegers(
			left.ClauseSpan().EndByte(),
			right.ClauseSpan().EndByte(),
		)
	case leftSubject.Span().EndByte() != rightSubject.Span().EndByte():
		return compareIntegers(
			leftSubject.Span().EndByte(),
			rightSubject.Span().EndByte(),
		)
	default:
		return compareIntegers(
			leftPredicate.Span().EndByte(),
			rightPredicate.Span().EndByte(),
		)
	}
}

func compareIntegers(left int, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}
