package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/lawrevisionlist"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/application/listlawrevisions"
	"github.com/geonwoo-jeong/japanese-law-mcp/internal/model"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListLawRevisionsToolReturnsNormalizedHistory(t *testing.T) {
	t.Parallel()

	remainInForce := false
	revision := mustMCPListLawRevision(t, model.LawRevisionValues{
		LawID: "law-1", RevisionID: "revision-2", Title: "試験法",
		RevisionKind:  model.LawRevisionKindPartialAmendment,
		RemainInForce: &remainInForce,
		CurrentStatus: model.LawRevisionCurrentStatusCurrent,
	})
	result := mustMCPListLawRevisionsResult(t, "law-1", revision)
	port := &recordingListLawRevisionsPort{result: result}
	callResult, err := callListLawRevisions(
		context.Background(),
		port,
		json.RawMessage(`{"lawIdOrNumber":" law-1 "}`),
	)
	if err != nil {
		t.Fatalf("callListLawRevisions() のエラー = %v", err)
	}
	if callResult.IsError || port.calls != 1 || port.request.LawIDOrNumber() != "law-1" {
		t.Fatalf("MCP 結果又は呼出し = %#v / %#v", callResult, port)
	}
	content := callResult.Content[0].(*sdk.TextContent).Text
	var payload map[string]any
	if err := json.Unmarshal([]byte(content), &payload); err != nil {
		t.Fatalf("公開 JSON を解析できません: %v", err)
	}
	items, ok := payload["items"].([]any)
	if !ok || len(items) != 1 || payload["lawId"] != "law-1" || payload["totalCount"] != float64(1) {
		t.Fatalf("公開 payload = %#v", payload)
	}
	item := items[0].(map[string]any)
	if item["revisionKind"] != "partial_amendment" ||
		item["remainInForce"] != false || item["currentStatus"] != "current" {
		t.Fatalf("正規化済み履歴 = %#v", item)
	}
	for _, hidden := range []string{"ref", "provenance", "page", "amendmentType", "mission"} {
		if _, exists := item[hidden]; exists {
			t.Fatalf("内部又は provider 固有項目 %q が公開されました: %#v", hidden, item)
		}
	}
	structured, ok := callResult.StructuredContent.(json.RawMessage)
	if !ok || string(structured) != content {
		t.Fatalf("structuredContent と text が一致しません: %#v", callResult.StructuredContent)
	}
}

func TestListLawRevisionsToolProjectsLawRevisionJSONWithoutLoss(t *testing.T) {
	t.Parallel()

	remainInForce := false
	revision := mustMCPListLawRevision(t, model.LawRevisionValues{
		LawID:                     "law-1",
		RevisionID:                "revision-2",
		Title:                     "試験法",
		LawType:                   "Act",
		LawNumber:                 "令和六年法律第四十六号",
		TitleKana:                 "しけんほう",
		Abbreviation:              "試験法",
		Category:                  "行政組織",
		PromulgationDate:          mustMCPDatePointer(t, "2024-06-07"),
		SourceUpdatedAt:           "2024-06-07T09:30:00+09:00",
		AmendmentPromulgationDate: mustMCPDatePointer(t, "2024-06-07"),
		EffectiveDate:             mustMCPDatePointer(t, "2024-12-01"),
		EffectiveDateNote:         "一部の規定を除く",
		ScheduledEffectiveDate:    mustMCPDatePointer(t, "2025-04-01"),
		AmendmentLawID:            "506AC0000000046",
		AmendmentLawTitle:         "改正法",
		AmendmentLawTitleKana:     "かいせいほう",
		AmendmentLawNumber:        "令和六年法律第四十六号",
		RevisionKind:              model.LawRevisionKindPartialAmendment,
		RepealStatus:              model.LawRevisionRepealStatusExpired,
		RepealRecordedDate:        mustMCPDatePointer(t, "2026-01-01"),
		RemainInForce:             &remainInForce,
		CurrentStatus:             model.LawRevisionCurrentStatusPrevious,
	})

	expectedPayload, err := json.Marshal(revision)
	if err != nil {
		t.Fatalf("LawRevision を JSON 化できません: %v", err)
	}
	var expected map[string]any
	if err := json.Unmarshal(expectedPayload, &expected); err != nil {
		t.Fatalf("LawRevision JSON を解析できません: %v", err)
	}

	actualPayload, err := json.Marshal(mapListLawRevisionOutputItem(revision))
	if err != nil {
		t.Fatalf("公開 item を JSON 化できません: %v", err)
	}
	var actual map[string]any
	if err := json.Unmarshal(actualPayload, &actual); err != nil {
		t.Fatalf("公開 item JSON を解析できません: %v", err)
	}
	if !reflect.DeepEqual(actual, expected) {
		t.Fatalf("公開 item JSON = %#v, want %#v", actual, expected)
	}
}

