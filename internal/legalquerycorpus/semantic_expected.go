package legalquerycorpus

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	maximumExpectedMeanings          = 16
	maximumSelectedExpectedMeanings  = 2
	maximumSelectedExpectedStepCount = 4
)

// SemanticExpectedKind は、semantic fixture の期待値 variant を表す。
type SemanticExpectedKind string

const (
	// SemanticExpectedKindPlan は、意味判定の期待投影を表す。
	SemanticExpectedKindPlan SemanticExpectedKind = "plan"
	// SemanticExpectedKindRequestError は、公開入力エラーの期待投影を表す。
	SemanticExpectedKindRequestError SemanticExpectedKind = "request_error"
)

// SemanticExpected は、semantic fixture が許可する二つの期待値 variant である。
type SemanticExpected interface {
	Kind() SemanticExpectedKind
	Validate() error
	semanticExpected()
}

// ExpectedPlanValues は、意味判定の期待投影を構成する値を保持する。
type ExpectedPlanValues struct {
	Decision           legalquery.PlanDecision
	ReasonCodes        []legalquery.ReasonCode
	Meanings           []ExpectedMeaning
	SelectedMeaningIDs []string
}

// ExpectedPlan は、内部 score や候補 ID を持たない意味判定の期待投影である。
type ExpectedPlan struct {
	decision           legalquery.PlanDecision
	reasonCodes        []legalquery.ReasonCode
	meanings           []ExpectedMeaning
	selectedMeaningIDs []string
	initialized        bool
}

// NewExpectedPlan は、decision、正解意味および選択参照を検証して返す。
func NewExpectedPlan(values ExpectedPlanValues) (ExpectedPlan, error) {
	meanings, err := cloneExpectedMeanings(values.Meanings)
	if err != nil {
		return ExpectedPlan{}, err
	}
	plan := ExpectedPlan{
		decision:           values.Decision,
		reasonCodes:        append([]legalquery.ReasonCode{}, values.ReasonCodes...),
		meanings:           meanings,
		selectedMeaningIDs: cloneStrings(values.SelectedMeaningIDs),
		initialized:        true,
	}
	if err := plan.Validate(); err != nil {
		return ExpectedPlan{}, err
	}
	return plan, nil
}

// Kind は、plan variant を返す。
func (p ExpectedPlan) Kind() SemanticExpectedKind {
	return SemanticExpectedKindPlan
}

// Decision は、期待する選択または非実行方法を返す。
func (p ExpectedPlan) Decision() legalquery.PlanDecision {
	return p.decision
}

// ReasonCodes は、順序を含む決定理由の複製を返す。
func (p ExpectedPlan) ReasonCodes() []legalquery.ReasonCode {
	return append([]legalquery.ReasonCode{}, p.reasonCodes...)
}

// Meanings は、主正解を先頭にした正しい意味の複製を返す。
func (p ExpectedPlan) Meanings() []ExpectedMeaning {
	meanings, err := cloneExpectedMeanings(p.meanings)
	if err != nil {
		panic(fmt.Sprintf("検証済み ExpectedPlan の meanings 複製に失敗しました: %v", err))
	}
	return meanings
}

// SelectedMeaningIDs は、selection 順の意味参照の複製を返す。
func (p ExpectedPlan) SelectedMeaningIDs() []string {
	return cloneStrings(p.selectedMeaningIDs)
}

// Validate は、decision、reason、意味および selection の整合を確認する。
func (p ExpectedPlan) Validate() error {
	if !p.initialized {
		return fmt.Errorf("ExpectedPlan は NewExpectedPlan で作成しなければなりません")
	}
	if err := validateExpectedReasonCodes(p.decision, p.reasonCodes); err != nil {
		return err
	}
	references, err := validateExpectedMeanings(p.meanings)
	if err != nil {
		return err
	}
	if err := validateSelectedMeaningIDs(
		p.decision,
		p.selectedMeaningIDs,
		references,
	); err != nil {
		return err
	}
	if len(p.reasonCodes) == 1 &&
		p.reasonCodes[0] ==
			legalquery.ReasonCodeStandaloneStructuredQuery &&
		len(p.meanings) != 0 {
		return fmt.Errorf("standalone structured query は meanings を持てません")
	}
	return validateExpectedPlanShape(
		p.decision,
		len(p.meanings),
		len(p.selectedMeaningIDs),
	)
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedPlan) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExpectedPlan は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}

