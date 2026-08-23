package lawv2

import (
	"fmt"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

const (
	lawRevisionResourceType  = "law"
	lawRevisionMediaType     = "application/json"
	lawRevisionMappingMethod = "SOT-IF-057"
)

func mapLawRevisions(
	response lawRevisionResponse,
	retrievedAt time.Time,
	endpointURL string,
) ([]model.SourcedResource[model.LawRevision], error) {
	source := informationSource()
	legalSource, err := model.NewLegalSource(source)
	if err != nil {
		return invalidMappedLawRevisions()
	}
	promulgationDate, err := optionalLawRevisionDate(response.promulgationDate)
	if err != nil {
		return nil, err
	}
	items := make([]model.SourcedResource[model.LawRevision], len(response.revisions))
	for index, revision := range response.revisions {
		item, mapErr := mapLawRevision(
			response,
			revision,
			promulgationDate,
			source,
			legalSource,
			retrievedAt,
			endpointURL,
			index,
		)
		if mapErr != nil {
			return nil, mapErr
		}
		items[index] = item
	}
	return items, nil
}

func mapLawRevision(
	response lawRevisionResponse,
	revision lawRevisionRecord,
	promulgationDate *model.Date,
	source model.InformationSource,
	legalSource model.LegalSource,
	retrievedAt time.Time,
	endpointURL string,
	index int,
) (model.SourcedResource[model.LawRevision], error) {
	amendmentPromulgationDate, err := optionalLawRevisionDate(
		revision.amendmentPromulgationDate,
	)
	if err != nil {
		return invalidMappedLawRevision()
	}
	effectiveDate, err := optionalLawRevisionDate(revision.effectiveDate)
	if err != nil {
		return invalidMappedLawRevision()
	}
	scheduledEffectiveDate, err := optionalLawRevisionDate(
		revision.scheduledEffectiveDate,
	)
	if err != nil {
		return invalidMappedLawRevision()
	}
	repealRecordedDate, err := optionalLawRevisionDate(revision.repealRecordedDate)
	if err != nil {
		return invalidMappedLawRevision()
	}
	revisionKind, err := mapLawRevisionKind(revision)
	if err != nil {
		return invalidMappedLawRevision()
	}
	repealStatus, err := mapLawRevisionRepealStatus(revision.repealStatus)
	if err != nil {
		return invalidMappedLawRevision()
	}
	currentStatus, err := mapLawRevisionCurrentStatus(revision.currentStatus)
	if err != nil {
		return invalidMappedLawRevision()
	}
	data, err := model.NewLawRevision(model.LawRevisionValues{
		LawID:                     response.lawID,
		RevisionID:                revision.revisionID,
		Title:                     revision.title,
		LawType:                   revision.lawType,
		LawNumber:                 response.lawNumber,
		TitleKana:                 revision.titleKana,
		Abbreviation:              revision.abbreviation,
		Category:                  revision.category,
		PromulgationDate:          promulgationDate,
		SourceUpdatedAt:           revision.sourceUpdatedAt,
		AmendmentPromulgationDate: amendmentPromulgationDate,
		EffectiveDate:             effectiveDate,
		EffectiveDateNote:         revision.effectiveDateNote,
		ScheduledEffectiveDate:    scheduledEffectiveDate,
		AmendmentLawID:            revision.amendmentLawID,
		AmendmentLawTitle:         revision.amendmentLawTitle,
		AmendmentLawTitleKana:     revision.amendmentLawTitleKana,
		AmendmentLawNumber:        revision.amendmentLawNumber,
		RevisionKind:              revisionKind,
		RepealStatus:              repealStatus,
		RepealRecordedDate:        repealRecordedDate,
		RemainInForce:             revision.remainInForce,
		CurrentStatus:             currentStatus,
		Source:                    legalSource,
	})
	if err != nil {
		return invalidMappedLawRevision()
	}
	key, err := model.NewSourceResourceKey(model.SourceResourceKeyValues{
		SourceID:     providerID,
		ResourceType: lawRevisionResourceType,
		ResourceID:   response.lawID,
		VersionID:    revision.revisionID,
	})
	if err != nil {
		return invalidMappedLawRevision()
	}
	ref, err := model.NewSourceResourceRef(model.SourceResourceRefValues{
		ProviderID: providerID,
		Key:        key,
	})
	if err != nil {
		return invalidMappedLawRevision()
	}
	provenance, err := model.NewProvenance(model.ProvenanceValues{
		Source:         source,
		ResourceKey:    key,
		URL:            endpointURL,
		RetrievedAt:    retrievedAt,
		MediaType:      lawRevisionMediaType,
		Location:       fmt.Sprintf("revisions[%d]", index),
		Transformation: model.ProvenanceTransformationNormalized,
		MethodID:       lawRevisionMappingMethod,
	})
	if err != nil {
		return invalidMappedLawRevision()
	}
	resource, err := model.NewSourcedResource(
		model.SourcedResourceValues[model.LawRevision]{
			Ref:        ref,
			Provenance: []model.Provenance{provenance},
			Data:       data,
		},
	)
	if err != nil {
		return invalidMappedLawRevision()
	}
	return resource, nil
}

func optionalLawRevisionDate(value string) (*model.Date, error) {
	if value == "" {
		return nil, nil
	}
	date, err := model.NewDate(value)
	if err != nil {
		return nil, newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return &date, nil
}

func mapLawRevisionKind(revision lawRevisionRecord) (model.LawRevisionKind, error) {
	if err := validateLawRevisionEnums(revision); err != nil {
		return "", err
	}
	switch {
	case revision.amendmentType == "8" ||
		revision.repealStatus == "Repeal" ||
		revision.repealStatus == "Expire" ||
		revision.repealStatus == "LossOfEffectiveness":
		return model.LawRevisionKindRepeal, nil
	case revision.mission == "Partial":
		return model.LawRevisionKindPartialAmendment, nil
	case revision.amendmentType == "3" && revision.mission == "New":
		return model.LawRevisionKindAffectedLaw, nil
	case revision.amendmentType == "1" && revision.mission == "New":
		return model.LawRevisionKindEnactment, nil
	case revision.amendmentType == "" &&
		revision.mission == "" &&
		revision.repealStatus == "":
		return "", nil
	default:
		return "", newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func validateLawRevisionEnums(revision lawRevisionRecord) error {
	if !isLawRevisionEnum(revision.amendmentType, "1", "3", "8") ||
		!isLawRevisionEnum(revision.mission, "New", "Partial") ||
		!isLawRevisionEnum(
			revision.repealStatus,
			"None",
			"Repeal",
			"Expire",
			"Suspend",
			"LossOfEffectiveness",
		) ||
		!isLawRevisionEnum(
			revision.currentStatus,
			"CurrentEnforced",
			"UnEnforced",
			"PreviousEnforced",
			"Repeal",
		) {
		return newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
	return nil
}

func isLawRevisionEnum(value string, values ...string) bool {
	if value == "" {
		return true
	}
	for _, candidate := range values {
		if value == candidate {
			return true
		}
	}
	return false
}

func mapLawRevisionRepealStatus(
	value string,
) (model.LawRevisionRepealStatus, error) {
	switch value {
	case "":
		return "", nil
	case "None":
		return model.LawRevisionRepealStatusNone, nil
	case "Repeal":
		return model.LawRevisionRepealStatusRepealed, nil
	case "Expire":
		return model.LawRevisionRepealStatusExpired, nil
	case "Suspend":
		return model.LawRevisionRepealStatusSuspended, nil
	case "LossOfEffectiveness":
		return model.LawRevisionRepealStatusLossOfEffectiveness, nil
	default:
		return "", newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func mapLawRevisionCurrentStatus(
	value string,
) (model.LawRevisionCurrentStatus, error) {
	switch value {
	case "":
		return "", nil
	case "CurrentEnforced":
		return model.LawRevisionCurrentStatusCurrent, nil
	case "UnEnforced":
		return model.LawRevisionCurrentStatusFuture, nil
	case "PreviousEnforced":
		return model.LawRevisionCurrentStatusPrevious, nil
	case "Repeal":
		return model.LawRevisionCurrentStatusRepealed, nil
	default:
		return "", newLawRevisionSourceError(
			model.SourceErrorCodeInvalidSourceResponse,
			"",
		)
	}
}

func invalidMappedLawRevisions() ([]model.SourcedResource[model.LawRevision], error) {
	return nil, newLawRevisionSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}

func invalidMappedLawRevision() (model.SourcedResource[model.LawRevision], error) {
	return model.SourcedResource[model.LawRevision]{}, newLawRevisionSourceError(
		model.SourceErrorCodeInvalidSourceResponse,
		"",
	)
}
