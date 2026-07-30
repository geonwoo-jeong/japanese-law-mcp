package defaultprofile

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/judicialdecisionsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawarticleread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawcontentsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawdocumentread"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawsearch"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/legalquerycorpus"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	executionLawProviderID      = "e-gov-law-api-v2"
	executionLawSourceID        = "e-gov-law-api-v2"
	executionJudicialProviderID = "courts-hanrei-html"
	executionJudicialSourceID   = "courts-hanrei"
)

var executionRetrievedAt = time.Date(
	2026,
	time.July,
	28,
	0,
	0,
	0,
	0,
	time.UTC,
)

type executionFixturePayloads struct {
	lawInformationSource      model.InformationSource
	lawSource                 model.LegalSource
	judicialInformationSource model.InformationSource
}

func newExecutionFixturePayloads() (executionFixturePayloads, error) {
	lawInformationSource, err := newExecutionInformationSource(
		executionLawSourceID,
		"https://laws.e-gov.go.jp/",
	)
	if err != nil {
		return executionFixturePayloads{}, err
	}
	lawSource, err := model.NewLegalSource(lawInformationSource)
	if err != nil {
		return executionFixturePayloads{}, err
	}
	judicialInformationSource, err := newExecutionInformationSource(
		executionJudicialSourceID,
		"https://www.courts.go.jp/hanrei/",
	)
	if err != nil {
		return executionFixturePayloads{}, err
	}
	return executionFixturePayloads{
		lawInformationSource:      lawInformationSource,
		lawSource:                 lawSource,
		judicialInformationSource: judicialInformationSource,
	}, nil
}

func (p executionFixturePayloads) lawSearchPage(
	_ string,
	count int,
) (lawsearch.Page, error) {
	items := make(
		[]model.SourcedResource[model.LawSummary],
		0,
		count,
	)
	for index := 0; index < count; index++ {
		item, err := p.lawSummaryResource(index + 1)
		if err != nil {
			return lawsearch.Page{}, err
		}
		items = append(items, item)
	}
	page, err := exactSourcePage(count)
	if err != nil {
		return lawsearch.Page{}, err
	}
	return lawsearch.NewPage(lawsearch.PageValues{
		Items: items,
		Page:  page,
	})
}

func (p executionFixturePayloads) lawContentPage(
	_ string,
	count int,
) (lawcontentsearch.Page, error) {
	items := make(
		[]model.SourcedResource[model.LawContentMatch],
		0,
		count,
	)
	for index := 0; index < count; index++ {
		item, err := p.lawContentResource(index + 1)
		if err != nil {
			return lawcontentsearch.Page{}, err
		}
		items = append(items, item)
	}
	page, err := exactSourcePage(count)
	if err != nil {
		return lawcontentsearch.Page{}, err
	}
	return lawcontentsearch.NewPage(lawcontentsearch.PageValues{
		Items: items,
		Page:  page,
	})
}

func (p executionFixturePayloads) judicialSearchPage(
	_ string,
	count int,
) (judicialdecisionsearch.Page, error) {
	items := make(
		[]model.SourcedResource[model.JudicialDecisionSummary],
		0,
		count,
	)
	for index := 0; index < count; index++ {
		item, err := p.judicialSummaryResource(index + 1)
		if err != nil {
			return judicialdecisionsearch.Page{}, err
		}
		items = append(items, item)
	}
	page, err := exactSourcePage(count)
	if err != nil {
		return judicialdecisionsearch.Page{}, err
	}
	return judicialdecisionsearch.NewPage(
		judicialdecisionsearch.PageValues{
			Items: items,
			Page:  page,
		},
	)
}

func (p executionFixturePayloads) lawSummaryResource(
	index int,
) (model.SourcedResource[model.LawSummary], error) {
	lawID := fmt.Sprintf("evaluation-law-%04d", index)
	revisionID := fmt.Sprintf("evaluation-revision-%04d", index)
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "評価用法令",
		Source:     p.lawSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, err
	}
	ref, err := newExecutionRef(
		executionLawProviderID,
		executionLawSourceID,
		"law",
		lawID,
		revisionID,
	)
	if err != nil {
		return model.SourcedResource[model.LawSummary]{}, err
	}
	return newExecutionSourcedResource(
		p.lawInformationSource,
		ref,
		"",
		"application/json",
		law,
	)
}

