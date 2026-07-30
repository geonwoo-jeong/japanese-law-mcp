package legalquery

import (
	"fmt"
	"slices"
	"sort"
)

// QueryProfileSet は、共通の順位校正を持つ query profile の固定集合である。
type QueryProfileSet struct {
	profiles       []QueryProfile
	metadata       []QueryProfileMetadata
	composer       CandidateComposer
	profileVersion string
	rankingVersion string
	selection      QuerySelectionPolicy
}

// QueryProfileSetResult は、全 profile の contribution を安定集計した結果である。
type QueryProfileSetResult struct {
	profileVersion        string
	rankingVersion        string
	rankedCandidates      []LegalQueryCandidate
	signals               []CandidateGenerationSignal
	selectionMode         QuerySelectionMode
	hedgePairs            []CandidateHedgePair
	compositionConstraint QueryCompositionConstraint
	selection             QuerySelectionPolicy
}

// ProfileVersion は、active profile set 全体の不透明な版を返す。
func (s QueryProfileSet) ProfileVersion() string {
	return s.profileVersion
}

// RankingVersion は、profile set 全体が共有する順位校正版を返す。
func (s QueryProfileSet) RankingVersion() string {
	return s.rankingVersion
}

// NewQueryProfileSet は、同じ ranking 校正を持つ profile を固定順で保持する。
func NewQueryProfileSet(profiles []QueryProfile) (QueryProfileSet, error) {
	if len(profiles) == 0 || len(profiles) > maximumProfileOrdinal {
		return QueryProfileSet{}, fmt.Errorf(
			"profiles は一件以上 %d 件以下でなければなりません",
			maximumProfileOrdinal,
		)
	}
	metadata := make([]QueryProfileMetadata, 0, len(profiles))
	seenProfileIDs := make(map[string]struct{}, len(profiles))
	var calibration string
	for index, profile := range profiles {
		value, err := validateProfileSetMember(
			profile,
			index,
			seenProfileIDs,
		)
		if err != nil {
			return QueryProfileSet{}, err
		}
		currentCalibration := rankingCalibrationSignature(value)
		if index > 0 && currentCalibration != calibration {
			return QueryProfileSet{}, fmt.Errorf(
				"profiles は同じ rankingVersion と校正規則を必要とします",
			)
		}
		if index == 0 {
			calibration = currentCalibration
		}
		metadata = append(metadata, value)
	}
	composer, err := NewCandidateComposer(
		defaultCandidateCompositionVersion,
	)
	if err != nil {
		return QueryProfileSet{}, fmt.Errorf(
			"candidate composer を初期化できません: %w",
			err,
		)
	}
	return QueryProfileSet{
		profiles: append([]QueryProfile(nil), profiles...),
		metadata: append([]QueryProfileMetadata(nil), metadata...),
		composer: composer,
		profileVersion: queryProfileSetVersion(
			metadata,
			composer.Version(),
		),
		rankingVersion: metadata[0].RankingVersion(),
		selection:      metadata[0].Selection(),
	}, nil
}

// Validate は、構築済みの profile、metadata および集合版が現在も一致することを確認する。
func (s QueryProfileSet) Validate() error {
	rebuilt, err := NewQueryProfileSet(s.profiles)
	if err != nil {
		return fmt.Errorf("profile set の構成が有効ではありません: %w", err)
	}
	if len(s.metadata) != len(rebuilt.metadata) {
		return fmt.Errorf("profile set の profiles と metadata の件数が一致しません")
	}
	for index := range s.metadata {
		if queryProfileMetadataSignature(s.metadata[index]) !=
			queryProfileMetadataSignature(rebuilt.metadata[index]) {
			return fmt.Errorf(
				"profiles[%d] の metadata が構築時と一致しません",
				index,
			)
		}
	}
	if s.profileVersion != rebuilt.profileVersion {
		return fmt.Errorf("profile set version が構築時の metadata と一致しません")
	}
	if err := s.composer.Validate(); err != nil {
		return fmt.Errorf("candidate composer が有効ではありません: %w", err)
	}
	if s.composer.Version() != rebuilt.composer.Version() {
		return fmt.Errorf("compositionVersion が構築時と一致しません")
	}
	if s.rankingVersion != rebuilt.rankingVersion {
		return fmt.Errorf("rankingVersion が構築時の metadata と一致しません")
	}
	if s.selection != rebuilt.selection {
		return fmt.Errorf("selection policy が構築時の metadata と一致しません")
	}
	return nil
}

