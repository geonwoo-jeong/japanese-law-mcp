package core

import (
	"fmt"
	"slices"
	"sort"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

const maximumGeneratedCandidates = 16

type candidateDraft struct {
	evidence                     map[legalquery.EvidenceCode]struct{}
	concepts                     []legalquery.LegalConceptSource
	requiredPacks                []string
	steps                        []stepDraft
	aliasRankings                []lawAliasRankingFact
	preserveOfficialAlias        bool
	preserveMorphologicalContext bool
	implicitResourceWeakGeneral  bool
	sharedTerminal               bool
}

type stepDraft struct {
	startByte        int
	topicOrdinal     int
	input            legalquery.LogicalInput
	evidenceBindings []profileevidence.EvidenceValues
}

// Generate は、位置付き前処理事実だけから法令コア候補を生成する。
func (p *Profile) Generate(
	input legalquery.CandidateGenerationInput,
	scope legalquery.CandidateIDScope,
) (legalquery.CandidateGeneration, error) {
	if p == nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf("core profile が初期化されていません")
	}
	if err := p.metadata.Validate(); err != nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf("profile metadata が有効ではありません: %w", err)
	}
	if err := input.Validate(); err != nil {
		return legalquery.CandidateGeneration{}, fmt.Errorf("candidate generation input が有効ではありません: %w", err)
	}
	if input.StandaloneStructuredQuery() {
		return p.newGeneration(
			nil,
			[]legalquery.CandidateGenerationSignal{
				legalquery.CandidateSignalStandaloneStructuredQuery,
			},
			legalquery.QuerySelectionModeAutomatic,
			nil,
			nil,
			legalquery.QueryCompositionConstraintNone,
		)
	}
	rawCues, err := p.resolveCues(input.CueMentions())
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	cues := rawCues
	var relationV2ContentDrafts []candidateDraft
	if p.intentEvidenceMode == cueIntentEvidenceRelationV2 ||
		p.intentEvidenceMode == cueIntentEvidenceCore {
		cues, err = p.resolveRelationV2Cues(input, cues)
		if err != nil {
			return legalquery.CandidateGeneration{}, err
		}
		if p.intentEvidenceMode == cueIntentEvidenceRelationV2 {
			relationV2ContentDrafts, err =
				p.buildRelationV2MentionTargetDrafts(input, cues)
			if err != nil {
				return legalquery.CandidateGeneration{}, err
			}
		}
	}
	if err := p.validateConceptMentions(input.LegalConceptMentions()); err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	signals := p.generationSignals(input, cues)
	if input.Language() == legalquery.QueryLanguageNonJapanese {
		return p.newGeneration(
			nil,
			signals,
			legalquery.QuerySelectionModeAutomatic,
			nil,
			nil,
			legalquery.QueryCompositionConstraintNone,
		)
	}
	if hasTooManySeparatedSubjects(input, cues) {
		return p.newGeneration(
			nil,
			signals,
			legalquery.QuerySelectionModeClarificationRequired,
			nil,
			nil,
			legalquery.QueryCompositionConstraintStepLimitExceeded,
		)
	}
	if p.intentEvidenceMode == cueIntentEvidenceCore {
		return p.generateCoreEvidence(
			input,
			cues,
			relationV2ContentDrafts,
			signals,
			scope,
		)
	}

	drafts, err := p.generateDrafts(input, cues, relationV2ContentDrafts)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	drafts = withAsOfEvidence(drafts)
	if p.intentEvidenceMode == cueIntentEvidenceRelationV2 {
		drafts = p.retainRelationV2SupportedDrafts(
			input,
			cues,
			drafts,
			signals,
		)
	} else {
		drafts = retainSupportedDraftsForUnsupportedRequest(
			drafts,
			cues,
			signals,
		)
	}
	candidates, stepStartBytes, err := p.materializeCandidates(
		input,
		cues,
		drafts,
		scope,
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	mode := p.selectionMode(input, cues, candidates)
	pairs, err := p.hedgePairs(input, cues, candidates, mode)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	members, err := buildCompositionMembers(
		candidates,
		stepStartBytes,
		mode,
	)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	return p.newGeneration(
		candidates,
		signals,
		mode,
		pairs,
		members,
		legalquery.QueryCompositionConstraintNone,
	)
}

