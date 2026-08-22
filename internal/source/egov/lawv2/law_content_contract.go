package lawv2

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"errors"
	"io"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

//go:embed fixtures/law-content-contract.json
var recordedLawContentContractJSON string

type recordedLawContentContractField struct {
	Required     bool   `json:"required"`
	Type         string `json:"type"`
	Nullable     bool   `json:"nullable"`
	MinimumItems int    `json:"minimumItems"`
}

type recordedLawContentContract struct {
	CheckedOn             string                          `json:"checkedOn"`
	Operation             string                          `json:"operation"`
	SuccessMediaType      string                          `json:"successMediaType"`
	ResponseType          string                          `json:"responseType"`
	TotalCount            recordedLawContentContractField `json:"totalCount"`
	SentenceCount         recordedLawContentContractField `json:"sentenceCount"`
	NextOffset            recordedLawContentContractField `json:"nextOffset"`
	Items                 recordedLawContentContractField `json:"items"`
	LawInfo               recordedLawContentContractField `json:"lawInfo"`
	RevisionInfo          recordedLawContentContractField `json:"revisionInfo"`
	Sentences             recordedLawContentContractField `json:"sentences"`
	LawID                 recordedLawContentContractField `json:"lawId"`
	RevisionID            recordedLawContentContractField `json:"revisionId"`
	Title                 recordedLawContentContractField `json:"title"`
	Position              recordedLawContentContractField `json:"position"`
	Text                  recordedLawContentContractField `json:"text"`
	LawNumber             recordedLawContentContractField `json:"lawNumber"`
	PromulgationDate      recordedLawContentContractField `json:"promulgationDate"`
	RevisionEffectiveDate recordedLawContentContractField `json:"revisionEffectiveDate"`
}

func expectedLawContentContract() recordedLawContentContract {
	requiredInteger := recordedLawContentContractField{Required: true, Type: "integer"}
	requiredObject := recordedLawContentContractField{Required: true, Type: "object"}
	requiredString := recordedLawContentContractField{Required: true, Type: "string"}
	optionalString := recordedLawContentContractField{Type: "string", Nullable: true}
	return recordedLawContentContract{
		CheckedOn:        "2026-07-30",
		Operation:        "GET /keyword",
		SuccessMediaType: "application/json",
		ResponseType:     "object",
		TotalCount:       requiredInteger,
		SentenceCount:    requiredInteger,
		NextOffset: recordedLawContentContractField{
			Type:     "integer",
			Nullable: true,
		},
		Items: recordedLawContentContractField{
			Required: true,
			Type:     "array",
		},
		LawInfo:      requiredObject,
		RevisionInfo: requiredObject,
		Sentences: recordedLawContentContractField{
			Required:     true,
			Type:         "array",
			MinimumItems: 1,
		},
		LawID:                 requiredString,
		RevisionID:            requiredString,
		Title:                 requiredString,
		Position:              requiredString,
		Text:                  requiredString,
		LawNumber:             optionalString,
		PromulgationDate:      optionalString,
		RevisionEffectiveDate: optionalString,
	}
}

// verifyRecordedLawContentContract は、保存済み公式契約の変更検証を
// 個別 runtime 応答の判定から分離する。
func verifyRecordedLawContentContract(body []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(body))
	var actual recordedLawContentContract
	if err := decoder.Decode(&actual); err != nil {
		return newLawContentSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return newLawContentSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	if actual != expectedLawContentContract() {
		return newLawContentSourceError(model.SourceErrorCodeSourceContractChanged, "")
	}
	return nil
}

func verifyEmbeddedLawContentContract() error {
	return verifyRecordedLawContentContract([]byte(recordedLawContentContractJSON))
}
