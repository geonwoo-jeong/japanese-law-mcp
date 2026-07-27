package hanrei

import (
	"fmt"
	"strings"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	readMappingMethodID = "SOT-IF-045"
	readHTMLMediaType   = "text/html"
)

func mapReadDetails(
	response readResponse,
	fetched fetchedReadResponse,
	ref model.SourceResourceRef,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	if err := validateReadMappingInput(response, fetched, ref); err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	summary, err := newReadSummary(response, fetched)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	details, err := newReadDetails(response, summary)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	provenance, err := newReadProvenance(fetched, ref)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.JudicialDecisionDetails]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       details,
		},
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, invalidReadResponseError()
	}
	return resource, nil
}

func newReadSummary(
	response readResponse,
	fetched fetchedReadResponse,
) (model.JudicialDecisionSummary, error) {
	category, exists := categoryForDetailNumber(fetched.categoryNumber)
	if !exists {
		return model.JudicialDecisionSummary{}, invalidReadResponseError()
	}
	categoryLabel, exists := readCategoryLabelForDetailNumber(fetched.categoryNumber)
	if !exists || response.sourceCategoryLabel != categoryLabel {
		return model.JudicialDecisionSummary{}, invalidReadResponseError()
	}
	caseNumber, err := requireReadField(response.fields, "事件番号")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	decisionDateText, err := requireReadField(response.fields, "裁判年月日")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	decisionDate, err := parseReadJapaneseDecisionDate(decisionDateText)
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	courtName, err := requireReadField(response.fields, "法廷名", "裁判所名・部", "裁判所名")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	documents, err := mapSearchDocuments(response.documents, fetched.fetchedURL)
	if err != nil {
		return model.JudicialDecisionSummary{}, invalidReadResponseError()
	}
	caseName, err := optionalReadField(response.fields, "事件名")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	branchName, err := optionalReadField(response.fields, "支部名")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	divisionName, err := optionalReadField(response.fields, "部名")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	decisionType, err := optionalReadField(response.fields, "裁判種別")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	outcome, err := optionalReadField(response.fields, "結果", "判決結果")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	summary, err := model.NewJudicialDecisionSummary(model.JudicialDecisionSummaryValues{
		DecisionID:          fetched.decisionID,
		PublicationCategory: category,
		SourceCategoryLabel: response.sourceCategoryLabel,
		CaseNumber:          caseNumber,
		CaseName:            caseName,
		DecisionDate:        decisionDate,
		CourtName:           courtName,
		BranchName:          branchName,
		DivisionName:        divisionName,
		DecisionType:        decisionType,
		Outcome:             outcome,
		DetailURL:           fetched.fetchedURL,
		Documents:           documents,
		Source:              informationSource(),
	})
	if err != nil {
		return model.JudicialDecisionSummary{}, invalidReadResponseError()
	}
	return summary, nil
}

func newReadDetails(
	response readResponse,
	summary model.JudicialDecisionSummary,
) (model.JudicialDecisionDetails, error) {
	reporterCitation, err := optionalReadField(
		response.fields,
		"判例集等巻・号・頁",
		"高裁判例集登載巻・号・頁",
	)
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	lowerCourtName, err := optionalReadField(response.fields, "原審裁判所名")
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	lowerCourtCaseNumber, err := optionalReadField(response.fields, "原審事件番号")
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	lowerCourtDecisionDate, err := optionalReadDateField(
		response.fields,
		"原審裁判年月日",
	)
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	holdingText, err := optionalReadField(
		response.fields,
		"判示事項",
		"判示事項の要旨",
	)
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	summaryText, err := optionalReadField(response.fields, "裁判要旨", "要旨")
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	referencedProvisionsText, err := optionalReadField(response.fields, "参照法条")
	if err != nil {
		return model.JudicialDecisionDetails{}, err
	}
	details, err := model.NewJudicialDecisionDetails(model.JudicialDecisionDetailsValues{
		Summary:                  summary,
		ReporterCitation:         reporterCitation,
		LowerCourtName:           lowerCourtName,
		LowerCourtCaseNumber:     lowerCourtCaseNumber,
		LowerCourtDecisionDate:   lowerCourtDecisionDate,
		HoldingText:              holdingText,
		SummaryText:              summaryText,
		ReferencedProvisionsText: referencedProvisionsText,
	})
	if err != nil {
		return model.JudicialDecisionDetails{}, invalidReadResponseError()
	}
	return details, nil
}

