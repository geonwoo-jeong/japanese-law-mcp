package model

const (
	judicialCitationExcerptMaxBytes = 256
	judicialCitationDecisionEdgeMax = 64
	judicialCitationLawEdgeMax      = 32
)

// JudicialCitationResultStatus は、引用追跡結果の完了状態を表す。
type JudicialCitationResultStatus string

const (
	JudicialCitationResultStatusComplete JudicialCitationResultStatus = "complete"
	JudicialCitationResultStatusPartial  JudicialCitationResultStatus = "partial"
)

// JudicialCitationNodeType は、引用 graph のノード種別を表す。
type JudicialCitationNodeType string

const (
	JudicialCitationNodeTypeDecision          JudicialCitationNodeType = "judicial_decision"
	JudicialCitationNodeTypeLawProvision      JudicialCitationNodeType = "law_provision"
	JudicialCitationNodeTypeDecisionReference JudicialCitationNodeType = "judicial_decision_reference"
)

// JudicialCitationRelationType は、引用 graph の関係種別を表す。
type JudicialCitationRelationType string

const (
	JudicialCitationRelationTypeCitesDecision          JudicialCitationRelationType = "cites_judicial_decision"
	JudicialCitationRelationTypePossibleCitesDecision  JudicialCitationRelationType = "possible_cites_judicial_decision"
	JudicialCitationRelationTypeReferencesLawProvision JudicialCitationRelationType = "references_law_provision"
	JudicialCitationRelationTypeHasLowerCourtDecision  JudicialCitationRelationType = "has_lower_court_decision"
)

// JudicialCitationEvidenceLevel は、引用関係の確認水準を表す。
type JudicialCitationEvidenceLevel string

const (
	JudicialCitationEvidenceLevelOfficialMetadata        JudicialCitationEvidenceLevel = "official_metadata"
	JudicialCitationEvidenceLevelExactTextMatch          JudicialCitationEvidenceLevel = "exact_text_match"
	JudicialCitationEvidenceLevelOfficialSearchCandidate JudicialCitationEvidenceLevel = "official_search_candidate"
)

// JudicialCitationMentionType は、未解決言及の種別を表す。
type JudicialCitationMentionType string

const (
	JudicialCitationMentionTypeDecision     JudicialCitationMentionType = "judicial_decision"
	JudicialCitationMentionTypeLawProvision JudicialCitationMentionType = "law_provision"
)

// JudicialCitationUnresolvedReason は、言及を edge にしなかった理由を表す。
type JudicialCitationUnresolvedReason string

const (
	JudicialCitationUnresolvedReasonAmbiguousTarget        JudicialCitationUnresolvedReason = "ambiguous_target"
	JudicialCitationUnresolvedReasonNoPublishedTargetMatch JudicialCitationUnresolvedReason = "no_published_target_match"
	JudicialCitationUnresolvedReasonInsufficientIdentity   JudicialCitationUnresolvedReason = "insufficient_identity"
	JudicialCitationUnresolvedReasonUnsupportedReference   JudicialCitationUnresolvedReason = "unsupported_reference_form"
	JudicialCitationUnresolvedReasonUnregisteredLawName    JudicialCitationUnresolvedReason = "unregistered_law_name"
	JudicialCitationUnresolvedReasonAmbiguousLawLocation   JudicialCitationUnresolvedReason = "ambiguous_law_location"
	JudicialCitationUnresolvedReasonFuzzyMatchOnly         JudicialCitationUnresolvedReason = "fuzzy_match_only"
)

// JudicialCitationRequestedDirection は、追跡を要求した方向を表す。
type JudicialCitationRequestedDirection string

const (
	JudicialCitationRequestedDirectionOutgoing JudicialCitationRequestedDirection = "outgoing"
	JudicialCitationRequestedDirectionIncoming JudicialCitationRequestedDirection = "incoming"
	JudicialCitationRequestedDirectionBoth     JudicialCitationRequestedDirection = "both"
)

// JudicialCitationDirectionStatus は、方向ごとの処理状態を表す。
type JudicialCitationDirectionStatus string

const (
	JudicialCitationDirectionStatusComplete     JudicialCitationDirectionStatus = "complete"
	JudicialCitationDirectionStatusPartial      JudicialCitationDirectionStatus = "partial"
	JudicialCitationDirectionStatusUnavailable  JudicialCitationDirectionStatus = "unavailable"
	JudicialCitationDirectionStatusNotRequested JudicialCitationDirectionStatus = "not_requested"
)

// JudicialCitationMethod は、完了した公式情報の取得方法を表す。
type JudicialCitationMethod string