// Collect は、全 contribution を部分結果なしで回収し、安定順位を確定する。
func (s QueryProfileSet) Collect(
	preprocessed PreprocessResult,
) (QueryProfileSetResult, error) {
	if err := s.Validate(); err != nil {
		return QueryProfileSetResult{}, fmt.Errorf(
			"profile set が有効ではありません: %w",
			err,
		)
	}
	if err := preprocessed.Validate(); err != nil {
		return QueryProfileSetResult{}, fmt.Errorf(
			"preprocessed が有効ではありません: %w",
			err,
		)
	}
	aggregate := newProfileSetAggregate()
	for index, profile := range s.profiles {
		currentMetadata := profile.Metadata()
		if err := currentMetadata.Validate(); err != nil {
			return QueryProfileSetResult{}, fmt.Errorf(
				"profiles[%d] の metadata が変更後に無効です: %w",
				index,
				err,
			)
		}
		if queryProfileMetadataSignature(currentMetadata) !=
			queryProfileMetadataSignature(s.metadata[index]) {
			return QueryProfileSetResult{}, fmt.Errorf(
				"profiles[%d] の metadata が構築後に変更されました",
				index,
			)
		}
		scope, err := NewCandidateIDScope(index + 1)
		if err != nil {
			return QueryProfileSetResult{}, err
		}
		contribution, err := collectProfileCandidatesForMetadata(
			profile,
			currentMetadata,
			preprocessed,
			scope,
		)
		if err != nil {
			return QueryProfileSetResult{}, fmt.Errorf(
				"profiles[%d] の contribution を回収できません: %w",
				index,
				err,
			)
		}
		if err := aggregate.add(
			contribution,
			currentMetadata.Score(),
			index+1,
		); err != nil {
			return QueryProfileSetResult{}, fmt.Errorf(
				"profiles[%d] の contribution を集約できません: %w",
				index,
				err,
			)
		}
	}
	composerScope, err := NewCandidateIDScope(len(s.profiles) + 1)
	if err != nil {
		return QueryProfileSetResult{}, err
	}
	if err := aggregate.compose(
		preprocessed.Query(),
		s.composer,
		s.metadata[0].Score(),
		composerScope,
	); err != nil {
		return QueryProfileSetResult{}, fmt.Errorf(
			"profile contributions を合成できません: %w",
			err,
		)
	}
	return aggregate.result(
		s.profileVersion,
		s.rankingVersion,
		s.selection,
	)
}

// ProfileVersion は、active profile set 全体の不透明な版を返す。
func (r QueryProfileSetResult) ProfileVersion() string {
	return r.profileVersion
}

// RankingVersion は、候補 score の共通校正版を返す。
func (r QueryProfileSetResult) RankingVersion() string {
	return r.rankingVersion
}

// RankedCandidates は、安定した意味順位の深い複製を返す。
func (r QueryProfileSetResult) RankedCandidates() []LegalQueryCandidate {
	values, err := cloneLegalQueryCandidates(r.rankedCandidates)
	if err != nil {
		panic(fmt.Sprintf("検証済み profile set result を複製できません: %v", err))
	}
	return values
}

// Signals は、固定順で集約した安全信号の複製を返す。
func (r QueryProfileSetResult) Signals() []CandidateGenerationSignal {
	return append([]CandidateGenerationSignal(nil), r.signals...)
}

// SelectionMode は、全 contribution を通した自動選択の可否を返す。
func (r QueryProfileSetResult) SelectionMode() QuerySelectionMode {
	return r.selectionMode
}

// HedgePairs は、各 profile が明示した独立候補対の複製を返す。
func (r QueryProfileSetResult) HedgePairs() []CandidateHedgePair {
	return append([]CandidateHedgePair(nil), r.hedgePairs...)
}

// CompositionConstraint は、profile または候補合成で検出した非実行制約を返す。
func (r QueryProfileSetResult) CompositionConstraint() QueryCompositionConstraint {
	return r.compositionConstraint
}

func (r QueryProfileSetResult) selectionPolicy() QuerySelectionPolicy {
	return r.selection
}

func validateProfileSetMember(
	profile QueryProfile,
	index int,
	seen map[string]struct{},
) (QueryProfileMetadata, error) {
	if isNilInterfaceValue(profile) {
		return QueryProfileMetadata{}, fmt.Errorf("profiles[%d] は必須です", index)
	}
	metadata := profile.Metadata()
	if err := metadata.Validate(); err != nil {
		return QueryProfileMetadata{}, fmt.Errorf(
			"profiles[%d] の metadata が有効ではありません: %w",
			index,
			err,
		)
	}
	if _, exists := seen[metadata.ProfileID()]; exists {
		return QueryProfileMetadata{}, fmt.Errorf(
			"profileId %q を重複させることはできません",
			metadata.ProfileID(),
		)
	}
	seen[metadata.ProfileID()] = struct{}{}
	return metadata, nil
}