func retainSupportedDraftsForUnsupportedRequest(
	drafts []candidateDraft,
	cues resolvedCues,
	signals []legalquery.CandidateGenerationSignal,
) []candidateDraft {
	if !hasUnsupportedSignal(signals) {
		return drafts
	}
	result := make([]candidateDraft, 0, len(drafts))
	for _, draft := range drafts {
		if !hasDraftEvidence(draft, legalquery.EvidenceExplicitTask) &&
			!hasDeterministicRetrievalEvidence(draft.evidence) {
			continue
		}
		if hasUnsupportedTaskOrResourceSignal(signals) &&
			(!hasGroundedRetrievalEvidence(draft.evidence) ||
				hasDraftEvidence(
					draft,
					legalquery.EvidenceUniqueTypoCorrection,
				) ||
				!hasTaskCueAtOrAfterDraft(draft, cues)) {
			continue
		}
		result = append(result, draft)
	}
	return result
}

func hasUnsupportedSignal(
	signals []legalquery.CandidateGenerationSignal,
) bool {
	for _, signal := range signals {
		switch signal {
		case legalquery.CandidateSignalUnsupportedLegalAdvice,
			legalquery.CandidateSignalUnsupportedTranslation,
			legalquery.CandidateSignalUnsupportedTaskOrResource:
			return true
		default:
		}
	}
	return false
}

func hasUnsupportedTaskOrResourceSignal(
	signals []legalquery.CandidateGenerationSignal,
) bool {
	for _, signal := range signals {
		if signal ==
			legalquery.CandidateSignalUnsupportedTaskOrResource {
			return true
		}
	}
	return false
}

func hasDraftEvidence(
	draft candidateDraft,
	code legalquery.EvidenceCode,
) bool {
	_, exists := draft.evidence[code]
	return exists
}

func hasDeterministicRetrievalEvidence(
	evidence map[legalquery.EvidenceCode]struct{},
) bool {
	for _, code := range []legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceStructuredReference,
	} {
		if _, exists := evidence[code]; exists {
			return true
		}
	}
	return false
}

func hasTaskCueAtOrAfterDraft(
	draft candidateDraft,
	cues resolvedCues,
) bool {
	if len(draft.steps) == 0 {
		return false
	}
	startByte := draft.steps[0].startByte
	for _, step := range draft.steps[1:] {
		if step.startByte < startByte {
			startByte = step.startByte
		}
	}
	for _, value := range []string{"search", "read", "list_updates"} {
		for _, mention := range cues.mentions[cueMeaningKey("task", value)] {
			if mention.Span().StartByte() >= startByte {
				return true
			}
		}
	}
	return false
}

func hasGroundedRetrievalEvidence(
	evidence map[legalquery.EvidenceCode]struct{},
) bool {
	grounded := [...]legalquery.EvidenceCode{
		legalquery.EvidenceOfficialIdentifier,
		legalquery.EvidenceStructuredReference,
		legalquery.EvidenceExplicitResource,
		legalquery.EvidenceOfficialAlias,
		legalquery.EvidenceLegalConcept,
	}
	for _, code := range grounded {
		if _, exists := evidence[code]; exists {
			return true
		}
	}
	return false
}

