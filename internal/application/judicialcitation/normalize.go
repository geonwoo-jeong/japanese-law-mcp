package judicialcitation

import (
	"context"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialcitationnormalize"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type LowerCourtDecisionReference struct {
	courtName        string
	caseNumber       string
	caseNumberSearch string
	decisionDate     *model.Date
}

func (r LowerCourtDecisionReference) CourtName() string        { return r.courtName }
func (r LowerCourtDecisionReference) CaseNumber() string       { return r.caseNumber }
func (r LowerCourtDecisionReference) CaseNumberSearch() string { return r.caseNumberSearch }
func (r LowerCourtDecisionReference) DecisionDate() (model.Date, bool) {
	if r.decisionDate == nil {
		return model.Date{}, false
	}
	return *r.decisionDate, true
}

func NormalizeLowerCourtDecision(
	details model.JudicialDecisionDetails,
) (LowerCourtDecisionReference, bool, error) {
	courtName, hasCourt := details.LowerCourtName()
	caseNumber, hasCaseNumber := details.LowerCourtCaseNumber()
	if !hasCourt || !hasCaseNumber {
		return LowerCourtDecisionReference{}, false, nil
	}
	normalized, ok, err := judicialcitationnormalize.NormalizeLowerCourtDecision(
		courtName,
		caseNumber,
	)
	if err != nil {
		return LowerCourtDecisionReference{}, false, fmt.Errorf(
			"原審事件番号を正規化できません: %w",
			err,
		)
	}
	if !ok {
		return LowerCourtDecisionReference{}, false, nil
	}
	reference := LowerCourtDecisionReference{
		courtName:        normalized.CourtName(),
		caseNumber:       caseNumber,
		caseNumberSearch: normalized.CaseNumberSearch(),
	}
	if value, exists := details.LowerCourtDecisionDate(); exists {
		date := value
		reference.decisionDate = &date
	}
	return reference, true, nil
}

type ProvisionNormalizationResult struct {
	references []model.JudicialCitationLawReference
	unresolved []model.JudicialCitationUnresolvedMention
}

func (r ProvisionNormalizationResult) References() []model.JudicialCitationLawReference {
	return append([]model.JudicialCitationLawReference(nil), r.references...)
}

func (r ProvisionNormalizationResult) Unresolved() []model.JudicialCitationUnresolvedMention {
	return append([]model.JudicialCitationUnresolvedMention(nil), r.unresolved...)
}

func NormalizeReferencedProvisions(
	ctx context.Context,
	resolver judicialcitationnormalize.ExactLawAliasResolver,
	details model.JudicialDecisionDetails,
	provenance model.Provenance,
) (ProvisionNormalizationResult, error) {
	if ctx == nil {
		return ProvisionNormalizationResult{}, fmt.Errorf("context は必須です")
	}
	if err := ctx.Err(); err != nil {
		return ProvisionNormalizationResult{}, err
	}
	if err := provenance.Validate(); err != nil {
		return ProvisionNormalizationResult{}, fmt.Errorf("provenance が有効ではありません: %w", err)
	}
	text, exists := details.ReferencedProvisionsText()
	if !exists {
		return ProvisionNormalizationResult{
			references: []model.JudicialCitationLawReference{},
			unresolved: []model.JudicialCitationUnresolvedMention{},
		}, nil
	}
	segments := judicialcitationnormalize.SplitReferencedProvisionText(text)
	references := make([]model.JudicialCitationLawReference, 0, len(segments))
	unresolved := make([]model.JudicialCitationUnresolvedMention, 0, len(segments))
	for _, segment := range segments {
		if err := ctx.Err(); err != nil {
			return ProvisionNormalizationResult{}, err
		}
		resolved, reason, err := judicialcitationnormalize.NormalizeReferencedProvision(
			ctx,
			segment,
			resolver,
		)
		if err != nil {
			return ProvisionNormalizationResult{}, err
		}
		if reason != "" {
			mention, err := newLawProvisionUnresolvedMention(
				segment,
				reason,
				provenance,
			)
			if err != nil {
				return ProvisionNormalizationResult{}, err
			}
			unresolved = append(unresolved, mention)
			continue
		}
		reference, err := model.NewJudicialCitationLawReference(
			model.JudicialCitationLawReferenceValues{
				LawID:    resolved.LawID(),
				LawTitle: resolved.LawTitle(),
				Location: resolved.Location(),
			},
		)
		if err != nil {
			return ProvisionNormalizationResult{}, err
		}
		references = append(references, reference)
	}
	return ProvisionNormalizationResult{
		references: references,
		unresolved: unresolved,
	}, nil
}

func newLawProvisionUnresolvedMention(
	text string,
	reason model.JudicialCitationUnresolvedReason,
	provenance model.Provenance,
) (model.JudicialCitationUnresolvedMention, error) {
	return model.NewJudicialCitationUnresolvedMention(
		model.JudicialCitationUnresolvedMentionValues{
			MentionType: model.JudicialCitationMentionTypeLawProvision,
			MentionText: text,
			Reason:      reason,
			Provenance:  provenance,
		},
	)
}