func (ExpectedPlan) semanticExpected() {}

// RequestErrorField は、入力エラーを期待する公開 request 項目を表す。
type RequestErrorField string

const (
	// RequestErrorFieldQuery は、query の入力エラーを表す。
	RequestErrorFieldQuery RequestErrorField = "query"
	// RequestErrorFieldRef は、ref の入力エラーを表す。
	RequestErrorFieldRef RequestErrorField = "ref"
	// RequestErrorFieldLimitPerAttempt は、limitPerAttempt の入力エラーを表す。
	RequestErrorFieldLimitPerAttempt RequestErrorField = "limitPerAttempt"
)

// ExpectedRequestErrorValues は、公開入力エラーの期待値を保持する。
type ExpectedRequestErrorValues struct {
	ErrorCode model.ErrorCode
	Field     RequestErrorField
}

// ExpectedRequestError は、公開文言を golden にしない入力エラーの期待投影である。
type ExpectedRequestError struct {
	errorCode   model.ErrorCode
	field       RequestErrorField
	initialized bool
}

// NewExpectedRequestError は、許可した分類と field だけを持つ期待値を返す。
func NewExpectedRequestError(
	values ExpectedRequestErrorValues,
) (ExpectedRequestError, error) {
	expected := ExpectedRequestError{
		errorCode:   values.ErrorCode,
		field:       values.Field,
		initialized: true,
	}
	if err := expected.Validate(); err != nil {
		return ExpectedRequestError{}, err
	}
	return expected, nil
}

// Kind は、request_error variant を返す。
func (e ExpectedRequestError) Kind() SemanticExpectedKind {
	return SemanticExpectedKindRequestError
}

// ErrorCode は、invalid_argument を返す。
func (e ExpectedRequestError) ErrorCode() model.ErrorCode {
	return e.errorCode
}

// Field は、修正が必要な公開 request 項目を返す。
func (e ExpectedRequestError) Field() RequestErrorField {
	return e.field
}

// Validate は、入力エラーの固定分類と field を確認する。
func (e ExpectedRequestError) Validate() error {
	if !e.initialized {
		return fmt.Errorf(
			"ExpectedRequestError は NewExpectedRequestError で作成しなければなりません",
		)
	}
	if e.errorCode != model.ErrorCodeInvalidArgument {
		return fmt.Errorf("expected request error の errorCode は invalid_argument でなければなりません")
	}
	switch e.field {
	case RequestErrorFieldQuery,
		RequestErrorFieldRef,
		RequestErrorFieldLimitPerAttempt:
		return nil
	default:
		return fmt.Errorf("expected request error の field が定義されていません")
	}
}

// UnmarshalJSON は、version 別 DTO を介さない直接復元を拒否する。
func (*ExpectedRequestError) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"ExpectedRequestError は JSON から直接復元できません。version 別 DTO を使用してください",
	)
}

func (ExpectedRequestError) semanticExpected() {}

func cloneSemanticExpected(value SemanticExpected) (SemanticExpected, error) {
	switch typed := value.(type) {
	case ExpectedPlan:
		return NewExpectedPlan(ExpectedPlanValues{
			Decision:           typed.Decision(),
			ReasonCodes:        typed.ReasonCodes(),
			Meanings:           typed.Meanings(),
			SelectedMeaningIDs: typed.SelectedMeaningIDs(),
		})
	case ExpectedRequestError:
		return NewExpectedRequestError(ExpectedRequestErrorValues{
			ErrorCode: typed.ErrorCode(),
			Field:     typed.Field(),
		})
	default:
		return nil, fmt.Errorf(
			"semantic expected は検証済みの値 variant でなければなりません",
		)
	}
}

