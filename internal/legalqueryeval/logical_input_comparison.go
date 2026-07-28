package legalqueryeval

import (
	"fmt"
	"slices"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func compareLogicalInput(
	expected legalquery.LogicalInput,
	actual legalquery.LogicalInput,
) (bool, error) {
	switch expectedValue := expected.(type) {
	case legalquery.LawSearchIntentV1:
		actualValue, ok := actual.(legalquery.LawSearchIntentV1)
		return ok && compareLawSearchInput(expectedValue, actualValue), nil
	case legalquery.LawContentSearchIntentV1:
		actualValue, ok := actual.(legalquery.LawContentSearchIntentV1)
		return ok && compareLawContentSearchInput(expectedValue, actualValue), nil
	case legalquery.LawReadIntentV1:
		actualValue, ok := actual.(legalquery.LawReadIntentV1)
		return ok && compareLawReadInput(expectedValue, actualValue), nil
	case legalquery.LawArticleReadIntentV1:
		actualValue, ok := actual.(legalquery.LawArticleReadIntentV1)
		return ok && compareLawArticleReadInput(expectedValue, actualValue), nil
	case legalquery.LawUpdateListIntentV1:
		actualValue, ok := actual.(legalquery.LawUpdateListIntentV1)
		return ok && compareDate(expectedValue.Date(), actualValue.Date()), nil
	case legalquery.JudicialDecisionSearchIntentV1:
		actualValue, ok := actual.(legalquery.JudicialDecisionSearchIntentV1)
		return ok && expectedValue.Query() == actualValue.Query(), nil
	case legalquery.JudicialDecisionReadIntentV1:
		actualValue, ok := actual.(legalquery.JudicialDecisionReadIntentV1)
		return ok && compareSourceResourceRef(expectedValue.Ref(), actualValue.Ref()), nil
	default:
		return false, fmt.Errorf("比較対象の logicalInput variant が定義されていません")
	}
}

func compareLawSearchInput(
	expected legalquery.LawSearchIntentV1,
	actual legalquery.LawSearchIntentV1,
) bool {
	expectedDate, expectedHasDate := expected.AsOf()
	actualDate, actualHasDate := actual.AsOf()
	return expected.Query() == actual.Query() &&
		compareOptionalDate(
			expectedDate,
			expectedHasDate,
			actualDate,
			actualHasDate,
		)
}

func compareLawContentSearchInput(
	expected legalquery.LawContentSearchIntentV1,
	actual legalquery.LawContentSearchIntentV1,
) bool {
	expectedDate, expectedHasDate := expected.AsOf()
	actualDate, actualHasDate := actual.AsOf()
	return slices.Equal(expected.AllTerms(), actual.AllTerms()) &&
		slices.Equal(expected.AnyTerms(), actual.AnyTerms()) &&
		slices.Equal(expected.ExcludeTerms(), actual.ExcludeTerms()) &&
		compareOptionalDate(
			expectedDate,
			expectedHasDate,
			actualDate,
			actualHasDate,
		)
}

func compareLawReadInput(
	expected legalquery.LawReadIntentV1,
	actual legalquery.LawReadIntentV1,
) bool {
	expectedLawID, expectedHasLawID := expected.LawID()
	actualLawID, actualHasLawID := actual.LawID()
	expectedRevisionID, expectedHasRevisionID := expected.RevisionID()
	actualRevisionID, actualHasRevisionID := actual.RevisionID()
	expectedDate, expectedHasDate := expected.AsOf()
	actualDate, actualHasDate := actual.AsOf()
	expectedRef, expectedHasRef := expected.Ref()
	actualRef, actualHasRef := actual.Ref()

	return compareOptionalString(
		expectedLawID,
		expectedHasLawID,
		actualLawID,
		actualHasLawID,
	) &&
		compareOptionalString(
			expectedRevisionID,
			expectedHasRevisionID,
			actualRevisionID,
			actualHasRevisionID,
		) &&
		compareOptionalDate(
			expectedDate,
			expectedHasDate,
			actualDate,
			actualHasDate,
		) &&
		compareOptionalRef(
			expectedRef,
			expectedHasRef,
			actualRef,
			actualHasRef,
		)
}

func compareLawArticleReadInput(
	expected legalquery.LawArticleReadIntentV1,
	actual legalquery.LawArticleReadIntentV1,
) bool {
	expectedLawID, expectedHasLawID := expected.LawID()
	actualLawID, actualHasLawID := actual.LawID()
	expectedRef, expectedHasRef := expected.Ref()
	actualRef, actualHasRef := actual.Ref()
	expectedDate, expectedHasDate := expected.AsOf()
	actualDate, actualHasDate := actual.AsOf()

	return compareOptionalString(
		expectedLawID,
		expectedHasLawID,
		actualLawID,
		actualHasLawID,
	) &&
		compareOptionalRef(
			expectedRef,
			expectedHasRef,
			actualRef,
			actualHasRef,
		) &&
		compareLawArticleLocation(expected.Location(), actual.Location()) &&
		compareOptionalDate(
			expectedDate,
			expectedHasDate,
			actualDate,
			actualHasDate,
		)
}

func compareOptionalString(
	expected string,
	expectedExists bool,
	actual string,
	actualExists bool,
) bool {
	return expectedExists == actualExists &&
		(!expectedExists || expected == actual)
}

func compareOptionalDate(
	expected model.Date,
	expectedExists bool,
	actual model.Date,
	actualExists bool,
) bool {
	return expectedExists == actualExists &&
		(!expectedExists || compareDate(expected, actual))
}

func compareDate(expected model.Date, actual model.Date) bool {
	return expected.String() == actual.String()
}

func compareOptionalRef(
	expected model.SourceResourceRef,
	expectedExists bool,
	actual model.SourceResourceRef,
	actualExists bool,
) bool {
	return expectedExists == actualExists &&
		(!expectedExists || compareSourceResourceRef(expected, actual))
}

func compareSourceResourceRef(
	expected model.SourceResourceRef,
	actual model.SourceResourceRef,
) bool {
	return expected.ProviderID() == actual.ProviderID() &&
		compareSourceResourceKey(expected.Key(), actual.Key())
}

func compareSourceResourceKey(
	expected model.SourceResourceKey,
	actual model.SourceResourceKey,
) bool {
	expectedVersionID, expectedHasVersionID := expected.VersionID()
	actualVersionID, actualHasVersionID := actual.VersionID()
	return expected.SourceID() == actual.SourceID() &&
		expected.ResourceType() == actual.ResourceType() &&
		expected.ResourceID() == actual.ResourceID() &&
		compareOptionalString(
			expectedVersionID,
			expectedHasVersionID,
			actualVersionID,
			actualHasVersionID,
		)
}

func compareLawArticleLocation(
	expected model.LawArticleLocation,
	actual model.LawArticleLocation,
) bool {
	expectedParagraph, expectedHasParagraph := expected.ParagraphNumber()
	actualParagraph, actualHasParagraph := actual.ParagraphNumber()
	return expected.Provision() == actual.Provision() &&
		expected.ArticleNumber() == actual.ArticleNumber() &&
		expectedHasParagraph == actualHasParagraph &&
		(!expectedHasParagraph || expectedParagraph == actualParagraph)
}
