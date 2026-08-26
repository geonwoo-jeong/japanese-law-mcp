package judicialcitingcandidatesearch

import (
	"encoding/json"
	"fmt"
	"net/url"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const evidenceMethodID = "SOT-IF-073"

// CandidateValues は、被引用候補の作成値を保持する。
type CandidateValues struct {
	Decision model.SourcedResource[model.JudicialDecisionSummary]
	Evidence []model.JudicialCitationEvidence
}

// Candidate は、公式検索で観測した裁判例と根拠を不変に保持する。
type Candidate struct {
	decision    model.SourcedResource[model.JudicialDecisionSummary]
	evidence    []model.JudicialCitationEvidence
	initialized bool
}

// NewCandidate は、候補と official_search_candidate 根拠を検証する。
func NewCandidate(values CandidateValues) (Candidate, error) {
	candidate := Candidate{
		decision:    values.Decision,
		evidence:    slices.Clone(values.Evidence),
		initialized: true,
	}
	if err := candidate.Validate(); err != nil {
		return Candidate{}, err
	}
	return candidate, nil
}

func (c Candidate) Decision() model.SourcedResource[model.JudicialDecisionSummary] {
	return c.decision
}

func (c Candidate) Evidence() []model.JudicialCitationEvidence {
	return slices.Clone(c.evidence)
}

func (c Candidate) Validate() error {
	if !c.initialized {
		return fmt.Errorf("Candidate は NewCandidate で作成しなければなりません")
	}
	if err := c.decision.Validate(); err != nil {
		return fmt.Errorf("decision が有効ではありません: %w", err)
	}
	key := c.decision.Ref().Key()
	if key.ResourceType() != "judicial-decision" {
		return fmt.Errorf("decision.ref.key.resourceType が judicial-decision ではありません")
	}
	if _, exists := key.VersionID(); exists {
		return fmt.Errorf("decision.ref.key.versionId は指定できません")
	}
	decisionProvenance := c.decision.Provenance()
	lastDecisionProvenance := decisionProvenance[len(decisionProvenance)-1]
	decisionMethodID, exists := lastDecisionProvenance.MethodID()
	decisionURL, urlErr := url.Parse(lastDecisionProvenance.URL())
	if !exists || decisionMethodID != evidenceMethodID || urlErr != nil || decisionURL.RawQuery != "" {
		return fmt.Errorf("decision の最後の provenance は検索 query を除いた SOT-IF-073 でなければなりません")
	}
	if len(c.evidence) == 0 {
		return fmt.Errorf("evidence は一件以上必要です")
	}
	for index, evidence := range c.evidence {
		if err := evidence.Validate(); err != nil {
			return fmt.Errorf("evidence[%d] が有効ではありません: %w", index, err)
		}
		if evidence.EvidenceLevel() != model.JudicialCitationEvidenceLevelOfficialSearchCandidate {
			return fmt.Errorf("evidence[%d].evidenceLevel が official_search_candidate ではありません", index)
		}
		if _, exists := evidence.Excerpt(); exists {
			return fmt.Errorf("evidence[%d] に検索結果本文の excerpt を保持できません", index)
		}
		provenance := evidence.Provenance()
		methodID, exists := provenance.MethodID()
		if !exists || methodID != evidenceMethodID || provenance.ResourceKey() != key {
			return fmt.Errorf("evidence[%d].provenance が SOT-IF-073 の候補資源と一致しません", index)
		}
		parsedURL, err := url.Parse(provenance.URL())
		if err != nil || parsedURL.RawQuery != "" {
			return fmt.Errorf("evidence[%d].provenance.url に検索 query を保持できません", index)
		}
	}
	return nil
}

func (c Candidate) MarshalJSON() ([]byte, error) {
	if err := c.Validate(); err != nil {
		return nil, err
	}
	return json.Marshal(struct {
		Decision model.SourcedResource[model.JudicialDecisionSummary] `json:"decision"`
		Evidence []model.JudicialCitationEvidence                     `json:"evidence"`
	}{c.decision, slices.Clone(c.evidence)})
}

func (*Candidate) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"Candidate は JSON から直接復元できません。NewCandidate を使用してください",
	)
}