func cloneExpectedMeanings(values []ExpectedMeaning) ([]ExpectedMeaning, error) {
	cloned := make([]ExpectedMeaning, 0, len(values))
	for _, meaning := range values {
		if err := meaning.Validate(); err != nil {
			return nil, fmt.Errorf("expected plan の meaning が有効ではありません: %w", err)
		}
		cloned = append(cloned, meaning.clone())
	}
	return cloned, nil
}

type expectedMeaningReference struct {
	index     int
	stepCount int
	packCount int
}

func validateExpectedMeanings(
	values []ExpectedMeaning,
) (map[string]expectedMeaningReference, error) {
	if len(values) > maximumExpectedMeanings {
		return nil, fmt.Errorf("expected plan の meanings は十六件以下でなければなりません")
	}
	references := make(map[string]expectedMeaningReference, len(values))
	for index, meaning := range values {
		if err := meaning.Validate(); err != nil {
			return nil, fmt.Errorf("expected plan の meaning が有効ではありません: %w", err)
		}
		if _, exists := references[meaning.MeaningID()]; exists {
			return nil, fmt.Errorf("expected plan の meaningId を重複させることはできません")
		}
		references[meaning.MeaningID()] = expectedMeaningReference{
			index:     index,
			stepCount: len(meaning.Steps()),
			packCount: len(meaning.RequiredPacks()),
		}
	}
	return references, nil
}

func validateSelectedMeaningIDs(
	decision legalquery.PlanDecision,
	values []string,
	references map[string]expectedMeaningReference,
) error {
	if len(values) > maximumSelectedExpectedMeanings {
		return fmt.Errorf("selectedMeaningIds は二件以下でなければなりません")
	}
	previousIndex := -1
	stepCount := 0
	for _, value := range values {
		reference, exists := references[value]
		if !exists {
			return fmt.Errorf("selectedMeaningIds は meanings の要素だけを参照できます")
		}
		if reference.index <= previousIndex {
			return fmt.Errorf("selectedMeaningIds は meanings の順位を重複なく保持してください")
		}
		if decision == legalquery.PlanDecisionCapabilityUnavailable &&
			reference.packCount == 0 {
			return fmt.Errorf("capability_unavailable の選択意味には requiredPacks が必要です")
		}
		previousIndex = reference.index
		stepCount += reference.stepCount
	}
	if stepCount > maximumSelectedExpectedStepCount {
		return fmt.Errorf("選択した期待意味の step 合計は四件以下でなければなりません")
	}
	return nil
}

func validateExpectedPlanShape(
	decision legalquery.PlanDecision,
	meaningCount int,
	selectedCount int,
) error {
	switch decision {
	case legalquery.PlanDecisionSingle:
		return requireExpectedCounts(meaningCount, selectedCount, 1, 16, 1, 1)
	case legalquery.PlanDecisionHedged:
		return requireExpectedCounts(meaningCount, selectedCount, 2, 16, 2, 2)
	case legalquery.PlanDecisionNeedsClarification:
		return requireExpectedCounts(meaningCount, selectedCount, 1, 16, 0, 2)
	case legalquery.PlanDecisionCapabilityUnavailable:
		return requireExpectedCounts(meaningCount, selectedCount, 1, 16, 1, 2)
	case legalquery.PlanDecisionUnsupported:
		return requireExpectedCounts(meaningCount, selectedCount, 0, 16, 0, 0)
	default:
		return fmt.Errorf("expected plan の decision が定義されていません")
	}
}

