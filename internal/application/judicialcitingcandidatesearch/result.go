package judicialcitingcandidatesearch

import (
	"encoding/json"
	"fmt"
	"reflect"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

// ResultStatus は、候補検索 operation の完了状態を表す。
type ResultStatus string

const (
	ResultStatusComplete ResultStatus = "complete"
	ResultStatusPartial  ResultStatus = "partial"
	SearchStatusComplete              = ResultStatusComplete
	SearchStatusPartial               = ResultStatusPartial
)

// IssueValues は、失敗した検索と安全な共通情報源エラーの作成値を保持する。
type IssueValues struct {
	SearchKind  SearchKind
	SourceError model.SourceError
}

// Issue は、失敗した検索種類と共通情報源エラーを保持する。
type Issue struct {
	searchKind  SearchKind
	sourceError model.SourceError
	errorResult model.ErrorResult
	initialized bool
}

func NewIssue(values IssueValues) (Issue, error) {
	if err := values.SourceError.Validate(); err != nil {
		return Issue{}, fmt.Errorf("issue の情報源エラーが有効ではありません: %w", err)
	}
	if !allowedIssueSourceCode(values.SourceError.Code()) {
		return Issue{}, fmt.Errorf("issue の情報源エラー code が capability 契約外です")
	}
	if values.SourceError.CapabilityID() != CapabilityID {
		return Issue{}, fmt.Errorf("issue の情報源エラー capabilityId が候補検索ではありません")
	}
	errorResult, err := model.NewErrorResultFromSourceError(values.SourceError)
	if err != nil {
		return Issue{}, fmt.Errorf("issue の公開エラーを構成できません: %w", err)
	}
	issue := Issue{
		searchKind:  values.SearchKind,
		sourceError: values.SourceError,
		errorResult: errorResult,
		initialized: true,
	}
	if err := issue.Validate(); err != nil {
		return Issue{}, err
	}
	return issue, nil
}

func allowedIssueSourceCode(code model.SourceErrorCode) bool {
	switch code {
	case model.SourceErrorCodeUnsupportedCapability,
		model.SourceErrorCodeUnsupportedQuery,
		model.SourceErrorCodeConfigurationRequired,
		model.SourceErrorCodeRateLimited,
		model.SourceErrorCodeSourceTimeout,
		model.SourceErrorCodeSourceUnavailable,
		model.SourceErrorCodeSourceBusy,
		model.SourceErrorCodeSourceContractChanged,
		model.SourceErrorCodeInvalidSourceResponse,
		model.SourceErrorCodeSourceResponseTooLarge,
		model.SourceErrorCodeSourceProcessingLimit,
		model.SourceErrorCodeUnsafeSourceContent:
		return true
	default:
		return false
	}
}

func (i Issue) SearchKind() SearchKind         { return i.searchKind }
func (i Issue) ErrorResult() model.ErrorResult { return i.errorResult }

func (i Issue) Validate() error {
	if !i.initialized {
		return fmt.Errorf("Issue は NewIssue で作成しなければなりません")
	}
	if err := (Attempt{SearchKind: i.searchKind, Status: AttemptStatusFailed}).Validate(); err != nil {
		return err
	}
	if err := i.sourceError.Validate(); err != nil {
		return fmt.Errorf("情報源エラーが有効ではありません: %w", err)
	}
	if !allowedIssueSourceCode(i.sourceError.Code()) {
		return fmt.Errorf("情報源エラー code が capability 契約外です")
	}
	if i.sourceError.CapabilityID() != CapabilityID {
		return fmt.Errorf("情報源エラー capabilityId が候補検索ではありません")
	}
	if err := i.errorResult.Validate(); err != nil {
		return fmt.Errorf("error が有効ではありません: %w", err)
	}
	details, exists := i.errorResult.Details()
	if !exists {
		return fmt.Errorf("error.details は必須です")
	}
	for key, expected := range map[string]string{
		"providerId":   i.sourceError.ProviderID(),
		"sourceId":     i.sourceError.SourceID(),
		"capabilityId": CapabilityID,
	} {
		value, ok := details[key].(string)
		if !ok || value == "" {
			return fmt.Errorf("error.details.%s は必須です", key)
		}
		if value != expected {
			return fmt.Errorf("error.details.%s が情報源エラーと一致しません", key)
		}
	}
	expected, err := model.NewErrorResultFromSourceError(i.sourceError)
	if err != nil {
		return fmt.Errorf("情報源エラーから公開エラーを再構成できません: %w", err)
	}
	expectedDetails, _ := expected.Details()
	if i.errorResult.Code() != expected.Code() ||
		i.errorResult.Message() != expected.Message() ||
		i.errorResult.Retryable() != expected.Retryable() ||
		!reflect.DeepEqual(details, expectedDetails) {
		return fmt.Errorf("error が情報源エラーから導出した値と一致しません")
	}
	return nil
}

func (i Issue) MarshalJSON() ([]byte, error) {
	if err := i.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		SearchKind SearchKind        `json:"searchKind"`
		Error      model.ErrorResult `json:"error"`
	}{i.searchKind, i.errorResult})
}

