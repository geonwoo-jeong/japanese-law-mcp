package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ExpectedStepValues は、評価上の意味署名を構成する step 値を保持する。
type ExpectedStepValues struct {
	Task         legalquery.Task
	Resource     legalquery.Resource
	InputKind    legalquery.LogicalInputKind
	LogicalInput legalquery.LogicalInput
}

// ExpectedStep は、route と内部 ID を持たない評価用 logical step である。
type ExpectedStep struct {
	task         legalquery.Task
	resource     legalquery.Resource
	inputKind    legalquery.LogicalInputKind
	logicalInput legalquery.LogicalInput
	initialized  bool
}

// NewExpectedStep は、七つの許可された対応を検証して不変な step を返す。
func NewExpectedStep(values ExpectedStepValues) (ExpectedStep, error) {
	input, err := cloneExpectedLogicalInput(values.LogicalInput)
	if err != nil {
		return ExpectedStep{}, err
	}
	step := ExpectedStep{
		task:         values.Task,
		resource:     values.Resource,
		inputKind:    values.InputKind,
		logicalInput: input,
		initialized:  true,
	}
	if err := step.Validate(); err != nil {
		return ExpectedStep{}, err
	}
	return step, nil
}

// Task は、step が行う作業を返す。
func (s ExpectedStep) Task() legalquery.Task {
	return s.task
}

// Resource は、step が対象にする資源を返す。
func (s ExpectedStep) Resource() legalquery.Resource {
	return s.resource
}

// InputKind は、logical input の variant を返す。
func (s ExpectedStep) InputKind() legalquery.LogicalInputKind {
	return s.inputKind
}

// LogicalInput は、provider 選択前の取得条件の複製を返す。
func (s ExpectedStep) LogicalInput() legalquery.LogicalInput {
	input, err := cloneExpectedLogicalInput(s.logicalInput)
	if err != nil {
		panic(fmt.Sprintf("検証済み expected logicalInput の複製に失敗しました: %v", err))
	}
	return input
}

// Validate は、初期化、七つの対応および logical input を確認する。
func (s ExpectedStep) Validate() error {
	if !s.initialized {
		return fmt.Errorf("ExpectedStep は NewExpectedStep で作成しなければなりません")
	}
	task, resource, exists := expectedStepSpecificationFor(s.inputKind)
	if !exists {
		return fmt.Errorf("expected step の inputKind が定義されていません")
	}
	if s.task != task || s.resource != resource {
		return fmt.Errorf("expected step の task、resource および inputKind の対応が許可されていません")
	}
	input, err := cloneExpectedLogicalInput(s.logicalInput)
	if err != nil {
		return err
	}
	if input.InputKind() != s.inputKind {
		return fmt.Errorf("expected step の inputKind と logicalInput が一致しません")
	}
	if err := input.Validate(); err != nil {
		return fmt.Errorf("expected step の logicalInput が有効ではありません: %w", err)
	}
	return nil
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedStep) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExpectedStep は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}

func expectedStepSpecificationFor(
	kind legalquery.LogicalInputKind,
) (legalquery.Task, legalquery.Resource, bool) {
	switch kind {
	case legalquery.InputKindLawSearch:
		return legalquery.TaskSearch, legalquery.ResourceLaw, true
	case legalquery.InputKindLawContentSearch:
		return legalquery.TaskSearch, legalquery.ResourceLawProvision, true
	case legalquery.InputKindLawRead:
		return legalquery.TaskRead, legalquery.ResourceLaw, true
	case legalquery.InputKindLawArticleRead:
		return legalquery.TaskRead, legalquery.ResourceLawProvision, true
	case legalquery.InputKindLawUpdates:
		return legalquery.TaskListUpdates, legalquery.ResourceLaw, true
	case legalquery.InputKindJudicialDecisionSearch:
		return legalquery.TaskSearch, legalquery.ResourceJudicialDecision, true
	case legalquery.InputKindJudicialDecisionRead:
		return legalquery.TaskRead, legalquery.ResourceJudicialDecision, true
	default:
		return "", "", false
	}
}

func cloneExpectedLogicalInput(
	input legalquery.LogicalInput,
) (legalquery.LogicalInput, error) {
	switch value := input.(type) {
	case legalquery.LawSearchIntentV1:
		return cloneExpectedLawSearch(value)
	case legalquery.LawContentSearchIntentV1:
		return cloneExpectedLawContentSearch(value)
	case legalquery.LawReadIntentV1:
		return cloneExpectedLawRead(value)
	case legalquery.LawArticleReadIntentV1:
		return cloneExpectedLawArticleRead(value)
	case legalquery.LawUpdateListIntentV1:
		return legalquery.NewLawUpdateListIntentV1(
			legalquery.LawUpdateListIntentV1Values{Date: value.Date()},
		)
	case legalquery.JudicialDecisionSearchIntentV1:
		return legalquery.NewJudicialDecisionSearchIntentV1(
			legalquery.JudicialDecisionSearchIntentV1Values{Query: value.Query()},
		)
	case legalquery.JudicialDecisionReadIntentV1:
		return legalquery.NewJudicialDecisionReadIntentV1(
			legalquery.JudicialDecisionReadIntentV1Values{Ref: value.Ref()},
		)
	default:
		return nil, fmt.Errorf("expected logicalInput は許可された値型でなければなりません")
	}
}

func cloneExpectedLawSearch(
	value legalquery.LawSearchIntentV1,
) (legalquery.LogicalInput, error) {
	return legalquery.NewLawSearchIntentV1(legalquery.LawSearchIntentV1Values{
		Query: value.Query(),
		AsOf:  expectedDatePointer(value.AsOf()),
	})
}

func cloneExpectedLawContentSearch(
	value legalquery.LawContentSearchIntentV1,
) (legalquery.LogicalInput, error) {
	return legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     value.AllTerms(),
			AnyTerms:     value.AnyTerms(),
			ExcludeTerms: value.ExcludeTerms(),
			AsOf:         expectedDatePointer(value.AsOf()),
		},
	)
}

func cloneExpectedLawRead(
	value legalquery.LawReadIntentV1,
) (legalquery.LogicalInput, error) {
	lawID, _ := value.LawID()
	revisionID, _ := value.RevisionID()
	return legalquery.NewLawReadIntentV1(legalquery.LawReadIntentV1Values{
		LawID:      lawID,
		RevisionID: revisionID,
		AsOf:       expectedDatePointer(value.AsOf()),
		Ref:        expectedRefPointer(value.Ref()),
	})
}

func cloneExpectedLawArticleRead(
	value legalquery.LawArticleReadIntentV1,
) (legalquery.LogicalInput, error) {
	lawID, _ := value.LawID()
	return legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    lawID,
			Ref:      expectedRefPointer(value.Ref()),
			Location: value.Location(),
			AsOf:     expectedDatePointer(value.AsOf()),
		},
	)
}

func expectedDatePointer(value model.Date, exists bool) *model.Date {
	if !exists {
		return nil
	}
	cloned := value
	return &cloned
}

func expectedRefPointer(
	value model.SourceResourceRef,
	exists bool,
) *model.SourceResourceRef {
	if !exists {
		return nil
	}
	cloned := value
	return &cloned
}
