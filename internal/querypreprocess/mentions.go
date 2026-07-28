package querypreprocess

import (
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
)

func (p *Preprocessor) buildLawMentions(
	query string,
	drafts []lawDraft,
) ([]legalquery.LawNameMention, error) {
	drafts = deduplicateLawDrafts(drafts)
	mentions := make([]legalquery.LawNameMention, 0, len(drafts))
	for _, draft := range drafts {
		entry, exists := p.lawsByID[draft.lawID]
		if !exists {
			return nil, fmt.Errorf("未知の lawId %q が前処理結果にあります", draft.lawID)
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: draft.startByte,
			EndByte:   draft.endByte,
		})
		if err != nil {
			return nil, err
		}
		mention, err := legalquery.NewLawNameMention(
			legalquery.LawNameMentionValues{
				Span:       span,
				Surface:    query[draft.startByte:draft.endByte],
				LawID:      entry.ResourceID,
				RevisionID: entry.RevisionID,
				LawNumber:  entry.LawNumber,
				Canonical:  entry.Canonical,
				MatchKind:  draft.matchKind,
			},
		)
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}

func (p *Preprocessor) buildConceptMentions(
	query string,
	drafts []conceptDraft,
	lawMentions []legalquery.LawNameMention,
) ([]legalquery.LegalConceptMention, error) {
	drafts = deduplicateConceptDrafts(drafts)
	mentions := make([]legalquery.LegalConceptMention, 0, len(drafts))
	for _, draft := range drafts {
		if lawNameDominatesConcept(draft, lawMentions) {
			continue
		}
		entry, exists := p.conceptsByID[draft.conceptID]
		if !exists {
			return nil, fmt.Errorf(
				"未知の conceptId %q が前処理結果にあります",
				draft.conceptID,
			)
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: draft.startByte,
			EndByte:   draft.endByte,
		})
		if err != nil {
			return nil, err
		}
		mention, err := legalquery.NewLegalConceptMention(
			legalquery.LegalConceptMentionValues{
				Span:      span,
				Surface:   query[draft.startByte:draft.endByte],
				ConceptID: entry.ConceptID,
				Canonical: entry.Canonical,
				MatchKind: draft.matchKind,
			},
		)
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}

func lawNameDominatesConcept(
	concept conceptDraft,
	laws []legalquery.LawNameMention,
) bool {
	for _, law := range laws {
		span := law.Span()
		if span.StartByte() != concept.startByte ||
			span.EndByte() != concept.endByte {
			continue
		}
		switch law.MatchKind() {
		case legalquery.PreprocessMatchExact,
			legalquery.PreprocessMatchComparisonNormalized,
			legalquery.PreprocessMatchRegisteredTerm:
			return true
		}
	}
	return false
}

func (p *Preprocessor) buildCueMentions(
	query string,
	drafts []cueDraft,
) ([]legalquery.CueMention, error) {
	drafts = deduplicateCueDrafts(drafts)
	mentions := make([]legalquery.CueMention, 0, len(drafts))
	for _, draft := range drafts {
		if _, exists := p.cuesByKey[cueKey(draft.profileID, draft.cueID)]; !exists {
			return nil, fmt.Errorf(
				"未知の profileId=%q, cueId=%q が前処理結果にあります",
				draft.profileID,
				draft.cueID,
			)
		}
		span, err := legalquery.NewQuerySpan(legalquery.QuerySpanValues{
			StartByte: draft.startByte,
			EndByte:   draft.endByte,
		})
		if err != nil {
			return nil, err
		}
		mention, err := legalquery.NewCueMention(legalquery.CueMentionValues{
			Span:      span,
			Surface:   query[draft.startByte:draft.endByte],
			ProfileID: draft.profileID,
			CueID:     draft.cueID,
			MatchKind: draft.matchKind,
		})
		if err != nil {
			return nil, err
		}
		mentions = append(mentions, mention)
	}
	return mentions, nil
}
