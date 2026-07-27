package legalquery

import (
	"encoding/json"
	"fmt"
)

// LegalQueryCompletedResult は、非空成功を一件以上持つ実行結果である。
type LegalQueryCompletedResult struct {
	legalQueryResultCore
}

// NewLegalQueryCompletedResult は、失敗のない非空成功から completed を作る。
func NewLegalQueryCompletedResult(
	plan LegalQueryPlan,
	attempts []LegalQueryAttempt,
) (LegalQueryCompletedResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusCompleted,
		attempts,
	)
	if err != nil {
		return LegalQueryCompletedResult{}, err
	}
	return LegalQueryCompletedResult{legalQueryResultCore: core}, nil
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryCompletedResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryCompletedResult")
}

// LegalQueryEmptyResult は、すべての collection が正常な空である実行結果である。
type LegalQueryEmptyResult struct {
	legalQueryResultCore
}

// NewLegalQueryEmptyResult は、空の collection 成功だけから empty を作る。
func NewLegalQueryEmptyResult(
	plan LegalQueryPlan,
	attempts []LegalQueryAttempt,
) (LegalQueryEmptyResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusEmpty,
		attempts,
	)
	if err != nil {
		return LegalQueryEmptyResult{}, err
	}
	return LegalQueryEmptyResult{legalQueryResultCore: core}, nil
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryEmptyResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryEmptyResult")
}

// LegalQueryPartialResult は、成功と失敗が混在する実行結果である。
type LegalQueryPartialResult struct {
	legalQueryResultCore
}

// NewLegalQueryPartialResult は、一件以上の成功と失敗から partial を作る。
func NewLegalQueryPartialResult(
	plan LegalQueryPlan,
	attempts []LegalQueryAttempt,
) (LegalQueryPartialResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusPartial,
		attempts,
	)
	if err != nil {
		return LegalQueryPartialResult{}, err
	}
	return LegalQueryPartialResult{legalQueryResultCore: core}, nil
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryPartialResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryPartialResult")
}

// LegalQueryNeedsClarificationResult は、外部実行前の固定質問を返す。
type LegalQueryNeedsClarificationResult struct {
	legalQueryResultCore
	clarification LegalQueryClarification
}

// NewLegalQueryNeedsClarificationResult は、plan と固定質問から非実行結果を作る。
func NewLegalQueryNeedsClarificationResult(
	plan LegalQueryPlan,
	questions []LegalQueryQuestion,
) (LegalQueryNeedsClarificationResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusNeedsClarification,
		[]LegalQueryAttempt{},
	)
	if err != nil {
		return LegalQueryNeedsClarificationResult{}, err
	}
	clarification, err := NewLegalQueryClarification(plan, questions)
	if err != nil {
		return LegalQueryNeedsClarificationResult{}, err
	}
	result := LegalQueryNeedsClarificationResult{
		legalQueryResultCore: core,
		clarification:        clarification,
	}
	if err := result.Validate(); err != nil {
		return LegalQueryNeedsClarificationResult{}, err
	}
	return result, nil
}

// Clarification は、理由と固定質問の複製を返す。
func (r LegalQueryNeedsClarificationResult) Clarification() LegalQueryClarification {
	return LegalQueryClarification{
		reasonCodes: append([]ReasonCode{}, r.clarification.reasonCodes...),
		questions:   append([]LegalQueryQuestion{}, r.clarification.questions...),
	}
}

// Validate は、共通結果と clarification の理由を照合する。
func (r LegalQueryNeedsClarificationResult) Validate() error {
	if err := r.legalQueryResultCore.Validate(); err != nil {
		return err
	}
	if err := r.clarification.Validate(); err != nil {
		return fmt.Errorf("clarification が有効ではありません: %w", err)
	}
	if !equalReasonCodes(
		r.reasonCodes,
		r.clarification.reasonCodes,
	) {
		return fmt.Errorf("clarification.reasonCodes が plan の理由と一致しません")
	}
	return nil
}

// MarshalJSON は、共通項目に clarification だけを追加する。
func (r LegalQueryNeedsClarificationResult) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	common := r.jsonValue()
	return json.Marshal(struct {
		Status          LegalQueryResultStatus     `json:"status"`
		Decision        LegalQueryResultDecision   `json:"decision"`
		Language        string                     `json:"language"`
		Interpretations []LegalQueryInterpretation `json:"interpretations"`
		Attempts        []LegalQueryAttempt        `json:"attempts"`
		Notices         []string                   `json:"notices"`
		Clarification   LegalQueryClarification    `json:"clarification"`
	}{
		Status:          common.Status,
		Decision:        common.Decision,
		Language:        common.Language,
		Interpretations: common.Interpretations,
		Attempts:        common.Attempts,
		Notices:         common.Notices,
		Clarification:   r.Clarification(),
	})
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryNeedsClarificationResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryNeedsClarificationResult")
}

// LegalQueryCapabilityUnavailableResult は、必要な拡張 pack が無効な非実行結果である。
type LegalQueryCapabilityUnavailableResult struct {
	legalQueryResultCore
}

// NewLegalQueryCapabilityUnavailableResult は、pack_disabled plan を公開結果へ変換する。
func NewLegalQueryCapabilityUnavailableResult(
	plan LegalQueryPlan,
) (LegalQueryCapabilityUnavailableResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusCapabilityUnavailable,
		[]LegalQueryAttempt{},
	)
	if err != nil {
		return LegalQueryCapabilityUnavailableResult{}, err
	}
	return LegalQueryCapabilityUnavailableResult{legalQueryResultCore: core}, nil
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryCapabilityUnavailableResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryCapabilityUnavailableResult")
}

// LegalQueryUnsupportedResult は、統合照会の採用範囲外を表す非実行結果である。
type LegalQueryUnsupportedResult struct {
	legalQueryResultCore
}

// NewLegalQueryUnsupportedResult は、unsupported plan を固定 notice へ変換する。
func NewLegalQueryUnsupportedResult(
	plan LegalQueryPlan,
) (LegalQueryUnsupportedResult, error) {
	core, err := newLegalQueryResultCore(
		plan,
		LegalQueryResultStatusUnsupported,
		[]LegalQueryAttempt{},
	)
	if err != nil {
		return LegalQueryUnsupportedResult{}, err
	}
	return LegalQueryUnsupportedResult{legalQueryResultCore: core}, nil
}

// UnmarshalJSON は、result assembler を介さない直接復元を拒否する。
func (*LegalQueryUnsupportedResult) UnmarshalJSON(_ []byte) error {
	return directResultRestoreError("LegalQueryUnsupportedResult")
}

func directResultRestoreError(typeName string) error {
	return fmt.Errorf(
		"%s は JSON から直接復元できません。対応する constructor を使用してください",
		typeName,
	)
}

func equalReasonCodes(left []ReasonCode, right []ReasonCode) bool {
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
