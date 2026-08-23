package lawv2

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"
	"reflect"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

//go:embed fixtures/law-revisions-contract.json
var recordedLawRevisionContractJSON string

type recordedLawRevisionContractField struct {
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type recordedLawRevisionContract struct {
	CheckedOn        string                           `json:"checkedOn"`
	Operation        string                           `json:"operation"`
	SuccessMediaType string                           `json:"successMediaType"`
	ResponseType     string                           `json:"responseType"`
	LawInfo          recordedLawRevisionContractField `json:"lawInfo"`
	Revisions        recordedLawRevisionContractField `json:"revisions"`
	LawID            recordedLawRevisionContractField `json:"lawId"`
	RevisionID       recordedLawRevisionContractField `json:"revisionId"`
	Title            recordedLawRevisionContractField `json:"title"`
	RemainInForce    recordedLawRevisionContractField `json:"remainInForce"`
	AmendmentTypes   []string                         `json:"amendmentTypes"`
	Missions         []string                         `json:"missions"`
	RepealStatuses   []string                         `json:"repealStatuses"`
	CurrentStatuses  []string                         `json:"currentStatuses"`
}

func expectedLawRevisionContract() recordedLawRevisionContract {
	requiredObject := recordedLawRevisionContractField{
		Required: true,
		Type:     "object",
	}
	requiredArray := recordedLawRevisionContractField{
		Required: true,
		Type:     "array",
	}
	optionalString := recordedLawRevisionContractField{Type: "string"}
	return recordedLawRevisionContract{
		CheckedOn:        "2026-08-23",
		Operation:        "GET /law_revisions/{law_id_or_num}",
		SuccessMediaType: "application/json",
		ResponseType:     "object",
		LawInfo:          requiredObject,
		Revisions:        requiredArray,
		LawID:            optionalString,
		RevisionID:       optionalString,
		Title:            optionalString,
		RemainInForce: recordedLawRevisionContractField{
			Type:     "boolean",
			Nullable: true,
		},
		AmendmentTypes: []string{"1", "3", "8"},
		Missions:       []string{"New", "Partial"},
		RepealStatuses: []string{
			"None",
			"Repeal",
			"Expire",
			"Suspend",
			"LossOfEffectiveness",
		},
		CurrentStatuses: []string{
			"CurrentEnforced",
			"UnEnforced",
			"PreviousEnforced",
			"Repeal",
		},
	}
}

func verifyRecordedLawRevisionContract(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var actual recordedLawRevisionContract
	if err := decoder.Decode(&actual); err != nil {
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	if !reflect.DeepEqual(actual, expectedLawRevisionContract()) {
		return newLawRevisionSourceError(
			model.SourceErrorCodeSourceContractChanged,
			"",
		)
	}
	return nil
}

func verifyEmbeddedLawRevisionContract() error {
	return verifyRecordedLawRevisionContract([]byte(recordedLawRevisionContractJSON))
}
