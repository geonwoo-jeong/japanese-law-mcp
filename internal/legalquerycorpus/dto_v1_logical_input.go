package legalquerycorpus

import (
	"encoding/json"
	"fmt"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/legalquery"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

type expectedResourceKeyV1DTO struct {
	SourceID     *string `json:"sourceId"`
	ResourceType *string `json:"resourceType"`
	ResourceID   *string `json:"resourceId"`
	VersionID    *string `json:"versionId,omitempty"`
}

type expectedResourceRefV1DTO struct {
	ProviderID *string                   `json:"providerId"`
	Key        *expectedResourceKeyV1DTO `json:"key"`
}

type expectedLawArticleLocationV1DTO struct {
	Provision       *string `json:"provision"`
	ArticleNumber   *string `json:"articleNumber"`
	ParagraphNumber *int    `json:"paragraphNumber,omitempty"`
}

type lawSearchInputV1DTO struct {
	Query *string `json:"query"`
	AsOf  *string `json:"asOf,omitempty"`
}

type lawContentSearchInputV1DTO struct {
	AllTerms     *[]string `json:"allTerms"`
	AnyTerms     *[]string `json:"anyTerms"`
	ExcludeTerms *[]string `json:"excludeTerms"`
	AsOf         *string   `json:"asOf,omitempty"`
}

type lawReadInputV1DTO struct {
	LawID      *string                   `json:"lawId,omitempty"`
	RevisionID *string                   `json:"revisionId,omitempty"`
	AsOf       *string                   `json:"asOf,omitempty"`
	Ref        *expectedResourceRefV1DTO `json:"ref,omitempty"`
}

type lawArticleReadInputV1DTO struct {
	LawID    *string                          `json:"lawId,omitempty"`
	Ref      *expectedResourceRefV1DTO        `json:"ref,omitempty"`
	Location *expectedLawArticleLocationV1DTO `json:"location"`
	AsOf     *string                          `json:"asOf,omitempty"`
}

type lawUpdatesInputV1DTO struct {
	Date *string `json:"date"`
}

type judicialSearchInputV1DTO struct {
	Query *string `json:"query"`
}

type judicialReadInputV1DTO struct {
	Ref *expectedResourceRefV1DTO `json:"ref"`
}

func convertExpectedLogicalInputV1(
	kind legalquery.LogicalInputKind,
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	switch kind {
	case legalquery.InputKindLawSearch:
		return convertLawSearchInputV1(data)
	case legalquery.InputKindLawContentSearch:
		return convertLawContentSearchInputV1(data)
	case legalquery.InputKindLawRead:
		return convertLawReadInputV1(data)
	case legalquery.InputKindLawArticleRead:
		return convertLawArticleReadInputV1(data)
	case legalquery.InputKindLawUpdates:
		return convertLawUpdatesInputV1(data)
	case legalquery.InputKindJudicialDecisionSearch:
		return convertJudicialSearchInputV1(data)
	case legalquery.InputKindJudicialDecisionRead:
		return convertJudicialReadInputV1(data)
	default:
		return nil, fmt.Errorf("expected step の inputKind が定義されていません")
	}
}

func convertLawSearchInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto lawSearchInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.Query == nil {
		return nil, fmt.Errorf("law_search logicalInput の query は必須です")
	}
	asOf, err := convertExpectedOptionalDate(dto.AsOf)
	if err != nil {
		return nil, err
	}
	return legalquery.NewLawSearchIntentV1(
		legalquery.LawSearchIntentV1Values{Query: *dto.Query, AsOf: asOf},
	)
}

func convertLawContentSearchInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto lawContentSearchInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.AllTerms == nil || dto.AnyTerms == nil || dto.ExcludeTerms == nil {
		return nil, fmt.Errorf("law_content_search logicalInput の検索語配列は必須です")
	}
	asOf, err := convertExpectedOptionalDate(dto.AsOf)
	if err != nil {
		return nil, err
	}
	return legalquery.NewLawContentSearchIntentV1(
		legalquery.LawContentSearchIntentV1Values{
			AllTerms:     *dto.AllTerms,
			AnyTerms:     *dto.AnyTerms,
			ExcludeTerms: *dto.ExcludeTerms,
			AsOf:         asOf,
		},
	)
}

func convertLawReadInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto lawReadInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	asOf, err := convertExpectedOptionalDate(dto.AsOf)
	if err != nil {
		return nil, err
	}
	ref, err := convertExpectedOptionalRef(dto.Ref)
	if err != nil {
		return nil, err
	}
	return legalquery.NewLawReadIntentV1(legalquery.LawReadIntentV1Values{
		LawID:      optionalStringValue(dto.LawID),
		RevisionID: optionalStringValue(dto.RevisionID),
		AsOf:       asOf,
		Ref:        ref,
	})
}

func convertLawArticleReadInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto lawArticleReadInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.Location == nil {
		return nil, fmt.Errorf("law_article_read logicalInput の location は必須です")
	}
	location, err := convertExpectedLawArticleLocation(*dto.Location)
	if err != nil {
		return nil, err
	}
	asOf, err := convertExpectedOptionalDate(dto.AsOf)
	if err != nil {
		return nil, err
	}
	ref, err := convertExpectedOptionalRef(dto.Ref)
	if err != nil {
		return nil, err
	}
	return legalquery.NewLawArticleReadIntentV1(
		legalquery.LawArticleReadIntentV1Values{
			LawID:    optionalStringValue(dto.LawID),
			Ref:      ref,
			Location: location,
			AsOf:     asOf,
		},
	)
}

func convertLawUpdatesInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto lawUpdatesInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.Date == nil {
		return nil, fmt.Errorf("law_updates logicalInput の date は必須です")
	}
	date, err := model.NewDate(*dto.Date)
	if err != nil {
		return nil, fmt.Errorf("law_updates logicalInput の date が有効ではありません")
	}
	return legalquery.NewLawUpdateListIntentV1(
		legalquery.LawUpdateListIntentV1Values{Date: date},
	)
}

func convertJudicialSearchInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto judicialSearchInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.Query == nil {
		return nil, fmt.Errorf("judicial_decision_search logicalInput の query は必須です")
	}
	return legalquery.NewJudicialDecisionSearchIntentV1(
		legalquery.JudicialDecisionSearchIntentV1Values{Query: *dto.Query},
	)
}

func convertJudicialReadInputV1(
	data json.RawMessage,
) (legalquery.LogicalInput, error) {
	var dto judicialReadInputV1DTO
	if err := decodeJSONV1(data, &dto); err != nil {
		return nil, err
	}
	if dto.Ref == nil {
		return nil, fmt.Errorf("judicial_decision_read logicalInput の ref は必須です")
	}
	ref, err := convertExpectedRef(*dto.Ref)
	if err != nil {
		return nil, err
	}
	return legalquery.NewJudicialDecisionReadIntentV1(
		legalquery.JudicialDecisionReadIntentV1Values{Ref: ref},
	)
}

func convertExpectedOptionalDate(value *string) (*model.Date, error) {
	if value == nil {
		return nil, nil
	}
	date, err := model.NewDate(*value)
	if err != nil {
		return nil, fmt.Errorf("expected logicalInput の日付が有効ではありません")
	}
	return &date, nil
}

func convertExpectedLawArticleLocation(
	dto expectedLawArticleLocationV1DTO,
) (model.LawArticleLocation, error) {
	if dto.Provision == nil || dto.ArticleNumber == nil {
		return model.LawArticleLocation{}, fmt.Errorf("law article location の必須項目が不足しています")
	}
	location, err := model.NewLawArticleLocation(model.LawArticleLocationValues{
		Provision:       model.LawArticleProvision(*dto.Provision),
		ArticleNumber:   *dto.ArticleNumber,
		ParagraphNumber: dto.ParagraphNumber,
	})
	if err != nil {
		return model.LawArticleLocation{}, fmt.Errorf("law article location が有効ではありません")
	}
	return location, nil
}

func convertExpectedOptionalRef(
	dto *expectedResourceRefV1DTO,
) (*model.SourceResourceRef, error) {
	if dto == nil {
		return nil, nil
	}
	ref, err := convertExpectedRef(*dto)
	if err != nil {
		return nil, err
	}
	return &ref, nil
}

func convertExpectedRef(
	dto expectedResourceRefV1DTO,
) (model.SourceResourceRef, error) {
	if dto.ProviderID == nil || dto.Key == nil {
		return model.SourceResourceRef{}, fmt.Errorf("expected ref の必須項目が不足しています")
	}
	key, err := convertExpectedResourceKey(*dto.Key)
	if err != nil {
		return model.SourceResourceRef{}, err
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: *dto.ProviderID,
		Key:        key,
	})
	if err != nil {
		return model.SourceResourceRef{}, fmt.Errorf("expected ref が有効ではありません")
	}
	return ref, nil
}

func convertExpectedResourceKey(
	dto expectedResourceKeyV1DTO,
) (model.SourceResourceKey, error) {
	if dto.SourceID == nil || dto.ResourceType == nil || dto.ResourceID == nil {
		return model.SourceResourceKey{}, fmt.Errorf("expected ref.key の必須項目が不足しています")
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     *dto.SourceID,
		ResourceType: *dto.ResourceType,
		ResourceID:   *dto.ResourceID,
		VersionID:    optionalStringValue(dto.VersionID),
	})
	if err != nil {
		return model.SourceResourceKey{}, fmt.Errorf("expected ref.key が有効ではありません")
	}
	return key, nil
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
