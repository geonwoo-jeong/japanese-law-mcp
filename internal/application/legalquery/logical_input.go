package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// LogicalInputKind は、provider 選択前の logical input variant を表す。
type LogicalInputKind string

const (
	// InputKindLawSearch は、法令名検索の logical input を表す。
	InputKindLawSearch LogicalInputKind = "law_search"
	// InputKindLawContentSearch は、法令本文検索の logical input を表す。
	InputKindLawContentSearch LogicalInputKind = "law_content_search"
	// InputKindLawRead は、法令本文読取りの logical input を表す。
	InputKindLawRead LogicalInputKind = "law_read"
	// InputKindLawArticleRead は、法令条文読取りの logical input を表す。
	InputKindLawArticleRead LogicalInputKind = "law_article_read"
	// InputKindLawUpdates は、法令更新一覧の logical input を表す。
	InputKindLawUpdates LogicalInputKind = "law_updates"
	// InputKindJudicialDecisionSearch は、裁判例検索の logical input を表す。
	InputKindJudicialDecisionSearch LogicalInputKind = "judicial_decision_search"
	// InputKindJudicialDecisionRead は、裁判例読取りの logical input を表す。
	InputKindJudicialDecisionRead LogicalInputKind = "judicial_decision_read"
)

// LogicalInput は、provider 選択前の取得条件として許可した variant の閉じた集合である。
type LogicalInput interface {
	InputKind() LogicalInputKind
	Validate() error
	legalQueryLogicalInput()
}

func cloneLogicalInput(input LogicalInput) (LogicalInput, error) {
	switch value := input.(type) {
	case LawSearchIntentV1:
		return value.clone(), nil
	case LawContentSearchIntentV1:
		return value.clone(), nil
	case LawReadIntentV1:
		return value.clone(), nil
	case LawArticleReadIntentV1:
		return value.clone(), nil
	case LawUpdateListIntentV1:
		return value, nil
	case JudicialDecisionSearchIntentV1:
		return value, nil
	case JudicialDecisionReadIntentV1:
		return value, nil
	default:
		return nil, fmt.Errorf("logicalInput は許可された値型の variant でなければなりません")
	}
}

func mustCloneLogicalInput(input LogicalInput) LogicalInput {
	cloned, err := cloneLogicalInput(input)
	if err != nil {
		panic(fmt.Sprintf("検証済み logicalInput の複製に失敗しました: %v", err))
	}
	return cloned
}

func cloneOptionalDate(value *model.Date) *model.Date {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneOptionalResourceRef(
	value *model.SourceResourceRef,
) *model.SourceResourceRef {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func optionalDate(value *model.Date) (model.Date, bool) {
	if value == nil {
		return model.Date{}, false
	}
	return *value, true
}

func optionalResourceRef(
	value *model.SourceResourceRef,
) (model.SourceResourceRef, bool) {
	if value == nil {
		return model.SourceResourceRef{}, false
	}
	return *value, true
}
