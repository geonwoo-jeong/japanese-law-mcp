package hanrei

import (
	"fmt"
	"net/url"
	"path"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	judicialDecisionResourceType = "judicial-decision"
	searchMappingMethodID        = "SOT-IF-044"
	searchHTMLMediaType          = "text/html"
)

var (
	detailPathPattern = regexp.MustCompile(
		`^/hanrei/([0-9]+)/detail([2-8])/index[.]html$`,
	)
	japaneseDecisionDatePattern = regexp.MustCompile(
		`^(明治|大正|昭和|平成|令和)(元|[1-9][0-9]*)年([1-9][0-9]?)月([1-9][0-9]?)日$`,
	)
)

func mapSearchPage(
	response searchResponse,
	fetchedURL string,
	retrievedAt time.Time,
	limit int,
) (judicialdecisionsearch.Page, error) {
	if err := validateSearchMappingInput(response, fetchedURL, retrievedAt, limit); err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	items, err := mapSearchRows(response.rows, fetchedURL, retrievedAt, limit)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return newJudicialSearchPage(items)
}

func mapSearchRows(
	rows []searchResultRow,
	fetchedURL string,
	retrievedAt time.Time,
	limit int,
) ([]model.SourcedResource[model.JudicialDecisionSummary], error) {
	items := make(
		[]model.SourcedResource[model.JudicialDecisionSummary],
		0,
		limit,
	)
	for _, row := range rows {
		for _, detailLink := range row.detailLinks {
			item, err := mapSearchRow(row, detailLink, fetchedURL, retrievedAt)
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			if len(items) == limit {
				return items, nil
			}
		}
	}
	return items, nil
}

func validateSearchMappingInput(
	response searchResponse,
	fetchedURL string,
	retrievedAt time.Time,
	limit int,
) error {
	if limit < 1 ||
		retrievedAt.IsZero() ||
		!isCourtsOfficialURL(fetchedURL) ||
		response.totalCount < 0 ||
		response.totalCount < len(response.rows) {
		return invalidSearchResponseError()
	}
	if len(response.rows) == 0 && response.totalCount != 0 {
		return invalidSearchResponseError()
	}
	for _, row := range response.rows {
		if len(row.detailLinks) == 0 {
			return invalidSearchResponseError()
		}
	}
	return nil
}

func mapSearchRow(
	row searchResultRow,
	detailLink searchDetailLink,
	fetchedURL string,
	retrievedAt time.Time,
) (model.SourcedResource[model.JudicialDecisionSummary], error) {
	detailURL, decisionID, categoryNumber, err := decodeDetailURL(
		fetchedURL,
		detailLink.href,
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	category, err := mapPublicationCategory(
		detailLink.sourceCategoryLabel,
		categoryNumber,
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	decisionDate, err := parseJapaneseDecisionDate(row.decisionDate)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	documents, err := mapSearchDocuments(row.documents, fetchedURL)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	return buildSearchResource(searchResourceValues{
		row:            row,
		detailLink:     detailLink,
		detailURL:      detailURL,
		decisionID:     decisionID,
		categoryNumber: categoryNumber,
		category:       category,
		decisionDate:   decisionDate,
		documents:      documents,
		fetchedURL:     fetchedURL,
		retrievedAt:    retrievedAt,
	})
}

type searchResourceValues struct {
	row            searchResultRow
	detailLink     searchDetailLink
	detailURL      string
	decisionID     string
	categoryNumber string
	category       model.JudicialPublicationCategory
	decisionDate   model.Date
	documents      []model.JudicialDocumentLink
	fetchedURL     string
	retrievedAt    time.Time
}

func buildSearchResource(
	values searchResourceValues,
) (model.SourcedResource[model.JudicialDecisionSummary], error) {
	source := informationSource()
	summary, err := newJudicialDecisionSummary(values, source)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	key, err := newJudicialDecisionKey(values.decisionID, values.categoryNumber)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, invalidSearchResponseError()
	}
	provenance, err := newSearchProvenance(values, source, key)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionSummary]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       summary,
		},
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, invalidSearchResponseError()
	}
	return resource, nil
}

func newJudicialDecisionSummary(
	values searchResourceValues,
	source model.InformationSource,
) (model.JudicialDecisionSummary, error) {
	summary, err := model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          values.decisionID,
			PublicationCategory: values.category,
			SourceCategoryLabel: values.detailLink.sourceCategoryLabel,
			CaseNumber:          values.row.caseNumber,
			CaseName:            optionalSearchString(values.row.caseName),
			DecisionDate:        values.decisionDate,
			CourtName:           values.row.courtName,
			BranchName:          optionalSearchString(values.row.branchName),
			DivisionName:        optionalSearchString(values.row.divisionName),
			DecisionType:        optionalSearchString(values.row.decisionType),
			Outcome:             optionalSearchString(values.row.outcome),
			DetailURL:           values.detailURL,
			Documents:           values.documents,
			Source:              source,
		},
	)
	if err != nil {
		return model.JudicialDecisionSummary{}, invalidSearchResponseError()
	}
	return summary, nil
}