func newReadProvenance(
	fetched fetchedReadResponse,
	ref model.SourceResourceRef,
) (model.Provenance, error) {
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         informationSource(),
		ResourceKey:    ref.Key(),
		URL:            fetched.fetchedURL,
		RetrievedAt:    fetched.retrievedAt,
		MediaType:      readHTMLMediaType,
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       readMappingMethodID,
	})
	if err != nil {
		return model.Provenance{}, invalidReadResponseError()
	}
	return provenance, nil
}

func validateReadMappingInput(
	response readResponse,
	fetched fetchedReadResponse,
	ref model.SourceResourceRef,
) error {
	match := detailResourceIDPattern.FindStringSubmatch(ref.Key().ResourceID())
	if response.sourceCategoryLabel == "" ||
		!isCourtsOfficialURL(fetched.fetchedURL) ||
		fetched.retrievedAt.IsZero() ||
		fetched.decisionID == "" ||
		fetched.categoryNumber == "" ||
		ref.ProviderID() != providerID ||
		ref.Key().SourceID() != sourceID ||
		ref.Key().ResourceType() != judicialDecisionResourceType ||
		len(match) != 3 ||
		match[1] != fetched.decisionID ||
		match[2] != fetched.categoryNumber ||
		fetched.fetchedURL != readDetailURL(fetched.decisionID, fetched.categoryNumber) {
		return invalidReadResponseError()
	}
	if _, exists := ref.Key().VersionID(); exists {
		return invalidReadResponseError()
	}
	return nil
}

func requireReadField(fields []readField, labels ...string) (string, error) {
	value, exists, err := readFieldValue(fields, labels...)
	if err != nil {
		return "", err
	}
	if !exists || value == "" {
		return "", invalidReadResponseError()
	}
	return value, nil
}

func optionalReadField(fields []readField, labels ...string) (*string, error) {
	value, exists, err := readFieldValue(fields, labels...)
	if err != nil {
		return nil, err
	}
	if !exists || value == "" {
		return nil, nil
	}
	cloned := value
	return &cloned, nil
}

func optionalReadDateField(
	fields []readField,
	labels ...string,
) (*model.Date, error) {
	value, exists, err := readFieldValue(fields, labels...)
	if err != nil {
		return nil, err
	}
	if !exists || value == "" {
		return nil, nil
	}
	parsed, err := parseReadJapaneseDecisionDate(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func readFieldValue(fields []readField, labels ...string) (string, bool, error) {
	accepted := make(map[string]struct{}, len(labels))
	for _, label := range labels {
		accepted[label] = struct{}{}
	}
	var selected string
	found := false
	for _, field := range fields {
		if _, ok := accepted[field.label]; !ok {
			continue
		}
		value := strings.TrimSpace(field.value)
		if value == "" {
			continue
		}
		if !found {
			selected = value
			found = true
			continue
		}
		if selected != value {
			return "", false, invalidReadResponseError()
		}
	}
	return selected, found, nil
}

func parseReadJapaneseDecisionDate(value string) (model.Date, error) {
	parsed, err := parseJapaneseDecisionDate(value)
	if err != nil {
		return model.Date{}, invalidReadResponseError()
	}
	return parsed, nil
}

func readCategoryLabelForDetailNumber(value string) (string, bool) {
	switch value {
	case "2":
		return "最高裁判所", true
	case "3":
		return "高等裁判所", true
	case "4":
		return "下級裁判所(速報)", true
	case "5":
		return "行政事件", true
	case "6":
		return "労働事件", true
	case "7":
		return "知的財産事件", true
	case "8":
		return "知的財産高等裁判所", true
	default:
		return "", false
	}
}

func readDetailURL(decisionID string, categoryNumber string) string {
	return fmt.Sprintf(
		"https://www.courts.go.jp/hanrei/%s/detail%s/index.html",
		decisionID,
		categoryNumber,
	)
}
