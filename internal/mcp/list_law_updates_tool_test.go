package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawupdatelist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawupdates"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListLawUpdatesToolReturnsAllPublicFields(t *testing.T) {
	t.Parallel()

	port := &recordingListLawUpdatesPort{
		result: mustListLawUpdatesResult(t),
	}
	result, err := callListLawUpdates(
		context.Background(),
		port,
		json.RawMessage(`{"date":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("SOT-IF-038: callListLawUpdates() のエラー = %v", err)
	}
	if result.IsError {
		t.Fatalf("SOT-IF-038: list_law_updates がエラーを返した: %#v", result)
	}
	if port.calls != 1 || port.request.Date().String() != "2026-07-26" {
		t.Fatalf(
			"SOT-IF-038: port 呼出し = %d, date = %q",
			port.calls,
			port.request.Date().String(),
		)
	}

	text := result.Content[0].(*sdk.TextContent).Text
	structured, ok := result.StructuredContent.(json.RawMessage)
	if !ok {
		t.Fatalf("SOT-IF-007: structuredContent の型 = %T", result.StructuredContent)
	}
	if text != string(structured) {
		t.Fatal("SOT-IF-007: content と structuredContent が一致しない")
	}

	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("SOT-IF-038: 成功結果を解析できない: %v", err)
	}
	if payload["date"] != "2026-07-26" || payload["totalCount"] != float64(2) {
		t.Fatalf("SOT-IF-038: 公開件数 = %#v", payload)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 2 {
		t.Fatalf("SOT-IF-038: items = %#v", payload["items"])
	}
	full := items[0].(map[string]any)
	wantFull := map[string]any{
		"updatedOn":                 "2026-07-26",
		"lawId":                     "law-001",
		"title":                     "行政手続法",
		"lawType":                   "法律",
		"lawNumber":                 "平成五年法律第八十八号",
		"titleKana":                 "ぎょうせいてつづきほう",
		"previousTitle":             "旧行政手続法",
		"promulgationDate":          "2024-01-01",
		"amendmentTitle":            "行政手続法の一部を改正する法律",
		"amendmentLawNumber":        "令和七年法律第一号",
		"amendmentPromulgationDate": "2025-12-01",
		"effectiveDate":             "2026-04-01",
		"effectiveDateNote":         "一部の規定を除く",
		"documentUrl":               "https://laws.e-gov.go.jp/law/law-001",
		"enforcementPending":        false,
		"authorityReviewPending":    true,
		"source": map[string]any{
			"id":         "e-gov-law-api-v1",
			"name":       "e-Gov 法令 API Version 1",
			"authority":  "official",
			"serviceUrl": "https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/",
		},
	}
	if !reflect.DeepEqual(full, wantFull) {
		t.Fatalf("SOT-IF-038: 全項目 = %#v, want %#v", full, wantFull)
	}
	minimal := items[1].(map[string]any)
	for _, key := range []string{
		"lawType",
		"lawNumber",
		"titleKana",
		"previousTitle",
		"promulgationDate",
		"amendmentTitle",
		"amendmentLawNumber",
		"amendmentPromulgationDate",
		"effectiveDate",
		"effectiveDateNote",
		"documentUrl",
		"enforcementPending",
		"authorityReviewPending",
		"ref",
		"provenance",
		"page",
	} {
		if _, exists := minimal[key]; exists {
			t.Fatalf("SOT-IF-038: 省略または非公開の %s が出力された: %#v", key, minimal)
		}
	}
}

func TestListLawUpdatesToolReturnsNonNilEmptyItems(t *testing.T) {
	t.Parallel()

	date := mustMCPListLawUpdatesDate(t, "2026-07-26")
	empty, err := listlawupdates.NewResult(listlawupdates.ResultValues{Date: date})
	if err != nil {
		t.Fatalf("試験用空結果を作成できない: %v", err)
	}
	result, err := callListLawUpdates(
		context.Background(),
		&recordingListLawUpdatesPort{result: empty},
		json.RawMessage(`{"date":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("callListLawUpdates() のエラー = %v", err)
	}
	if got := result.Content[0].(*sdk.TextContent).Text; got !=
		`{"date":"2026-07-26","totalCount":0,"items":[]}` {
		t.Fatalf("SOT-IF-038: 空結果 = %s", got)
	}
}

func TestListLawUpdatesToolRejectsInvalidInput(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"入力の欠落":       nil,
		"date の欠落":    json.RawMessage(`{}`),
		"date の null": json.RawMessage(`{"date":null}`),
		"date の型":     json.RawMessage(`{"date":20260726}`),
		"date の形式":    json.RawMessage(`{"date":"2026/07/26"}`),
		"存在しない日":      json.RawMessage(`{"date":"2026-02-30"}`),
		"未定義項目":       json.RawMessage(`{"date":"2026-07-26","other":true}`),
		"配列":          json.RawMessage(`[]`),
		"文字列":         json.RawMessage(`"2026-07-26"`),
		"null object": json.RawMessage(`null`),
	}
	for name, arguments := range tests {
		name, arguments := name, arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			port := &recordingListLawUpdatesPort{}
			result, err := callListLawUpdates(
				context.Background(),
				port,
				arguments,
			)
			if err != nil {
				t.Fatalf("callListLawUpdates() のエラー = %v", err)
			}
			assertListLawUpdatesErrorCode(t, result, "invalid_argument")
			if port.calls != 0 {
				t.Fatalf("不正入力で port を %d 回呼び出した", port.calls)
			}
		})
	}
}

