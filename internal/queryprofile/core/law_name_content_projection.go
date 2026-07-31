package core

import (
	"fmt"
	"slices"
	"strings"
	"unicode"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/queryprofile/profileevidence"
)

type coreTopicOption struct {
	startByte  int
	input      legalquery.LawContentSearchIntentV1
	evidence   []profileevidence.EvidenceValues
	concepts   []legalquery.LegalConceptSource
	meaningKey string
}

// coreValidatedContentTopic は、共有末尾列の task/resource 検証を終えた
// 一つの topic span だけを法令名 projector へ渡すための非公開の証票である。
// SharedTerminalSequence 自体の検証と消費は後続段階が所有する。
type coreValidatedContentTopic struct {
	span                legalquery.QuerySpan
	baseEvidence        []profileevidence.EvidenceValues
	lawProvisionBinding coreLawProvisionBindingKind
}

type coreLawProvisionBindingKind uint8

const (
	// coreLawProvisionBindingExplicitResource は、位置付き resource cue を表す。
	coreLawProvisionBindingExplicitResource coreLawProvisionBindingKind = iota + 1
	// coreLawProvisionBindingTerminalTask は、SOT-ARCH-039 が許可する
	// 「教えて」系 terminal relation 自体による閉じた resource 束縛を表す。
	coreLawProvisionBindingTerminalTask
)

func newCoreValidatedContentTopic(
	span legalquery.QuerySpan,
	baseEvidence []profileevidence.EvidenceValues,
	lawProvisionBinding coreLawProvisionBindingKind,
) (coreValidatedContentTopic, error) {
	if err := span.Validate(); err != nil {
		return coreValidatedContentTopic{}, fmt.Errorf(
			"本文検索 topic span が有効ではありません: %w",
			err,
		)
	}
	taskCount := 0
	resourceCount := 0
	for _, value := range baseEvidence {
		if value.FactID == "" ||
			value.Layer != profileevidence.LayerExplicitTaskResource {
			return coreValidatedContentTopic{}, fmt.Errorf(
				"本文検索 topic の明示根拠が有効ではありません",
			)
		}
		switch value.Code {
		case legalquery.EvidenceExplicitTask:
			taskCount++
		case legalquery.EvidenceExplicitResource:
			resourceCount++
		default:
			return coreValidatedContentTopic{}, fmt.Errorf(
				"本文検索 topic に許可されていない明示根拠があります",
			)
		}
	}
	if taskCount != 1 {
		return coreValidatedContentTopic{}, fmt.Errorf(
			"本文検索 topic の task 根拠が一意ではありません",
		)
	}
	switch lawProvisionBinding {
	case coreLawProvisionBindingExplicitResource:
		if resourceCount != 1 {
			return coreValidatedContentTopic{}, fmt.Errorf(
				"本文検索 topic の明示 resource 根拠が一意ではありません",
			)
		}
	case coreLawProvisionBindingTerminalTask:
		if resourceCount != 0 {
			return coreValidatedContentTopic{}, fmt.Errorf(
				"terminal task 束縛に明示 resource 根拠を混在できません",
			)
		}
	default:
		return coreValidatedContentTopic{}, fmt.Errorf(
			"本文検索 topic の law provision 束縛が定義されていません",
		)
	}
	return coreValidatedContentTopic{
		span:                span,
		baseEvidence:        slices.Clone(baseEvidence),
		lawProvisionBinding: lawProvisionBinding,
	}, nil
}

type coreLawNameProjectionContext struct {
	lawNameSpan         legalquery.QuerySpan
	individualTopic     *legalquery.QuerySpan
	sharedTerminalTopic *coreValidatedContentTopic
}