type profileSetAggregate struct {
	candidates            []LegalQueryCandidate
	signals               map[CandidateGenerationSignal]struct{}
	selectionMode         QuerySelectionMode
	hedgePairs            []CandidateHedgePair
	contributions         []candidateCompositionContribution
	compositionConstraint QueryCompositionConstraint
	candidateIDs          map[string]struct{}
	stepIDs               map[string]struct{}
	meanings              map[string]struct{}
}

func newProfileSetAggregate() profileSetAggregate {
	return profileSetAggregate{
		signals:               make(map[CandidateGenerationSignal]struct{}),
		selectionMode:         QuerySelectionModeAutomatic,
		compositionConstraint: QueryCompositionConstraintNone,
		candidateIDs:          make(map[string]struct{}),
		stepIDs:               make(map[string]struct{}),
		meanings:              make(map[string]struct{}),
	}
}

func (a *profileSetAggregate) add(
	contribution QueryProfileContribution,
	score QueryScorePolicy,
	profileOrdinal int,
) error {
	candidates := contribution.Candidates()
	if err := validateContributionCandidateOrder(candidates, score); err != nil {
		return err
	}
	for _, candidate := range candidates {
		if err := a.addCandidate(candidate); err != nil {
			return err
		}
	}
	for _, signal := range contribution.Signals() {
		a.signals[signal] = struct{}{}
	}
	if contribution.SelectionMode() ==
		QuerySelectionModeClarificationRequired {
		a.selectionMode = QuerySelectionModeClarificationRequired
	}
	if contribution.CompositionConstraint() ==
		QueryCompositionConstraintStepLimitExceeded {
		a.compositionConstraint =
			QueryCompositionConstraintStepLimitExceeded
	}
	a.hedgePairs = append(a.hedgePairs, contribution.HedgePairs()...)
	a.contributions = append(
		a.contributions,
		candidateCompositionContribution{
			profileOrdinal: profileOrdinal,
			candidates:     candidates,
			members:        contribution.CompositionMembers(),
			hedgePairs:     contribution.HedgePairs(),
			selectionMode:  contribution.SelectionMode(),
		},
	)
	return nil
}

func (a *profileSetAggregate) compose(
	query string,
	composer CandidateComposer,
	score QueryScorePolicy,
	scope CandidateIDScope,
) error {
	composed, err := composer.compose(
		query,
		a.contributions,
		a.candidates,
		a.hedgePairs,
		score,
		scope,
	)
	if err != nil {
		return err
	}
	a.candidates = composed.candidates
	a.hedgePairs = composed.hedgePairs
	switch {
	case composed.constraint ==
		QueryCompositionConstraintStepLimitExceeded:
		a.compositionConstraint =
			QueryCompositionConstraintStepLimitExceeded
	case composed.constraint == QueryCompositionConstraintIneligible &&
		a.compositionConstraint == QueryCompositionConstraintNone:
		a.compositionConstraint = QueryCompositionConstraintIneligible
	}
	return a.rebuildCandidateIndexes()
}

func (a *profileSetAggregate) rebuildCandidateIndexes() error {
	a.candidateIDs = make(map[string]struct{}, len(a.candidates))
	a.stepIDs = make(map[string]struct{})
	a.meanings = make(map[string]struct{}, len(a.candidates))
	candidates := append([]LegalQueryCandidate(nil), a.candidates...)
	a.candidates = nil
	for _, candidate := range candidates {
		if err := a.addCandidate(candidate); err != nil {
			return err
		}
	}
	return nil
}

func (a *profileSetAggregate) addCandidate(
	candidate LegalQueryCandidate,
) error {
	if _, exists := a.candidateIDs[candidate.CandidateID()]; exists {
		return fmt.Errorf("profile 間で candidateId が重複しています")
	}
	for _, step := range candidate.Steps() {
		if _, exists := a.stepIDs[step.StepID()]; exists {
			return fmt.Errorf("profile 間で stepId が重複しています")
		}
	}
	signature, err := legalQueryCandidateMeaningSignature(candidate)
	if err != nil {
		return err
	}
	if _, exists := a.meanings[signature]; exists {
		return fmt.Errorf("profile 間で候補の意味署名が重複しています")
	}
	a.candidateIDs[candidate.CandidateID()] = struct{}{}
	for _, step := range candidate.Steps() {
		a.stepIDs[step.StepID()] = struct{}{}
	}
	a.meanings[signature] = struct{}{}
	a.candidates = append(a.candidates, candidate)
	return nil
}