func TestListLawUpdatesToolPreservesSourceErrorAndMapsUnknownError(t *testing.T) {
	t.Parallel()

	sourceFailure := &recordingListLawUpdatesPort{
		err: mustListLawUpdatesSourceError(
			t,
			model.SourceErrorCodeUnsupportedQuery,
		),
	}
	result, err := callListLawUpdates(
		context.Background(),
		sourceFailure,
		json.RawMessage(`{"date":"2020-11-23"}`),
	)
	if err != nil {
		t.Fatalf("callListLawUpdates() のエラー = %v", err)
	}
	assertListLawUpdatesErrorCode(t, result, "unsupported_query")
	if sourceFailure.calls != 1 ||
		sourceFailure.request.Date().String() != "2020-11-23" {
		t.Fatalf(
			"SOT-IF-034/038: provider 対象範囲を公開入力で拒否した: %#v",
			sourceFailure,
		)
	}

	result, err = callListLawUpdates(
		context.Background(),
		&recordingListLawUpdatesPort{
			err: fmt.Errorf(
				"provider page が不正です: %w",
				listlawupdates.ErrInvalidSourceResponse,
			),
		},
		json.RawMessage(`{"date":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("callListLawUpdates() のエラー = %v", err)
	}
	assertListLawUpdatesErrorCode(t, result, "invalid_source_response")

	result, err = callListLawUpdates(
		context.Background(),
		&recordingListLawUpdatesPort{err: errors.New("unknown")},
		json.RawMessage(`{"date":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("callListLawUpdates() のエラー = %v", err)
	}
	assertListLawUpdatesErrorCode(t, result, "internal_error")

	result, err = callListLawUpdates(
		context.Background(),
		nil,
		json.RawMessage(`{"date":"2026-07-26"}`),
	)
	if err != nil {
		t.Fatalf("callListLawUpdates() のエラー = %v", err)
	}
	assertListLawUpdatesErrorCode(t, result, "internal_error")
}

func assertListLawUpdatesErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) {
	t.Helper()

	if !result.IsError {
		t.Fatal("SOT-IF-007: IsError = false")
	}
	if result.StructuredContent != nil {
		t.Fatal("SOT-IF-007: エラー結果に structuredContent がある")
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(
		[]byte(result.Content[0].(*sdk.TextContent).Text),
		&payload,
	); err != nil {
		t.Fatalf("ErrorResult を解析できない: %v", err)
	}
	if payload.Code != want {
		t.Fatalf("code = %q, want %q", payload.Code, want)
	}
}

type recordingListLawUpdatesPort struct {
	result  listlawupdates.Result
	err     error
	request listlawupdates.Request
	calls   int
}

func (p *recordingListLawUpdatesPort) List(
	_ context.Context,
	request listlawupdates.Request,
) (listlawupdates.Result, error) {
	p.calls++
	p.request = request
	return p.result, p.err
}

func mustListLawUpdatesResult(t *testing.T) listlawupdates.Result {
	t.Helper()

	date := mustMCPListLawUpdatesDate(t, "2026-07-26")
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v1",
		Name:       "e-Gov 法令 API Version 1",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/",
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できない: %v", err)
	}
	source, err := model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("試験用 LegalSource を作成できない: %v", err)
	}
	promulgationDate := mustMCPListLawUpdatesDate(t, "2024-01-01")
	amendmentPromulgationDate := mustMCPListLawUpdatesDate(t, "2025-12-01")
	effectiveDate := mustMCPListLawUpdatesDate(t, "2026-04-01")
	enforcementPending := false
	authorityReviewPending := true
	full, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn:                 date,
		LawID:                     "law-001",
		Title:                     "行政手続法",
		LawType:                   "法律",
		LawNumber:                 "平成五年法律第八十八号",
		TitleKana:                 "ぎょうせいてつづきほう",
		PreviousTitle:             "旧行政手続法",
		PromulgationDate:          &promulgationDate,
		AmendmentTitle:            "行政手続法の一部を改正する法律",
		AmendmentLawNumber:        "令和七年法律第一号",
		AmendmentPromulgationDate: &amendmentPromulgationDate,
		EffectiveDate:             &effectiveDate,
		EffectiveDateNote:         "一部の規定を除く",
		DocumentURL:               "https://laws.e-gov.go.jp/law/law-001",
		EnforcementPending:        &enforcementPending,
		AuthorityReviewPending:    &authorityReviewPending,
		Source:                    source,
	})
	if err != nil {
		t.Fatalf("全項目の LawUpdate を作成できない: %v", err)
	}
	minimal, err := model.NewLawUpdate(model.LawUpdateValues{
		UpdatedOn: date,
		LawID:     "law-002",
		Title:     "試験法",
		Source:    source,
	})
	if err != nil {
		t.Fatalf("最小 LawUpdate を作成できない: %v", err)
	}
	result, err := listlawupdates.NewResult(listlawupdates.ResultValues{
		Date:       date,
		TotalCount: 2,
		Items:      []model.LawUpdate{full, minimal},
	})
	if err != nil {
		t.Fatalf("試験用 Result を作成できない: %v", err)
	}
	return result
}