func (p *Profile) generateDrafts(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	additionalContent []candidateDraft,
) ([]candidateDraft, error) {
	targets := buildLawTargets(input, cues)
	if reservedPackOnlyRequest(input, cues) {
		return nil, nil
	}
	update, err := buildUpdateCandidate(input, cues)
	if err != nil {
		return nil, err
	}
	content, err := p.buildContentCandidates(input, cues, len(targets) > 0)
	if err != nil {
		return nil, err
	}
	if len(additionalContent) > 0 {
		content = p.withoutRelationV2MentionNounDrafts(
			input,
			content,
			additionalContent,
		)
	}
	for _, draft := range additionalContent {
		content = append(content, cloneDraft(draft))
	}
	if (cues.has("operator", "dual_candidate") ||
		isCoreResourceChoice(input, cues)) &&
		len(targets) == 0 {
		alternatives, alternativesErr :=
			buildLawResourceAlternativeDrafts(input, cues)
		if alternativesErr != nil {
			return nil, alternativesErr
		}
		if len(alternatives) > 0 {
			return mergeUpdateIntoDrafts(alternatives, update), nil
		}
	}
	explicitResources, handled, err :=
		buildExplicitLawAndContentSearchCandidate(
			cues,
			len(targets) > 0,
			content,
		)
	if err != nil {
		return nil, err
	}
	if handled {
		return mergeUpdateIntoDrafts(explicitResources, update), nil
	}

	explicitRead := cues.has("task", "read")
	explicitSearch := cues.has("task", "search")
	readRequested := explicitRead || len(input.ArticleMentions()) > 0
	searchRequested := explicitSearch || (!readRequested && len(targets) > 0)
	asOf := selectedAsOfDate(input, cues, update != nil)
	documentRequested := shouldReadDocument(input, cues, readRequested)

	read, err := buildReadCandidates(
		input,
		targets,
		readRequested,
		explicitRead,
		documentRequested,
		hasSingleTrailingReadTask(input, cues, targets),
		asOf,
	)
	if err != nil {
		return nil, err
	}
	searchTargets := targets
	groundedSearchBeforeRead := false
	if explicitSearch && explicitRead {
		if preceding := refPrecedingLawSearchTargets(input, cues); len(preceding) > 0 {
			searchTargets = preceding
			groundedSearchBeforeRead = true
		}
	}
	search, err := buildLawSearchCandidates(
		input,
		searchTargets,
		searchRequested,
		explicitSearch,
		documentRequested,
		separatesSubjects(cues),
		asOf,
	)
	if err != nil {
		return nil, err
	}
	refFollowup, handled, err := buildRefFollowupLawSearchDrafts(
		input,
		cues,
		read,
		content,
		update,
		asOf,
	)
	if err != nil {
		return nil, err
	}
	if handled {
		return refFollowup, nil
	}

	switch {
	case groundedSearchBeforeRead &&
		len(search) == 1 && len(read) > 0:
		searchAndRead := combineDraftSets(search, read, nil)
		return combineDraftSets(searchAndRead, content, update), nil
	case len(read) > 0:
		return combineDraftSets(read, content, update), nil
	case len(search) > 0:
		return combineDraftSets(search, content, update), nil
	case len(content) > 0:
		return mergeUpdateIntoDrafts(content, update), nil
	case update != nil:
		return []candidateDraft{*update}, nil
	default:
		return nil, nil
	}
}

func reservedPackOnlyRequest(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	if !cues.has("reserved_pack", "judicial-cases") {
		if ref, exists := input.Ref(); !exists ||
			ref.Key().ResourceType() != "judicial-decision" {
			return false
		}
	}
	if len(input.LawNameMentions()) > 0 ||
		len(input.IdentifierMentions()) > 0 ||
		len(input.ArticleMentions()) > 0 ||
		cues.has("resource", "law") ||
		cues.has("resource", "law_provision") ||
		cues.has("resource", "updates") {
		return false
	}
	for _, term := range input.QueryTermMentions() {
		if term.Surface() == "参照" {
			return false
		}
	}
	return true
}

func combineDraftSets(
	primary []candidateDraft,
	content []candidateDraft,
	update *candidateDraft,
) []candidateDraft {
	if len(content) == 0 {
		return mergeUpdateIntoDrafts(primary, update)
	}
	combined := make([]candidateDraft, 0, len(primary)*len(content))
	for _, left := range primary {
		for _, right := range content {
			current := cloneDraft(left)
			mergeDraft(&current, right)
			if update != nil {
				mergeDraft(&current, *update)
			}
			combined = append(combined, current)
		}
	}
	return combined
}

func mergeUpdateIntoDrafts(
	values []candidateDraft,
	update *candidateDraft,
) []candidateDraft {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		current := cloneDraft(value)
		if update != nil {
			mergeDraft(&current, *update)
		}
		result = append(result, current)
	}
	return result
}

func mergeDraft(target *candidateDraft, source candidateDraft) {
	target.preserveOfficialAlias =
		target.preserveOfficialAlias ||
			source.preserveOfficialAlias ||
			(hasDraftEvidence(
				*target,
				legalquery.EvidenceOfficialIdentifier,
			) &&
				hasDraftEvidence(
					source,
					legalquery.EvidenceOfficialAlias,
				)) ||
			(hasDraftEvidence(
				*target,
				legalquery.EvidenceOfficialAlias,
			) &&
				hasDraftEvidence(
					source,
					legalquery.EvidenceOfficialIdentifier,
				))
	target.steps = append(target.steps, source.steps...)
	for code := range source.evidence {
		target.evidence[code] = struct{}{}
	}
	target.concepts = append(target.concepts, source.concepts...)
	target.requiredPacks = append(target.requiredPacks, source.requiredPacks...)
	target.aliasRankings = mergeLawAliasRankingFacts(
		target.aliasRankings,
		source.aliasRankings,
	)
	target.preserveMorphologicalContext =
		target.preserveMorphologicalContext ||
			source.preserveMorphologicalContext
	target.implicitResourceWeakGeneral =
		target.implicitResourceWeakGeneral ||
			source.implicitResourceWeakGeneral
}

