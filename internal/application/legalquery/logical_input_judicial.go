package legalquery

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// JudicialDecisionSearchIntentV1Values は、裁判例検索の logical input 値を保持する。
type JudicialDecisionSearchIntentV1Values struct {
	Query string
}

// JudicialDecisionSearchIntentV1 は、裁判例検索の provider 非依存条件である。
type JudicialDecisionSearchIntentV1 struct {
	query string
}

// NewJudicialDecisionSearchIntentV1 は、裁判例検索能力と同じ制約で検証した条件を返す。
func NewJudicialDecisionSearchIntentV1(
	values JudicialDecisionSearchIntentV1Values,
) (JudicialDecisionSearchIntentV1, error) {
	request, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{Query: values.Query},
	)
	if err != nil {
		return JudicialDecisionSearchIntentV1{}, fmt.Errorf(
			"judicial decision search intent が有効ではありません: %w",
			err,
		)
	}
	return JudicialDecisionSearchIntentV1{query: request.Query()}, nil
}

// Query は、正規化済みの裁判例検索語を返す。
func (i JudicialDecisionSearchIntentV1) Query() string {
	return i.query
}

// InputKind は、judicial_decision_search variant を返す。
func (i JudicialDecisionSearchIntentV1) InputKind() LogicalInputKind {
	return InputKindJudicialDecisionSearch
}

// Validate は、judicial-decision.search@1 へ変換できることを検証する。
func (i JudicialDecisionSearchIntentV1) Validate() error {
	_, err := judicialdecisionsearch.NewRequest(
		judicialdecisionsearch.RequestValues{Query: i.query},
	)
	return err
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*JudicialDecisionSearchIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialDecisionSearchIntentV1 は JSON から直接復元できません。NewJudicialDecisionSearchIntentV1 を使用してください",
	)
}

func (JudicialDecisionSearchIntentV1) legalQueryLogicalInput() {}

// JudicialDecisionReadIntentV1Values は、裁判例読取りの logical input 値を保持する。
type JudicialDecisionReadIntentV1Values struct {
	Ref model.SourceResourceRef
}

// JudicialDecisionReadIntentV1 は、入力 ref だけを保持する裁判例読取り条件である。
type JudicialDecisionReadIntentV1 struct {
	ref model.SourceResourceRef
}

// NewJudicialDecisionReadIntentV1 は、構造検証済みの裁判例参照を返す。
func NewJudicialDecisionReadIntentV1(
	values JudicialDecisionReadIntentV1Values,
) (JudicialDecisionReadIntentV1, error) {
	request, err := judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{Ref: values.Ref},
	)
	if err != nil {
		return JudicialDecisionReadIntentV1{}, fmt.Errorf(
			"judicial decision read intent が有効ではありません: %w",
			err,
		)
	}
	return JudicialDecisionReadIntentV1{ref: request.Ref()}, nil
}

// Ref は、入力で受け取った裁判例参照を返す。
func (i JudicialDecisionReadIntentV1) Ref() model.SourceResourceRef {
	return i.ref
}

// InputKind は、judicial_decision_read variant を返す。
func (i JudicialDecisionReadIntentV1) InputKind() LogicalInputKind {
	return InputKindJudicialDecisionRead
}

// Validate は、judicial-decision.read@1 へ変換できる参照を検証する。
func (i JudicialDecisionReadIntentV1) Validate() error {
	_, err := judicialdecisionread.NewRequest(
		judicialdecisionread.RequestValues{Ref: i.ref},
	)
	return err
}

// UnmarshalJSON は、constructor を介さない直接復元を拒否する。
func (*JudicialDecisionReadIntentV1) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialDecisionReadIntentV1 は JSON から直接復元できません。NewJudicialDecisionReadIntentV1 を使用してください",
	)
}

func (JudicialDecisionReadIntentV1) legalQueryLogicalInput() {}