func newJudicialDecisionKey(
	decisionID string,
	categoryNumber string,
) (model.SourceResourceKey, error) {
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: judicialDecisionResourceType,
		ResourceID:   decisionID + "/detail" + categoryNumber,
	})
	if err != nil {
		return model.SourceResourceKey{}, invalidSearchResponseError()
	}
	return key, nil
}

func newSearchProvenance(
	values searchResourceValues,
	source model.InformationSource,
	key model.SourceResourceKey,
) (model.Provenance, error) {
	if values.row.location == "" {
		return model.Provenance{}, invalidSearchResponseError()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            values.fetchedURL,
		RetrievedAt:    values.retrievedAt,
		MediaType:      searchHTMLMediaType,
		Location:       values.row.location,
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       searchMappingMethodID,
	})
	if err != nil {
		return model.Provenance{}, invalidSearchResponseError()
	}
	return provenance, nil
}

func newJudicialSearchPage(
	items []model.SourcedResource[model.JudicialDecisionSummary],
) (judicialdecisionsearch.Page, error) {
	sourcePage, err := model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: len(items),
	})
	if err != nil {
		return judicialdecisionsearch.Page{}, invalidSearchResponseError()
	}
	page, err := judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{Items: items, Page: sourcePage},
	)
	if err != nil {
		return judicialdecisionsearch.Page{}, invalidSearchResponseError()
	}
	return page, nil
}

func mapSearchDocuments(
	values []searchDocumentLink,
	baseURL string,
) ([]model.JudicialDocumentLink, error) {
	documents := make([]model.JudicialDocumentLink, len(values))
	for index, value := range values {
		resolved, valid := resolveCourtsURL(baseURL, value.href)
		if !valid ||
			!strings.EqualFold(path.Ext(resolved.Path), ".pdf") ||
			resolved.Fragment != "" {
			return nil, invalidSearchResponseError()
		}
		document, err := model.NewJudicialDocumentLink(
			model.JudicialDocumentLinkValues{
				Kind:      documentKind(value.label),
				Label:     value.label,
				MediaType: model.JudicialDocumentMediaTypePDF,
				URL:       resolved.String(),
			},
		)
		if err != nil {
			return nil, invalidSearchResponseError()
		}
		documents[index] = document
	}
	return documents, nil
}

func documentKind(label string) model.JudicialDocumentKind {
	switch label {
	case "全文":
		return model.JudicialDocumentKindFullText
	case "要旨":
		return model.JudicialDocumentKindSummary
	default:
		return model.JudicialDocumentKindAttachment
	}
}

func mapPublicationCategory(
	label string,
	categoryNumber string,
) (model.JudicialPublicationCategory, error) {
	labelCategory, labelExists := categoryForLabel(label)
	pathCategory, pathExists := categoryForDetailNumber(categoryNumber)
	if !labelExists || !pathExists || labelCategory != pathCategory {
		return "", invalidSearchResponseError()
	}
	return pathCategory, nil
}

func categoryForLabel(
	label string,
) (model.JudicialPublicationCategory, bool) {
	switch label {
	case "最高裁判例":
		return model.JudicialPublicationCategorySupremeCourt, true
	case "高裁判例":
		return model.JudicialPublicationCategoryHighCourt, true
	case "下級裁裁判例":
		return model.JudicialPublicationCategoryLowerCourt, true
	case "行政事件裁判例":
		return model.JudicialPublicationCategoryAdministrative, true
	case "労働事件裁判例":
		return model.JudicialPublicationCategoryLabor, true
	case "知的財産裁判例", "知財高裁裁判例":
		return model.JudicialPublicationCategoryIntellectualProperty, true
	default:
		return "", false
	}
}

func categoryForDetailNumber(
	value string,
) (model.JudicialPublicationCategory, bool) {
	switch value {
	case "2":
		return model.JudicialPublicationCategorySupremeCourt, true
	case "3":
		return model.JudicialPublicationCategoryHighCourt, true
	case "4":
		return model.JudicialPublicationCategoryLowerCourt, true
	case "5":
		return model.JudicialPublicationCategoryAdministrative, true
	case "6":
		return model.JudicialPublicationCategoryLabor, true
	case "7", "8":
		return model.JudicialPublicationCategoryIntellectualProperty, true
	default:
		return "", false
	}
}

