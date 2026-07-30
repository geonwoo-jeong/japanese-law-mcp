package legalquery

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

const maxPreprocessCueTaskRelations = 128

// CueSyntaxRole は、共通前処理へ注入した cue の構文上の役割である。
type CueSyntaxRole string

const (
	// CueSyntaxRoleNone は、task relation の構成要素にしない cue を表す。
	CueSyntaxRoleNone CueSyntaxRole = "none"
	// CueSyntaxRoleTaskExpression は、登録表現自体が完結した task であることを表す。
	CueSyntaxRoleTaskExpression CueSyntaxRole = "task_expression"
	// CueSyntaxRoleTaskObject は、task の対象となる cue を表す。
	CueSyntaxRoleTaskObject CueSyntaxRole = "task_object"
	// CueSyntaxRoleTaskPredicate は、先行する task object と結び付く述語を表す。
	CueSyntaxRoleTaskPredicate CueSyntaxRole = "task_predicate"
)

// CueTaskRelationKind は、cue 出現から確認した task 関係の形を表す。
type CueTaskRelationKind string

const (
	// CueTaskRelationDirectTask は、完結した task 表現を表す。
	CueTaskRelationDirectTask CueTaskRelationKind = "direct_task"
	// CueTaskRelationObjectPredicate は、目的語と述語の直接接続を表す。
	CueTaskRelationObjectPredicate CueTaskRelationKind = "object_predicate"
	// CueTaskRelationStandaloneTask は、短縮された単独 task を表す。
	CueTaskRelationStandaloneTask CueTaskRelationKind = "standalone_task"
)

// CueTaskRelationRefValues は、同じ前処理結果にある cue 出現の参照を構築する値である。
type CueTaskRelationRefValues struct {
	ProfileID string
	CueID     string
	Span      QuerySpan
}

// CueTaskRelationRef は、profile、cue および原文 span で一つの cue 出現を参照する。
type CueTaskRelationRef struct {
	profileID string
	cueID     string
	span      QuerySpan
}

// NewCueTaskRelationRef は、必須 ID と span を持つ cue 出現参照を返す。
func NewCueTaskRelationRef(
	values CueTaskRelationRefValues,
) (CueTaskRelationRef, error) {
	ref := CueTaskRelationRef{
		profileID: values.ProfileID,
		cueID:     values.CueID,
		span:      values.Span,
	}
	if err := ref.Validate(); err != nil {
		return CueTaskRelationRef{}, err
	}
	return ref, nil
}

// ProfileID は、参照先 cue の profile ID を返す。
func (r CueTaskRelationRef) ProfileID() string {
	return r.profileID
}

// CueID は、参照先 cue の ID を返す。
func (r CueTaskRelationRef) CueID() string {
	return r.cueID
}

// Span は、参照先 cue の原文 span を返す。
func (r CueTaskRelationRef) Span() QuerySpan {
	return r.span
}

// Validate は、cue 出現参照の必須 ID と span を確認する。
func (r CueTaskRelationRef) Validate() error {
	if err := validateRequiredInternalText("profileId", r.profileID); err != nil {
		return err
	}
	if err := validateRequiredInternalText("cueId", r.cueID); err != nil {
		return err
	}
	if err := r.span.Validate(); err != nil {
		return fmt.Errorf("span が有効ではありません: %w", err)
	}
	return nil
}

// CueTaskRelationValues は、検証済み cue 出現から task 関係を構築する値である。
// Query と syntax role は構築時の検証だけに使用し、relation へ保持しない。
type CueTaskRelationValues struct {
	Query         string
	Subject       CueMention
	Predicate     CueMention
	SubjectRole   CueSyntaxRole
	PredicateRole CueSyntaxRole
	ClauseSpan    QuerySpan
	Kind          CueTaskRelationKind
}

// CueTaskRelation は、同じ節にある cue 出現の task 関係を表す不変な sidecar である。
type CueTaskRelation struct {
	subject    CueTaskRelationRef
	predicate  CueTaskRelationRef
	clauseSpan QuerySpan
	kind       CueTaskRelationKind
}

