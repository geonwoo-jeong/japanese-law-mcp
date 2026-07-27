package legalquery

import "fmt"

const (
	// MaxRankedCandidates は、一つの plan に保持できる候補数である。
	MaxRankedCandidates = 16
	// MaxSelectedCandidates は、一つの plan で選択できる候補数である。
	MaxSelectedCandidates = 2
	// MaxParallelCandidates は、同時に実行できる独立候補数である。
	MaxParallelCandidates = 2
	// MaxCapabilityCalls は、一つの plan で実行できる論理能力呼出し数である。
	MaxCapabilityCalls = 4
	// MaxItemsPerCollectionStep は、一つの collection step の公開上限である。
	MaxItemsPerCollectionStep = 20
	// MaxReturnedItems は、一つの統合照会が公開できる item 総数である。
	MaxReturnedItems = 40
	// FirstPageOnly は、統合照会内で継続取得しないことを表す。
	FirstPageOnly = true
)

// LegalQueryStepBudget は、一つの実行 step に確定した item 予算を表す。
type LegalQueryStepBudget struct {
	candidateID    string
	stepID         string
	reservedItems  int
	effectiveLimit *int
}

// CandidateID は、予算を割り当てた候補 ID を返す。
func (b LegalQueryStepBudget) CandidateID() string {
	return b.candidateID
}

// StepID は、予算を割り当てた step ID を返す。
func (b LegalQueryStepBudget) StepID() string {
	return b.stepID
}

// ReservedItems は、read step のために先に予約した item 数を返す。
func (b LegalQueryStepBudget) ReservedItems() int {
	return b.reservedItems
}

// EffectiveLimit は、collection step の確定上限と有無を返す。
func (b LegalQueryStepBudget) EffectiveLimit() (int, bool) {
	if b.effectiveLimit == nil {
		return 0, false
	}
	return *b.effectiveLimit, true
}

// Validate は、read または collection の排他的な予算構造を確認する。
func (b LegalQueryStepBudget) Validate() error {
	if err := validateQueryPlanID("candidateId", b.candidateID); err != nil {
		return err
	}
	if err := validateQueryPlanID("stepId", b.stepID); err != nil {
		return err
	}
	switch b.reservedItems {
	case 1:
		if b.effectiveLimit != nil {
			return fmt.Errorf("read step に effectiveLimit を指定できません")
		}
	case 0:
		if b.effectiveLimit == nil {
			return fmt.Errorf("collection step には effectiveLimit が必要です")
		}
		if *b.effectiveLimit < 1 ||
			*b.effectiveLimit > MaxItemsPerCollectionStep {
			return fmt.Errorf("effectiveLimit は 1 以上 20 以下でなければなりません")
		}
	default:
		return fmt.Errorf("reservedItems は read の 1 または collection の 0 でなければなりません")
	}
	return nil
}

// UnmarshalJSON は、plan を介さない直接復元を拒否する。
func (*LegalQueryStepBudget) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryStepBudget は JSON から直接復元できません",
	)
}

func (b LegalQueryStepBudget) clone() LegalQueryStepBudget {
	cloned := b
	if b.effectiveLimit != nil {
		effectiveLimit := *b.effectiveLimit
		cloned.effectiveLimit = &effectiveLimit
	}
	return cloned
}

// LegalQueryBudget は、一照会の固定上限と step ごとの確定値を保持する。
type LegalQueryBudget struct {
	maxRankedCandidates   int
	maxSelectedCandidates int
	maxParallelCandidates int
	maxCapabilityCalls    int
	maxItemsPerCollection int
	maxReturnedItems      int
	firstPageOnly         bool
	limitPerAttempt       int
	readStepCount         int
	collectionStepCount   int
	stepBudgets           []LegalQueryStepBudget
}

// MaxRankedCandidates は、plan が保持できる候補数を返す。
func (b LegalQueryBudget) MaxRankedCandidates() int {
	return b.maxRankedCandidates
}

// MaxSelectedCandidates は、選択できる候補数を返す。
func (b LegalQueryBudget) MaxSelectedCandidates() int {
	return b.maxSelectedCandidates
}

// MaxParallelCandidates は、同時実行できる候補数を返す。
func (b LegalQueryBudget) MaxParallelCandidates() int {
	return b.maxParallelCandidates
}

// MaxCapabilityCalls は、論理能力呼出し数の上限を返す。
func (b LegalQueryBudget) MaxCapabilityCalls() int {
	return b.maxCapabilityCalls
}

// MaxItemsPerCollectionStep は、一 collection step の公開上限を返す。
func (b LegalQueryBudget) MaxItemsPerCollectionStep() int {
	return b.maxItemsPerCollection
}

// MaxReturnedItems は、一照会の公開 item 総数上限を返す。
func (b LegalQueryBudget) MaxReturnedItems() int {
	return b.maxReturnedItems
}

// FirstPageOnly は、継続取得を行わない固定値を返す。
func (b LegalQueryBudget) FirstPageOnly() bool {
	return b.firstPageOnly
}