const (
	JudicialCitationMethodOfficialDetailMetadata JudicialCitationMethod = "official_detail_metadata"
	JudicialCitationMethodOfficialPDFText        JudicialCitationMethod = "official_pdf_text"
	JudicialCitationMethodOfficialCaseSearch     JudicialCitationMethod = "official_case_search"
)

// JudicialCitationIssueDirection は、issue が属する方向を表す。
type JudicialCitationIssueDirection string

const (
	JudicialCitationIssueDirectionOutgoing JudicialCitationIssueDirection = "outgoing"
	JudicialCitationIssueDirectionIncoming JudicialCitationIssueDirection = "incoming"
	JudicialCitationIssueDirectionShared   JudicialCitationIssueDirection = "shared"
)

// JudicialCitationIssueStage は、issue が発生した処理段階を表す。
type JudicialCitationIssueStage string

const (
	JudicialCitationIssueStageOfficialDetailMetadata JudicialCitationIssueStage = "official_detail_metadata"
	JudicialCitationIssueStageOfficialPDFText        JudicialCitationIssueStage = "official_pdf_text"
	JudicialCitationIssueStageOfficialCaseSearch     JudicialCitationIssueStage = "official_case_search"
	JudicialCitationIssueStageLawReferenceResolution JudicialCitationIssueStage = "law_reference_resolution"
)

func (s JudicialCitationResultStatus) valid() bool {
	return s == JudicialCitationResultStatusComplete || s == JudicialCitationResultStatusPartial
}

func (t JudicialCitationNodeType) valid() bool {
	return t == JudicialCitationNodeTypeDecision ||
		t == JudicialCitationNodeTypeLawProvision ||
		t == JudicialCitationNodeTypeDecisionReference
}

func (t JudicialCitationRelationType) valid() bool {
	return t == JudicialCitationRelationTypeCitesDecision ||
		t == JudicialCitationRelationTypePossibleCitesDecision ||
		t == JudicialCitationRelationTypeReferencesLawProvision ||
		t == JudicialCitationRelationTypeHasLowerCourtDecision
}

func (l JudicialCitationEvidenceLevel) valid() bool {
	return l == JudicialCitationEvidenceLevelOfficialMetadata ||
		l == JudicialCitationEvidenceLevelExactTextMatch ||
		l == JudicialCitationEvidenceLevelOfficialSearchCandidate
}

func (t JudicialCitationMentionType) valid() bool {
	return t == JudicialCitationMentionTypeDecision || t == JudicialCitationMentionTypeLawProvision
}

func (r JudicialCitationUnresolvedReason) valid() bool {
	return r == JudicialCitationUnresolvedReasonAmbiguousTarget ||
		r == JudicialCitationUnresolvedReasonNoPublishedTargetMatch ||
		r == JudicialCitationUnresolvedReasonInsufficientIdentity ||
		r == JudicialCitationUnresolvedReasonUnsupportedReference ||
		r == JudicialCitationUnresolvedReasonUnregisteredLawName ||
		r == JudicialCitationUnresolvedReasonAmbiguousLawLocation ||
		r == JudicialCitationUnresolvedReasonFuzzyMatchOnly
}

func (d JudicialCitationRequestedDirection) valid() bool {
	return d == JudicialCitationRequestedDirectionOutgoing ||
		d == JudicialCitationRequestedDirectionIncoming ||
		d == JudicialCitationRequestedDirectionBoth
}

func (s JudicialCitationDirectionStatus) valid() bool {
	return s == JudicialCitationDirectionStatusComplete ||
		s == JudicialCitationDirectionStatusPartial ||
		s == JudicialCitationDirectionStatusUnavailable ||
		s == JudicialCitationDirectionStatusNotRequested
}

func (m JudicialCitationMethod) valid() bool {
	return m == JudicialCitationMethodOfficialDetailMetadata ||
		m == JudicialCitationMethodOfficialPDFText ||
		m == JudicialCitationMethodOfficialCaseSearch
}

func (d JudicialCitationIssueDirection) valid() bool {
	return d == JudicialCitationIssueDirectionOutgoing ||
		d == JudicialCitationIssueDirectionIncoming ||
		d == JudicialCitationIssueDirectionShared
}

func (s JudicialCitationIssueStage) valid() bool {
	return s == JudicialCitationIssueStageOfficialDetailMetadata ||
		s == JudicialCitationIssueStageOfficialPDFText ||
		s == JudicialCitationIssueStageOfficialCaseSearch ||
		s == JudicialCitationIssueStageLawReferenceResolution
}
