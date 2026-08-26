package judicialcitingcandidatesearch

import (
	"encoding/json"
	"fmt"
	"slices"
)

// SearchKind は、固定された検索値の種類を表す。
type SearchKind string

const (
	SearchKindCaseNumber       SearchKind = "case_number"
	SearchKindReporterCitation SearchKind = "reporter_citation"
)

// AttemptStatus は、検索一回の完了状態を表す。
type AttemptStatus string

const (
	AttemptStatusComplete AttemptStatus = "complete"
	AttemptStatusFailed   AttemptStatus = "failed"
)

// CoverageAttemptValues は、検索 attempt の作成値を保持する。
type CoverageAttemptValues struct {
	SearchKind SearchKind    `json:"searchKind"`
	Status     AttemptStatus `json:"status"`
}

// CoverageAttempt は、検索値を含めずに実行順と完了状態だけを保持する。
type CoverageAttempt struct {
	SearchKind SearchKind    `json:"searchKind"`
	Status     AttemptStatus `json:"status"`
}

// Attempt は、内部呼出しとの互換性を保つ CoverageAttempt の別名である。
type Attempt = CoverageAttempt

// NewCoverageAttempt は、閉じた検索種類と状態を検証する。
func NewCoverageAttempt(values CoverageAttemptValues) (CoverageAttempt, error) {
	attempt := CoverageAttempt{SearchKind: values.SearchKind, Status: values.Status}
	if err := attempt.Validate(); err != nil {
		return CoverageAttempt{}, err
	}
	return attempt, nil
}

func (a CoverageAttempt) Validate() error {
	if a.SearchKind != SearchKindCaseNumber && a.SearchKind != SearchKindReporterCitation {
		return fmt.Errorf("searchKind が定義されていません")
	}
	if a.Status != AttemptStatusComplete && a.Status != AttemptStatusFailed {
		return fmt.Errorf("status が定義されていません")
	}
	return nil
}

// CoverageValues は、検索 operation の観測範囲を保持する。
type CoverageValues struct {
	Attempts              []CoverageAttempt
	ObservedItemCount     int
	DedupedCandidateCount int
	DeduplicatedItemCount int
	Truncated             bool
}

// Coverage は、検索語を含まない候補検索の観測範囲である。
type Coverage struct {
	attempts              []CoverageAttempt
	observedItemCount     int
	dedupedCandidateCount int
	truncated             bool
	initialized           bool
}

func NewCoverage(values CoverageValues) (Coverage, error) {
	deduplicatedCount := values.DedupedCandidateCount
	if deduplicatedCount == 0 && values.DeduplicatedItemCount != 0 {
		deduplicatedCount = values.DeduplicatedItemCount
	}
	if values.DedupedCandidateCount != 0 && values.DeduplicatedItemCount != 0 &&
		values.DedupedCandidateCount != values.DeduplicatedItemCount {
		return Coverage{}, fmt.Errorf("候補件数の互換 field が一致しません")
	}
	coverage := Coverage{
		attempts:              slices.Clone(values.Attempts),
		observedItemCount:     values.ObservedItemCount,
		dedupedCandidateCount: deduplicatedCount,
		truncated:             values.Truncated,
		initialized:           true,
	}
	if err := coverage.Validate(); err != nil {
		return Coverage{}, err
	}
	return coverage, nil
}

func (c Coverage) Attempts() []CoverageAttempt { return slices.Clone(c.attempts) }
func (c Coverage) ObservedItemCount() int      { return c.observedItemCount }
func (c Coverage) DedupedCandidateCount() int  { return c.dedupedCandidateCount }
func (c Coverage) Truncated() bool             { return c.truncated }

func (c Coverage) Validate() error {
	if !c.initialized {
		return fmt.Errorf("Coverage は NewCoverage で作成しなければなりません")
	}
	if len(c.attempts) < 1 || len(c.attempts) > 2 {
		return fmt.Errorf("attempts は一件以上二件以下でなければなりません")
	}
	if c.attempts[0].SearchKind != SearchKindCaseNumber {
		return fmt.Errorf("最初の attempt は case_number でなければなりません")
	}
	if len(c.attempts) == 2 && c.attempts[1].SearchKind != SearchKindReporterCitation {
		return fmt.Errorf("二番目の attempt は reporter_citation でなければなりません")
	}
	for index, attempt := range c.attempts {
		if err := attempt.Validate(); err != nil {
			return fmt.Errorf("attempts[%d] が有効ではありません: %w", index, err)
		}
	}
	if c.observedItemCount < 0 || c.dedupedCandidateCount < 0 ||
		c.dedupedCandidateCount > c.observedItemCount {
		return fmt.Errorf("候補件数が有効ではありません")
	}
	return nil
}

func (c Coverage) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Attempts              []CoverageAttempt `json:"attempts"`
		ObservedItemCount     int               `json:"observedItemCount"`
		DedupedCandidateCount int               `json:"dedupedCandidateCount"`
		Truncated             bool              `json:"truncated"`
	}{slices.Clone(c.attempts), c.observedItemCount, c.dedupedCandidateCount, c.truncated})
}

func (*Coverage) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf("Coverage は JSON から直接復元できません。NewCoverage を使用してください")
}