// NewCueTaskRelation は、原文、cue 出現、syntax role および kind の対応を検証する。
func NewCueTaskRelation(
	values CueTaskRelationValues,
) (CueTaskRelation, error) {
	if !utf8.ValidString(values.Query) {
		return CueTaskRelation{}, fmt.Errorf("query は有効な UTF-8 でなければなりません")
	}
	if err := validateCueMentionForRelation(
		values.Query,
		"subject",
		values.Subject,
	); err != nil {
		return CueTaskRelation{}, err
	}
	if err := validateCueMentionForRelation(
		values.Query,
		"predicate",
		values.Predicate,
	); err != nil {
		return CueTaskRelation{}, err
	}
	if err := validateCueSyntaxRole(values.SubjectRole); err != nil {
		return CueTaskRelation{}, fmt.Errorf("subjectRole が有効ではありません: %w", err)
	}
	if err := validateCueSyntaxRole(values.PredicateRole); err != nil {
		return CueTaskRelation{}, fmt.Errorf("predicateRole が有効ではありません: %w", err)
	}

	subject, err := newCueTaskRelationRefFromMention(values.Subject)
	if err != nil {
		return CueTaskRelation{}, fmt.Errorf("subject が有効ではありません: %w", err)
	}
	predicate, err := newCueTaskRelationRefFromMention(values.Predicate)
	if err != nil {
		return CueTaskRelation{}, fmt.Errorf("predicate が有効ではありません: %w", err)
	}
	relation := CueTaskRelation{
		subject:    subject,
		predicate:  predicate,
		clauseSpan: values.ClauseSpan,
		kind:       values.Kind,
	}
	if err := relation.Validate(); err != nil {
		return CueTaskRelation{}, err
	}
	if err := validateRelationSpanInQuery(
		values.Query,
		"clauseSpan",
		values.ClauseSpan,
	); err != nil {
		return CueTaskRelation{}, err
	}
	if err := validateCueTaskRelationRoles(
		relation.kind,
		values.SubjectRole,
		values.PredicateRole,
	); err != nil {
		return CueTaskRelation{}, err
	}
	if err := validateCueTaskRelationText(values.Query, relation); err != nil {
		return CueTaskRelation{}, err
	}
	return relation, nil
}

// Subject は、task として解釈する cue 出現の参照を返す。
func (r CueTaskRelation) Subject() CueTaskRelationRef {
	return cloneCueTaskRelationRef(r.subject)
}

// Predicate は、task 述語となる cue 出現の参照を返す。
func (r CueTaskRelation) Predicate() CueTaskRelationRef {
	return cloneCueTaskRelationRef(r.predicate)
}

// ClauseSpan は、relation を確認した一つの節の原文 span を返す。
func (r CueTaskRelation) ClauseSpan() QuerySpan {
	return r.clauseSpan
}

// Kind は、検証済み task 関係の形を返す。
func (r CueTaskRelation) Kind() CueTaskRelationKind {
	return r.kind
}

// Validate は、参照、節への包含、profile および kind の構造を確認する。
// syntax role と原文上の直接接続は constructor だけで検証する。
func (r CueTaskRelation) Validate() error {
	if err := r.subject.Validate(); err != nil {
		return fmt.Errorf("subject が有効ではありません: %w", err)
	}
	if err := r.predicate.Validate(); err != nil {
		return fmt.Errorf("predicate が有効ではありません: %w", err)
	}
	if err := r.clauseSpan.Validate(); err != nil {
		return fmt.Errorf("clauseSpan が有効ではありません: %w", err)
	}
	if !querySpanContains(r.clauseSpan, r.subject.Span()) ||
		!querySpanContains(r.clauseSpan, r.predicate.Span()) {
		return fmt.Errorf("clauseSpan は subject と predicate を含まなければなりません")
	}
	if r.subject.ProfileID() != r.predicate.ProfileID() {
		return fmt.Errorf("subject と predicate の profileId は一致しなければなりません")
	}
	switch r.kind {
	case CueTaskRelationDirectTask, CueTaskRelationStandaloneTask:
		if r.subject != r.predicate {
			return fmt.Errorf("%s は同じ cue 出現を参照しなければなりません", r.kind)
		}
	case CueTaskRelationObjectPredicate:
		if r.subject.Span().EndByte() > r.predicate.Span().StartByte() {
			return fmt.Errorf("object_predicate の subject は predicate より前でなければなりません")
		}
	default:
		return fmt.Errorf("cue task relation の kind が定義されていません")
	}
	return nil
}