func TestListLawRevisionsToolRejectsInvalidInputWithoutCallingPort(t *testing.T) {
	t.Parallel()

	tests := map[string]json.RawMessage{
		"欠落":       nil,
		"object 空": json.RawMessage(`{}`),
		"null":     json.RawMessage(`{"lawIdOrNumber":null}`),
		"型":        json.RawMessage(`{"lawIdOrNumber":1}`),
		"空":        json.RawMessage(`{"lawIdOrNumber":" "}`),
		"未知項目":     json.RawMessage(`{"lawIdOrNumber":"law-1","other":true}`),
		"配列":       json.RawMessage(`[]`),
	}
	for name, arguments := range tests {
		name, arguments := name, arguments
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			port := &recordingListLawRevisionsPort{}
			result, err := callListLawRevisions(context.Background(), port, arguments)
			if err != nil {
				t.Fatalf("callListLawRevisions() のエラー = %v", err)
			}
			assertListLawRevisionsErrorCode(t, result, "invalid_argument")
			if port.calls != 0 {
				t.Fatalf("不正入力で port を %d 回呼び出しました", port.calls)
			}
		})
	}
}

func TestListLawRevisionsToolReturnsNonNilEmptyItemsAndPreservesErrors(t *testing.T) {
	t.Parallel()

	empty := mustMCPListLawRevisionsResult(t, "law-1")
	result, err := callListLawRevisions(
		context.Background(),
		&recordingListLawRevisionsPort{result: empty},
		json.RawMessage(`{"lawIdOrNumber":"law-1"}`),
	)
	if err != nil {
		t.Fatalf("空結果のエラー = %v", err)
	}
	if got := result.Content[0].(*sdk.TextContent).Text; got !=
		`{"lawId":"law-1","totalCount":0,"items":[]}` {
		t.Fatalf("空結果 = %s", got)
	}

	result, err = callListLawRevisions(
		context.Background(),
		&recordingListLawRevisionsPort{err: lawrevisionlist.ErrNotFound},
		json.RawMessage(`{"lawIdOrNumber":"missing"}`),
	)
	if err != nil {
		t.Fatalf("情報源エラーの処理に失敗しました: %v", err)
	}
	assertListLawRevisionsErrorCode(t, result, "not_found")

	sourceError := mustMCPListLawRevisionsSourceError(
		t,
		model.SourceErrorCodeSourceUnavailable,
	)
	result, err = callListLawRevisions(
		context.Background(),
		&recordingListLawRevisionsPort{err: sourceError},
		json.RawMessage(`{"lawIdOrNumber":"law-1"}`),
	)
	if err != nil {
		t.Fatalf("情報源エラーの処理に失敗しました: %v", err)
	}
	assertListLawRevisionsErrorCode(t, result, "source_unavailable")

	result, err = callListLawRevisions(
		context.Background(),
		&recordingListLawRevisionsPort{err: errors.New("unknown")},
		json.RawMessage(`{"lawIdOrNumber":"law-1"}`),
	)
	if err != nil {
		t.Fatalf("未知エラーの処理に失敗しました: %v", err)
	}
	assertListLawRevisionsErrorCode(t, result, "internal_error")
}

func TestListLawRevisionsToolFailsClosedForNilPortVariants(t *testing.T) {
	t.Parallel()

	var typedNil *recordingListLawRevisionsPort
	for name, port := range map[string]listlawrevisions.Port{
		"nil":       nil,
		"typed nil": typedNil,
	} {
		name, port := name, port
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			result, err := callListLawRevisions(
				context.Background(),
				port,
				json.RawMessage(`{"lawIdOrNumber":"law-1"}`),
			)
			if err != nil {
				t.Fatalf("callListLawRevisions() のエラー = %v", err)
			}
			assertListLawRevisionsErrorCode(t, result, "internal_error")
		})
	}
}

type recordingListLawRevisionsPort struct {
	result  listlawrevisions.Result
	err     error
	calls   int
	request listlawrevisions.Request
}