func (p executionFixturePayloads) lawContentResource(
	index int,
) (model.SourcedResource[model.LawContentMatch], error) {
	lawID := fmt.Sprintf("evaluation-law-%04d", index)
	revisionID := fmt.Sprintf("evaluation-revision-%04d", index)
	location := fmt.Sprintf("main:article=%d", index)
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "評価用法令",
		Source:     p.lawSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     p.lawSource,
		LawID:      lawID,
		RevisionID: revisionID,
		Location:   location,
		URL:        "https://laws.e-gov.go.jp/",
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	match, err := model.NewLawContentMatch(model.LawContentMatchValues{
		Law:      law,
		Location: location,
		Text:     "評価用本文",
		Citation: citation,
	})
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	ref, err := newExecutionRef(
		executionLawProviderID,
		executionLawSourceID,
		"law",
		lawID,
		revisionID,
	)
	if err != nil {
		return model.SourcedResource[model.LawContentMatch]{}, err
	}
	return newExecutionSourcedResource(
		p.lawInformationSource,
		ref,
		location,
		"application/json",
		match,
	)
}

func (p executionFixturePayloads) lawDocument(
	input legalquery.LawReadIntentV1,
) (model.SourcedResource[model.LawDocumentRepresentation], error) {
	providerID := executionLawProviderID
	sourceID := executionLawSourceID
	lawID, _ := input.LawID()
	revisionID, _ := input.RevisionID()
	if revisionID == "" {
		revisionID = "evaluation-revision-0001"
	}
	if inputRef, exists := input.Ref(); exists {
		providerID = inputRef.ProviderID()
		sourceID = inputRef.Key().SourceID()
		lawID = inputRef.Key().ResourceID()
		if value, hasVersion := inputRef.Key().VersionID(); hasVersion {
			revisionID = value
		}
	}
	informationSource, err := newExecutionInformationSource(
		sourceID,
		"https://laws.e-gov.go.jp/",
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	legalSource, err := model.NewLegalSource(informationSource)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "評価用法令",
		Source:     legalSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      lawID,
		RevisionID: revisionID,
		URL:        "https://laws.e-gov.go.jp/",
	})
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	var asOf *model.Date
	if value, exists := input.AsOf(); exists {
		asOf = &value
	}
	document, err := model.NewLawDocumentRepresentation(
		model.LawDocumentRepresentationValues{
			Law:      law,
			AsOf:     asOf,
			Format:   model.LawDocumentFormatText,
			Content:  "評価用法令本文",
			Citation: citation,
		},
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	ref, err := newExecutionRef(
		providerID,
		sourceID,
		"law",
		lawID,
		revisionID,
	)
	if err != nil {
		return model.SourcedResource[model.LawDocumentRepresentation]{}, err
	}
	return newExecutionSourcedResource(
		informationSource,
		ref,
		"",
		"application/json",
		document,
	)
}

