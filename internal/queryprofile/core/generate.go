package core

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const maximumGeneratedCandidates = 16

type candidateDraft struct {
	evidence                     map[legalquery.EvidenceCode]struct{}
	concepts                     []legalquery.LegalConceptSource
	requiredPacks                []string
	steps                        []stepDraft
	aliasRankings                []lawAliasRankingFact
	preserveMorphologicalContext bool
	implicitResourceWeakGeneral  bool
}

type stepDraft struct {
	startByte int
	input     legalquery.LogicalInput
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
	cues, err := p.resolveCues(input.CueMentions())
	if err != nil {
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

	drafts, err := p.generateDrafts(input, cues)
	if err != nil {
		return legalquery.CandidateGeneration{}, err
	}
	drafts = withAsOfEvidence(drafts)
	drafts = retainGroundedDraftsForUnsupportedResource(drafts, signals)
	candidates, stepStartBytes, err := p.materializeCandidates(
		input,
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

func retainGroundedDraftsForUnsupportedResource(
	drafts []candidateDraft,
	signals []legalquery.CandidateGenerationSignal,
) []candidateDraft {
	if !hasUnsupportedTaskOrResourceSignal(signals) {
		return drafts
	}
	result := make([]candidateDraft, 0, len(drafts))
	for _, draft := range drafts {
		if hasGroundedRetrievalEvidence(draft.evidence) {
			result = append(result, draft)
		}
	}
	return result
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
) ([]candidateDraft, error) {
	targets := buildLawTargets(input)
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
		asOf,
	)
	if err != nil {
		return nil, err
	}
	search, err := buildLawSearchCandidates(
		input,
		targets,
		searchRequested,
		explicitSearch,
		documentRequested,
		separatesSubjects(cues),
		asOf,
	)
	if err != nil {
		return nil, err
	}

	switch {
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
	return candidateDraft{
		evidence:                     evidence,
		concepts:                     append([]legalquery.LegalConceptSource(nil), value.concepts...),
		requiredPacks:                append([]string(nil), value.requiredPacks...),
		steps:                        append([]stepDraft(nil), value.steps...),
		aliasRankings:                append([]lawAliasRankingFact(nil), value.aliasRankings...),
		preserveMorphologicalContext: value.preserveMorphologicalContext,
		implicitResourceWeakGeneral:  value.implicitResourceWeakGeneral,
	}
}

func (p *Profile) materializeCandidates(
	input legalquery.CandidateGenerationInput,
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
		aggregated,
	)
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
		prepared = append(prepared, preparedDraft{
			draft:      current.draft,
			evidence:   evidence,
			score:      score,
			confidence: confidence,
			signature:  current.signature,
			rankingSignature: weakGeneralRankingSignature(
				current.draft,
				current.signature,
			),
		})
	}
	prepared = withLawAliasCollisionRankingSignatures(prepared)
	sort.SliceStable(prepared, func(left, right int) bool {
		return comparePreparedDrafts(prepared[left], prepared[right]) < 0
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
	drafts []aggregatedDraft,
) bool {
	if len(drafts) <= maximumGeneratedCandidates {
		return false
	}
	targets := buildLawTargets(input)
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

type preparedDraft struct {
	draft            candidateDraft
	evidence         []legalquery.EvidenceCode
	score            int
	confidence       legalquery.Confidence
	signature        string
	rankingSignature string
}

func comparePreparedDrafts(left preparedDraft, right preparedDraft) int {
	if left.score != right.score {
		return right.score - left.score
	}
	leftEvidence := evidenceSignature(left.evidence)
	rightEvidence := evidenceSignature(right.evidence)
	if leftEvidence != rightEvidence {
		return strings.Compare(leftEvidence, rightEvidence)
	}
	if len(left.draft.steps) != len(right.draft.steps) {
		return len(left.draft.steps) - len(right.draft.steps)
	}
	if left.rankingSignature != right.rankingSignature {
		return strings.Compare(left.rankingSignature, right.rankingSignature)
	}
	return sourcePosition(left.draft) - sourcePosition(right.draft)
}

func normalizeEvidence(
	values map[legalquery.EvidenceCode]struct{},
	preserveLegalConcept bool,
	preserveMorphologicalContext bool,
) []legalquery.EvidenceCode {
	result := make(map[legalquery.EvidenceCode]struct{}, len(values))
	for code := range values {
		result[code] = struct{}{}
	}
	if _, exists := result[legalquery.EvidenceOfficialIdentifier]; exists {
		delete(result, legalquery.EvidenceOfficialAlias)
		if !preserveLegalConcept {
			delete(result, legalquery.EvidenceLegalConcept)
		}
		delete(result, legalquery.EvidenceMorphologicalContext)
		delete(result, legalquery.EvidenceUniqueTypoCorrection)
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceOfficialAlias]; exists {
		delete(result, legalquery.EvidenceMorphologicalContext)
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceLegalConcept]; exists {
		if !preserveMorphologicalContext {
			delete(result, legalquery.EvidenceMorphologicalContext)
		}
		delete(result, legalquery.EvidenceGeneralTerm)
	} else if _, exists := result[legalquery.EvidenceMorphologicalContext]; exists {
		delete(result, legalquery.EvidenceGeneralTerm)
	}
	ordered := make([]legalquery.EvidenceCode, 0, len(result))
	for _, code := range evidenceOrder {
		if _, exists := result[code]; exists {
			ordered = append(ordered, code)
		}
	}
	return ordered
}

func preservesLegalConceptForDistinctStep(draft candidateDraft) bool {
	if len(draft.concepts) == 0 || len(draft.steps) < 2 {
		return false
	}
	for _, step := range draft.steps {
		if step.input.InputKind() ==
			legalquery.InputKindLawContentSearch {
			return true
		}
	}
	return false
}

var evidenceOrder = []legalquery.EvidenceCode{
	legalquery.EvidenceOfficialIdentifier,
	legalquery.EvidenceStructuredReference,
	legalquery.EvidenceExplicitTask,
	legalquery.EvidenceExplicitResource,
	legalquery.EvidenceOfficialAlias,
	legalquery.EvidenceLegalConcept,
	legalquery.EvidenceMorphologicalContext,
	legalquery.EvidenceUniqueTypoCorrection,
	legalquery.EvidenceGeneralTerm,
}

func evidenceSignature(values []legalquery.EvidenceCode) string {
	parts := make([]string, 0, len(values))
	for _, value := range values {
		parts = append(parts, string(value))
	}
	return strings.Join(parts, "\x00")
}

func uniqueConceptSources(
	values []legalquery.LegalConceptSource,
) []legalquery.LegalConceptSource {
	byID := make(map[string]legalquery.LegalConceptSource, len(values))
	for _, value := range values {
		byID[value.ConceptID()] = value
	}
	ids := make([]string, 0, len(byID))
	for conceptID := range byID {
		ids = append(ids, conceptID)
	}
	slices.Sort(ids)
	result := make([]legalquery.LegalConceptSource, 0, len(ids))
	for _, conceptID := range ids {
		result = append(result, byID[conceptID])
	}
	return result
}

func sourcePosition(value candidateDraft) int {
	if len(value.steps) == 0 {
		return 0
	}
	position := value.steps[0].startByte
	for _, step := range value.steps[1:] {
		if step.startByte < position {
			position = step.startByte
		}
	}
	return position
}

func draftMeaningSignature(value candidateDraft) (string, error) {
	parts := make([]string, 0, len(value.steps)+1)
	packs := append([]string(nil), value.requiredPacks...)
	slices.Sort(packs)
	parts = append(parts, strings.Join(slices.Compact(packs), ","))
	for _, step := range value.steps {
		signature, err := logicalInputSignature(step.input)
		if err != nil {
			return "", err
		}
		parts = append(parts, signature)
	}
	return strings.Join(parts, "\x1f"), nil
}

func logicalInputSignature(value legalquery.LogicalInput) (string, error) {
	switch input := value.(type) {
	case legalquery.LawSearchIntentV1:
		return "01-law-search|" + input.Query() + "|" +
			optionalDateSignature(input.AsOf()), nil
	case legalquery.LawContentSearchIntentV1:
		return "02-law-content|" +
			strings.Join(input.AllTerms(), "\x00") + "|" +
			strings.Join(input.AnyTerms(), "\x00") + "|" +
			strings.Join(input.ExcludeTerms(), "\x00") + "|" +
			optionalDateSignature(input.AsOf()), nil
	case legalquery.LawReadIntentV1:
		if ref, exists := input.Ref(); exists {
			return "03-law-read|ref|" + resourceRefSignature(ref), nil
		}
		lawID, _ := input.LawID()
		revisionID, _ := input.RevisionID()
		return "03-law-read|" + lawID + "|" + revisionID + "|" +
			optionalDateSignature(input.AsOf()), nil
	case legalquery.LawArticleReadIntentV1:
		target := ""
		if ref, exists := input.Ref(); exists {
			target = resourceRefSignature(ref)
		} else {
			target, _ = input.LawID()
		}
		location := input.Location()
		paragraph, _ := location.ParagraphNumber()
		return "04-law-article|" + target + "|" +
			string(location.Provision()) + "|" +
			location.ArticleNumber() + "|" +
			strconv.Itoa(paragraph) + "|" +
			optionalDateSignature(input.AsOf()), nil
	case legalquery.LawUpdateListIntentV1:
		return "05-law-updates|" + input.Date().String(), nil
	default:
		return "", fmt.Errorf("core profile が未対応の logical input を生成しました")
	}
}

func optionalDateSignature(date model.Date, exists bool) string {
	if !exists {
		return ""
	}
	return date.String()
}

func resourceRefSignature(ref model.SourceResourceRef) string {
	key := ref.Key()
	version, _ := key.VersionID()
	return ref.ProviderID() + "|" +
		key.SourceID() + "|" +
		key.ResourceType() + "|" +
		key.ResourceID() + "|" +
		version
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
