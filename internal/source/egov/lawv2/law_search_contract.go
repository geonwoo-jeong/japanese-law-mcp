package lawv2

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

//go:embed fixtures/law-search-contract.json
var recordedLawSearchContractJSON []byte

type recordedLawSearchContractField struct {
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type recordedLawSearchContract struct {
	CheckedOn        string                         `json:"checkedOn"`
	Operation        string                         `json:"operation"`
	SuccessMediaType string                         `json:"successMediaType"`
	ResponseType     string                         `json:"responseType"`
	TotalCount       recordedLawSearchContractField `json:"totalCount"`
	Count            recordedLawSearchContractField `json:"count"`
	NextOffset       recordedLawSearchContractField `json:"nextOffset"`
	Laws             recordedLawSearchContractField `json:"laws"`
	LawInfo          recordedLawSearchContractField `json:"lawInfo"`
	RevisionInfo     recordedLawSearchContractField `json:"revisionInfo"`
	LawID            recordedLawSearchContractField `json:"lawId"`
	RevisionID       recordedLawSearchContractField `json:"revisionId"`
	Title            recordedLawSearchContractField `json:"title"`
}

func expectedLawSearchContract() recordedLawSearchContract {
	return recordedLawSearchContract{
		CheckedOn:        "2026-07-30",
		Operation:        "GET /laws",
		SuccessMediaType: "application/json",
		ResponseType:     "object",
		TotalCount:       recordedLawSearchContractField{Required: true, Type: "integer"},
		Count:            recordedLawSearchContractField{Required: true, Type: "integer"},
		NextOffset: recordedLawSearchContractField{
			Type:     "integer",
			Nullable: true,
		},
		Laws: recordedLawSearchContractField{
			Required: true,
			Type:     "array",
		},
		LawInfo: recordedLawSearchContractField{
			Required: true,
			Type:     "object",
		},
		RevisionInfo: recordedLawSearchContractField{
			Required: true,
			Type:     "object",
		},
		LawID: recordedLawSearchContractField{
			Required: true,
			Type:     "string",
		},
		RevisionID: recordedLawSearchContractField{
			Required: true,
			Type:     "string",
		},
		Title: recordedLawSearchContractField{
			Required: true,
			Type:     "string",
		},
	}
}

// verifyRecordedLawSearchContract は、保存済み公式契約の意図的な変更だけを
// runtime 応答の不正とは別の情報源エラーへ分類する。
func verifyRecordedLawSearchContract(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var actual recordedLawSearchContract
	if err := decoder.Decode(&actual); err != nil {
		return newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	if actual != expectedLawSearchContract() {
		return newSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return nil
}

func verifyEmbeddedLawSearchContract() error {
	return verifyRecordedLawSearchContract(recordedLawSearchContractJSON)
}
