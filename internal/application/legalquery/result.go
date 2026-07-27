package legalquery

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// LegalQueryResultStatus は、統合照会の公開結果状態を表す。
type LegalQueryResultStatus string

const (
	// LegalQueryResultStatusCompleted は、一件以上の非空成功を表す。
	LegalQueryResultStatusCompleted LegalQueryResultStatus = "completed"
	// LegalQueryResultStatusEmpty は、すべての実行結果が正常な空であることを表す。
	LegalQueryResultStatusEmpty LegalQueryResultStatus = "empty"
	// LegalQueryResultStatusPartial は、成功と失敗が混在したことを表す。
	LegalQueryResultStatusPartial LegalQueryResultStatus = "partial"
	// LegalQueryResultStatusNeedsClarification は、外部実行前の明確化要求を表す。
	LegalQueryResultStatusNeedsClarification LegalQueryResultStatus = "needs_clarification"
	// LegalQueryResultStatusCapabilityUnavailable は、必要な pack が無効であることを表す。
	LegalQueryResultStatusCapabilityUnavailable LegalQueryResultStatus = "capability_unavailable"
	// LegalQueryResultStatusUnsupported は、採用範囲外の要求を表す。
	LegalQueryResultStatusUnsupported LegalQueryResultStatus = "unsupported"
)

// LegalQueryResultDecision は、公開する実行方法または非実行を表す。
type LegalQueryResultDecision string

const (
	// LegalQueryResultDecisionSingle は、一つの解釈を実行したことを表す。
	LegalQueryResultDecisionSingle LegalQueryResultDecision = "single"
	// LegalQueryResultDecisionHedged は、二つの解釈を実行したことを表す。
	LegalQueryResultDecisionHedged LegalQueryResultDecision = "hedged"
	// LegalQueryResultDecisionNoExecution は、外部情報源を呼び出していないことを表す。
	LegalQueryResultDecisionNoExecution LegalQueryResultDecision = "no_execution"
)

const (
	legalQueryFirstPageNotice        = "最初のページだけを返しています。続きが必要な場合は対応する専門ツールを使用してください。"
	legalQuerySeparateAttemptsNotice = "複数の解釈または step の件数、順位および継続位置は統合していません。"
	legalQueryPartialFailureNotice   = "一部の step が失敗しました。attempts の error を確認してください。"
	legalQueryPackDisabledNotice     = "必要な拡張パックが無効です。interpretations の requiredPacks を確認してください。"
	legalQueryNonJapaneseNotice      = "日本語の法情報取得要求を入力してください。"
	legalQueryMixedUnsupportedNotice = "法情報の取得要求と法的助言、翻訳または対象外の要求を分けて入力してください。"
	legalQueryUnsupportedScopeNotice = "指定した task または resource は統合照会の採用範囲外です。対応する専門ツールを確認してください。"
)

// LegalQueryResult は、六つの公開成功結果だけを許す閉じた interface である。
type LegalQueryResult interface {
	Status() LegalQueryResultStatus
	Decision() LegalQueryResultDecision
	Language() string
	Interpretations() []LegalQueryInterpretation
	Attempts() []LegalQueryAttempt
	Notices() []string
	Validate() error
	isLegalQueryResult()
}

type legalQueryResultCore struct {
	status          LegalQueryResultStatus
	decision        LegalQueryResultDecision
	language        string
	interpretations []LegalQueryInterpretation
	attempts        []LegalQueryAttempt
	notices         []string
	planDecision    PlanDecision
	reasonCodes     []ReasonCode
}

func newLegalQueryResultCore(
	plan LegalQueryPlan,
	status LegalQueryResultStatus,
	attempts []LegalQueryAttempt,
) (legalQueryResultCore, error) {
	if err := plan.Validate(); err != nil {
		return legalQueryResultCore{}, fmt.Errorf("plan が有効ではありません: %w", err)
	}
	decision, err := resultDecisionFor(plan.Decision(), status)
	if err != nil {
		return legalQueryResultCore{}, err
	}
	interpretations, err := NewLegalQueryInterpretations(plan)
	if err != nil {
		return legalQueryResultCore{}, err
	}
	clonedAttempts, err := cloneLegalQueryAttempts(attempts)
	if err != nil {
		return legalQueryResultCore{}, err
	}
	core := legalQueryResultCore{
		status:          status,
		decision:        decision,
		language:        legalQueryLanguage,
		interpretations: cloneLegalQueryInterpretations(interpretations),
		attempts:        clonedAttempts,
		planDecision:    plan.Decision(),
		reasonCodes:     plan.ReasonCodes(),
	}
	core.notices = deriveLegalQueryNotices(
		core.status,
		core.attempts,
		core.reasonCodes,
	)
	if err := core.Validate(); err != nil {
		return legalQueryResultCore{}, err
	}
	return core, nil
}