func mustMCPListLawUpdatesDate(t *testing.T, value string) model.Date {
	t.Helper()

	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("試験用日付を作成できない: %v", err)
	}
	return date
}

func mustListLawUpdatesSourceError(
	t *testing.T,
	code model.SourceErrorCode,
) error {
	t.Helper()

	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID:         "e-gov-law-api-v1",
		Name:       "e-Gov 法令 API Version 1",
		Publisher:  "デジタル庁",
		Authority:  model.AuthorityOfficial,
		ServiceURL: "https://laws.e-gov.go.jp/docs/law-data-basic/8529371-law-api-v1/",
	})
	if err != nil {
		t.Fatalf("試験用 InformationSource を作成できない: %v", err)
	}
	verifiedAt := mustMCPListLawUpdatesDate(t, "2026-07-26")
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID:           lawupdatelist.CapabilityID,
		MajorVersion: lawupdatelist.MajorVersion,
		Level:        model.CapabilityLevelExtended,
		Stability:    model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("試験用 ProviderCapability を作成できない: %v", err)
	}
	provider, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID:             "e-gov-law-api-v1",
		Source:                 informationSource,
		AdapterContractVersion: "1.0.0",
		UpstreamSpecVersion:    "1",
		VerifiedAt:             verifiedAt,
		InterfaceType:          model.InterfaceTypeAPI,
		Capabilities:           []model.ProviderCapability{capability},
	})
	if err != nil {
		t.Fatalf("試験用 ProviderDescriptor を作成できない: %v", err)
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code:       code,
		Provider:   provider,
		Capability: capability,
		Operation:  listLawUpdatesTestOperation("GET /api/1/updatelawlists/{yyyyMMdd}"),
	})
	if err != nil {
		t.Fatalf("試験用 SourceError を作成できない: %v", err)
	}
	return sourceError
}

type listLawUpdatesTestOperation string

func (listLawUpdatesTestOperation) SourceOperationProviderID() string {
	return "e-gov-law-api-v1"
}

func (o listLawUpdatesTestOperation) SourceOperationName() string {
	return string(o)
}

func (o listLawUpdatesTestOperation) ValidateSourceOperation() error {
	if o != "GET /api/1/updatelawlists/{yyyyMMdd}" {
		return errors.New("試験 operation が定義されていません")
	}
	return nil
}
