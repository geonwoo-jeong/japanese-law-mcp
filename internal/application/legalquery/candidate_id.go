package legalquery

import (
	"fmt"
	"strconv"
)

const (
	maximumProfileOrdinal   = 9999
	maximumCandidateOrdinal = 16
	maximumStepOrdinal      = 4
)

// CandidateIDScope は、profile ごとの候補 ID 名前空間を表す。
type CandidateIDScope struct {
	profileOrdinal int
}

// NewCandidateIDScope は、composition 順に割り当てた ID 名前空間を返す。
func NewCandidateIDScope(profileOrdinal int) (CandidateIDScope, error) {
	if profileOrdinal < 1 || profileOrdinal > maximumProfileOrdinal {
		return CandidateIDScope{}, fmt.Errorf(
			"profile ordinal は 1 以上 %d 以下でなければなりません",
			maximumProfileOrdinal,
		)
	}
	return CandidateIDScope{profileOrdinal: profileOrdinal}, nil
}

// CandidateID は、入力断片を含まない候補 ID を返す。
func (s CandidateIDScope) CandidateID(
	candidateOrdinal int,
) (string, error) {
	if err := s.validate(); err != nil {
		return "", err
	}
	if candidateOrdinal < 1 ||
		candidateOrdinal > maximumCandidateOrdinal {
		return "", fmt.Errorf(
			"candidate ordinal は 1 以上 %d 以下でなければなりません",
			maximumCandidateOrdinal,
		)
	}
	return "candidate-" +
		strconv.Itoa(s.profileOrdinal) +
		"-" +
		strconv.Itoa(candidateOrdinal), nil
}

// StepID は、入力断片を含まない plan 全体で一意な step ID を返す。
func (s CandidateIDScope) StepID(
	candidateOrdinal int,
	stepOrdinal int,
) (string, error) {
	_, err := s.CandidateID(candidateOrdinal)
	if err != nil {
		return "", err
	}
	if stepOrdinal < 1 || stepOrdinal > maximumStepOrdinal {
		return "", fmt.Errorf(
			"step ordinal は 1 以上 %d 以下でなければなりません",
			maximumStepOrdinal,
		)
	}
	return "step-" +
		strconv.Itoa(s.profileOrdinal) +
		"-" +
		strconv.Itoa(candidateOrdinal) +
		"-" +
		strconv.Itoa(stepOrdinal), nil
}

func (s CandidateIDScope) validate() error {
	if s.profileOrdinal < 1 || s.profileOrdinal > maximumProfileOrdinal {
		return fmt.Errorf("candidate ID scope は初期化されていません")
	}
	return nil
}