func cloneDraft(value candidateDraft) candidateDraft {
	evidence := make(map[legalquery.EvidenceCode]struct{}, len(value.evidence))
	for code := range value.evidence {
		evidence[code] = struct{}{}
	}
	steps := append([]stepDraft(nil), value.steps...)
	for index := range steps {
		steps[index].evidenceBindings = append(
			[]profileevidence.EvidenceValues(nil),
			steps[index].evidenceBindings...,
		)
	}
	return candidateDraft{
		evidence:                     evidence,
		concepts:                     append([]legalquery.LegalConceptSource(nil), value.concepts...),
		requiredPacks:                append([]string(nil), value.requiredPacks...),
		steps:                        steps,
		aliasRankings:                append([]lawAliasRankingFact(nil), value.aliasRankings...),
		preserveOfficialAlias:        value.preserveOfficialAlias,
		preserveMorphologicalContext: value.preserveMorphologicalContext,
		implicitResourceWeakGeneral:  value.implicitResourceWeakGeneral,
		sharedTerminal:               value.sharedTerminal,
	}
}

func (p *Profile) materializeCandidates(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []candidateDraft,
	scope legalquery.CandidateIDScope,
) (
	[]legalquery.LegalQueryCandidate,
	[][]int,
	error,
) {
	aggregated := make([]aggregatedDraft, 0, len(drafts))
	meaningIndexes := make(map[string]int, len(drafts))
	for _, original := range drafts {
		draft := cloneDraft(original)
		sort.SliceStable(draft.steps, func(left, right int) bool {
			return draft.steps[left].startByte < draft.steps[right].startByte
		})
		if len(draft.steps) == 0 || len(draft.steps) > 4 {
			return nil, nil, fmt.Errorf(
				"一候補の logical step は一件以上四件以下でなければなりません",
			)
		}
		signature, err := draftMeaningSignature(draft)
		if err != nil {
			return nil, nil, err
		}
		if index, exists := meaningIndexes[signature]; exists {
			mergeEquivalentDraft(&aggregated[index].draft, draft)
			continue
		}
		meaningIndexes[signature] = len(aggregated)
		aggregated = append(aggregated, aggregatedDraft{
			draft:     draft,
			signature: signature,
		})
	}
	rankAliasCollisionOverflow := canRankAliasCollisionOverflow(
		input,
		cues,
		aggregated,
	)
	rankAliasCollisionGroupsBySource :=
		p.rankAliasCollisionGroupsBySource &&
			rankAliasCollisionOverflow &&
			hasMultipleLawAliasCollisionGroups(input, cues)
	if len(aggregated) > maximumGeneratedCandidates &&
		!rankAliasCollisionOverflow {
		return nil, nil, fmt.Errorf(
			"core profile の候補は %d 件以下でなければなりません",
			maximumGeneratedCandidates,
		)
	}

	prepared := make([]preparedDraft, 0, len(aggregated))
	for _, current := range aggregated {
		evidence := normalizeEvidence(
			current.draft.evidence,
			current.draft.preserveOfficialAlias,
			preservesLegalConceptForDistinctStep(current.draft),
			current.draft.preserveMorphologicalContext,
		)
		score, err := p.metadata.Score().Score(evidence)
		if err != nil {
			return nil, nil, err
		}
		confidence, err := p.metadata.Score().ConfidenceFor(score)
		if err != nil {
			return nil, nil, err
		}
		rankingSignature, err := draftMeaningRankingSignature(current.draft)
		if err != nil {
			return nil, nil, err
		}
		prepared = append(prepared, preparedDraft{
			draft:      current.draft,
			evidence:   evidence,
			score:      score,
			confidence: confidence,
			signature:  current.signature,
			rankingSignature: weakGeneralRankingSignature(
				current.draft,
				rankingSignature,
			),
		})
	}
	prepared = withLawAliasCollisionRankingSignatures(prepared)
	sort.SliceStable(prepared, func(left, right int) bool {
		return comparePreparedDrafts(
			prepared[left],
			prepared[right],
			rankAliasCollisionGroupsBySource,
		) < 0
	})
	if rankAliasCollisionOverflow {
		prepared = prepared[:maximumGeneratedCandidates]
	}

	candidates := make([]legalquery.LegalQueryCandidate, 0, len(prepared))
	stepStartBytes := make([][]int, 0, len(prepared))
	for index, current := range prepared {
		inputs := make([]legalquery.LogicalInput, 0, len(current.draft.steps))
		startBytes := make([]int, 0, len(current.draft.steps))
		for _, step := range current.draft.steps {
			inputs = append(inputs, step.input)
			startBytes = append(startBytes, step.startByte)
		}
		var concepts []legalquery.LegalConceptSource
		if slices.Contains(
			current.evidence,
			legalquery.EvidenceLegalConcept,
		) {
			concepts = uniqueConceptSources(current.draft.concepts)
		}
		packs := append([]string(nil), current.draft.requiredPacks...)
		slices.Sort(packs)
		packs = slices.Compact(packs)
		candidate, err := legalquery.AssembleLegalQueryCandidate(
			legalquery.CandidateAssemblyValues{
				IDScope:          scope,
				CandidateOrdinal: index + 1,
				SemanticScore:    current.score,
				Confidence:       current.confidence,
				EvidenceCodes:    current.evidence,
				ConceptSources:   concepts,
				RequiredPacks:    packs,
				LogicalInputs:    inputs,
			},
		)
		if err != nil {
			return nil, nil, err
		}
		candidates = append(candidates, candidate)
		stepStartBytes = append(stepStartBytes, startBytes)
	}
	return candidates, stepStartBytes, nil
}