// LimitPerAttempt は、request で既定値適用後の希望上限を返す。
func (b LegalQueryBudget) LimitPerAttempt() int {
	return b.limitPerAttempt
}

// ReadStepCount は、実行予定の read step 数を返す。
func (b LegalQueryBudget) ReadStepCount() int {
	return b.readStepCount
}

// CollectionStepCount は、実行予定の collection step 数を返す。
func (b LegalQueryBudget) CollectionStepCount() int {
	return b.collectionStepCount
}

// StepBudgets は、計画順の step 予算の複製を返す。
func (b LegalQueryBudget) StepBudgets() []LegalQueryStepBudget {
	cloned := make([]LegalQueryStepBudget, 0, len(b.stepBudgets))
	for _, value := range b.stepBudgets {
		cloned = append(cloned, value.clone())
	}
	return cloned
}

// Validate は、固定上限、R/C と全 step の確定値を確認する。
func (b LegalQueryBudget) Validate() error {
	if !b.hasFixedLimits() {
		return fmt.Errorf("固定予算が SOT-MODEL-023 と一致しません")
	}
	if b.limitPerAttempt < 1 || b.limitPerAttempt > MaxLimitPerAttempt {
		return fmt.Errorf("limitPerAttempt は 1 以上 20 以下でなければなりません")
	}
	if b.readStepCount < 0 || b.collectionStepCount < 0 ||
		b.readStepCount+b.collectionStepCount > MaxCapabilityCalls {
		return fmt.Errorf("read と collection の step 総数は 0 以上 4 以下でなければなりません")
	}
	if len(b.stepBudgets) != b.readStepCount+b.collectionStepCount {
		return fmt.Errorf("stepBudgets の件数が read と collection の合計と一致しません")
	}
	return b.validateSteps()
}

// UnmarshalJSON は、plan を介さない直接復元を拒否する。
func (*LegalQueryBudget) UnmarshalJSON(_ []byte) error {
	return fmt.Errorf(
		"LegalQueryBudget は JSON から直接復元できません",
	)
}

func newLegalQueryBudget(
	decision PlanDecision,
	limitPerAttempt int,
	rankedCandidates []LegalQueryCandidate,
	selected []LegalQueryPlanSelection,
) (LegalQueryBudget, error) {
	budget := fixedLegalQueryBudget(limitPerAttempt)
	if !isExecutingDecision(decision) {
		if err := budget.Validate(); err != nil {
			return LegalQueryBudget{}, err
		}
		return budget, nil
	}
	references := make(map[string]LegalQueryCandidate, len(rankedCandidates))
	for _, candidate := range rankedCandidates {
		references[candidate.CandidateID()] = candidate
	}
	steps, err := flattenSelectedSteps(selected, references)
	if err != nil {
		return LegalQueryBudget{}, err
	}
	for _, value := range steps {
		switch value.step.Task() {
		case TaskRead:
			budget.readStepCount++
		case TaskSearch, TaskListUpdates:
			budget.collectionStepCount++
		default:
			return LegalQueryBudget{}, fmt.Errorf("予算を割り当てられない task です")
		}
	}
	effectiveLimit, hasLimit, err := allocateEffectiveLimit(
		limitPerAttempt,
		budget.readStepCount,
		budget.collectionStepCount,
	)
	if err != nil {
		return LegalQueryBudget{}, err
	}
	budget.stepBudgets = make(
		[]LegalQueryStepBudget,
		0,
		len(steps),
	)
	for _, value := range steps {
		budget.stepBudgets = append(
			budget.stepBudgets,
			newStepBudget(value, effectiveLimit, hasLimit),
		)
	}
	if err := budget.Validate(); err != nil {
		return LegalQueryBudget{}, err
	}
	return budget, nil
}

type selectedStep struct {
	candidateID string
	step        LegalQueryCandidateStep
}

func flattenSelectedSteps(
	selected []LegalQueryPlanSelection,
	references map[string]LegalQueryCandidate,
) ([]selectedStep, error) {
	values := make([]selectedStep, 0, MaxCapabilityCalls)
	for _, selection := range selected {
		candidate, exists := references[selection.CandidateID()]
		if !exists {
			return nil, fmt.Errorf("selected は rankedCandidates の候補だけを参照できます")
		}
		for _, step := range candidate.Steps() {
			values = append(values, selectedStep{
				candidateID: selection.CandidateID(),
				step:        step,
			})
			if len(values) > MaxCapabilityCalls {
				return nil, fmt.Errorf("論理 capability 呼出しは四回以下でなければなりません")
			}
		}
	}
	return values, nil
}

func newStepBudget(
	value selectedStep,
	effectiveLimit int,
	hasLimit bool,
) LegalQueryStepBudget {
	budget := LegalQueryStepBudget{
		candidateID: value.candidateID,
		stepID:      value.step.StepID(),
	}
	if value.step.Task() == TaskRead {
		budget.reservedItems = 1
		return budget
	}
	if hasLimit {
		limit := effectiveLimit
		budget.effectiveLimit = &limit
	}
	return budget
}

