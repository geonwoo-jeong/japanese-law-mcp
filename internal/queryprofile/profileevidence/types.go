// Package profileevidence は、SOT-ARCH-031 と SOT-ARCH-032 に従い、
// query profile の評価中だけ保持する根拠対応と cluster key を扱う。
package profileevidence

import "github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"

const (
	maximumLocalIDBytes = 64
	// maximumFacts は、前処理の最大 256 mention と任意の入力 ref を合わせた上限である。
	maximumFacts  = 257
	maximumDrafts = legalquery.MaxRankedCandidates
	// maximumStepEvidence は、各 fact の基本 code と誤記補助 code を合わせた上限である。
	maximumStepEvidence = maximumFacts * 2
)

// Layer は、位置付き事実を評価する意図根拠レイヤである。
type Layer string

const (
	// LayerBoundary は、構造化入力または非実行境界を表す。
	LayerBoundary Layer = "boundary"
	// LayerExplicitTaskResource は、明示された task と resource を表す。
	LayerExplicitTaskResource Layer = "explicit_task_resource"
	// LayerTargetAnchor は、task と resource に束縛した取得対象を表す。
	LayerTargetAnchor Layer = "target_anchor"
	// LayerSemanticExpansion は、法概念または一般検索語による補助を表す。
	LayerSemanticExpansion Layer = "semantic_expansion"
	// LayerClarificationOrReject は、明確化または非実行への終端を表す。
	LayerClarificationOrReject Layer = "clarification_or_reject"
)

// FactValues は、profile が検証した一つの事実を登録する値である。
type FactValues struct {
	FactID string
	Span   *legalquery.QuerySpan
}

// EvidenceValues は、一つの step と登録済み事実の対応を表す。
type EvidenceValues struct {
	FactID              string
	Layer               Layer
	Code                legalquery.EvidenceCode
	IndependentPositive bool
	ClusterSpan         bool
}

// StepValues は、候補 draft 内の一つの step と根拠対応を表す。
type StepValues struct {
	StepID        string
	SourceOrdinal int
	TopicOrdinal  int
	Evidence      []EvidenceValues
}

// DraftValues は、一つの候補 draft の根拠対応を表す。
type DraftValues struct {
	DraftID string
	Steps   []StepValues
}

// MappingValues は、一回の profile 評価に必要な根拠対応を表す。
type MappingValues struct {
	ProfileID string
	Facts     []FactValues
	Drafts    []DraftValues
}

// Evidence は、検証済みの一つの step 根拠である。
type Evidence struct {
	factID              string
	layer               Layer
	code                legalquery.EvidenceCode
	span                *legalquery.QuerySpan
	independentPositive bool
	clusterSpan         bool
}

// FactID は、profile 評価内だけで有効な事実 ID を返す。
func (e Evidence) FactID() string {
	return e.factID
}

// Layer は、事実を評価した意図根拠レイヤを返す。
func (e Evidence) Layer() Layer {
	return e.layer
}

// Code は、候補へ寄与する根拠コードを返す。
func (e Evidence) Code() legalquery.EvidenceCode {
	return e.code
}

// Span は、原文 span の値と有無を返す。
func (e Evidence) Span() (legalquery.QuerySpan, bool) {
	if e.span == nil {
		return legalquery.QuerySpan{}, false
	}
	return *e.span, true
}

// IndependentPositive は、profile が独立した正の根拠と認めたかを返す。
func (e Evidence) IndependentPositive() bool {
	return e.independentPositive
}

// ClusterSpan は、この span を cluster key に利用できるかを返す。
func (e Evidence) ClusterSpan() bool {
	return e.clusterSpan
}

type fact struct {
	factID string
	span   *legalquery.QuerySpan
}

type step struct {
	stepID        string
	sourceOrdinal int
	topicOrdinal  int
	evidence      []Evidence
}

type draft struct {
	draftID  string
	steps    []step
	stepByID map[string]int
}

// Mapping は、一 request の一 profile 評価中だけ使う不変な根拠対応である。
type Mapping struct {
	profileID string
	facts     map[string]fact
	drafts    map[string]draft
}

// ProfileID は、この対応を所有する profile ID を返す。
func (m Mapping) ProfileID() string {
	return m.profileID
}
