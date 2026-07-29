package core

import (
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type articleLocation struct {
	startByte int
	location  model.LawArticleLocation
}

func articleLocationsForTargetGroup(
	input legalquery.CandidateGenerationInput,
	groups [][]lawTarget,
	groupIndex int,
) ([]articleLocation, error) {
	articles := input.ArticleMentions()
	paragraphs := input.ParagraphMentions()
	result := make([]articleLocation, 0)
	for articleIndex, article := range articles {
		if targetGroupForArticle(groups, article) != groupIndex {
			continue
		}
		nextArticleStart := int(^uint(0) >> 1)
		if articleIndex+1 < len(articles) {
			nextArticleStart = articles[articleIndex+1].Span().StartByte()
		}
		matchingParagraphs := make([]legalquery.ParagraphMention, 0)
		for _, paragraph := range paragraphs {
			start := paragraph.Span().StartByte()
			if start >= article.Span().EndByte() &&
				start < nextArticleStart {
				matchingParagraphs = append(matchingParagraphs, paragraph)
			}
		}
		if len(matchingParagraphs) == 0 {
			location, err := model.NewLawArticleLocation(
				model.LawArticleLocationValues{
					Provision:     article.Provision(),
					ArticleNumber: article.ArticleNumber(),
				},
			)
			if err != nil {
				return nil, fmt.Errorf("条位置を構築できません: %w", err)
			}
			result = append(result, articleLocation{
				startByte: article.Span().StartByte(),
				location:  location,
			})
			continue
		}
		for _, paragraph := range matchingParagraphs {
			number := paragraph.ParagraphNumber()
			location, err := model.NewLawArticleLocation(
				model.LawArticleLocationValues{
					Provision:       article.Provision(),
					ArticleNumber:   article.ArticleNumber(),
					ParagraphNumber: &number,
				},
			)
			if err != nil {
				return nil, fmt.Errorf("条項位置を構築できません: %w", err)
			}
			result = append(result, articleLocation{
				startByte: paragraph.Span().StartByte(),
				location:  location,
			})
		}
	}
	return result, nil
}

func targetGroupForArticle(
	groups [][]lawTarget,
	article legalquery.ArticleMention,
) int {
	if len(groups) == 0 {
		return -1
	}
	if len(groups) == 1 || groups[0][0].viaRef != nil {
		return 0
	}
	selected := -1
	for index, group := range groups {
		target := group[0]
		if target.viaRef != nil {
			continue
		}
		if target.endByte <= article.Span().StartByte() {
			selected = index
			continue
		}
		break
	}
	return selected
}

func selectedAsOfDate(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	updatePresent bool,
) *model.Date {
	dates := input.DateMentions()
	if len(dates) == 0 {
		return nil
	}
	if updatePresent && !cues.has("operator", "as_of") {
		return nil
	}
	date := dates[0].Date()
	return &date
}

func withAsOfEvidence(values []candidateDraft) []candidateDraft {
	result := make([]candidateDraft, 0, len(values))
	for _, value := range values {
		current := cloneDraft(value)
		if draftUsesAsOf(current) {
			current.evidence[legalquery.EvidenceStructuredReference] =
				struct{}{}
		}
		result = append(result, current)
	}
	return result
}

func draftUsesAsOf(value candidateDraft) bool {
	for _, step := range value.steps {
		switch input := step.input.(type) {
		case legalquery.LawSearchIntentV1:
			if _, exists := input.AsOf(); exists {
				return true
			}
		case legalquery.LawContentSearchIntentV1:
			if _, exists := input.AsOf(); exists {
				return true
			}
		case legalquery.LawReadIntentV1:
			if _, exists := input.AsOf(); exists {
				return true
			}
		case legalquery.LawArticleReadIntentV1:
			if _, exists := input.AsOf(); exists {
				return true
			}
		}
	}
	return false
}

func shouldReadDocument(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	readRequested bool,
) bool {
	if !readRequested {
		return false
	}
	if len(input.ArticleMentions()) == 0 {
		return explicitDocumentReadRequested(input, cues)
	}
	readCues := cues.mentions[cueMeaningKey("task", "read")]
	searchCues := cues.mentions[cueMeaningKey("task", "search")]
	for _, connector := range cues.mentions[cueMeaningKey("operator", "document_article")] {
		if !explicitRefDocumentBeforeArticle(input, connector, cues) {
			continue
		}
		for _, read := range readCues {
			if connector.Span().EndByte() <= read.Span().StartByte() &&
				!cueStartsBetween(
					searchCues,
					connector.Span().EndByte(),
					read.Span().StartByte(),
				) {
				return true
			}
		}
	}
	return false
}

func explicitRefDocumentBeforeArticle(
	input legalquery.CandidateGenerationInput,
	connector legalquery.CueMention,
	cues resolvedCues,
) bool {
	ref, exists := input.Ref()
	if !exists || ref.Key().ResourceType() != "law" {
		return false
	}
	articles := input.ArticleMentions()
	if len(articles) == 0 {
		return false
	}
	firstArticleStart := articles[0].Span().StartByte()
	firstArticleEnd := articles[0].Span().EndByte()
	for _, article := range articles[1:] {
		if article.Span().StartByte() < firstArticleStart {
			firstArticleStart = article.Span().StartByte()
			firstArticleEnd = article.Span().EndByte()
		}
	}
	if connector.Span().StartByte() > firstArticleStart ||
		connector.Span().EndByte() < firstArticleStart ||
		connector.Span().EndByte() > firstArticleEnd {
		return false
	}
	hasRefTerm := false
	for _, term := range input.QueryTermMentions() {
		span := term.Span()
		if term.Surface() == "参照" &&
			span.EndByte() <= connector.Span().StartByte() &&
			!hasInterveningRefSubject(
				input,
				cues,
				span.EndByte(),
				connector.Span().StartByte(),
			) {
			hasRefTerm = true
			continue
		}
		if connector.Span().EndByte() <= span.StartByte() &&
			span.EndByte() <= firstArticleStart {
			return false
		}
	}
	return hasRefTerm
}

func explicitDocumentReadRequested(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
) bool {
	ref, hasRef := input.Ref()
	if !hasRef {
		return true
	}
	if ref.Key().ResourceType() != "law" {
		return false
	}
	readCues := cues.mentions[cueMeaningKey("task", "read")]
	searchCues := cues.mentions[cueMeaningKey("task", "search")]
	for _, term := range input.QueryTermMentions() {
		if term.Surface() == "参照" &&
			refTermHasDirectRead(
				term,
				input,
				cues,
				readCues,
				searchCues,
			) {
			return true
		}
	}
	resourceGroups := [][]legalquery.CueMention{
		cues.mentions[cueMeaningKey("resource", "law")],
		cues.mentions[cueMeaningKey("resource", "law_provision")],
	}
	for groupIndex, resources := range resourceGroups {
		for _, resource := range resources {
			if groupIndex == 1 &&
				(!strings.Contains(resource.Surface(), "本文") ||
					overlapsReservedJudicialCue(resource, cues)) {
				continue
			}
			for _, read := range readCues {
				if resource.Span().EndByte() <= read.Span().StartByte() &&
					!cueStartsBetween(
						searchCues,
						resource.Span().EndByte(),
						read.Span().StartByte(),
					) &&
					!hasInterveningRefSubject(
						input,
						cues,
						resource.Span().EndByte(),
						read.Span().StartByte(),
					) {
					return true
				}
			}
		}
	}
	return false
}

func refTermHasDirectRead(
	term legalquery.QueryTermMention,
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	readCues []legalquery.CueMention,
	searchCues []legalquery.CueMention,
) bool {
	for _, read := range readCues {
		if term.Span().EndByte() <= read.Span().StartByte() &&
			!cueStartsBetween(
				searchCues,
				term.Span().EndByte(),
				read.Span().StartByte(),
			) &&
			!hasInterveningRefSubject(
				input,
				cues,
				term.Span().EndByte(),
				read.Span().StartByte(),
			) {
			return true
		}
	}
	return false
}

func hasInterveningRefSubject(
	input legalquery.CandidateGenerationInput,
	cues resolvedCues,
	startByte int,
	endByte int,
) bool {
	for _, term := range input.QueryTermMentions() {
		start := term.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, concept := range input.LegalConceptMentions() {
		start := concept.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, law := range input.LawNameMentions() {
		start := law.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, identifier := range input.IdentifierMentions() {
		start := identifier.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, article := range input.ArticleMentions() {
		start := article.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, paragraph := range input.ParagraphMentions() {
		start := paragraph.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, caseNumber := range input.CaseNumberMentions() {
		start := caseNumber.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, date := range input.DateMentions() {
		start := date.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	for _, judicial := range cues.mentions[cueMeaningKey("reserved_pack", "judicial-cases")] {
		start := judicial.Span().StartByte()
		if startByte <= start && start < endByte {
			return true
		}
	}
	return false
}