// Status は、result concrete object の固定状態を返す。
func (c legalQueryResultCore) Status() LegalQueryResultStatus {
	return c.status
}

// Decision は、実行方法または固定値 no_execution を返す。
func (c legalQueryResultCore) Decision() LegalQueryResultDecision {
	return c.decision
}

// Language は、固定値 ja を返す。
func (c legalQueryResultCore) Language() string {
	return c.language
}

// Interpretations は、内部候補値を含まない公開解釈の複製を返す。
func (c legalQueryResultCore) Interpretations() []LegalQueryInterpretation {
	return cloneLegalQueryInterpretations(c.interpretations)
}

// Attempts は、型を保った実行結果の複製を返す。
func (c legalQueryResultCore) Attempts() []LegalQueryAttempt {
	attempts, err := cloneLegalQueryAttempts(c.attempts)
	if err != nil {
		panic(fmt.Sprintf("検証済み attempt の複製に失敗しました: %v", err))
	}
	return attempts
}

// Notices は、result assembler が導出した固定注意の複製を返す。
func (c legalQueryResultCore) Notices() []string {
	return append([]string{}, c.notices...)
}

// Validate は、状態、解釈、attempt、固定注意および plan 由来値を照合する。
func (c legalQueryResultCore) Validate() error {
	if c.language != legalQueryLanguage {
		return fmt.Errorf("language は ja でなければなりません")
	}
	expectedDecision, err := resultDecisionFor(c.planDecision, c.status)
	if err != nil {
		return err
	}
	if c.decision != expectedDecision {
		return fmt.Errorf("decision が plan と status から導出した値と一致しません")
	}
	if err := validateReasonCodes(c.planDecision, c.reasonCodes); err != nil {
		return fmt.Errorf("reasonCodes が有効ではありません: %w", err)
	}
	if err := validateResultInterpretations(
		c.status,
		c.decision,
		c.interpretations,
	); err != nil {
		return err
	}
	if err := validateResultAttempts(
		c.status,
		c.interpretations,
		c.attempts,
	); err != nil {
		return err
	}
	expectedNotices := deriveLegalQueryNotices(
		c.status,
		c.attempts,
		c.reasonCodes,
	)
	if !equalStrings(c.notices, expectedNotices) {
		return fmt.Errorf("notices が状態、attempt および reasonCodes から導出した値と一致しません")
	}
	return nil
}

// MarshalJSON は、内部 plan、候補 ID、score および継続位置を除外する。
func (c legalQueryResultCore) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(c.jsonValue())
}

func (c legalQueryResultCore) jsonValue() legalQueryResultJSON {
	return legalQueryResultJSON{
		Status:          c.status,
		Decision:        c.decision,
		Language:        c.language,
		Interpretations: c.Interpretations(),
		Attempts:        c.Attempts(),
		Notices:         c.Notices(),
	}
}

func (legalQueryResultCore) isLegalQueryResult() {}

type legalQueryResultJSON struct {
	Status          LegalQueryResultStatus     `json:"status"`
	Decision        LegalQueryResultDecision   `json:"decision"`
	Language        string                     `json:"language"`
	Interpretations []LegalQueryInterpretation `json:"interpretations"`
	Attempts        []LegalQueryAttempt        `json:"attempts"`
	Notices         []string                   `json:"notices"`
}

func cloneLegalQueryInterpretations(
	values []LegalQueryInterpretation,
) []LegalQueryInterpretation {
	cloned := make([]LegalQueryInterpretation, 0, len(values))
	for _, value := range values {
		cloned = append(cloned, LegalQueryInterpretation{
			interpretationID: value.interpretationID,
			confidence:       value.confidence,
			evidenceCodes:    append([]EvidenceCode{}, value.evidenceCodes...),
			conceptSources:   append([]LegalConceptSource{}, value.conceptSources...),
			availability:     value.availability,
			requiredPacks:    append([]string{}, value.requiredPacks...),
			steps:            append([]LegalQueryStepSummary{}, value.steps...),
		})
	}
	return cloned
}

func cloneLegalQueryAttempts(
	values []LegalQueryAttempt,
) ([]LegalQueryAttempt, error) {
	cloned := make([]LegalQueryAttempt, 0, len(values))
	for index, value := range values {
		if isNilLegalQueryAttempt(value) {
			return nil, fmt.Errorf("attempts[%d] は nil にできません", index)
		}
		if err := value.Validate(); err != nil {
			return nil, fmt.Errorf("attempts[%d] が有効ではありません: %w", index, err)
		}
		cloned = append(cloned, value.cloneLegalQueryAttempt())
	}
	return cloned, nil
}

func isNilLegalQueryAttempt(value LegalQueryAttempt) bool {
	if value == nil {
		return true
	}
	reflected := reflect.ValueOf(value)
	return reflected.Kind() == reflect.Pointer && reflected.IsNil()
}

func equalStrings(left []string, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}