func (p executionFixturePayloads) lawArticle(
	input legalquery.LawArticleReadIntentV1,
) (model.SourcedResource[model.LawArticleFragment], error) {
	providerID := executionLawProviderID
	sourceID := executionLawSourceID
	lawID, _ := input.LawID()
	revisionID := "evaluation-revision-0001"
	if inputRef, exists := input.Ref(); exists {
		providerID = inputRef.ProviderID()
		sourceID = inputRef.Key().SourceID()
		lawID = inputRef.Key().ResourceID()
		if value, hasVersion := inputRef.Key().VersionID(); hasVersion {
			revisionID = value
		}
	}
	informationSource, err := newExecutionInformationSource(
		sourceID,
		"https://laws.e-gov.go.jp/",
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	legalSource, err := model.NewLegalSource(informationSource)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	law, err := model.NewLawSummary(model.LawSummaryValues{
		LawID:      lawID,
		RevisionID: revisionID,
		Title:      "評価用法令",
		Source:     legalSource,
	})
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	location := executionArticleLocation(input.Location())
	citation, err := model.NewCitation(model.CitationValues{
		Source:     legalSource,
		LawID:      lawID,
		RevisionID: revisionID,
		Location:   location,
		URL:        "https://laws.e-gov.go.jp/",
	})
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	fragment, err := model.NewLawArticleFragment(model.LawArticleFragmentValues{
		Law:      law,
		Location: input.Location(),
		Format:   model.LawArticleFormatText,
		Content:  "評価用条文",
		Citation: citation,
	})
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	ref, err := newExecutionRef(
		providerID,
		sourceID,
		"law",
		lawID,
		revisionID,
	)
	if err != nil {
		return model.SourcedResource[model.LawArticleFragment]{}, err
	}
	return newExecutionSourcedResource(
		informationSource,
		ref,
		location,
		"application/json",
		fragment,
	)
}

func (p executionFixturePayloads) lawUpdatePage(
	input legalquery.LawUpdateListIntentV1,
	count int,
) (lawupdatelist.Page, error) {
	date := input.Date()
	ref, err := newExecutionRef(
		executionLawProviderID,
		executionLawSourceID,
		"law-update-list",
		date.String(),
		"",
	)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	items := make(
		[]model.SourcedResource[model.LawUpdate],
		0,
		count,
	)
	for index := 0; index < count; index++ {
		update, updateErr := model.NewLawUpdate(model.LawUpdateValues{
			UpdatedOn: date,
			LawID:     fmt.Sprintf("evaluation-law-update-%04d", index+1),
			Title:     "評価用法令更新",
			Source:    p.lawSource,
		})
		if updateErr != nil {
			return lawupdatelist.Page{}, updateErr
		}
		item, itemErr := newExecutionSourcedResource(
			p.lawInformationSource,
			ref,
			"",
			"application/json",
			update,
		)
		if itemErr != nil {
			return lawupdatelist.Page{}, itemErr
		}
		items = append(items, item)
	}
	page, err := exactSourcePage(count)
	if err != nil {
		return lawupdatelist.Page{}, err
	}
	return lawupdatelist.NewPage(lawupdatelist.PageValues{
		Items: items,
		Page:  page,
		Date:  date,
	})
}

func (p executionFixturePayloads) judicialSummaryResource(
	index int,
) (model.SourcedResource[model.JudicialDecisionSummary], error) {
	resourceID := fmt.Sprintf("evaluation-%04d/detail2", index)
	summary, err := p.judicialSummary(
		fmt.Sprintf("evaluation-%04d", index),
		resourceID,
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	ref, err := newExecutionRef(
		executionJudicialProviderID,
		executionJudicialSourceID,
		"judicial-decision",
		resourceID,
		"",
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionSummary]{}, err
	}
	return newExecutionSourcedResource(
		p.judicialInformationSource,
		ref,
		"",
		"text/html",
		summary,
	)
}

func (p executionFixturePayloads) judicialDecision(
	input legalquery.JudicialDecisionReadIntentV1,
) (model.SourcedResource[model.JudicialDecisionDetails], error) {
	ref := input.Ref()
	informationSource, err := newExecutionInformationSource(
		ref.Key().SourceID(),
		"https://www.courts.go.jp/hanrei/",
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	summary, err := judicialSummaryForSource(
		informationSource,
		ref.Key().ResourceID(),
		ref.Key().ResourceID(),
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	details, err := model.NewJudicialDecisionDetails(
		model.JudicialDecisionDetailsValues{Summary: summary},
	)
	if err != nil {
		return model.SourcedResource[model.JudicialDecisionDetails]{}, err
	}
	return newExecutionSourcedResource(
		informationSource,
		ref,
		"",
		"text/html",
		details,
	)
}

func (p executionFixturePayloads) judicialSummary(
	decisionID string,
	resourceID string,
) (model.JudicialDecisionSummary, error) {
	return judicialSummaryForSource(
		p.judicialInformationSource,
		decisionID,
		resourceID,
	)
}

func judicialSummaryForSource(
	source model.InformationSource,
	decisionID string,
	resourceID string,
) (model.JudicialDecisionSummary, error) {
	decisionDate, err := model.NewDate("2025-03-03")
	if err != nil {
		return model.JudicialDecisionSummary{}, err
	}
	return model.NewJudicialDecisionSummary(
		model.JudicialDecisionSummaryValues{
			DecisionID:          decisionID,
			PublicationCategory: model.JudicialPublicationCategorySupremeCourt,
			SourceCategoryLabel: "最高裁判例",
			CaseNumber:          "令和6年（受）第1号",
			DecisionDate:        decisionDate,
			CourtName:           "最高裁判所",
			DetailURL:           "https://www.courts.go.jp/hanrei/" + resourceID,
			Documents:           []model.JudicialDocumentLink{},
			Source:              source,
		},
	)
}

func executionArticleLocation(location model.LawArticleLocation) string {
	value := fmt.Sprintf(
		"%s:article=%s",
		location.Provision(),
		location.ArticleNumber(),
	)
	if paragraph, exists := location.ParagraphNumber(); exists {
		return fmt.Sprintf("%s;paragraph=%d", value, paragraph)
	}
	return value
}

func exactSourcePage(count int) (model.SourcePage, error) {
	total := count
	return model.NewSourcePage(model.SourcePageValues{
		ReturnedCount: count,
		TotalCount:    &total,
		TotalRelation: model.TotalRelationExact,
	})
}

func newExecutionInformationSource(
	sourceID string,
	serviceURL string,
) (model.InformationSource, error) {
	return model.NewInformationSource(model.InformationSourceValues{
		ID:         sourceID,
		Name:       "評価用公式情報源",
		Publisher:  "公的機関",
		Authority:  model.AuthorityOfficial,
		ServiceURL: serviceURL,
	})
}

func newExecutionRef(
	providerID string,
	sourceID string,
	resourceType string,
	resourceID string,
	versionID string,
) (model.SourceResourceRef, error) {
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     sourceID,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		VersionID:    versionID,
	})
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	return model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
}

func newExecutionSourcedResource[T interface{ Validate() error }](
	source model.InformationSource,
	ref model.SourceResourceRef,
	location string,
	mediaType string,
	data T,
) (model.SourcedResource[T], error) {
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    ref.Key(),
		URL:            source.ServiceURL(),
		RetrievedAt:    executionRetrievedAt,
		MediaType:      mediaType,
		Location:       location,
		Transformation: model.ProvenanceTransformationUnchanged,
	})
	if err != nil {
		return model.SourcedResource[T]{}, err
	}
	return model.NewSourcedResource(model.SourcedResourceValues[T]{
		Ref:        ref,
		Provenance: []model.Provenance{provenance},
		Data:       data,
	})
}

