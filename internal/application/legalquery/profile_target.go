package legalquery

import "fmt"

// QueryProfileTargetValues は、profile の対象能力を構築する値である。
type QueryProfileTargetValues struct {
	Task      Task
	Resource  Resource
	InputKind LogicalInputKind
}

// QueryProfileTarget は、profile が生成できる task/resource/inputKind を表す。
type QueryProfileTarget struct {
	task      Task
	resource  Resource
	inputKind LogicalInputKind
}

// NewQueryProfileTarget は、七つの許可済み対応に限定した target を返す。
func NewQueryProfileTarget(
	values QueryProfileTargetValues,
) (QueryProfileTarget, error) {
	target := QueryProfileTarget{
		task:      values.Task,
		resource:  values.Resource,
		inputKind: values.InputKind,
	}
	if err := target.Validate(); err != nil {
		return QueryProfileTarget{}, err
	}
	return target, nil
}

// Task は、対象 task を返す。
func (t QueryProfileTarget) Task() Task {
	return t.task
}

// Resource は、対象 resource を返す。
func (t QueryProfileTarget) Resource() Resource {
	return t.resource
}

// InputKind は、対象 logical input variant を返す。
func (t QueryProfileTarget) InputKind() LogicalInputKind {
	return t.inputKind
}

// Validate は、task/resource/inputKind の対応を確認する。
func (t QueryProfileTarget) Validate() error {
	specification, exists := stepSpecificationFor(t.inputKind)
	if !exists {
		return fmt.Errorf("profile target の inputKind が定義されていません")
	}
	if t.task != specification.task || t.resource != specification.resource {
		return fmt.Errorf("profile target の task/resource/inputKind 対応が許可されていません")
	}
	return nil
}
