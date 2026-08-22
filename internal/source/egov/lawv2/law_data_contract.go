package lawv2

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

//go:embed fixtures/law-data-contract.json
var recordedLawDataContractJSON []byte

type recordedLawDataContractField struct {
	Required bool   `json:"required"`
	Type     string `json:"type"`
}

type recordedLawDataContract struct {
	CheckedOn             string                       `json:"checkedOn"`
	Operation             string                       `json:"operation"`
	SuccessMediaType      string                       `json:"successMediaType"`
	RootElement           string                       `json:"rootElement"`
	LawInfo               recordedLawDataContractField `json:"lawInfo"`
	RevisionInfo          recordedLawDataContractField `json:"revisionInfo"`
	LawFullText           recordedLawDataContractField `json:"lawFullText"`
	Law                   recordedLawDataContractField `json:"law"`
	LawID                 recordedLawDataContractField `json:"lawId"`
	RevisionID            recordedLawDataContractField `json:"revisionId"`
	Title                 recordedLawDataContractField `json:"title"`
	LawNumber             recordedLawDataContractField `json:"lawNumber"`
	PromulgationDate      recordedLawDataContractField `json:"promulgationDate"`
	RevisionEffectiveDate recordedLawDataContractField `json:"revisionEffectiveDate"`
}

func expectedLawDataContract() recordedLawDataContract {
	requiredElement := recordedLawDataContractField{
		Required: true,
		Type:     "element",
	}
	requiredString := recordedLawDataContractField{
		Required: true,
		Type:     "string",
	}
	optionalString := recordedLawDataContractField{Type: "string"}
	return recordedLawDataContract{
		CheckedOn:             "2026-07-30",
		Operation:             "GET /law_data/{law_id_or_num_or_revision_id}",
		SuccessMediaType:      "application/xml",
		RootElement:           "law_data_response",
		LawInfo:               requiredElement,
		RevisionInfo:          requiredElement,
		LawFullText:           requiredElement,
		Law:                   requiredElement,
		LawID:                 requiredString,
		RevisionID:            requiredString,
		Title:                 requiredString,
		LawNumber:             optionalString,
		PromulgationDate:      optionalString,
		RevisionEffectiveDate: optionalString,
	}
}

// verifyRecordedLawDataContract は、保存済み公式契約の変更検証を
// 個別 runtime XML の判定から分離する。
func verifyRecordedLawDataContract(
	body []byte,
	sourceError sourceErrorFactory,
) error {
	if sourceError == nil {
		return fmt.Errorf("e-Gov law_data 契約のエラー生成関数は必須です")
	}
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	var actual recordedLawDataContract
	if err := decoder.Decode(&actual); err != nil {
		return sourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return sourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	if actual != expectedLawDataContract() {
		return sourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return nil
}

func verifyEmbeddedLawDataContract(sourceError sourceErrorFactory) error {
	return verifyRecordedLawDataContract(recordedLawDataContractJSON, sourceError)
}
