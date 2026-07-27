package legalquery

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// JudicialCasesCoverageNotice は、裁判例の収録範囲を表す固定の注意である。
const JudicialCasesCoverageNotice = "裁判所の裁判例検索には、すべての判決等が掲載されているわけではありません。掲載情報だけから先例性、拘束力、確定性または現在の有効性を判断できません。"

// LegalQueryAttemptOutcome は、一つの実行 step の結果を表す。
type LegalQueryAttemptOutcome string

const (
	// LegalQueryAttemptOutcomeCompleted は、型付きの非空結果を表す。
	LegalQueryAttemptOutcomeCompleted LegalQueryAttemptOutcome = "completed"
	// LegalQueryAttemptOutcomeEmpty は、collection の正常な空結果を表す。
	LegalQueryAttemptOutcomeEmpty LegalQueryAttemptOutcome = "empty"
	// LegalQueryAttemptOutcomeFailed は、公開可能な実行時エラーを表す。
	LegalQueryAttemptOutcomeFailed LegalQueryAttemptOutcome = "failed"
)

// LegalQueryAttemptResultKind は、成功 attempt の payload variant を表す。
type LegalQueryAttemptResultKind string

const (
	// LegalQueryResultKindLawSearch は、法令検索結果を表す。
	LegalQueryResultKindLawSearch LegalQueryAttemptResultKind = "law_search"
	// LegalQueryResultKindLawContentSearch は、法令本文検索結果を表す。
	LegalQueryResultKindLawContentSearch LegalQueryAttemptResultKind = "law_content_search"
	// LegalQueryResultKindLawDocument は、法令本文の読取り結果を表す。
	LegalQueryResultKindLawDocument LegalQueryAttemptResultKind = "law_document"
	// LegalQueryResultKindLawArticle は、条文の読取り結果を表す。
	LegalQueryResultKindLawArticle LegalQueryAttemptResultKind = "law_article"
	// LegalQueryResultKindLawUpdates は、法令更新一覧を表す。
	LegalQueryResultKindLawUpdates LegalQueryAttemptResultKind = "law_updates"
	// LegalQueryResultKindJudicialSearch は、裁判例検索結果を表す。
	LegalQueryResultKindJudicialSearch LegalQueryAttemptResultKind = "judicial_decision_search"
	// LegalQueryResultKindJudicialDecision は、裁判例詳細を表す。
	LegalQueryResultKindJudicialDecision LegalQueryAttemptResultKind = "judicial_decision"
)

// LegalQueryAttempt は、型を保った一つの実行結果を表す閉じた interface である。
type LegalQueryAttempt interface {
	InterpretationID() string
	StepID() string
	CapabilityID() string
	CapabilityMajorVersion() int
	Outcome() LegalQueryAttemptOutcome
	Validate() error
	isLegalQueryAttempt()
	cloneLegalQueryAttempt() LegalQueryAttempt
}

type legalQueryAttemptHeader struct {
	interpretationID       string
	stepID                 string
	capabilityID           string
	capabilityMajorVersion int
	inputKind              LogicalInputKind
	logicalInput           LogicalInput
}

func newLegalQueryAttemptHeader(
	interpretationID string,
	step LegalQueryCandidateStep,
) (legalQueryAttemptHeader, error) {
	if !isInterpretationID(interpretationID) {
		return legalQueryAttemptHeader{}, fmt.Errorf(
			"interpretationId は interpretation-1 または interpretation-2 でなければなりません",
		)
	}
	if err := step.Validate(); err != nil {
		return legalQueryAttemptHeader{}, fmt.Errorf("step が有効ではありません: %w", err)
	}
	return legalQueryAttemptHeader{
		interpretationID:       interpretationID,
		stepID:                 step.StepID(),
		capabilityID:           step.CapabilityID(),
		capabilityMajorVersion: step.CapabilityMajorVersion(),
		inputKind:              step.InputKind(),
		logicalInput:           step.LogicalInput(),
	}, nil
}