func fixedLegalQueryBudget(limitPerAttempt int) LegalQueryBudget {
	return LegalQueryBudget{
		maxRankedCandidates:   MaxRankedCandidates,
		maxSelectedCandidates: MaxSelectedCandidates,
		maxParallelCandidates: MaxParallelCandidates,
		maxCapabilityCalls:    MaxCapabilityCalls,
		maxItemsPerCollection: MaxItemsPerCollectionStep,
		maxReturnedItems:      MaxReturnedItems,
		firstPageOnly:         FirstPageOnly,
		limitPerAttempt:       limitPerAttempt,
		stepBudgets:           []LegalQueryStepBudget{},
	}
}

func allocateEffectiveLimit(
	limitPerAttempt int,
	readSteps int,
	collectionSteps int,
) (int, bool, error) {
	if limitPerAttempt < 1 || limitPerAttempt > MaxLimitPerAttempt {
		return 0, false, fmt.Errorf("limitPerAttempt は 1 以上 20 以下でなければなりません")
	}
	if readSteps < 0 || collectionSteps < 0 ||
		readSteps+collectionSteps > MaxCapabilityCalls {
		return 0, false, fmt.Errorf("read と collection の step 総数は 0 以上 4 以下でなければなりません")
	}
	if collectionSteps == 0 {
		return 0, false, nil
	}
	remainingItems := MaxReturnedItems - readSteps
	effectiveLimit := min(
		limitPerAttempt,
		remainingItems/collectionSteps,
		MaxItemsPerCollectionStep,
	)
	if effectiveLimit < 1 {
		return 0, false, fmt.Errorf("effectiveLimit を一件以上確保できません")
	}
	return effectiveLimit, true, nil
}

func (b LegalQueryBudget) validateSteps() error {
	expectedLimit, hasLimit, err := allocateEffectiveLimit(
		b.limitPerAttempt,
		b.readStepCount,
		b.collectionStepCount,
	)
	if err != nil {
		return err
	}
	readSteps := 0
	collectionSteps := 0
	stepIDs := make(map[string]struct{}, len(b.stepBudgets))
	for index, stepBudget := range b.stepBudgets {
		if err := stepBudget.Validate(); err != nil {
			return fmt.Errorf("stepBudgets[%d] が有効ではありません: %w", index, err)
		}
		if _, exists := stepIDs[stepBudget.StepID()]; exists {
			return fmt.Errorf("stepBudgets の stepId を重複させることはできません")
		}
		stepIDs[stepBudget.StepID()] = struct{}{}
		if stepBudget.ReservedItems() == 1 {
			readSteps++
			continue
		}
		collectionSteps++
		limit, exists := stepBudget.EffectiveLimit()
		if !hasLimit || !exists || limit != expectedLimit {
			return fmt.Errorf("collection step の effectiveLimit が固定配分と一致しません")
		}
	}
	if readSteps != b.readStepCount ||
		collectionSteps != b.collectionStepCount {
		return fmt.Errorf("stepBudgets の種別件数が R/C と一致しません")
	}
	return nil
}

func (b LegalQueryBudget) hasFixedLimits() bool {
	return b.maxRankedCandidates == MaxRankedCandidates &&
		b.maxSelectedCandidates == MaxSelectedCandidates &&
		b.maxParallelCandidates == MaxParallelCandidates &&
		b.maxCapabilityCalls == MaxCapabilityCalls &&
		b.maxItemsPerCollection == MaxItemsPerCollectionStep &&
		b.maxReturnedItems == MaxReturnedItems &&
		b.firstPageOnly
}

func (b LegalQueryBudget) clone() LegalQueryBudget {
	cloned := b
	cloned.stepBudgets = b.StepBudgets()
	return cloned
}

func (b LegalQueryBudget) equal(other LegalQueryBudget) bool {
	if b.maxRankedCandidates != other.maxRankedCandidates ||
		b.maxSelectedCandidates != other.maxSelectedCandidates ||
		b.maxParallelCandidates != other.maxParallelCandidates ||
		b.maxCapabilityCalls != other.maxCapabilityCalls ||
		b.maxItemsPerCollection != other.maxItemsPerCollection ||
		b.maxReturnedItems != other.maxReturnedItems ||
		b.firstPageOnly != other.firstPageOnly ||
		b.limitPerAttempt != other.limitPerAttempt ||
		b.readStepCount != other.readStepCount ||
		b.collectionStepCount != other.collectionStepCount ||
		len(b.stepBudgets) != len(other.stepBudgets) {
		return false
	}
	for index := range b.stepBudgets {
		if !b.stepBudgets[index].equal(other.stepBudgets[index]) {
			return false
		}
	}
	return true
}

func (b LegalQueryStepBudget) equal(other LegalQueryStepBudget) bool {
	if b.candidateID != other.candidateID ||
		b.stepID != other.stepID ||
		b.reservedItems != other.reservedItems {
		return false
	}
	left, leftExists := b.EffectiveLimit()
	right, rightExists := other.EffectiveLimit()
	return leftExists == rightExists && (!leftExists || left == right)
}

func isExecutingDecision(value PlanDecision) bool {
	return value == PlanDecisionSingle || value == PlanDecisionHedged
}
