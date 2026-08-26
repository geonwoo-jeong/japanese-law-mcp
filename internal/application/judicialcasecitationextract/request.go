package judicialcasecitationextract

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	// CapabilityID は、判例引用抽出 capability の識別子である。
	CapabilityID = "judicial-decision.case-citation.extract"
	// MajorVersion は、判例引用抽出 capability のメジャーバージョンである。
	MajorVersion = 1
)

// RequestValues は、Request の作成に必要な境界値を保持する。
type RequestValues struct {
	Decision model.SourcedResource[model.JudicialDecisionDetails]
	Document model.JudicialDocumentLink
}

// Request は、同一 request 内で検証済みの裁判例詳細と全文 PDF を保持する。
type Request struct {
	decision    model.SourcedResource[model.JudicialDecisionDetails]
	document    model.JudicialDocumentLink
	initialized bool
}

// NewRequest は、裁判例詳細と全文 PDF 所属を検証した Request を返す。
func NewRequest(values RequestValues) (Request, error) {
	request := Request{
		decision:    values.Decision,
		document:    values.Document,
		initialized: true,
	}
	if err := request.Validate(); err != nil {
		return Request{}, err
	}
	return request, nil
}

func (r Request) Decision() model.SourcedResource[model.JudicialDecisionDetails] {
	return r.decision
}

func (r Request) Document() model.JudicialDocumentLink { return r.document }

// Validate は、裁判例詳細と全文 PDF の構造および所属を確認する。
func (r Request) Validate() error {
	if !r.initialized {
		return invalidDecisionArgument("は NewRequest で検証しなければなりません")
	}
	if err := validateDecision(r.decision); err != nil {
		return err
	}
	if err := r.document.Validate(); err != nil {
		return invalidDocumentArgument("は有効な JudicialDocumentLink でなければなりません")
	}
	if r.document.Kind() != model.JudicialDocumentKindFullText {
		return invalidDocumentArgument("の kind は full_text でなければなりません")
	}
	if r.document.MediaType() != model.JudicialDocumentMediaTypePDF {
		return invalidDocumentArgument("の mediaType は application/pdf でなければなりません")
	}
	documents := r.decision.Data().Summary().Documents()
	matches := 0
	for _, document := range documents {
		if document == r.document {
			matches++
		}
	}
	if matches != 1 {
		return invalidDocumentArgument("は decision.data.summary.documents に一回だけ完全一致で含まれなければなりません")
	}
	return nil
}

// ValidateProviderSource は、判例詳細が指定情報源に属することを確認する。
func (r Request) ValidateProviderSource(expectedSourceID string) error {
	if err := r.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(expectedSourceID) == "" {
		return invalidDecisionArgument("の sourceId は空にできません")
	}
	if r.decision.Ref().Key().SourceID() != expectedSourceID ||
		r.decision.Data().Summary().Source().ID() != expectedSourceID {
		return invalidDecisionArgument("の sourceId は provider が扱う情報源と一致しなければなりません")
	}
	return nil
}

func validateDecision(decision model.SourcedResource[model.JudicialDecisionDetails]) error {
	if err := decision.Validate(); err != nil {
		return invalidDecisionArgument("は有効な裁判例詳細を指さなければなりません")
	}
	ref := decision.Ref()
	key := ref.Key()
	if key.ResourceType() != "judicial-decision" {
		return invalidDecisionArgument("の resourceType は judicial-decision でなければなりません")
	}
	if _, exists := key.VersionID(); exists {
		return invalidDecisionArgument("に versionId は指定できません")
	}
	summary := decision.Data().Summary()
	if err := summary.Validate(); err != nil || summary.CaseNumber() == "" {
		return invalidDecisionArgument("の summary.caseNumber は必須です")
	}
	source := summary.Source()
	if err := source.Validate(); err != nil ||
		source.Authority() != model.AuthorityOfficial ||
		source.ID() != key.SourceID() {
		return invalidDecisionArgument("の summary.source は ref と一致する公式情報源でなければなりません")
	}
	provenance := decision.Provenance()
	last := provenance[len(provenance)-1]
	if last.Source() != source || last.ResourceKey() != key {
		return invalidDecisionArgument("の最後の provenance は ref と summary.source に一致しなければなりません")
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

func invalidDecisionArgument(reason string) error {
	return newArgumentError("decision", reason)
}

func invalidDocumentArgument(reason string) error {
	return newArgumentError("document", reason)
}
