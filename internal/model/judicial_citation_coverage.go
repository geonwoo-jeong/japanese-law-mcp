package model

import (
	"encoding/json"
	"fmt"
	"slices"
	"unicode/utf8"
)

// JudicialCitationCoverageValues は、方向別 coverage の作成値を保持する。
type JudicialCitationCoverageValues struct {
	RequestedDirection JudicialCitationRequestedDirection
	HopDepth           int
	Outgoing           JudicialCitationDirectionCoverage
	Incoming           JudicialCitationDirectionCoverage
}

// JudicialCitationCoverage は、要求方向と処理範囲を表す。
type JudicialCitationCoverage struct {
	requestedDirection JudicialCitationRequestedDirection
	hopDepth           int
	outgoing           JudicialCitationDirectionCoverage
	incoming           JudicialCitationDirectionCoverage
}

func NewJudicialCitationCoverage(values JudicialCitationCoverageValues) (JudicialCitationCoverage, error) {
	hopDepth := values.HopDepth
	if hopDepth == 0 {
		hopDepth = 1
	}
	coverage := JudicialCitationCoverage{
		requestedDirection: values.RequestedDirection,
		hopDepth:           hopDepth,
		outgoing:           values.Outgoing,
		incoming:           values.Incoming,
	}
	if err := coverage.Validate(); err != nil {
		return JudicialCitationCoverage{}, err
	}
	return coverage, nil
}

func (c JudicialCitationCoverage) RequestedDirection() JudicialCitationRequestedDirection {
	return c.requestedDirection
}
func (c JudicialCitationCoverage) HopDepth() int { return c.hopDepth }
func (c JudicialCitationCoverage) Outgoing() JudicialCitationDirectionCoverage {
	return c.outgoing
}
func (c JudicialCitationCoverage) Incoming() JudicialCitationDirectionCoverage {
	return c.incoming
}

func (c JudicialCitationCoverage) Validate() error {
	if !c.requestedDirection.valid() {
		return fmt.Errorf("requestedDirection が有効ではありません")
	}
	if c.hopDepth != 1 {
		return fmt.Errorf("hopDepth は 1 でなければなりません")
	}
	if err := c.outgoing.Validate(); err != nil {
		return fmt.Errorf("outgoing が有効ではありません: %w", err)
	}
	if err := c.incoming.Validate(); err != nil {
		return fmt.Errorf("incoming が有効ではありません: %w", err)
	}
	if err := c.outgoing.validateForDirection(JudicialCitationRequestedDirectionOutgoing); err != nil {
		return fmt.Errorf("outgoing が方向契約に適合しません: %w", err)
	}
	if err := c.incoming.validateForDirection(JudicialCitationRequestedDirectionIncoming); err != nil {
		return fmt.Errorf("incoming が方向契約に適合しません: %w", err)
	}
	wantsOutgoing := c.requestedDirection != JudicialCitationRequestedDirectionIncoming
	wantsIncoming := c.requestedDirection != JudicialCitationRequestedDirectionOutgoing
	if wantsOutgoing == (c.outgoing.Status() == JudicialCitationDirectionStatusNotRequested) {
		return fmt.Errorf("requestedDirection と outgoing.status が一致しません")
	}
	if wantsIncoming == (c.incoming.Status() == JudicialCitationDirectionStatusNotRequested) {
		return fmt.Errorf("requestedDirection と incoming.status が一致しません")
	}
	return nil
}

func (c JudicialCitationCoverage) requestedDirectionsComplete() bool {
	switch c.requestedDirection {
	case JudicialCitationRequestedDirectionOutgoing:
		return c.outgoing.Status() == JudicialCitationDirectionStatusComplete
	case JudicialCitationRequestedDirectionIncoming:
		return c.incoming.Status() == JudicialCitationDirectionStatusComplete
	case JudicialCitationRequestedDirectionBoth:
		return c.outgoing.Status() == JudicialCitationDirectionStatusComplete &&
			c.incoming.Status() == JudicialCitationDirectionStatusComplete
	default:
		return false
	}
}

func (c JudicialCitationCoverage) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		RequestedDirection JudicialCitationRequestedDirection `json:"requestedDirection"`
		HopDepth           int                                `json:"hopDepth"`
		Outgoing           JudicialCitationDirectionCoverage  `json:"outgoing"`
		Incoming           JudicialCitationDirectionCoverage  `json:"incoming"`
	}{c.requestedDirection, c.hopDepth, c.outgoing, c.incoming})
}

func (*JudicialCitationCoverage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationCoverage は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationCoverage を使用してください",
	)
}