func (p *recordingListLawRevisionsPort) List(
	_ context.Context,
	request listlawrevisions.Request,
) (listlawrevisions.Result, error) {
	p.calls++
	p.request = request
	return p.result, p.err
}

func mustMCPListLawRevision(
	t *testing.T,
	values model.LawRevisionValues,
) model.LawRevision {
	t.Helper()
	informationSource, err := model.NewInformationSource(model.InformationSourceValues{
		ID: "test-source", Name: "試験用情報源", Publisher: "試験機関",
		Authority: model.AuthorityOfficial, ServiceURL: "https://example.test/service",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	values.Source, err = model.NewLegalSource(informationSource)
	if err != nil {
		t.Fatalf("LegalSource を作成できません: %v", err)
	}
	revision, err := model.NewLawRevision(values)
	if err != nil {
		t.Fatalf("LawRevision を作成できません: %v", err)
	}
	return revision
}

func mustMCPListLawRevisionsResult(
	t *testing.T,
	lawID string,
	items ...model.LawRevision,
) listlawrevisions.Result {
	t.Helper()
	result, err := listlawrevisions.NewResult(listlawrevisions.ResultValues{
		LawID: lawID, TotalCount: len(items), Items: items,
	})
	if err != nil {
		t.Fatalf("Result を作成できません: %v", err)
	}
	return result
}

func mustMCPListLawRevisionsSourceError(
	t *testing.T,
	code model.SourceErrorCode,
) model.SourceError {
	t.Helper()
	source, err := model.NewInformationSource(model.InformationSourceValues{
		ID: "test-source", Name: "試験用情報源", Publisher: "試験機関",
		Authority: model.AuthorityOfficial, ServiceURL: "https://example.test/service",
	})
	if err != nil {
		t.Fatalf("InformationSource を作成できません: %v", err)
	}
	capability, err := model.NewProviderCapability(model.ProviderCapabilityValues{
		ID: "law.revision.list", MajorVersion: 1,
		Level: model.CapabilityLevelCore, Stability: model.CapabilityStabilityStable,
	})
	if err != nil {
		t.Fatalf("ProviderCapability を作成できません: %v", err)
	}
	verifiedAt, err := model.NewDate("2026-08-23")
	if err != nil {
		t.Fatalf("verifiedAt を作成できません: %v", err)
	}
	descriptor, err := model.NewProviderDescriptor(model.ProviderDescriptorValues{
		ProviderID: "test-provider", Source: source,
		AdapterContractVersion: "1.0.0", UpstreamSpecVersion: "test",
		VerifiedAt: verifiedAt, InterfaceType: model.InterfaceTypeAPI,
		CredentialRequired: false, Capabilities: []model.ProviderCapability{capability},
	})
	if err != nil {
		t.Fatalf("ProviderDescriptor を作成できません: %v", err)
	}
	sourceError, err := model.NewSourceError(model.SourceErrorValues{
		Code: code, Provider: descriptor, Capability: capability,
		Operation: listLawRevisionsTestOperation("GET /law_revisions/{law_id_or_num}"),
	})
	if err != nil {
		t.Fatalf("SourceError を作成できません: %v", err)
	}
	return sourceError
}

func mustMCPDatePointer(t *testing.T, value string) *model.Date {
	t.Helper()
	date, err := model.NewDate(value)
	if err != nil {
		t.Fatalf("Date を作成できません: %v", err)
	}
	return &date
}

type listLawRevisionsTestOperation string

func (listLawRevisionsTestOperation) SourceOperationProviderID() string {
	return "test-provider"
}

func (o listLawRevisionsTestOperation) SourceOperationName() string { return string(o) }

func (o listLawRevisionsTestOperation) ValidateSourceOperation() error {
	if o != "GET /law_revisions/{law_id_or_num}" {
		return errors.New("試験 operation が定義されていません")
	}
	return nil
}

func assertListLawRevisionsErrorCode(
	t *testing.T,
	result *sdk.CallToolResult,
	want string,
) {
	t.Helper()
	if !result.IsError || result.StructuredContent != nil {
		t.Fatalf("エラー結果 = %#v", result)
	}
	var payload struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal([]byte(result.Content[0].(*sdk.TextContent).Text), &payload); err != nil {
		t.Fatalf("ErrorResult を解析できません: %v", err)
	}
	if payload.Code != want {
		t.Fatalf("code = %q, want %q", payload.Code, want)
	}
}