func newCoreLawNameProjectionContext(
	lawNameSpan legalquery.QuerySpan,
	individualTopic *legalquery.QuerySpan,
	sharedTerminalTopic *coreValidatedContentTopic,
) (coreLawNameProjectionContext, error) {
	if err := lawNameSpan.Validate(); err != nil {
		return coreLawNameProjectionContext{}, fmt.Errorf(
			"法令名 span が有効ではありません: %w",
			err,
		)
	}
	result := coreLawNameProjectionContext{lawNameSpan: lawNameSpan}
	if individualTopic != nil {
		if err := individualTopic.Validate(); err != nil {
			return coreLawNameProjectionContext{}, fmt.Errorf(
				"個別主題 span が有効ではありません: %w",
				err,
			)
		}
		current := *individualTopic
		result.individualTopic = &current
	}
	if sharedTerminalTopic != nil {
		current, err := newCoreValidatedContentTopic(
			sharedTerminalTopic.span,
			sharedTerminalTopic.baseEvidence,
			sharedTerminalTopic.lawProvisionBinding,
		)
		if err != nil {
			return coreLawNameProjectionContext{}, fmt.Errorf(
				"共有末尾主題が有効ではありません: %w",
				err,
			)
		}
		result.sharedTerminalTopic = &current
	}
	return result, nil
}

// projectCoreLawName は一つの法令名出現について、引用句、明示した個別主題、
// 検証済み共有末尾 topic の順で最初に成立する経路だけを返す。
func (p *Profile) projectCoreLawName(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	context coreLawNameProjectionContext,
) (coreTopicOption, bool, error) {
	quoted, factID, quotedExists := exactQuotedTerm(
		input,
		context.lawNameSpan,
	)
	if quotedExists {
		if hasLawNameAtSpan(input.LawNameMentions(), quoted.Span()) {
			base, _, scoped := p.coreProjectionBaseEvidence(
				input,
				cues,
				context.lawNameSpan,
			)
			if scoped {
				return newCoreQuotedLawNameOption(quoted, factID, base)
			}
		}
		// 同じ span に引用句がある場合、経路 2 と 3 は成立しない。
		return coreTopicOption{}, false, nil
	}

	if context.individualTopic != nil &&
		sameQuerySpan(context.lawNameSpan, *context.individualTopic) {
		base, clause, scoped := p.coreProjectionBaseEvidence(
			input,
			cues,
			context.lawNameSpan,
		)
		if scoped && hasExplicitIndividualCueInClause(cues, clause) {
			return newCoreAliasLawNameOption(
				input,
				context.lawNameSpan,
				base,
			)
		}
	}

	if context.sharedTerminalTopic != nil &&
		sameQuerySpan(
			context.lawNameSpan,
			context.sharedTerminalTopic.span,
		) {
		return newCoreAliasLawNameOption(
			input,
			context.lawNameSpan,
			context.sharedTerminalTopic.baseEvidence,
		)
	}
	return coreTopicOption{}, false, nil
}

func newCoreQuotedLawNameOption(
	quoted legalquery.QueryTermMention,
	factID string,
	base []profileevidence.EvidenceValues,
) (coreTopicOption, bool, error) {
	option, err := newCoreTopicOption(
		quoted.Surface(),
		quoted.Span().StartByte(),
		append(slices.Clone(base), profileevidence.EvidenceValues{
			FactID:              factID,
			Layer:               profileevidence.LayerTargetAnchor,
			Code:                legalquery.EvidenceGeneralTerm,
			IndependentPositive: true,
			ClusterSpan:         true,
		}),
		nil,
		"quoted:"+factID,
	)
	return option, err == nil, err
}

func newCoreAliasLawNameOption(
	input legalquery.CandidateGenerationInput,
	span legalquery.QuerySpan,
	base []profileevidence.EvidenceValues,
) (coreTopicOption, bool, error) {
	var surface string
	bindings := slices.Clone(base)
	positiveAdded := false
	for index, mention := range input.LawNameMentions() {
		if !sameQuerySpan(mention.Span(), span) ||
			mention.MatchKind() ==
				legalquery.PreprocessMatchUniqueTypoCorrection {
			continue
		}
		if surface == "" {
			surface = mention.Surface()
		}
		if mention.Surface() != surface {
			return coreTopicOption{}, false, nil
		}
		bindings = append(bindings, profileevidence.EvidenceValues{
			FactID:              fmt.Sprintf("law-name-%d", index+1),
			Layer:               profileevidence.LayerTargetAnchor,
			Code:                legalquery.EvidenceOfficialAlias,
			IndependentPositive: !positiveAdded,
			ClusterSpan:         !positiveAdded,
		})
		positiveAdded = true
	}
	if surface == "" {
		return coreTopicOption{}, false, nil
	}
	option, err := newCoreTopicOption(
		strings.TrimFunc(surface, unicode.IsSpace),
		span.StartByte(),
		bindings,
		nil,
		fmt.Sprintf(
			"law-name:%d:%d:%s",
			span.StartByte(),
			span.EndByte(),
			surface,
		),
	)
	return option, err == nil, err
}