func (h legalQueryAttemptHeader) validate(
	expectedKind LogicalInputKind,
) error {
	if !isInterpretationID(h.interpretationID) {
		return fmt.Errorf("interpretationId が定義されていません")
	}
	if err := validateQueryPlanID("stepId", h.stepID); err != nil {
		return err
	}
	specification, exists := stepSpecificationFor(h.inputKind)
	if !exists ||
		h.inputKind != expectedKind ||
		h.capabilityID != specification.capabilityID ||
		h.capabilityMajorVersion != specification.majorVersion {
		return fmt.Errorf("attempt と step の capability variant が一致しません")
	}
	input, err := cloneLogicalInput(h.logicalInput)
	if err != nil {
		return fmt.Errorf("attempt の logical input が有効ではありません: %w", err)
	}
	if input.InputKind() != expectedKind {
		return fmt.Errorf("attempt の inputKind と logical input が一致しません")
	}
	return nil
}

type legalQueryAttemptHeaderJSON struct {
	InterpretationID       string `json:"interpretationId"`
	StepID                 string `json:"stepId"`
	CapabilityID           string `json:"capabilityId"`
	CapabilityMajorVersion int    `json:"capabilityMajorVersion"`
}

func (h legalQueryAttemptHeader) jsonValue() legalQueryAttemptHeaderJSON {
	return legalQueryAttemptHeaderJSON{
		InterpretationID:       h.interpretationID,
		StepID:                 h.stepID,
		CapabilityID:           h.capabilityID,
		CapabilityMajorVersion: h.capabilityMajorVersion,
	}
}

func marshalSuccessfulAttempt(
	header legalQueryAttemptHeader,
	outcome LegalQueryAttemptOutcome,
	resultKind LegalQueryAttemptResultKind,
	result any,
) ([]byte, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return nil, fmt.Errorf("attempt result を JSON に変換できません: %w", err)
	}
	headerJSON := header.jsonValue()
	return json.Marshal(struct {
		InterpretationID       string                      `json:"interpretationId"`
		StepID                 string                      `json:"stepId"`
		CapabilityID           string                      `json:"capabilityId"`
		CapabilityMajorVersion int                         `json:"capabilityMajorVersion"`
		Outcome                LegalQueryAttemptOutcome    `json:"outcome"`
		ResultKind             LegalQueryAttemptResultKind `json:"resultKind"`
		Result                 json.RawMessage             `json:"result"`
	}{
		InterpretationID:       headerJSON.InterpretationID,
		StepID:                 headerJSON.StepID,
		CapabilityID:           headerJSON.CapabilityID,
		CapabilityMajorVersion: headerJSON.CapabilityMajorVersion,
		Outcome:                outcome,
		ResultKind:             resultKind,
		Result:                 resultJSON,
	})
}

// LegalQueryFailedAttemptValues は、失敗 attempt の作成値を保持する。
type LegalQueryFailedAttemptValues struct {
	InterpretationID string
	Step             LegalQueryCandidateStep
	Error            model.ErrorResult
}

// LegalQueryFailedAttempt は、payload を持たない公開可能な step 失敗を表す。
type LegalQueryFailedAttempt struct {
	header legalQueryAttemptHeader
	err    model.ErrorResult
}

// NewLegalQueryFailedAttempt は、実行時エラーだけを持つ失敗 attempt を返す。
func NewLegalQueryFailedAttempt(
	values LegalQueryFailedAttemptValues,
) (LegalQueryFailedAttempt, error) {
	header, err := newLegalQueryAttemptHeader(
		values.InterpretationID,
		values.Step,
	)
	if err != nil {
		return LegalQueryFailedAttempt{}, err
	}
	attempt := LegalQueryFailedAttempt{
		header: header,
		err:    values.Error,
	}
	if err := attempt.Validate(); err != nil {
		return LegalQueryFailedAttempt{}, err
	}
	return attempt, nil
}

// InterpretationID は、対応する公開解釈 ID を返す。
func (a LegalQueryFailedAttempt) InterpretationID() string {
	return a.header.interpretationID
}

// StepID は、失敗した step ID を返す。
func (a LegalQueryFailedAttempt) StepID() string {
	return a.header.stepID
}