// JudicialCitationDirectionCoverageValues は、一方向の coverage 作成値を保持する。
type JudicialCitationDirectionCoverageValues struct {
	Status            JudicialCitationDirectionStatus
	Methods           []JudicialCitationMethod
	Truncated         bool
	Limit             *int
	AttemptedSearches *int
	CompletedSearches *int
}

// JudicialCitationDirectionCoverage は、一方向の処理状態と上限を表す。
type JudicialCitationDirectionCoverage struct {
	status            JudicialCitationDirectionStatus
	methods           []JudicialCitationMethod
	truncated         bool
	limit             *int
	attemptedSearches *int
	completedSearches *int
}

func NewJudicialCitationDirectionCoverage(
	values JudicialCitationDirectionCoverageValues,
) (JudicialCitationDirectionCoverage, error) {
	coverage := JudicialCitationDirectionCoverage{
		status:            values.Status,
		methods:           slices.Clone(values.Methods),
		truncated:         values.Truncated,
		limit:             cloneOptionalInt(values.Limit),
		attemptedSearches: cloneOptionalInt(values.AttemptedSearches),
		completedSearches: cloneOptionalInt(values.CompletedSearches),
	}
	if err := coverage.Validate(); err != nil {
		return JudicialCitationDirectionCoverage{}, err
	}
	return coverage, nil
}

func (c JudicialCitationDirectionCoverage) Status() JudicialCitationDirectionStatus {
	return c.status
}
func (c JudicialCitationDirectionCoverage) Methods() []JudicialCitationMethod {
	return slices.Clone(c.methods)
}
func (c JudicialCitationDirectionCoverage) Truncated() bool { return c.truncated }
func (c JudicialCitationDirectionCoverage) Limit() (int, bool) {
	return optionalIntValue(c.limit)
}
func (c JudicialCitationDirectionCoverage) AttemptedSearches() (int, bool) {
	return optionalIntValue(c.attemptedSearches)
}
func (c JudicialCitationDirectionCoverage) CompletedSearches() (int, bool) {
	return optionalIntValue(c.completedSearches)
}

func (c JudicialCitationDirectionCoverage) Validate() error {
	if !c.status.valid() {
		return fmt.Errorf("status が有効ではありません")
	}
	if c.methods == nil {
		return fmt.Errorf("methods は空配列または値を持つ配列でなければなりません")
	}
	seen := make(map[JudicialCitationMethod]struct{}, len(c.methods))
	for _, method := range c.methods {
		if !method.valid() {
			return fmt.Errorf("methods に不正な値があります")
		}
		if _, exists := seen[method]; exists {
			return fmt.Errorf("methods に重複があります")
		}
		seen[method] = struct{}{}
	}
	if c.limit != nil && (*c.limit < 1 || *c.limit > 10) {
		return fmt.Errorf("limit は 1 以上 10 以下でなければなりません")
	}
	if (c.attemptedSearches == nil) != (c.completedSearches == nil) {
		return fmt.Errorf("attemptedSearches と completedSearches は同時に指定しなければなりません")
	}
	if c.attemptedSearches != nil && (*c.attemptedSearches < 0 || *c.attemptedSearches > 2) {
		return fmt.Errorf("attemptedSearches は 0 以上 2 以下でなければなりません")
	}
	if c.completedSearches != nil &&
		(*c.completedSearches < 0 || *c.completedSearches > *c.attemptedSearches) {
		return fmt.Errorf("completedSearches は 0 以上 attemptedSearches 以下でなければなりません")
	}
	if c.status == JudicialCitationDirectionStatusNotRequested &&
		(len(c.methods) != 0 || c.truncated || c.limit != nil || c.attemptedSearches != nil) {
		return fmt.Errorf("not_requested は空の methods 以外の処理情報を持てません")
	}
	return nil
}