func (a profileSetAggregate) result(
	profileVersion string,
	rankingVersion string,
	selection QuerySelectionPolicy,
) (QueryProfileSetResult, error) {
	sort.SliceStable(a.candidates, func(left int, right int) bool {
		return a.candidates[left].SemanticScore() >
			a.candidates[right].SemanticScore()
	})
	candidates, err := cloneLegalQueryCandidates(a.candidates)
	if err != nil {
		return QueryProfileSetResult{}, err
	}
	result := QueryProfileSetResult{
		profileVersion:        profileVersion,
		rankingVersion:        rankingVersion,
		rankedCandidates:      candidates,
		signals:               orderedGenerationSignals(a.signals),
		selectionMode:         a.selectionMode,
		hedgePairs:            append([]CandidateHedgePair(nil), a.hedgePairs...),
		compositionConstraint: a.compositionConstraint,
		selection:             selection,
	}
	if err := result.validate(); err != nil {
		return QueryProfileSetResult{}, err
	}
	return result, nil
}

func validateContributionCandidateOrder(
	candidates []LegalQueryCandidate,
	score QueryScorePolicy,
) error {
	for index, candidate := range candidates {
		if index > 0 &&
			candidates[index-1].SemanticScore() <
				candidate.SemanticScore() {
			return fmt.Errorf("contribution candidates は score の非増加順が必要です")
		}
		expected, err := score.ConfidenceFor(candidate.SemanticScore())
		if err != nil {
			return err
		}
		if candidate.Confidence() != expected {
			return fmt.Errorf(
				"candidate %q の confidence が共通 score 規則と一致しません",
				candidate.CandidateID(),
			)
		}
	}
	return nil
}

func orderedGenerationSignals(
	values map[CandidateGenerationSignal]struct{},
) []CandidateGenerationSignal {
	result := make([]CandidateGenerationSignal, 0, len(values))
	for _, signal := range []CandidateGenerationSignal{
		CandidateSignalNonJapaneseQuery,
		CandidateSignalStandaloneStructuredQuery,
		CandidateSignalUnsupportedLegalAdvice,
		CandidateSignalUnsupportedTranslation,
		CandidateSignalUnsupportedTaskOrResource,
		CandidateSignalReservedPackRequest,
	} {
		if _, exists := values[signal]; exists {
			result = append(result, signal)
		}
	}
	return result
}

func (r QueryProfileSetResult) validate() error {
	if err := validateProfileVersion(r.profileVersion); err != nil {
		return fmt.Errorf("profileVersion が有効ではありません: %w", err)
	}
	if err := validateProfileVersion(r.rankingVersion); err != nil {
		return fmt.Errorf("rankingVersion が有効ではありません: %w", err)
	}
	if _, err := validateRankedCandidates(r.rankedCandidates); err != nil {
		return err
	}
	if !isQuerySelectionMode(r.selectionMode) {
		return fmt.Errorf("selectionMode が定義されていません")
	}
	if err := r.compositionConstraint.Validate(); err != nil {
		return fmt.Errorf("compositionConstraint が有効ではありません: %w", err)
	}
	if err := r.selection.Validate(); err != nil {
		return fmt.Errorf("selection policy が有効ではありません: %w", err)
	}
	if slices.Contains(
		r.signals,
		CandidateSignalStandaloneStructuredQuery,
	) {
		if len(r.rankedCandidates) != 0 {
			return fmt.Errorf("standalone structured query は候補を持てません")
		}
		if slices.Contains(
			r.signals,
			CandidateSignalNonJapaneseQuery,
		) {
			return fmt.Errorf("standalone structured query は non-Japanese signal と併存できません")
		}
		if len(r.signals) != 1 {
			return fmt.Errorf("standalone structured query は他の signal と併存できません")
		}
	}
	return validateAggregateHedgePairs(r.hedgePairs, r.rankedCandidates)
}

func validateAggregateHedgePairs(
	pairs []CandidateHedgePair,
	candidates []LegalQueryCandidate,
) error {
	references := make(map[string]LegalQueryCandidate, len(candidates))
	for _, candidate := range candidates {
		references[candidate.CandidateID()] = candidate
	}
	seen := make(map[string]struct{}, len(pairs))
	for index, pair := range pairs {
		if err := pair.Validate(); err != nil {
			return fmt.Errorf("hedgePairs[%d] が有効ではありません: %w", index, err)
		}
		first, firstExists := references[pair.FirstCandidateID()]
		second, secondExists := references[pair.SecondCandidateID()]
		if !firstExists || !secondExists {
			return fmt.Errorf("hedge pair は集約済み候補だけを参照できます")
		}
		key := pair.FirstCandidateID() + "\x00" + pair.SecondCandidateID()
		if _, exists := seen[key]; exists {
			return fmt.Errorf("hedge pair を重複させることはできません")
		}
		seen[key] = struct{}{}
		if len(first.Steps())+len(second.Steps()) > MaxCapabilityCalls {
			return fmt.Errorf("hedge pair の step は合計四件以下でなければなりません")
		}
	}
	return nil
}