func fixtureOutcomeError(action resolvedExecutionAction) error {
	switch outcome := action.outcome.(type) {
	case legalquerycorpus.CollectionSuccessOutcome,
		legalquerycorpus.ReadSuccessOutcome:
		return nil
	case legalquerycorpus.TimeoutOutcome:
		return newExecutedFixtureError(context.DeadlineExceeded)
	case legalquerycorpus.FailureOutcome:
		cause, err := fixtureFailureCause(outcome.ErrorCode(), action.step)
		if err != nil {
			return err
		}
		return newExecutedFixtureError(cause)
	default:
		return fmt.Errorf("execution outcome variant が定義されていません")
	}
}

func newExecutedFixtureError(cause error) error {
	executed, err := legalquery.NewExecutedStepError(cause)
	if err != nil {
		return err
	}
	return executed
}

func fixtureFailureCause(
	code model.ErrorCode,
	step legalquery.LegalQueryCandidateStep,
) (error, error) {
	switch code {
	case model.ErrorCodeNotFound:
		return lawdocumentread.ErrNotFound, nil
	case model.ErrorCodeAmbiguousLocation:
		return lawarticleread.ErrAmbiguousLocation, nil
	case model.ErrorCodeInternalError:
		return errors.New("評価用の内部失敗"), nil
	}
	sourceCode := model.SourceErrorCode(code)
	sourceError, err := newExecutionSourceError(sourceCode, step)
	if err != nil {
		return nil, err
	}
	return sourceError, nil
}

type executionSourceOperation string

const executionFixtureCallOperation executionSourceOperation = "fixture-call"

func (executionSourceOperation) SourceOperationProviderID() string {
	return "evaluation-fixture-provider"
}

func (o executionSourceOperation) SourceOperationName() string {
	return string(o)
}

func (o executionSourceOperation) ValidateSourceOperation() error {
	if o != executionFixtureCallOperation {
		return fmt.Errorf("operation が定義されていません")
	}
	return nil
}

func newExecutionSourceError(
	code model.SourceErrorCode,
	step legalquery.LegalQueryCandidateStep,
) (model.SourceError, error) {
	const providerID = "evaluation-fixture-provider"
	capability, err := model.NewProviderCapability(
		model.ProviderCapabilityValues{
			ID:           step.CapabilityID(),
			MajorVersion: step.CapabilityMajorVersion(),
			Level:        model.CapabilityLevelCore,
			Stability:    model.CapabilityStabilityStable,
		},
	)
	if err != nil {
		return model.SourceError{}, err
	}
	source, err := newExecutionInformationSource(
		"evaluation-fixture-source",
		"https://example.go.jp/",
	)
	if err != nil {
		return model.SourceError{}, err
	}
	verifiedAt, err := model.NewDate("2026-07-28")
	if err != nil {
		return model.SourceError{}, err
	}
	provider, err := model.NewProviderDescriptor(
		model.ProviderDescriptorValues{
			ProviderID:             providerID,
			Source:                 source,
			AdapterContractVersion: "1.0.0",
			UpstreamSpecVersion:    "2026-07-28",
			VerifiedAt:             verifiedAt,
			InterfaceType:          model.InterfaceTypeAPI,
			Capabilities:           []model.ProviderCapability{capability},
		},
	)
	if err != nil {
		return model.SourceError{}, err
	}
	return model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   provider,
		Capability: capability,
		Operation:  executionFixtureCallOperation,
	})
}