func validateCueMentionForRelation(
	query string,
	name string,
	mention CueMention,
) error {
	if err := mention.Validate(); err != nil {
		return fmt.Errorf("%s が有効ではありません: %w", name, err)
	}
	if err := validateMentionSurface(query, mention.Span(), mention.Surface()); err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func newCueTaskRelationRefFromMention(
	mention CueMention,
) (CueTaskRelationRef, error) {
	return NewCueTaskRelationRef(CueTaskRelationRefValues{
		ProfileID: mention.ProfileID(),
		CueID:     mention.CueID(),
		Span:      mention.Span(),
	})
}

func validateCueSyntaxRole(role CueSyntaxRole) error {
	switch role {
	case CueSyntaxRoleNone,
		CueSyntaxRoleTaskExpression,
		CueSyntaxRoleTaskObject,
		CueSyntaxRoleTaskPredicate:
		return nil
	default:
		return fmt.Errorf("cue syntaxRole が定義されていません")
	}
}

func validateCueTaskRelationRoles(
	kind CueTaskRelationKind,
	subjectRole CueSyntaxRole,
	predicateRole CueSyntaxRole,
) error {
	switch kind {
	case CueTaskRelationDirectTask:
		if subjectRole != CueSyntaxRoleTaskExpression ||
			predicateRole != CueSyntaxRoleTaskExpression {
			return fmt.Errorf(
				"direct_task は subject と predicate に task_expression を必要とします",
			)
		}
	case CueTaskRelationObjectPredicate:
		if subjectRole != CueSyntaxRoleTaskObject ||
			predicateRole != CueSyntaxRoleTaskPredicate {
			return fmt.Errorf(
				"object_predicate は task_object と task_predicate を必要とします",
			)
		}
	case CueTaskRelationStandaloneTask:
		if subjectRole != CueSyntaxRoleTaskObject ||
			predicateRole != CueSyntaxRoleTaskObject {
			return fmt.Errorf(
				"standalone_task は subject と predicate に task_object を必要とします",
			)
		}
	default:
		return fmt.Errorf("cue task relation の kind が定義されていません")
	}
	return nil
}

func validateCueTaskRelationText(query string, relation CueTaskRelation) error {
	clause := relation.ClauseSpan()
	subject := relation.Subject().Span()
	predicate := relation.Predicate().Span()
	switch relation.Kind() {
	case CueTaskRelationDirectTask:
		if !onlyUnicodeWhitespace(query[predicate.EndByte():clause.EndByte()]) {
			return fmt.Errorf("direct_task の cue より後ろは空白だけでなければなりません")
		}
	case CueTaskRelationObjectPredicate:
		connector := query[subject.EndByte():predicate.StartByte()]
		if strings.TrimFunc(connector, unicode.IsSpace) != "を" {
			return fmt.Errorf("object_predicate は空白と助詞 を だけで直接接続しなければなりません")
		}
		if !onlyUnicodeWhitespace(query[predicate.EndByte():clause.EndByte()]) {
			return fmt.Errorf("object_predicate の predicate より後ろは空白だけでなければなりません")
		}
	case CueTaskRelationStandaloneTask:
		if !onlyUnicodeWhitespace(query[clause.StartByte():subject.StartByte()]) ||
			!onlyUnicodeWhitespace(query[subject.EndByte():clause.EndByte()]) {
			return fmt.Errorf("standalone_task の cue の外側は空白だけでなければなりません")
		}
	}
	return nil
}

func validateRelationSpanInQuery(
	query string,
	name string,
	span QuerySpan,
) error {
	if err := span.Validate(); err != nil {
		return fmt.Errorf("%s が有効ではありません: %w", name, err)
	}
	if span.EndByte() > len(query) {
		return fmt.Errorf("%s が query の範囲を超えています", name)
	}
	if !isRuneBoundary(query, span.StartByte()) ||
		!isRuneBoundary(query, span.EndByte()) {
		return fmt.Errorf("%s の両端は UTF-8 rune 境界でなければなりません", name)
	}
	return nil
}

func querySpanContains(container QuerySpan, value QuerySpan) bool {
	return container.StartByte() <= value.StartByte() &&
		value.EndByte() <= container.EndByte()
}

func onlyUnicodeWhitespace(value string) bool {
	for _, character := range value {
		if !unicode.IsSpace(character) {
			return false
		}
	}
	return true
}

func cloneCueTaskRelationRef(value CueTaskRelationRef) CueTaskRelationRef {
	return CueTaskRelationRef{
		profileID: value.profileID,
		cueID:     value.cueID,
		span: QuerySpan{
			startByte: value.span.startByte,
			endByte:   value.span.endByte,
		},
	}
}

func cloneCueTaskRelations(values []CueTaskRelation) []CueTaskRelation {
	if len(values) == 0 {
		return nil
	}
	result := make([]CueTaskRelation, len(values))
	for index, value := range values {
		result[index] = CueTaskRelation{
			subject:    cloneCueTaskRelationRef(value.subject),
			predicate:  cloneCueTaskRelationRef(value.predicate),
			clauseSpan: value.clauseSpan,
			kind:       value.kind,
		}
	}
	return result
}

func validateCueTaskRelationSequence(
	query string,
	values []CueTaskRelation,
	cues []CueMention,
) error {
	if err := validateCueTaskRelationReferencesAndOrder(values, cues); err != nil {
		return err
	}
	for index, relation := range values {
		if err := validateRelationSpanInQuery(
			query,
			fmt.Sprintf("cueTaskRelations[%d].clauseSpan", index),
			relation.ClauseSpan(),
		); err != nil {
			return err
		}
		for name, ref := range map[string]CueTaskRelationRef{
			"subject":   relation.Subject(),
			"predicate": relation.Predicate(),
		} {
			if err := validateRelationSpanInQuery(
				query,
				fmt.Sprintf("cueTaskRelations[%d].%s.span", index, name),
				ref.Span(),
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateCueTaskRelationReferencesAndOrder(
	values []CueTaskRelation,
	cues []CueMention,
) error {
	if len(values) > maxPreprocessCueTaskRelations {
		return fmt.Errorf(
			"cueTaskRelations は %d 件以下でなければなりません",
			maxPreprocessCueTaskRelations,
		)
	}
	cueRefs := make(map[CueTaskRelationRef]struct{}, len(cues))
	for _, cue := range cues {
		ref, err := newCueTaskRelationRefFromMention(cue)
		if err != nil {
			return fmt.Errorf("cue mention の参照を作成できません: %w", err)
		}
		cueRefs[ref] = struct{}{}
	}
	for index, relation := range values {
		if err := relation.Validate(); err != nil {
			return fmt.Errorf(
				"cueTaskRelations[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		if _, exists := cueRefs[relation.Subject()]; !exists {
			return fmt.Errorf(
				"cueTaskRelations[%d].subject の参照先 cue が存在しません",
				index,
			)
		}
		if _, exists := cueRefs[relation.Predicate()]; !exists {
			return fmt.Errorf(
				"cueTaskRelations[%d].predicate の参照先 cue が存在しません",
				index,
			)
		}
		if index > 0 &&
			compareCueTaskRelation(values[index-1], relation) >= 0 {
			return fmt.Errorf(
				"cueTaskRelations は正規順で重複なく保持しなければなりません",
			)
		}
	}
	return nil
}

func compareCueTaskRelation(left CueTaskRelation, right CueTaskRelation) int {
	leftSubject := left.Subject()
	rightSubject := right.Subject()
	leftPredicate := left.Predicate()
	rightPredicate := right.Predicate()
	switch {
	case left.ClauseSpan().StartByte() < right.ClauseSpan().StartByte():
		return -1
	case left.ClauseSpan().StartByte() > right.ClauseSpan().StartByte():
		return 1
	case leftSubject.Span().StartByte() < rightSubject.Span().StartByte():
		return -1
	case leftSubject.Span().StartByte() > rightSubject.Span().StartByte():
		return 1
	case leftPredicate.Span().StartByte() < rightPredicate.Span().StartByte():
		return -1
	case leftPredicate.Span().StartByte() > rightPredicate.Span().StartByte():
		return 1
	case leftSubject.ProfileID() != rightSubject.ProfileID():
		return strings.Compare(leftSubject.ProfileID(), rightSubject.ProfileID())
	case leftSubject.CueID() != rightSubject.CueID():
		return strings.Compare(leftSubject.CueID(), rightSubject.CueID())
	case leftPredicate.CueID() != rightPredicate.CueID():
		return strings.Compare(leftPredicate.CueID(), rightPredicate.CueID())
	case left.Kind() != right.Kind():
		return strings.Compare(string(left.Kind()), string(right.Kind()))
	case left.ClauseSpan().EndByte() < right.ClauseSpan().EndByte():
		return -1
	case left.ClauseSpan().EndByte() > right.ClauseSpan().EndByte():
		return 1
	case leftSubject.Span().EndByte() < rightSubject.Span().EndByte():
		return -1
	case leftSubject.Span().EndByte() > rightSubject.Span().EndByte():
		return 1
	case leftPredicate.Span().EndByte() < rightPredicate.Span().EndByte():
		return -1
	case leftPredicate.Span().EndByte() > rightPredicate.Span().EndByte():
		return 1
	default:
		return 0
	}
}