func (c JudicialCitationDirectionCoverage) validateForDirection(
	direction JudicialCitationRequestedDirection,
) error {
	hasPDF := slices.Contains(c.methods, JudicialCitationMethodOfficialPDFText)
	hasSearch := slices.Contains(c.methods, JudicialCitationMethodOfficialCaseSearch)
	switch direction {
	case JudicialCitationRequestedDirectionOutgoing:
		if hasSearch || c.limit != nil || c.attemptedSearches != nil || c.completedSearches != nil {
			return fmt.Errorf("outgoing に候補検索の field 又は method は指定できません")
		}
		if c.status == JudicialCitationDirectionStatusComplete && !hasPDF {
			return fmt.Errorf("complete の outgoing には official_pdf_text が必要です")
		}
		if c.status == JudicialCitationDirectionStatusUnavailable && hasPDF {
			return fmt.Errorf("unavailable の outgoing は official_pdf_text を完了済みにできません")
		}
	case JudicialCitationRequestedDirectionIncoming:
		if hasPDF {
			return fmt.Errorf("incoming に official_pdf_text は指定できません")
		}
		if c.status == JudicialCitationDirectionStatusUnavailable {
			return fmt.Errorf("incoming は unavailable にできません")
		}
		if c.status == JudicialCitationDirectionStatusNotRequested {
			return nil
		}
		if c.limit == nil || c.attemptedSearches == nil || c.completedSearches == nil {
			return fmt.Errorf("要求した incoming には limit と検索回数が必要です")
		}
		if *c.attemptedSearches < 1 {
			return fmt.Errorf("要求した incoming の attemptedSearches は 1 以上でなければなりません")
		}
		if hasSearch != (*c.completedSearches > 0) {
			return fmt.Errorf("official_case_search と completedSearches が一致しません")
		}
		if c.status == JudicialCitationDirectionStatusComplete &&
			*c.completedSearches != *c.attemptedSearches {
			return fmt.Errorf("complete の incoming では全検索が完了していなければなりません")
		}
		if c.status == JudicialCitationDirectionStatusPartial &&
			*c.completedSearches >= *c.attemptedSearches {
			return fmt.Errorf("partial の incoming には未完了検索が必要です")
		}
	}
	return nil
}

func (c JudicialCitationDirectionCoverage) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Status            JudicialCitationDirectionStatus `json:"status"`
		Methods           []JudicialCitationMethod        `json:"methods"`
		Truncated         bool                            `json:"truncated"`
		Limit             *int                            `json:"limit,omitempty"`
		AttemptedSearches *int                            `json:"attemptedSearches,omitempty"`
		CompletedSearches *int                            `json:"completedSearches,omitempty"`
	}{
		c.status,
		slices.Clone(c.methods),
		c.truncated,
		cloneOptionalInt(c.limit),
		cloneOptionalInt(c.attemptedSearches),
		cloneOptionalInt(c.completedSearches),
	})
}

func (*JudicialCitationDirectionCoverage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationDirectionCoverage は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationDirectionCoverage を使用してください",
	)
}

// JudicialCitationIssueValues は、部分失敗又は制限の作成値を保持する。
type JudicialCitationIssueValues struct {
	Direction JudicialCitationIssueDirection
	Stage     JudicialCitationIssueStage
	Code      string
	Message   string
	Retryable bool
}

// JudicialCitationIssue は、利用者が次の対応を判断できる縮退情報を表す。
type JudicialCitationIssue struct {
	direction JudicialCitationIssueDirection
	stage     JudicialCitationIssueStage
	code      string
	message   string
	retryable bool
}

func NewJudicialCitationIssue(values JudicialCitationIssueValues) (JudicialCitationIssue, error) {
	issue := JudicialCitationIssue{
		direction: values.Direction,
		stage:     values.Stage,
		code:      values.Code,
		message:   values.Message,
		retryable: values.Retryable,
	}
	if err := issue.Validate(); err != nil {
		return JudicialCitationIssue{}, err
	}
	return issue, nil
}

func (i JudicialCitationIssue) Direction() JudicialCitationIssueDirection { return i.direction }
func (i JudicialCitationIssue) Stage() JudicialCitationIssueStage         { return i.stage }
func (i JudicialCitationIssue) Code() string                              { return i.code }
func (i JudicialCitationIssue) Message() string                           { return i.message }
func (i JudicialCitationIssue) Retryable() bool                           { return i.retryable }

func (i JudicialCitationIssue) Validate() error {
	if !i.direction.valid() {
		return fmt.Errorf("direction が有効ではありません")
	}
	if !i.stage.valid() {
		return fmt.Errorf("stage が有効ではありません")
	}
	if !utf8.ValidString(i.code) || i.code == "" {
		return fmt.Errorf("code は必須の UTF-8 文字列です")
	}
	if !utf8.ValidString(i.message) || i.message == "" {
		return fmt.Errorf("message は必須の UTF-8 文字列です")
	}
	return nil
}

func (i JudicialCitationIssue) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Direction JudicialCitationIssueDirection `json:"direction"`
		Stage     JudicialCitationIssueStage     `json:"stage"`
		Code      string                         `json:"code"`
		Message   string                         `json:"message"`
		Retryable bool                           `json:"retryable"`
	}{i.direction, i.stage, i.code, i.message, i.retryable})
}

func (*JudicialCitationIssue) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"JudicialCitationIssue は JSON から直接復元できません。境界専用の入力型から NewJudicialCitationIssue を使用してください",
	)
}

func optionalIntValue(value *int) (int, bool) {
	if value == nil {
		return 0, false
	}
	return *value, true
}