func (p *Profile) coreProjectionBaseEvidence(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	span legalquery.QuerySpan,
) ([]profileevidence.EvidenceValues, legalquery.QuerySpan, bool) {
	scope, exists := coreTaskScopeFor(
		input,
		cues,
		"search",
		[]legalquery.QuerySpan{span},
	)
	if !exists || coreHasCompetingReadScope(input, cues, span) {
		return nil, legalquery.QuerySpan{}, false
	}
	resourceFactID, exists := coreResourceFactIDInClause(
		input,
		cues,
		"law_provision",
		scope.relation.ClauseSpan(),
	)
	if !exists || len(coreResourceFactIDsInClause(
		input,
		cues,
		"law",
		scope.relation.ClauseSpan(),
	)) > 0 {
		return nil, legalquery.QuerySpan{}, false
	}
	return []profileevidence.EvidenceValues{
		{
			FactID: scope.taskFactID,
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitTask,
		},
		{
			FactID: resourceFactID,
			Layer:  profileevidence.LayerExplicitTaskResource,
			Code:   legalquery.EvidenceExplicitResource,
		},
	}, scope.relation.ClauseSpan(), true
}

func coreResourceFactIDInClause(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	resourceValue string,
	clause legalquery.QuerySpan,
) (string, bool) {
	values := coreResourceFactIDsInClause(
		input,
		cues,
		resourceValue,
		clause,
	)
	if len(values) != 1 {
		return "", false
	}
	return values[0], true
}

func coreHasCompetingReadScope(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	span legalquery.QuerySpan,
) bool {
	_, exists := coreTaskScopeFor(
		input,
		cues,
		"read",
		[]legalquery.QuerySpan{span},
	)
	return exists
}

func hasExplicitIndividualCueInClause(
	cues resolvedCues,
	clause legalquery.QuerySpan,
) bool {
	for _, mention := range cues.mentions[cueMeaningKey("operator", "individual")] {
		if !coreSpanContains(clause, mention.Span()) {
			continue
		}
		switch mention.Surface() {
		case "それぞれ", "個別に", "一つずつ", "各々":
			return true
		}
	}
	return false
}

func exactQuotedTerm(
	input legalquery.CandidateGenerationInput,
	span legalquery.QuerySpan,
) (legalquery.QueryTermMention, string, bool) {
	for index, mention := range input.QueryTermMentions() {
		if mention.Kind() == legalquery.QueryTermMentionQuotedPhrase &&
			sameQuerySpan(mention.Span(), span) {
			return mention, fmt.Sprintf("query-term-%d", index+1), true
		}
	}
	return legalquery.QueryTermMention{}, "", false
}

func hasLawNameAtSpan(
	mentions []legalquery.LawNameMention,
	span legalquery.QuerySpan,
) bool {
	for _, mention := range mentions {
		if sameQuerySpan(mention.Span(), span) {
			return true
		}
	}
	return false
}

func newCoreTopicOption(
	term string,
	startByte int,
	evidence []profileevidence.EvidenceValues,
	concepts []legalquery.LegalConceptSource,
	meaningKey string,
) (coreTopicOption, error) {
	input, err := newContentInput([]string{term}, nil, nil, nil)
	if err != nil {
		return coreTopicOption{}, err
	}
	return coreTopicOption{
		startByte:  startByte,
		input:      input,
		evidence:   slices.Clone(evidence),
		concepts:   slices.Clone(concepts),
		meaningKey: meaningKey,
	}, nil
}