func requireExpectedCounts(
	meaningCount int,
	selectedCount int,
	minimumMeanings int,
	maximumMeanings int,
	minimumSelected int,
	maximumSelected int,
) error {
	if meaningCount < minimumMeanings ||
		meaningCount > maximumMeanings ||
		selectedCount < minimumSelected ||
		selectedCount > maximumSelected {
		return fmt.Errorf("expected plan の meaning または selection 件数が decision と一致しません")
	}
	return nil
}

func validateExpectedReasonCodes(
	decision legalquery.PlanDecision,
	values []legalquery.ReasonCode,
) error {
	previous := -1
	for _, value := range values {
		rank, exists := expectedReasonRank(value)
		if !exists {
			return fmt.Errorf("expected plan の reasonCodes に未定義の値があります")
		}
		if rank <= previous {
			return fmt.Errorf("expected plan の reasonCodes は規定順で重複なく保持してください")
		}
		previous = rank
	}
	switch decision {
	case legalquery.PlanDecisionSingle:
		return requireExpectedExactReason(values, legalquery.ReasonCodeSingleClearCandidate)
	case legalquery.PlanDecisionHedged:
		return requireExpectedExactReason(values, legalquery.ReasonCodeHedgedCloseCandidates)
	case legalquery.PlanDecisionNeedsClarification:
		return requireExpectedReasons(
			values,
			1,
			2,
			legalquery.ReasonCodeBelowExecutionThreshold,
			legalquery.ReasonCodeAmbiguousCandidates,
		)
	case legalquery.PlanDecisionCapabilityUnavailable:
		return requireExpectedExactReason(values, legalquery.ReasonCodeRequiredPackDisabled)
	case legalquery.PlanDecisionUnsupported:
		if len(values) == 1 &&
			values[0] ==
				legalquery.ReasonCodeStandaloneStructuredQuery {
			return nil
		}
		return requireExpectedReasons(
			values,
			1,
			3,
			legalquery.ReasonCodeNonJapaneseQuery,
			legalquery.ReasonCodeMixedUnsupportedIntent,
			legalquery.ReasonCodeUnsupportedTaskOrResource,
		)
	default:
		return fmt.Errorf("expected plan の decision が定義されていません")
	}
}

func requireExpectedExactReason(
	values []legalquery.ReasonCode,
	expected legalquery.ReasonCode,
) error {
	if len(values) != 1 || values[0] != expected {
		return fmt.Errorf("expected plan の reasonCodes が decision と一致しません")
	}
	return nil
}

func requireExpectedReasons(
	values []legalquery.ReasonCode,
	minimum int,
	maximum int,
	allowed ...legalquery.ReasonCode,
) error {
	if len(values) < minimum || len(values) > maximum {
		return fmt.Errorf("expected plan の reasonCodes 件数が decision と一致しません")
	}
	for _, value := range values {
		if !containsExpectedReason(allowed, value) {
			return fmt.Errorf("expected plan の reasonCodes が decision と一致しません")
		}
	}
	return nil
}

func containsExpectedReason(
	values []legalquery.ReasonCode,
	target legalquery.ReasonCode,
) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func expectedReasonRank(value legalquery.ReasonCode) (int, bool) {
	switch value {
	case legalquery.ReasonCodeSingleClearCandidate:
		return 0, true
	case legalquery.ReasonCodeHedgedCloseCandidates:
		return 1, true
	case legalquery.ReasonCodeBelowExecutionThreshold:
		return 2, true
	case legalquery.ReasonCodeAmbiguousCandidates:
		return 3, true
	case legalquery.ReasonCodeRequiredPackDisabled:
		return 4, true
	case legalquery.ReasonCodeNonJapaneseQuery:
		return 5, true
	case legalquery.ReasonCodeStandaloneStructuredQuery:
		return 6, true
	case legalquery.ReasonCodeMixedUnsupportedIntent:
		return 7, true
	case legalquery.ReasonCodeUnsupportedTaskOrResource:
		return 8, true
	default:
		return 0, false
	}
}
