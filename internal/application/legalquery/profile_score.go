package legalquery

import "fmt"

const maximumQueryProfileScore = 1_000_000

// QueryEvidenceWeightValues は、一つの根拠 weight を構築する値である。
type QueryEvidenceWeightValues struct {
	Code   EvidenceCode
	Weight int
}

// QueryEvidenceWeight は、profile 内で固定した一つの根拠 weight である。
type QueryEvidenceWeight struct {
	code   EvidenceCode
	weight int
}

// NewQueryEvidenceWeight は、閉じた根拠と正の weight を返す。
func NewQueryEvidenceWeight(
	values QueryEvidenceWeightValues,
) (QueryEvidenceWeight, error) {
	weight := QueryEvidenceWeight{
		code:   values.Code,
		weight: values.Weight,
	}
	if err := weight.Validate(); err != nil {
		return QueryEvidenceWeight{}, err
	}
	return weight, nil
}

// Code は、根拠コードを返す。
func (w QueryEvidenceWeight) Code() EvidenceCode {
	return w.code
}

// Weight は、同じ profile 内だけで使う加点値を返す。
func (w QueryEvidenceWeight) Weight() int {
	return w.weight
}

// Validate は、根拠と weight の範囲を確認する。
func (w QueryEvidenceWeight) Validate() error {
	if _, exists := evidenceRank(w.code); !exists {
		return fmt.Errorf("evidence weight の code が定義されていません")
	}
	if w.weight < 1 || w.weight > maximumQueryProfileScore {
		return fmt.Errorf(
			"evidence weight は 1 以上 %d 以下でなければなりません",
			maximumQueryProfileScore,
		)
	}
	return nil
}

// QueryScorePolicyValues は、score と confidence の境界値である。
type QueryScorePolicyValues struct {
	Minimum            int
	Maximum            int
	EvidenceWeights    []QueryEvidenceWeight
	HighConfidenceAt   int
	MediumConfidenceAt int
}

// QueryScorePolicy は、版付き profile の score 規則を不変に保持する。
type QueryScorePolicy struct {
	minimum            int
	maximum            int
	evidenceWeights    []QueryEvidenceWeight
	highConfidenceAt   int
	mediumConfidenceAt int
}

// NewQueryScorePolicy は、全根拠の weight と confidence 境界を返す。
func NewQueryScorePolicy(
	values QueryScorePolicyValues,
) (QueryScorePolicy, error) {
	policy := QueryScorePolicy{
		minimum:            values.Minimum,
		maximum:            values.Maximum,
		evidenceWeights:    append([]QueryEvidenceWeight(nil), values.EvidenceWeights...),
		highConfidenceAt:   values.HighConfidenceAt,
		mediumConfidenceAt: values.MediumConfidenceAt,
	}
	if err := policy.Validate(); err != nil {
		return QueryScorePolicy{}, err
	}
	return policy, nil
}

// Minimum は、score の最小値を返す。
func (p QueryScorePolicy) Minimum() int {
	return p.minimum
}

// Maximum は、score の最大値を返す。
func (p QueryScorePolicy) Maximum() int {
	return p.maximum
}

// EvidenceWeights は、強い根拠から並ぶ weight の複製を返す。
func (p QueryScorePolicy) EvidenceWeights() []QueryEvidenceWeight {
	return append([]QueryEvidenceWeight(nil), p.evidenceWeights...)
}

// HighConfidenceAt は、high confidence の下限を返す。
func (p QueryScorePolicy) HighConfidenceAt() int {
	return p.highConfidenceAt
}

// MediumConfidenceAt は、medium confidence の下限を返す。
func (p QueryScorePolicy) MediumConfidenceAt() int {
	return p.mediumConfidenceAt
}

// Weight は、根拠 code の weight と有無を返す。
func (p QueryScorePolicy) Weight(code EvidenceCode) (int, bool) {
	for _, weight := range p.evidenceWeights {
		if weight.code == code {
			return weight.weight, true
		}
	}
	return 0, false
}

// ConfidenceFor は、score を同じ profile の confidence 区分へ変換する。
func (p QueryScorePolicy) ConfidenceFor(score int) (Confidence, error) {
	if score < p.minimum || score > p.maximum {
		return "", fmt.Errorf("semantic score が profile の範囲外です")
	}
	switch {
	case score >= p.highConfidenceAt:
		return ConfidenceHigh, nil
	case score >= p.mediumConfidenceAt:
		return ConfidenceMedium, nil
	default:
		return ConfidenceLow, nil
	}
}

// Score は、重複のない固定順 evidence code を加点する。
func (p QueryScorePolicy) Score(codes []EvidenceCode) (int, error) {
	if _, err := validateEvidenceCodes(codes); err != nil {
		return 0, err
	}
	score := p.minimum
	for _, code := range codes {
		weight, exists := p.Weight(code)
		if !exists {
			return 0, fmt.Errorf("evidence code の weight がありません")
		}
		score += weight
	}
	if score > p.maximum {
		return 0, fmt.Errorf("semantic score が profile の最大値を超えました")
	}
	return score, nil
}

// Validate は、全根拠の固定順、score 範囲および confidence 境界を確認する。
func (p QueryScorePolicy) Validate() error {
	if p.minimum < 0 ||
		p.maximum < p.minimum ||
		p.maximum > maximumQueryProfileScore {
		return fmt.Errorf("score の minimum/maximum が有効ではありません")
	}
	if len(p.evidenceWeights) != 9 {
		return fmt.Errorf("evidenceWeights は全九根拠を一件ずつ必要とします")
	}
	total := p.minimum
	for index, weight := range p.evidenceWeights {
		if err := weight.Validate(); err != nil {
			return fmt.Errorf("evidenceWeights[%d]: %w", index, err)
		}
		rank, _ := evidenceRank(weight.code)
		if rank != index {
			return fmt.Errorf("evidenceWeights は強い根拠から固定順に並べなければなりません")
		}
		total += weight.weight
		if total > maximumQueryProfileScore {
			return fmt.Errorf("evidence weight の合計が上限を超えました")
		}
	}
	if total != p.maximum {
		return fmt.Errorf("maximum は全 evidence weight の合計と一致しなければなりません")
	}
	if p.mediumConfidenceAt < p.minimum ||
		p.highConfidenceAt < p.mediumConfidenceAt ||
		p.highConfidenceAt > p.maximum {
		return fmt.Errorf("confidence の境界が score 範囲内の昇順ではありません")
	}
	return nil
}
