package lawv2

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
)

func TestParseAndMapLawContentKeepsEverySentenceLocation(t *testing.T) {
	t.Parallel()

	response, nextOffset, err := parseLawContentSearchResponse(
		context.Background(),
		fixture(t, "fixtures/law-content-normal.json"),
		2,
		0,
	)
	if err != nil {
		t.Fatalf("SOT-IF-010/028: parser のエラー = %v", err)
	}
	if response.totalCount != 3 ||
		response.sentenceCount != 2 ||
		nextOffset == nil ||
		*nextOffset != 2 {
		t.Fatalf("SOT-IF-010/028: response = %#v、nextOffset = %v", response, nextOffset)
	}
	retrievedAt := time.Date(2026, 7, 26, 12, 0, 0, 0, time.UTC)
	items, err := mapLawContentItems(response, retrievedAt)
	if err != nil {
		t.Fatalf("SOT-IF-028: mapper のエラー = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("SOT-IF-023/028: items = %d", len(items))
	}
	if items[0].Ref() != items[1].Ref() {
		t.Fatal("SOT-IF-023/028: 同じ法令リビジョンで ref が異なる")
	}
	if items[0].Data().Location() == items[1].Data().Location() {
		t.Fatal("SOT-IF-023/028: 異なる一致位置が失われた")
	}
	wantText := []string{
		"地方自治に関する第一の一致。",
		"<em>地方</em>自治に関する第二の一致。",
	}
	for index, item := range items {
		if item.Data().Text() != wantText[index] {
			t.Errorf(
				"SOT-IF-028: items[%d].text = %q、期待値 = %q",
				index,
				item.Data().Text(),
				wantText[index],
			)
		}
		assertLawContentLocations(t, item, retrievedAt)
	}
}

func TestParseLawContentAcceptsEmptyResult(t *testing.T) {
	t.Parallel()

	response, nextOffset, err := parseLawContentSearchResponse(
		context.Background(),
		fixture(t, "fixtures/law-content-empty.json"),
		20,
		0,
	)
	if err != nil {
		t.Fatalf("SOT-IF-023/028: 空結果のエラー = %v", err)
	}
	if response.totalCount != 0 ||
		response.sentenceCount != 0 ||
		len(response.items) != 0 ||
		nextOffset != nil {
		t.Fatalf("SOT-IF-023/028: 空結果 = %#v、nextOffset = %v", response, nextOffset)
	}
}

func TestParseLawContentRejectsPageInvariantViolations(t *testing.T) {
	t.Parallel()

	const item = `{
		"law_info":{"law_id":"322CO0000000016"},
		"revision_info":{
			"law_revision_id":"322CO0000000016_20240401_506CO0000000161",
			"law_title":"地方自治法施行令"
		},
		"sentences":[{"position":"MainProvision/Article[1]","text":"本文"}]
	}`
	tests := []struct {
		name   string
		body   string
		limit  int
		offset int
	}{
		{
			name:  "sentence_count と展開件数が異なる",
			body:  fmt.Sprintf(`{"total_count":2,"sentence_count":2,"next_offset":null,"items":[%s]}`, item),
			limit: 2,
		},
		{
			name:  "next_offset が offset と count の和ではない",
			body:  fmt.Sprintf(`{"total_count":3,"sentence_count":1,"next_offset":2,"items":[%s]}`, item),
			limit: 1,
		},
		{
			name:  "残件があるのに next_offset が null",
			body:  fmt.Sprintf(`{"total_count":2,"sentence_count":1,"next_offset":null,"items":[%s]}`, item),
			limit: 1,
		},
		{
			name:   "next_offset が int32 上限を超える",
			body:   fmt.Sprintf(`{"total_count":2147483648,"sentence_count":1,"next_offset":2147483648,"items":[%s]}`, item),
			limit:  1,
			offset: 2147483647,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			_, _, err := parseLawContentSearchResponse(
				context.Background(),
				[]byte(test.body),
				test.limit,
				test.offset,
			)
			assertLawContentSourceError(
				t,
				err,
				model.SourceErrorCodeInvalidSourceResponse,
			)
		})
	}
}

func TestParseLawContentUsesContentCapabilityForStructuralError(t *testing.T) {
	t.Parallel()

	_, _, err := parseLawContentSearchResponse(
		context.Background(),
		fixture(t, "fixtures/law-content-structural-error.json"),
		1,
		0,
	)
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SOT-IF-017/028: error = %T %v", err, err)
	}
	if sourceError.Code() != model.SourceErrorCodeSourceContractChanged ||
		sourceError.ProviderID() != providerID ||
		sourceError.SourceID() != providerID ||
		sourceError.CapabilityID() != "law.content.search" ||
		sourceError.Operation() != "GET /keyword" {
		t.Fatalf("SOT-IF-017/028: SourceError = %#v", sourceError)
	}
}

func assertLawContentLocations(
	t *testing.T,
	item model.SourcedResource[model.LawContentMatch],
	retrievedAt time.Time,
) {
	t.Helper()

	ref := item.Ref()
	versionID, hasVersionID := ref.Key().VersionID()
	if ref.ProviderID() != providerID ||
		ref.Key().SourceID() != providerID ||
		ref.Key().ResourceType() != "law" ||
		ref.Key().ResourceID() != item.Data().Law().LawID() ||
		!hasVersionID ||
		versionID != item.Data().Law().RevisionID() {
		t.Fatalf("SOT-IF-023/028: ref = %#v", ref)
	}
	location := item.Data().Location()
	citationLocation, exists := item.Data().Citation().Location()
	if !exists ||
		citationLocation != location ||
		item.Data().Citation().URL() !=
			"https://laws.e-gov.go.jp/law/322CO0000000016/20240401_506CO0000000161" {
		t.Fatalf("SOT-IF-023/028: citation = %#v", item.Data().Citation())
	}
	provenance := item.Provenance()
	if len(provenance) != 1 {
		t.Fatalf("SOT-IF-015/028: provenance = %#v", provenance)
	}
	provenanceLocation, exists := provenance[0].Location()
	methodID, hasMethodID := provenance[0].MethodID()
	if !exists ||
		provenanceLocation != location ||
		provenance[0].Transformation() != model.ProvenanceTransformationExtracted ||
		!hasMethodID ||
		methodID != "SOT-IF-028" ||
		!provenance[0].RetrievedAt().Equal(retrievedAt) {
		t.Fatalf("SOT-IF-015/028: provenance = %#v", provenance[0])
	}
}

func assertLawContentSourceError(
	t *testing.T,
	err error,
	want model.SourceErrorCode,
) {
	t.Helper()
	if err == nil {
		t.Fatalf("SOT-IF-017/028: error = nil、期待値 = %q", want)
	}
	var sourceError model.SourceError
	if !errors.As(err, &sourceError) {
		t.Fatalf("SOT-IF-017/028: error = %T、期待値 = model.SourceError", err)
	}
	if sourceError.Code() != want {
		t.Fatalf("SOT-IF-017/028: code = %q、期待値 = %q", sourceError.Code(), want)
	}
}
