package legalquerycorpus

import "fmt"

type rawRequestKeyV1DTO struct {
	SourceID     *string `json:"sourceId"`
	ResourceType *string `json:"resourceType"`
	ResourceID   *string `json:"resourceId"`
	VersionID    *string `json:"versionId,omitempty"`
}

type rawRequestRefV1DTO struct {
	ProviderID *string             `json:"providerId"`
	Key        *rawRequestKeyV1DTO `json:"key"`
}

type rawRequestV1DTO struct {
	Query           *string             `json:"query"`
	Ref             *rawRequestRefV1DTO `json:"ref,omitempty"`
	LimitPerAttempt *int                `json:"limitPerAttempt,omitempty"`
}

func convertRawRequestV1(dto rawRequestV1DTO) (Request, error) {
	if dto.Query == nil {
		return Request{}, fmt.Errorf("raw request の query は必須です")
	}
	var ref *RequestRef
	if dto.Ref != nil {
		if err := validateRawRequestRefV1DTO(*dto.Ref); err != nil {
			return Request{}, err
		}
		key, err := NewRequestKey(RequestKeyValues{
			SourceID:     *dto.Ref.Key.SourceID,
			ResourceType: *dto.Ref.Key.ResourceType,
			ResourceID:   *dto.Ref.Key.ResourceID,
			VersionID:    dto.Ref.Key.VersionID,
		})
		if err != nil {
			return Request{}, err
		}
		converted, err := NewRequestRef(RequestRefValues{
			ProviderID: *dto.Ref.ProviderID,
			Key:        key,
		})
		if err != nil {
			return Request{}, err
		}
		ref = &converted
	}
	return NewRequest(RequestValues{
		Query:           *dto.Query,
		Ref:             ref,
		LimitPerAttempt: dto.LimitPerAttempt,
	})
}

func validateRawRequestRefV1DTO(dto rawRequestRefV1DTO) error {
	if dto.ProviderID == nil || dto.Key == nil {
		return fmt.Errorf("raw request の ref に必須項目がありません")
	}
	if dto.Key.SourceID == nil ||
		dto.Key.ResourceType == nil ||
		dto.Key.ResourceID == nil {
		return fmt.Errorf("raw request の ref.key に必須項目がありません")
	}
	return nil
}