func canRankAliasCollisionOverflow(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	drafts []aggregatedDraft,
) bool {
	if len(drafts) <= maximumGeneratedCandidates {
		return false
	}
	targets := buildLawTargets(input, cues)
	groups := groupLawTargets(targets)
	if len(groups) == 0 ||
		len(groups) > 4 ||
		len(drafts) != len(targets) {
		return false
	}
	hasCollision := false
	for _, group := range groups {
		if len(group) > 1 {
			hasCollision = true
		}
	}
	if !hasCollision {
		return false
	}
	for _, current := range drafts {
		if len(current.draft.steps) != 1 {
			return false
		}
		if _, exists := current.draft.evidence[legalquery.EvidenceOfficialAlias]; !exists {
			return false
		}
	}
	return true
}

type aggregatedDraft struct {
	draft     candidateDraft
	signature string
}

func mergeEquivalentDraft(target *candidateDraft, source candidateDraft) {
	for code := range source.evidence {
		target.evidence[code] = struct{}{}
	}
	target.concepts = append(target.concepts, source.concepts...)
	target.requiredPacks = append(target.requiredPacks, source.requiredPacks...)
	target.aliasRankings = mergeLawAliasRankingFacts(
		target.aliasRankings,
		source.aliasRankings,
	)
	target.preserveOfficialAlias =
		target.preserveOfficialAlias ||
			source.preserveOfficialAlias
	target.preserveMorphologicalContext =
		target.preserveMorphologicalContext ||
			source.preserveMorphologicalContext
	target.implicitResourceWeakGeneral =
		target.implicitResourceWeakGeneral ||
			source.implicitResourceWeakGeneral
	for index := range target.steps {
		if source.steps[index].startByte < target.steps[index].startByte {
			target.steps[index].startByte = source.steps[index].startByte
		}
	}
}

func (p *Profile) newGeneration(
	candidates []legalquery.LegalQueryCandidate,
	signals []legalquery.CandidateGenerationSignal,
	mode legalquery.QuerySelectionMode,
	pairs []legalquery.CandidateHedgePair,
	members []legalquery.QueryCandidateCompositionMember,
	constraint legalquery.QueryCompositionConstraint,
) (legalquery.CandidateGeneration, error) {
	return legalquery.NewCandidateGeneration(legalquery.CandidateGenerationValues{
		ProfileID:             p.metadata.ProfileID(),
		ProfileVersion:        p.metadata.ProfileVersion(),
		RankingVersion:        p.metadata.RankingVersion(),
		Candidates:            candidates,
		Signals:               signals,
		SelectionMode:         mode,
		HedgePairs:            pairs,
		CompositionMembers:    members,
		CompositionConstraint: constraint,
	})
}
