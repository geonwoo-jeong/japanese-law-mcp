package legalquery

import "fmt"

// CandidateGenerationSignal は、selector が候補と分離して扱う入力信号である。
type CandidateGenerationSignal string

const (
	// CandidateSignalNonJapaneseQuery は、日本語入力境界外を表す。
	CandidateSignalNonJapaneseQuery CandidateGenerationSignal = "non_japanese_query"
	// CandidateSignalUnsupportedLegalAdvice は、法的助言の明示要求を表す。
	CandidateSignalUnsupportedLegalAdvice CandidateGenerationSignal = "unsupported_legal_advice"
	// CandidateSignalUnsupportedTranslation は、翻訳の明示要求を表す。
	CandidateSignalUnsupportedTranslation CandidateGenerationSignal = "unsupported_translation"
	// CandidateSignalUnsupportedTaskOrResource は、対象外 task/resource を表す。
	CandidateSignalUnsupportedTaskOrResource CandidateGenerationSignal = "unsupported_task_or_resource"
	// CandidateSignalReservedPackRequest は、予約済み拡張 pack の要求を表す。
	CandidateSignalReservedPackRequest CandidateGenerationSignal = "reserved_pack_request"
)

// CandidateGenerationValues は、一 profile の候補生成結果を構築する値である。
type CandidateGenerationValues struct {
	ProfileID      string
	ProfileVersion string
	Candidates     []LegalQueryCandidate
	Signals        []CandidateGenerationSignal
}

// CandidateGeneration は、一 profile の候補と安全信号を不変に保持する。
type CandidateGeneration struct {
	profileID      string
	profileVersion string
	candidates     []LegalQueryCandidate
	signals        []CandidateGenerationSignal
}

// NewCandidateGeneration は、候補、ID および信号を複製して検証する。
func NewCandidateGeneration(
	values CandidateGenerationValues,
) (CandidateGeneration, error) {
	candidates, err := cloneLegalQueryCandidates(values.Candidates)
	if err != nil {
		return CandidateGeneration{}, err
	}
	generation := CandidateGeneration{
		profileID:      values.ProfileID,
		profileVersion: values.ProfileVersion,
		candidates:     candidates,
		signals:        append([]CandidateGenerationSignal(nil), values.Signals...),
	}
	if err := generation.Validate(); err != nil {
		return CandidateGeneration{}, err
	}
	return generation, nil
}

// ProfileID は、候補を生成した profile ID を返す。
func (g CandidateGeneration) ProfileID() string {
	return g.profileID
}

// ProfileVersion は、候補を生成した profile version を返す。
func (g CandidateGeneration) ProfileVersion() string {
	return g.profileVersion
}

// Candidates は、候補配列の深い複製を返す。
func (g CandidateGeneration) Candidates() []LegalQueryCandidate {
	candidates, err := cloneLegalQueryCandidates(g.candidates)
	if err != nil {
		panic(fmt.Sprintf("検証済み候補生成結果の複製に失敗しました: %v", err))
	}
	return candidates
}

// Signals は、入力信号の複製を返す。
func (g CandidateGeneration) Signals() []CandidateGenerationSignal {
	return append([]CandidateGenerationSignal(nil), g.signals...)
}

// Validate は、profile、候補 ID、step ID および信号順を確認する。
func (g CandidateGeneration) Validate() error {
	if err := validateQueryPlanID("profileId", g.profileID); err != nil {
		return err
	}
	if err := validateProfileVersion(g.profileVersion); err != nil {
		return err
	}
	if len(g.candidates) > maximumCandidateOrdinal {
		return fmt.Errorf(
			"candidates は %d 件以下でなければなりません",
			maximumCandidateOrdinal,
		)
	}
	candidateIDs := make(map[string]struct{}, len(g.candidates))
	stepIDs := make(map[string]struct{})
	for index, candidate := range g.candidates {
		if err := candidate.Validate(); err != nil {
			return fmt.Errorf("candidates[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := candidateIDs[candidate.CandidateID()]; exists {
			return fmt.Errorf("candidateId を重複させることはできません")
		}
		candidateIDs[candidate.CandidateID()] = struct{}{}
		for _, step := range candidate.Steps() {
			if _, exists := stepIDs[step.StepID()]; exists {
				return fmt.Errorf("stepId を候補間で重複させることはできません")
			}
			stepIDs[step.StepID()] = struct{}{}
		}
	}
	previousRank := -1
	for _, signal := range g.signals {
		rank, exists := candidateGenerationSignalRank(signal)
		if !exists {
			return fmt.Errorf("signals に定義されていない値があります")
		}
		if rank <= previousRank {
			return fmt.Errorf("signals は重複させず固定順に並べなければなりません")
		}
		previousRank = rank
	}
	return nil
}

func candidateGenerationSignalRank(
	value CandidateGenerationSignal,
) (int, bool) {
	switch value {
	case CandidateSignalNonJapaneseQuery:
		return 0, true
	case CandidateSignalUnsupportedLegalAdvice:
		return 1, true
	case CandidateSignalUnsupportedTranslation:
		return 2, true
	case CandidateSignalUnsupportedTaskOrResource:
		return 3, true
	case CandidateSignalReservedPackRequest:
		return 4, true
	default:
		return 0, false
	}
}