// CapabilityID は、失敗した能力 ID を返す。
func (a LegalQueryFailedAttempt) CapabilityID() string {
	return a.header.capabilityID
}

// CapabilityMajorVersion は、失敗した能力のメジャーバージョンを返す。
func (a LegalQueryFailedAttempt) CapabilityMajorVersion() int {
	return a.header.capabilityMajorVersion
}

// Outcome は、固定値 failed を返す。
func (a LegalQueryFailedAttempt) Outcome() LegalQueryAttemptOutcome {
	return LegalQueryAttemptOutcomeFailed
}

// Error は、安全な公開エラーを返す。
func (a LegalQueryFailedAttempt) Error() model.ErrorResult {
	return a.err
}

// Validate は、step と公開エラーの実行時境界を確認する。
func (a LegalQueryFailedAttempt) Validate() error {
	if err := validateAttemptHeaderForAnyKnownStep(a.header); err != nil {
		return err
	}
	if err := a.err.Validate(); err != nil {
		return fmt.Errorf("error が有効ではありません: %w", err)
	}
	if !isAllowedFailedAttemptError(a.err.Code()) {
		return fmt.Errorf("error.code は failed attempt で公開できません")
	}
	return nil
}

// MarshalJSON は、resultKind と result を持たない失敗 attempt を表す。
func (a LegalQueryFailedAttempt) MarshalJSON() ([]byte, error) {
	if err := a.Validate(); err != nil {
		return nil, err
	}
	header := a.header.jsonValue()
	return json.Marshal(struct {
		InterpretationID       string                   `json:"interpretationId"`
		StepID                 string                   `json:"stepId"`
		CapabilityID           string                   `json:"capabilityId"`
		CapabilityMajorVersion int                      `json:"capabilityMajorVersion"`
		Outcome                LegalQueryAttemptOutcome `json:"outcome"`
		Error                  model.ErrorResult        `json:"error"`
	}{
		InterpretationID:       header.InterpretationID,
		StepID:                 header.StepID,
		CapabilityID:           header.CapabilityID,
		CapabilityMajorVersion: header.CapabilityMajorVersion,
		Outcome:                a.Outcome(),
		Error:                  a.err,
	})
}

// UnmarshalJSON は、executor を介さない直接復元を拒否する。
func (*LegalQueryFailedAttempt) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryFailedAttempt は JSON から直接復元できません。NewLegalQueryFailedAttempt を使用してください",
	)
}

func (LegalQueryFailedAttempt) isLegalQueryAttempt() {}

func (a LegalQueryFailedAttempt) cloneLegalQueryAttempt() LegalQueryAttempt {
	return a
}

func validateAttemptHeaderForAnyKnownStep(
	header legalQueryAttemptHeader,
) error {
	for _, inputKind := range allLogicalInputKinds() {
		if header.validate(inputKind) == nil {
			return nil
		}
	}
	return fmt.Errorf("attempt の step が採用済み capability と一致しません")
}

func allLogicalInputKinds() []LogicalInputKind {
	return []LogicalInputKind{
		InputKindLawSearch,
		InputKindLawContentSearch,
		InputKindLawRead,
		InputKindLawArticleRead,
		InputKindLawUpdates,
		InputKindJudicialDecisionSearch,
		InputKindJudicialDecisionRead,
	}
}

func isAllowedFailedAttemptError(code model.ErrorCode) bool {
	switch code {
	case model.ErrorCodeNotFound,
		model.ErrorCodeAmbiguousLocation,
		model.ErrorCodeUnsupportedQuery,
		model.ErrorCodeSourceAuthFailed,
		model.ErrorCodeRateLimited,
		model.ErrorCodeSourceTimeout,
		model.ErrorCodeSourceUnavailable,
		model.ErrorCodeSourceBusy,
		model.ErrorCodeSourceContractChanged,
		model.ErrorCodeInvalidSourceResponse,
		model.ErrorCodeSourceResponseTooLarge,
		model.ErrorCodeSourceProcessingLimit,
		model.ErrorCodeUnsafeSourceContent,
		model.ErrorCodeInternalError:
		return true
	default:
		return false
	}
}
