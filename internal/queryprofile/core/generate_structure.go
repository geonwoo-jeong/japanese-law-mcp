package core

import (
	"fmt"

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

func shouldReadDocument(
	input legalquery.CandidateGenerationInput,
	_ resolvedCues,
	readRequested bool,
) bool {
	return readRequested && len(input.ArticleMentions()) == 0
}
