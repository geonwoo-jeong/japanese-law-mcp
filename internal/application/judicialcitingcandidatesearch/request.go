// Package judicialcitingcandidatesearch は、被引用候補検索 capability の型付き境界を提供する。
package judicialcitingcandidatesearch

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、被引用候補検索 capability の識別子である。
	CapabilityID = "judicial-decision.citing-candidate.search"
	// MajorVersion は、被引用候補検索 capability のメジャーバージョンである。
	MajorVersion = 1
	// DefaultLimit は、limit を省略した場合の返却上限である。
	DefaultLimit = 5
	// MaxLimit は、一回の operation で返せる候補上限である。
	MaxLimit = 10
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Target model.SourcedResource[model.JudicialDecisionDetails]
	Limit  *int
}

// Request は、judicial-decision.citing-candidate.search@1 の検証済み入力である。
type Request struct {
	target      model.SourcedResource[model.JudicialDecisionDetails]
	limit       int
	initialized bool
}

// NewRequest は、対象裁判例と返却上限を検証した不変な Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	limit := DefaultLimit
	if values.Limit != nil {
		limit = *values.Limit
	}
	request := Request{target: values.Target, limit: limit, initialized: true}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

// Target は、引用される側の公式掲載裁判例を返す。
func (r Request) Target() model.SourcedResource[model.JudicialDecisionDetails] {
	return r.target
}

// Limit は、既定値適用後の返却上限を返す。
func (r Request) Limit() int { return r.limit }

// Validate は、外部呼出し前に対象と上限の全入力制約を確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return fmt.Errorf("Request は NewRequest で作成しなければなりません")
	}
	if r.limit < 1 || r.limit > MaxLimit {
		return newArgumentError("limit", "は 1 以上 10 以下でなければなりません")
	}
	if err := validateTarget(r.target); err != nil {
		return err
	}
	return nil
}

func validateTarget(
	target model.SourcedResource[model.JudicialDecisionDetails],
) error {
	if err := target.Validate(); err != nil {
		return newArgumentError("ref", "は有効な裁判例詳細を指さなければなりません")
	}
	ref := target.Ref()
	key := ref.Key()
	if key.ResourceType() != "judicial-decision" {
		return newArgumentError("ref", "の resourceType は judicial-decision でなければなりません")
	}
	if _, exists := key.VersionID(); exists {
		return newArgumentError("ref", "に versionId を指定できません")
	}
	summary := target.Data().Summary()
	if err := summary.Validate(); err != nil || summary.CaseNumber() == "" {
		return newArgumentError("target", "の summary.caseNumber は必須です")
	}
	source := summary.Source()
	if err := source.Validate(); err != nil ||
		source.Authority() != model.AuthorityOfficial ||
		key.SourceID() != source.ID() {
		return newArgumentError("target", "の summary.source は ref と一致する公式情報源でなければなりません")
	}
	provenance := target.Provenance()
	last := provenance[len(provenance)-1]
	if last.Source() != source || last.ResourceKey() != key {
		return newArgumentError("target", "の最後の provenance は ref と summary.source に一致しなければなりません")
	}
	return nil
}

// UnmarshalJSON は、境界値を介さない直接復元を拒否する。
func (*Request) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Request は JSON から直接復元できません。境界専用の入力型から NewRequest を使用してください",
	)
}

var _ json.Unmarshaler = (*Request)(nil)
