package legalquery

import "fmt"

// QueryCandidateCompositionRole は、候補が合成で担う役割を表す。
type QueryCandidateCompositionRole string

const (
	// QueryCandidateCompositionRoleRequiredMember は、明示意図の必須部分を表す。
	QueryCandidateCompositionRoleRequiredMember QueryCandidateCompositionRole = "required_member"
)

// QueryCandidateStepOriginValues は、候補 step の原文上の基準位置を構築する値である。
type QueryCandidateStepOriginValues struct {
	StepID          string
	SourceStartByte int
}

// QueryCandidateStepOrigin は、一つの候補 step と原文上の開始位置を結び付ける。
type QueryCandidateStepOrigin struct {
	stepID          string
	sourceStartByte int
}

// NewQueryCandidateStepOrigin は、非負の原文開始位置を持つ step origin を返す。
func NewQueryCandidateStepOrigin(
	values QueryCandidateStepOriginValues,
) (QueryCandidateStepOrigin, error) {
	origin := QueryCandidateStepOrigin{
		stepID:          values.StepID,
		sourceStartByte: values.SourceStartByte,
	}
	if err := origin.Validate(); err != nil {
		return QueryCandidateStepOrigin{}, err
	}
	return origin, nil
}

// StepID は、構成元候補の step ID を返す。
func (o QueryCandidateStepOrigin) StepID() string {
	return o.stepID
}

// SourceStartByte は、元の照会文 UTF-8 byte 列上の開始位置を返す。
func (o QueryCandidateStepOrigin) SourceStartByte() int {
	return o.sourceStartByte
}

// Validate は、step ID と非負の開始位置を確認する。
// 照会文の長さと UTF-8 rune 境界は、前処理結果を持つ Collect で確認する。
func (o QueryCandidateStepOrigin) Validate() error {
	if err := validateQueryPlanID("stepId", o.stepID); err != nil {
		return err
	}
	if o.sourceStartByte < 0 {
		return fmt.Errorf("sourceStartByte は 0 以上でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、profile を介さない直接復元を拒否する。
func (*QueryCandidateStepOrigin) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"QueryCandidateStepOrigin は JSON から直接復元できません",
	)
}

// QueryCandidateCompositionMemberValues は、候補の構成 sidecar を構築する値である。
type QueryCandidateCompositionMemberValues struct {
	CandidateID string
	Role        QueryCandidateCompositionRole
	StepOrigins []QueryCandidateStepOrigin
}

// QueryCandidateCompositionMember は、合成できる必須候補と全 step の原文位置を表す。
type QueryCandidateCompositionMember struct {
	candidateID string
	role        QueryCandidateCompositionRole
	stepOrigins []QueryCandidateStepOrigin
}

// NewQueryCandidateCompositionMember は、必須候補の位置付き sidecar を複製して返す。
func NewQueryCandidateCompositionMember(
	values QueryCandidateCompositionMemberValues,
) (QueryCandidateCompositionMember, error) {
	member := QueryCandidateCompositionMember{
		candidateID: values.CandidateID,
		role:        values.Role,
		stepOrigins: append(
			[]QueryCandidateStepOrigin(nil),
			values.StepOrigins...,
		),
	}
	if err := member.Validate(); err != nil {
		return QueryCandidateCompositionMember{}, err
	}
	return member, nil
}

// CandidateID は、同じ contribution 内の構成元候補 ID を返す。
func (m QueryCandidateCompositionMember) CandidateID() string {
	return m.candidateID
}

// Role は、候補が合成で担う固定の役割を返す。
func (m QueryCandidateCompositionMember) Role() QueryCandidateCompositionRole {
	return m.role
}

// StepOrigins は、構成元候補の step 順を保った原文位置の複製を返す。
func (m QueryCandidateCompositionMember) StepOrigins() []QueryCandidateStepOrigin {
	return append([]QueryCandidateStepOrigin(nil), m.stepOrigins...)
}

// Validate は、必須役割、step origin の件数、構造および一意性を確認する。
func (m QueryCandidateCompositionMember) Validate() error {
	if err := validateQueryPlanID("candidateId", m.candidateID); err != nil {
		return err
	}
	if m.role != QueryCandidateCompositionRoleRequiredMember {
		return fmt.Errorf("composition member の role は required_member でなければなりません")
	}
	if len(m.stepOrigins) < 1 || len(m.stepOrigins) > MaxCapabilityCalls {
		return fmt.Errorf("stepOrigins は一件以上四件以下でなければなりません")
	}
	seen := make(map[string]struct{}, len(m.stepOrigins))
	for index, origin := range m.stepOrigins {
		if err := origin.Validate(); err != nil {
			return fmt.Errorf(
				"stepOrigins[%d] が有効ではありません: %w",
				index,
				err,
			)
		}
		if _, exists := seen[origin.StepID()]; exists {
			return fmt.Errorf("stepOrigins の stepId を重複させることはできません")
		}
		seen[origin.StepID()] = struct{}{}
	}
	return nil
}

// UnmarshalJSON は、profile を介さない直接復元を拒否する。
func (*QueryCandidateCompositionMember) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"QueryCandidateCompositionMember は JSON から直接復元できません",
	)
}

func cloneQueryCandidateCompositionMembers(
	values []QueryCandidateCompositionMember,
) []QueryCandidateCompositionMember {
	result := make([]QueryCandidateCompositionMember, len(values))
	for index, value := range values {
		result[index] = QueryCandidateCompositionMember{
			candidateID: value.candidateID,
			role:        value.role,
			stepOrigins: value.StepOrigins(),
		}
	}
	return result
}