func (*Issue) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Issue は JSON から直接復元できません。NewIssue を使用してください")
}

// ResultValues は、候補検索結果の作成値を保持する。
type ResultValues struct {
	Status   ResultStatus
	Items    []Candidate
	Coverage Coverage
	Issues   []Issue
}

// Result は、一件以上成功した公式検索の候補を不変に保持する。
type Result struct {
	status      ResultStatus
	items       []Candidate
	coverage    Coverage
	issues      []Issue
	initialized bool
}

func NewResult(values ResultValues) (Result, error) {
	result := Result{
		status:      values.Status,
		items:       slices.Clone(values.Items),
		coverage:    values.Coverage,
		issues:      slices.Clone(values.Issues),
		initialized: true,
	}
	if err := result.Validate(); err != nil {
		return Result{}, err
	}
	return result, nil
}

func (r Result) Status() ResultStatus { return r.status }
func (r Result) Items() []Candidate   { return slices.Clone(r.items) }
func (r Result) Coverage() Coverage   { return r.coverage }
func (r Result) Issues() []Issue      { return slices.Clone(r.issues) }

func (r Result) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Result は NewResult で作成しなければなりません")
	}
	if r.status != ResultStatusComplete && r.status != ResultStatusPartial {
		return fmt.Errorf("status が定義されていません")
	}
	if err := r.coverage.Validate(); err != nil {
		return fmt.Errorf("coverage が有効ではありません: %w", err)
	}
	failed := 0
	completed := 0
	for _, attempt := range r.coverage.Attempts() {
		if attempt.Status == AttemptStatusFailed {
			failed++
		} else {
			completed++
		}
	}
	if completed == 0 {
		return fmt.Errorf("一件以上の検索が成功した場合だけ Result を作成できます")
	}
	if (failed == 0) != (r.status == ResultStatusComplete) || failed != len(r.issues) {
		return fmt.Errorf("status、attempts と issues が一致しません")
	}
	issueKinds := make(map[SearchKind]struct{}, len(r.issues))
	for index, issue := range r.issues {
		if err := issue.Validate(); err != nil {
			return fmt.Errorf("issues[%d] が有効ではありません: %w", index, err)
		}
		if !hasFailedAttempt(r.coverage.Attempts(), issue.SearchKind()) {
			return fmt.Errorf("issues[%d] に対応する failed attempt がありません", index)
		}
		if _, exists := issueKinds[issue.SearchKind()]; exists {
			return fmt.Errorf("issues[%d] の searchKind が重複しています", index)
		}
		issueKinds[issue.SearchKind()] = struct{}{}
	}
	if len(r.items) > MaxLimit || len(r.items) > r.coverage.DedupedCandidateCount() {
		return fmt.Errorf("items の件数が候補上限または coverage と一致しません")
	}
	if len(r.items) < r.coverage.DedupedCandidateCount() && !r.coverage.Truncated() {
		return fmt.Errorf("未返却候補がある場合は coverage.truncated が必要です")
	}
	seen := make(map[model.SourceResourceRef]struct{}, len(r.items))
	for index, item := range r.items {
		if err := item.Validate(); err != nil {
			return fmt.Errorf("items[%d] が有効ではありません: %w", index, err)
		}
		ref := item.Decision().Ref()
		if _, exists := seen[ref]; exists {
			return fmt.Errorf("items[%d] の ref が重複しています", index)
		}
		seen[ref] = struct{}{}
	}
	return nil
}

func hasFailedAttempt(attempts []Attempt, kind SearchKind) bool {
	for _, attempt := range attempts {
		if attempt.SearchKind == kind && attempt.Status == AttemptStatusFailed {
			return true
		}
	}
	return false
}

func (r Result) MarshalJSON() ([]byte, error) {
	if err := r.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Status   ResultStatus `json:"status"`
		Items    []Candidate  `json:"items"`
		Coverage Coverage     `json:"coverage"`
		Issues   []Issue      `json:"issues"`
	}{r.status, slices.Clone(r.items), r.coverage, slices.Clone(r.issues)})
}

func (*Result) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Result は JSON から直接復元できません。NewResult を使用してください")
}