func decodeDetailURL(
	baseURL string,
	href string,
) (string, string, string, error) {
	resolved, valid := resolveCourtsURL(baseURL, href)
	if !valid || resolved.RawQuery != "" || resolved.Fragment != "" {
		return "", "", "", invalidSearchResponseError()
	}
	match := detailPathPattern.FindStringSubmatch(resolved.Path)
	if match == nil {
		return "", "", "", invalidSearchResponseError()
	}
	return resolved.String(), match[1], match[2], nil
}

func canonicalDetailPath(baseURL string, href string) (string, bool) {
	resolved, valid := resolveCourtsURL(baseURL, href)
	if !valid || resolved.RawQuery != "" || resolved.Fragment != "" {
		return "", false
	}
	if detailPathPattern.FindStringSubmatch(resolved.Path) == nil {
		return "", false
	}
	return resolved.Path, true
}

func resolveCourtsURL(baseURL string, href string) (*url.URL, bool) {
	base, err := url.Parse(baseURL)
	if err != nil || !isCourtsURL(base) {
		return nil, false
	}
	reference, err := url.Parse(href)
	if err != nil || reference.User != nil || reference.Opaque != "" {
		return nil, false
	}
	resolved := base.ResolveReference(reference)
	if !isCourtsURL(resolved) {
		return nil, false
	}
	return resolved, true
}

func isCourtsOfficialURL(value string) bool {
	parsed, err := url.Parse(value)
	return err == nil && isCourtsURL(parsed)
}

func isCourtsURL(value *url.URL) bool {
	return value != nil &&
		strings.EqualFold(value.Scheme, "https") &&
		strings.EqualFold(value.Hostname(), "www.courts.go.jp") &&
		value.Port() == "" &&
		value.User == nil &&
		value.Opaque == "" &&
		strings.HasPrefix(value.Path, "/")
}

func parseJapaneseDecisionDate(value string) (model.Date, error) {
	match := japaneseDecisionDatePattern.FindStringSubmatch(value)
	if match == nil {
		return model.Date{}, invalidSearchResponseError()
	}
	eraYear, err := parseJapaneseEraYear(match[2])
	if err != nil {
		return model.Date{}, invalidSearchResponseError()
	}
	month, monthErr := strconv.Atoi(match[3])
	day, dayErr := strconv.Atoi(match[4])
	spec, exists := japaneseEra(match[1])
	if monthErr != nil || dayErr != nil || !exists || eraYear > 9999-spec.offset {
		return model.Date{}, invalidSearchResponseError()
	}
	date := time.Date(spec.offset+eraYear, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	if date.Year() != spec.offset+eraYear ||
		int(date.Month()) != month ||
		date.Day() != day ||
		date.Before(spec.start) ||
		(!spec.end.IsZero() && date.After(spec.end)) {
		return model.Date{}, invalidSearchResponseError()
	}
	parsed, err := model.NewDate(date.Format("2006-01-02"))
	if err != nil {
		return model.Date{}, invalidSearchResponseError()
	}
	return parsed, nil
}

type japaneseEraSpec struct {
	offset int
	start  time.Time
	end    time.Time
}

func japaneseEra(name string) (japaneseEraSpec, bool) {
	switch name {
	case "明治":
		return era(1867, 1868, 1, 25, 1912, 7, 29), true
	case "大正":
		return era(1911, 1912, 7, 30, 1926, 12, 24), true
	case "昭和":
		return era(1925, 1926, 12, 25, 1989, 1, 7), true
	case "平成":
		return era(1988, 1989, 1, 8, 2019, 4, 30), true
	case "令和":
		return japaneseEraSpec{
			offset: 2018,
			start:  time.Date(2019, 5, 1, 0, 0, 0, 0, time.UTC),
		}, true
	default:
		return japaneseEraSpec{}, false
	}
}

func era(
	offset int,
	startYear int,
	startMonth time.Month,
	startDay int,
	endYear int,
	endMonth time.Month,
	endDay int,
) japaneseEraSpec {
	return japaneseEraSpec{
		offset: offset,
		start: time.Date(
			startYear, startMonth, startDay, 0, 0, 0, 0, time.UTC,
		),
		end: time.Date(
			endYear, endMonth, endDay, 0, 0, 0, 0, time.UTC,
		),
	}
}

func parseJapaneseEraYear(value string) (int, error) {
	if value == "元" {
		return 1, nil
	}
	year, err := strconv.Atoi(value)
	if err != nil || year < 1 {
		return 0, fmt.Errorf("元号年が有効ではありません")
	}
	return year, nil
}

func optionalSearchString(value string) *string {
	if value == "" {
		return nil
	}
	cloned := value
	return &cloned
}
